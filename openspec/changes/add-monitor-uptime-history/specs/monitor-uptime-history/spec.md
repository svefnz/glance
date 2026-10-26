# Spec Delta

## Purpose

Records the historical availability of monitored services and presents it as a configurable uptime bar, so an operator can distinguish a service that just failed from one that has been failing for days.

## ADDED Requirements

### Requirement: Monitor checks are recorded as per-minute aggregates

The system SHALL persist every monitor check result, aggregated to one record per service per minute, holding the success count, the total check count, and the average response time for that minute.

#### Scenario: A successful check is recorded

- **WHEN** a monitored service answers successfully
- **THEN** the current minute's record for that service counts one success and one total

#### Scenario: A failed check is recorded

- **WHEN** a monitored service fails to answer
- **THEN** the current minute's record for that service counts one total and no success

#### Scenario: Repeated checks in one minute accumulate

- **WHEN** several checks for the same service occur within one minute
- **THEN** they accumulate into a single record for that minute rather than creating separate rows

### Requirement: Uptime history is configurable per widget

The `monitor` widget SHALL accept an `uptime` block with an `enabled` flag and a `range` value selecting the history window. The range MUST accept values from 1 hour to 90 days and MUST default to 30 days when unset.

#### Scenario: History is off by default

- **WHEN** a monitor widget has no `uptime` block
- **THEN** no uptime bar is rendered
- **AND** check results are still recorded so that enabling the bar later shows existing history

#### Scenario: A configured range selects the window

- **WHEN** the widget sets `uptime.range` to `7d`
- **THEN** the uptime bar covers the last seven days

#### Scenario: Range outside the supported bounds is rejected

- **WHEN** the widget sets a range shorter than 1 hour or longer than 90 days
- **THEN** the configuration is rejected with an error naming the supported bounds

#### Scenario: Unset range uses the default

- **WHEN** the widget enables uptime without a range
- **THEN** the bar covers the last 30 days

### Requirement: Uptime bar shows a fixed number of buckets

The uptime bar SHALL render a fixed number of buckets regardless of the configured range, with each bucket covering an equal slice of the window.

#### Scenario: Bucket count is independent of range

- **WHEN** the range is 1 hour and when it is 90 days
- **THEN** both render the same number of buckets
- **AND** the bucket width differs so that the buckets together span the configured window

#### Scenario: Empty buckets render as no data

- **WHEN** a bucket covers a period with no recorded checks
- **THEN** that bucket renders as having no data rather than as healthy or failing

### Requirement: Bucket colour reflects success rate

Each bucket SHALL be coloured by the success rate of the checks it covers: healthy when every check succeeded, degraded when the rate is at least 90 percent but not complete, failing below 90 percent, and no-data when the bucket holds no checks.

#### Scenario: Fully successful bucket is healthy

- **WHEN** every check in a bucket succeeded
- **THEN** the bucket is coloured as healthy

#### Scenario: Mostly successful bucket is degraded

- **WHEN** at least 90 percent of the checks in a bucket succeeded but not all of them
- **THEN** the bucket is coloured as degraded

#### Scenario: Poorly performing bucket is failing

- **WHEN** fewer than 90 percent of the checks in a bucket succeeded
- **THEN** the bucket is coloured as failing

### Requirement: History is retained only for the configured window

The system MUST delete records older than the longest configured retention window, so that the database does not grow without bound.

#### Scenario: Old records are pruned

- **WHEN** records older than the retention window exist
- **THEN** they are removed rather than kept indefinitely

#### Scenario: Records inside the window are kept

- **WHEN** records fall inside the retention window
- **THEN** they remain available to the uptime bar

### Requirement: History survives a restart

Recorded history SHALL persist across process restarts, stored on a path the deployment can mount.

#### Scenario: History is available after a restart

- **WHEN** the process restarts
- **THEN** the uptime bar still shows the history recorded before the restart

#### Scenario: Missing data path is reported

- **WHEN** the configured data path is not writable
- **THEN** the failure is reported rather than silently discarding history
