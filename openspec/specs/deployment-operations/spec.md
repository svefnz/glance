# deployment-operations Specification

## Purpose
Defines how the hosted dashboard instance is built, published, composed, secured, and wired into the reverse proxy.

## Requirements

### Requirement: Custom image is built from the fork

The deployed dashboard image SHALL be built from this repository's source rather than consumed from the upstream published image. Image publication MUST happen automatically from a tagged release.

#### Scenario: Tagged release publishes an image

- **WHEN** a version tag is pushed to the repository
- **THEN** a container image is built from this source tree
- **AND** the image is published to the project's GHCR package

#### Scenario: Deployment consumes the custom image

- **WHEN** the deployment is inspected
- **THEN** the compose definition references the custom image rather than the upstream image

### Requirement: Dashboard requires authentication

The dashboard SHALL require a username and password before serving any page or page-content response. Passwords MUST be stored as bcrypt hashes rather than plaintext.

#### Scenario: Unauthenticated request is rejected

- **WHEN** an unauthenticated client requests the dashboard root
- **THEN** the response does not include dashboard content
- **AND** the client is directed to authenticate

#### Scenario: Authenticated request is served

- **WHEN** a client supplies valid credentials
- **THEN** the requested page content is returned

#### Scenario: Passwords are not stored in plaintext

- **WHEN** the configuration file is inspected
- **THEN** no plaintext password value is present for any user

### Requirement: Real client addresses reach the dashboard

The dashboard SHALL determine the originating client address from reverse-proxy forwarding headers so that its failed-authentication protection applies per client rather than per proxy.

#### Scenario: Brute-force protection sees distinct clients

- **WHEN** two different clients each fail authentication repeatedly
- **THEN** each client is tracked by its own address

### Requirement: Container state is readable

The dashboard SHALL have read access to the container runtime so that the container-inventory widget can report state.

#### Scenario: Container states are available

- **WHEN** the operator views the container widget
- **THEN** each configured container's live state is reported rather than an error

### Requirement: Unused statistics agent is not deployed

The deployment MUST NOT run a statistics agent on a host where no dashboard widget consumes agent data, because local statistics are read directly from the container runtime in that case.

#### Scenario: Agent is absent

- **WHEN** the host's running containers are listed
- **THEN** no statistics agent container is present on a host that the dashboard does not query as an agent source

#### Scenario: Dashboard host agent is deliberate

- **WHEN** the dashboard host runs a statistics agent
- **THEN** that agent is a configured source for dashboard widgets
- **AND** it is bound to loopback with authentication enabled

### Requirement: Dead reverse-proxy entry is removed

The reverse proxy MUST NOT contain a route whose upstream has no listener.

#### Scenario: Dead route is gone

- **WHEN** the reverse-proxy configuration is inspected
- **THEN** no route targets a port without a listener

#### Scenario: Remaining routes are unaffected

- **WHEN** the reverse proxy is reloaded after the dead route is removed
- **THEN** all remaining service routes continue to respond as before

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
