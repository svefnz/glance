# Tasks

## 1. Storage layer

- [x] 1.1 Add the pure-Go SQLite driver to `go.mod` and verify `CGO_ENABLED=0 go build ./...` still succeeds, confirming no cgo is pulled in.
- [x] 1.2 Create the schema: a table keyed by site and minute holding success count, total count, and average response time. Verify by opening a fresh database and confirming the table exists with the expected columns.
- [x] 1.3 Implement the recording upsert that adds one check to the current minute's row, creating it when absent. Verify with a test that two checks in the same minute produce one row with a total of two.
- [x] 1.4 Implement the windowed query returning buckets over an arbitrary range, with bucket width derived from the range and a fixed bucket count. Verify with a test that a one-hour and a ninety-day range both return the same number of buckets spanning their window.
- [x] 1.5 Implement the retention pruner that deletes rows older than the longest configured window. Verify with a test that rows outside the window are removed and rows inside it remain.
- [x] 1.6 Make opening an unwritable data path return an error that names the path, and verify by pointing the path at a read-only directory and confirming startup fails with that error.

## 2. Configuration

- [x] 2.1 Add the `uptime` block to the monitor widget with `enabled` and `range` fields, defaulting the range to 30 days when the block is present but the range is unset. Verify that a widget with no `uptime` block records history and renders no bar.
- [x] 2.2 Validate the range against the supported bounds of 1 hour to 90 days and reject values outside them with an error naming the bounds. Verify with tests for a value below, at, and above each bound.
- [x] 2.3 Add a test asserting that enabling uptime without a range yields the 30-day default.

## 3. Recording integration

- [x] 3.1 Record every check result from the monitor widget's update path, including failures and timeouts, so that a failing service accumulates total without success. Verify by test that a timeout records a total with no success.
- [x] 3.2 Confirm recording happens whether or not the bar is enabled, and verify by recording with the bar off and then enabling it and observing non-empty buckets.
- [x] 3.3 Verify the pruner runs on its schedule without blocking the update path, and confirm by observing that a full update cycle still completes in the expected time with pruning enabled.

## 4. Presentation

- [x] 4.1 Rewrite the monitor widget template as a status-page layout: an aggregate banner stating the worst state across the widget's services, and one row per service carrying icon, name, state text, latency, and resource usage. Verify by rendering a widget with a mix of healthy and failing services and confirming the banner reports the failing state.
- [x] 4.2 Add the three-state classification, reporting a successful but slow service as degraded rather than healthy. Verify with a test that a slow success classifies as degraded and a failure classifies as failing.
- [x] 4.3 Render the uptime bar beneath each service, one element per bucket, with the bucket count independent of the range. Verify that a one-hour and a ninety-day configuration render the same number of elements.
- [x] 4.4 Colour each bucket by success rate using the healthy, degraded, failing, and no-data states, and verify each of the four with a test that builds buckets from known counters.
- [x] 4.5 Add the CSS for the banner, service rows, and buckets, and verify the rendered page shows the banner above the rows and the bar beneath each row.
- [x] 4.6 Verify the compact monitor variant still renders correctly after the template change, since it shares the site sub-template.

## 5. Deployment

- [x] 5.1 Add a writable data directory mount to the deployment compose file and verify the container can write the database file there.
- [x] 5.2 Deploy the new image and verify the database file appears on the host under the mounted directory.
- [x] 5.3 Restart the container and verify the uptime bar still shows history recorded before the restart.
- [x] 5.4 Enable uptime on the three host monitor widgets with the 30-day default and verify each renders a bar with buckets that reflect the recorded checks.
- [x] 5.5 Document the data path, the retention behaviour, and the backup expectation in the deployment notes, and verify the notes name the database file and the mount.

## Workflow follow-up

- Archive this change once every task above is complete, so the `monitor-uptime-history` capability is created and the `dashboard-layout` delta is merged.
