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

The deployment MUST NOT run a separate statistics agent on the same host as the dashboard, because local statistics are read directly.

#### Scenario: Agent is absent

- **WHEN** the host's running containers are listed
- **THEN** no statistics agent container is present

### Requirement: Dead reverse-proxy entry is removed

The reverse proxy MUST NOT contain a route whose upstream has no listener.

#### Scenario: Dead route is gone

- **WHEN** the reverse-proxy configuration is inspected
- **THEN** no route targets a port without a listener

#### Scenario: Remaining routes are unaffected

- **WHEN** the reverse proxy is reloaded after the dead route is removed
- **THEN** all remaining service routes continue to respond as before
