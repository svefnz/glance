# Spec Delta

## Purpose

Defines which widgets render their human-readable labels and state text in Chinese for this deployment, and which technical abbreviations deliberately remain in English.

## ADDED Requirements

### Requirement: Monitor state text renders in Chinese

The monitor widget SHALL display each service's state as Chinese text rather than an English word.

#### Scenario: Healthy service shows a Chinese state

- **WHEN** a monitored service answers successfully and quickly
- **THEN** its state reads a Chinese word for healthy rather than `Operational`

#### Scenario: Slow service shows a Chinese degraded state

- **WHEN** a monitored service answers successfully but slowly
- **THEN** its state reads a Chinese word for degraded rather than `Degraded`

#### Scenario: Failed service shows a Chinese failing state

- **WHEN** a monitored service fails to answer or returns an error status
- **THEN** its state reads a Chinese word for failing rather than `Down`

### Requirement: Monitor banner renders in Chinese

The monitor widget's aggregate banner SHALL summarise the widget's overall state in Chinese, covering the healthy, degraded, and failing cases.

#### Scenario: All-healthy banner is Chinese

- **WHEN** every service in a monitor widget is healthy
- **THEN** the banner reads a Chinese sentence rather than `All systems operational`

#### Scenario: Degraded banner is Chinese

- **WHEN** at least one service is degraded and none is failing
- **THEN** the banner reads a Chinese sentence describing partial degradation

#### Scenario: Failing banner is Chinese

- **WHEN** at least one service is failing
- **THEN** the banner reads a Chinese sentence describing unavailability

#### Scenario: Filtered empty state is Chinese

- **WHEN** a monitor widget configured to show only failing services has none
- **THEN** its empty state reads Chinese text rather than `All sites are online`

### Requirement: Server statistics labels render in Chinese

The server-statistics widget SHALL label its host metrics in Chinese.

#### Scenario: Metric labels are Chinese

- **WHEN** the operator views the server-statistics widget
- **THEN** the platform, memory, swap, disk, and temperature labels are Chinese words rather than `PLATFORM`, `RAM`, `SWAP`, `DISK`, and `TEMP C`

#### Scenario: Load average labels are Chinese

- **WHEN** the operator views the CPU section
- **THEN** the one-minute and fifteen-minute load labels are Chinese rather than `1M AVG` and `15M AVG`

#### Scenario: Uptime suffix is Chinese

- **WHEN** the operator views a reachable host's uptime
- **THEN** the suffix following the relative time is a Chinese word rather than `uptime`

#### Scenario: Unreachable state is Chinese

- **WHEN** a configured host cannot be reached
- **THEN** the widget reads a Chinese phrase rather than `unreachable`

#### Scenario: Unknown uptime is Chinese

- **WHEN** a host's boot time is unavailable
- **THEN** the uptime reads a Chinese word rather than `unknown`

### Requirement: Container usage wording renders in Chinese

The container-inventory widget SHALL label its memory usage and its empty state in Chinese.

#### Scenario: Memory tooltip is Chinese

- **WHEN** the operator hovers a container's usage figures
- **THEN** the memory line uses a Chinese label rather than `Memory`

#### Scenario: Empty inventory is Chinese

- **WHEN** no container matches the widget's filter
- **THEN** the widget reads a Chinese phrase rather than `No containers available to show.`

### Requirement: Technical abbreviations stay in English

Widgets MUST keep technical abbreviations and HTTP status descriptions in English, because they are read as units and codes rather than as prose.

#### Scenario: Units are unchanged

- **WHEN** a widget displays a value with a unit
- **THEN** the unit is `CPU`, `ms`, `MB`, `GB`, `%`, or `n/a` as before

#### Scenario: Status descriptions are unchanged

- **WHEN** a monitored service returns an error status
- **THEN** the status description remains an English phrase such as `Not Found` or `Server Error`
