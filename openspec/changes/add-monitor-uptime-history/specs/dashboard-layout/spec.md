# Spec Delta

## MODIFIED Requirements

### Requirement: Monitor covers only reachable service domains

The `monitor` widget SHALL list every service domain that has a live upstream, and MUST NOT list domains whose upstream has no listener. Domains that answer with a redirect status MUST be configured so the redirect counts as healthy.

#### Scenario: All live services are monitored

- **WHEN** the operator views the `monitor` widget
- **THEN** it lists entries for the GitLab, PDF translation, file share, Octopus, Grok, subscription API, and dashboard domains
- **AND** every entry reports an OK status

#### Scenario: Redirect-answering services report healthy

- **WHEN** a monitored service responds with an HTTP 302 or 307 redirect
- **THEN** the widget reports that service as OK rather than failing

#### Scenario: Dead upstream is not monitored

- **WHEN** a domain's upstream port has no listener
- **THEN** that domain does not appear in the `monitor` widget

#### Scenario: Service state is classified into three levels

- **WHEN** a monitored service answers slowly but successfully
- **THEN** the widget reports it as degraded rather than healthy or failing

#### Scenario: Overall state is summarised

- **WHEN** the operator views a monitor widget with several services
- **THEN** a banner states the aggregate state of all services in that widget

#### Scenario: Each service shows its availability history

- **WHEN** the operator views a monitor widget with uptime enabled
- **THEN** each service shows a bar of buckets covering the configured window
