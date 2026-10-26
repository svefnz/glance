# Spec Delta

## ADDED Requirements

### Requirement: Agent requests are authenticated

Every statistics agent SHALL require an `Authorization: Bearer` header matching its configured token, and SHALL reject requests that omit it or present a wrong value with a 401 response.

#### Scenario: Request without credentials is rejected

- **WHEN** a client requests an agent endpoint without an `Authorization` header
- **THEN** the agent responds with 401
- **AND** no host statistics or container data is returned

#### Scenario: Request with wrong token is rejected

- **WHEN** a client requests an agent endpoint with a token that does not match the configured value
- **THEN** the agent responds with 401

#### Scenario: Request with the configured token is served

- **WHEN** a client requests an agent endpoint with the configured token
- **THEN** the agent returns the requested data

### Requirement: Agent refuses to start without a token

An agent whose configuration carries no token MUST fail to start rather than serve unauthenticated requests.

#### Scenario: Missing token prevents startup

- **WHEN** an agent starts with no token in its configuration and no token in its environment
- **THEN** the process exits with an error
- **AND** no port is left listening

#### Scenario: Empty token is treated as missing

- **WHEN** an agent is configured with an empty token value
- **THEN** the process exits with an error

### Requirement: Every dashboard agent consumer sends its token

Each dashboard widget that contacts a statistics agent SHALL send the token configured for that agent, so that enabling agent authentication does not degrade any widget.

#### Scenario: Container usage is fetched with credentials

- **WHEN** the monitor widget requests container usage from an agent
- **THEN** the request carries the configured token
- **AND** the widget reports CPU and memory figures rather than blank values

#### Scenario: Container inventory is fetched with credentials

- **WHEN** the container-inventory widget requests container state from an agent
- **THEN** the request carries the configured token
- **AND** the widget reports container state rather than an error

#### Scenario: Host statistics are fetched with credentials

- **WHEN** the server-statistics widget requests host metrics from a remote agent
- **THEN** the request carries the configured token

### Requirement: Agent network exposure is limited to its client

Each agent MUST be reachable only by the dashboard host, so that the agent port is not usable by arbitrary internet clients.

#### Scenario: Agent on the dashboard host is loopback-only

- **WHEN** the agent running on the same host as the dashboard is inspected
- **THEN** it listens on loopback rather than a public interface

#### Scenario: Remote agent rejects other sources

- **WHEN** a client other than the dashboard host attempts to reach a remote agent's port
- **THEN** the connection is refused by the host firewall

#### Scenario: Dashboard host reaches the remote agent

- **WHEN** the dashboard requests data from a remote agent
- **THEN** the connection succeeds

### Requirement: Each monitored host runs its own agent

Every host that appears as a server-statistics or container source in the dashboard configuration SHALL have a running agent, so that no configured host renders empty.

#### Scenario: All configured hosts report statistics

- **WHEN** the dashboard renders its server-statistics and container widgets
- **THEN** each configured host reports CPU, memory, and container data

#### Scenario: A configured host without an agent is a defect

- **WHEN** a host is configured in the dashboard but has no reachable agent
- **THEN** the missing data is treated as a deployment fault to be fixed rather than an acceptable state

## MODIFIED Requirements

### Requirement: Unused statistics agent is not deployed

The deployment MUST NOT run a statistics agent on a host where no dashboard widget consumes agent data, because local statistics are read directly from the container runtime in that case.

#### Scenario: Agent is absent

- **WHEN** the host's running containers are listed
- **THEN** no statistics agent container is present on a host that the dashboard does not query as an agent source

#### Scenario: Dashboard host agent is deliberate

- **WHEN** the dashboard host runs a statistics agent
- **THEN** that agent is a configured source for dashboard widgets
- **AND** it is bound to loopback with authentication enabled
