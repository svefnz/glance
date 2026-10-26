# Design

## Context

See proposal.md - Why.

The monitor widget currently keeps only the latest result per site. `update()` calls `fetchStatusForSites`, which fans out one HTTP request per site with 20 workers, and writes the outcome into `siteStatus` fields that are overwritten on the next cycle. Nothing survives the update, and nothing survives a restart.

Two properties of the existing code shape the approach:

- The monitor widget's cache is configurable per widget, and the deployed configuration sets it to `cache: 1m`. Recording granularity therefore has to be coarser than the check interval or the record count grows with the check rate.
- The image is built with `CGO_ENABLED=0` in a scratch-like Alpine layer holding a single static binary. Any storage driver requiring cgo cannot be used.

The deployed instance monitors 26 sites across three hosts and already has several chronically failing entries, so the feature has real data to show from the first minute.

## Goals / Non-Goals

**Goals:**

- History that survives restart and is queryable over an arbitrary window from 1 hour to 90 days.
- A bucket count independent of the window, so the visual density does not change with configuration.
- Recording that costs one small write per site per minute, not one per check.
- A status-page presentation: aggregate banner, one row per service, availability bar.

**Non-Goals:**

- Incident history with titles, causes, and resolutions. Only availability is recorded.
- Per-check drill-down. Aggregates deliberately discard which specific check failed.
- Alerting or notification on state change.
- Multi-process or multi-instance support. One process owns the database file.
- Historical data migration. History begins when the feature is deployed.

## Decisions

### Store per-minute aggregates, not raw checks

A row per check would be roughly 130k rows per site over 90 days at a one-minute interval, and far more if a widget's cache is lowered. Aggregating to one row per site per minute bounds the table at 1440 rows per site per day regardless of check frequency.

The write is an upsert that adds to the current minute's counters. Because checks are frequent and the minute boundary is approximate, an insert-or-replace with accumulated values is simpler than reading then writing, and loses at most the in-flight minute on an unclean shutdown.

Alternative considered: store raw checks and aggregate at query time. Rejected — it makes the database the largest file on the host and buys a drill-down the specs explicitly exclude.

Alternative considered: store only the final bucket values. Rejected — changing `range` would invalidate all stored history, and the spec requires the range to be configurable at any time.

### Use a pure-Go SQLite driver

`modernc.org/sqlite` is a transpiled SQLite that needs no cgo. The alternative, `mattn/go-sqlite3`, requires cgo and would break the `CGO_ENABLED=0` build the Dockerfile relies on.

Cost: roughly 10-15 MB of image growth. Accepted because the alternative is a hand-rolled JSON store, which would need its own locking, compaction, and corruption handling — more code than the dependency is worth.

Alternative considered: JSON files per site with a bounded ring of buckets. Rejected because it forces either coarse buckets (range changes lose history) or unbounded files, and because concurrent writes from the 20-worker fan-out need explicit locking that SQLite provides for free.

### Fixed bucket count with derived width

The bar renders a constant number of buckets and each bucket spans `range / bucketCount`. This keeps the visual density identical whether the operator selects one hour or ninety days, which is what makes a one-hour view and a ninety-day view comparable at a glance.

Alternative considered: fixed bucket width (for example one day per bucket, as a public status page does). Rejected because a one-hour range would then render a single bucket.

### Recording is independent of the display flag

Checks are recorded even when `uptime.enabled` is false. Enabling the bar later therefore shows the history that accumulated while it was off, rather than starting empty. The cost is one write per site per minute whether or not anyone looks at the bar.

### Bucket colouring is a pure function of success rate

Green for a fully successful bucket, yellow at 90% and above, red below 90%, grey for no data. This is computed at query time from the stored counters, so changing the thresholds later does not require rewriting history.

Alternative considered: colour by average response time. Rejected — a slow but working service and a broken one would look the same, and the specs define the bar as availability, not latency.

### Retention prunes to the longest configured window

The pruner reads the maximum configured `range` and deletes rows older than it. It runs on the same schedule as recording rather than on every write, so the cost is amortised.

If no widget enables uptime, the default 30-day window still governs retention, so the database does not grow unbounded merely because the feature is off.

### Three-state classification is derived, not stored

Healthy, degraded, failing are computed from the check result: success and fast is healthy, success but slow is degraded, failure is failing. Thresholds live in code, not in the database, so they can change without a migration. The aggregate banner reports the worst state across the widget's services.

## Risks / Trade-offs

- **A write per site per minute from a 20-worker fan-out contends on one database file** → SQLite serialises writes; with 26 sites the write rate is under one per second, far below what a single connection handles. If contention appears, batching the minute's writes into one transaction resolves it.
- **The database file is the only copy of history** → Losing it loses the bar's history, not the dashboard. Documented in the deployment notes; the file is small enough to back up with the config.
- **Image growth of 10-15 MB** → Accepted. The alternative is a hand-rolled store with its own bug surface.
- **An unwritable data path silently degrades to no history** → Mitigated by failing loudly at startup when the path cannot be opened, as required by the spec's "Missing data path is reported" scenario.
- **The first days of a 30- or 90-day window are mostly empty** → Expected and rendered as no-data rather than as healthy, so an empty bar is not mistaken for a perfect record.
- **Retention pruning on a large window may be slow the first time** → The first prune after enabling a long window deletes nothing (no old rows exist). Subsequent prunes are incremental.

## Migration Plan

1. Add the storage layer and recording, with the bar still disabled in configuration. Verify rows accumulate and survive a restart.
2. Enable `uptime` on one monitor widget with a short range and verify the bar renders buckets that match the recorded data.
3. Add the writable `./data` mount to the deployment compose file and recreate the container.
4. Roll back by removing the `uptime` blocks and the mount; the database file is inert and can be deleted. The image must be rolled back too, since the previous image has no SQLite driver and no storage code.
