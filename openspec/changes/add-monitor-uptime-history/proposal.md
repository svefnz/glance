# Proposal

## Why

The `monitor` widget reports only the current state of each service: a status code, a response time, and whether the last check succeeded. An operator looking at the dashboard cannot tell whether a service has been flaky for a week or has been solid for months — the previous result is discarded on every update.

The deployed Server page has 26 monitored sites across three hosts, several of which intermittently fail (`hermes` has no DNS record, `llm` resets connections, `RSSHub` and two agents report errors). Without history there is no way to distinguish a service that broke an hour ago from one that has been down for days, and no way to see whether an intermittent failure is getting worse.

## What Changes

- Record every monitor check as a per-minute aggregate (success count, total count, average response time) in a local SQLite database.
- Render an uptime bar under each monitored site: a fixed 60 buckets covering a configurable window.
- Add a per-widget `uptime` configuration block with `enabled` and `range` (1 hour to 90 days, default 30 days).
- Colour each bucket by success rate: green when every check succeeded, yellow at 90% or above, red below 90%, grey when there is no data.
- Restyle the monitor widget in the style of a public status page: an overall status banner at the top, one row per service with icon, name, state text, latency, and resource usage, and the uptime bar beneath.
- Classify each service into three states — healthy, degraded, or failing — instead of the current two.

**BREAKING**: the monitor widget gains a SQLite dependency and a writable data directory. A deployment that does not mount a writable path loses history on restart, and an image built without the SQLite driver will not start.

## Capabilities

### New Capabilities

- `monitor-uptime-history`: recording, aggregating, and presenting the historical availability of monitored services, including the configurable retention window and the bucket colouring rules.

### Modified Capabilities

- `dashboard-layout`: the `monitor` widget's presentation changes from a list of status codes to a status-page layout with an overall banner, per-service state classification, and an uptime bar.

## Impact

- `internal/glance/widget-monitor.go` — record check results, parse the new `uptime` config block, compute the three-state classification.
- New file for storage: schema, per-minute aggregation, windowed query, retention pruning.
- `internal/glance/templates/monitor.html` and `monitor-compact.html` — status-page layout and the uptime bar.
- `internal/glance/static/css/widget-monitor.css` — banner, row, and bucket styling.
- `internal/glance/config.go` — validate the `range` value against the supported window.
- New dependency: a pure-Go SQLite driver, since the image is built with `CGO_ENABLED=0`.
- Deployment: a writable `./data` directory mounted into the container.
- Container image grows by roughly 10-15 MB from the SQLite driver.
- Operational: the database file grows with monitored sites times minutes retained; pruning must run or the file grows without bound.
