# dashboard-layout Specification

## Purpose
Defines how the hosted dashboard is organized into pages and columns, and which widgets belong on each page for this particular deployment.

## Requirements

### Requirement: Two-page navigation structure

The dashboard SHALL expose exactly two pages: `Home` for service operations and `News` for information flow. Navigation between them MUST be available from the desktop navigation bar and the mobile navigation menu.

#### Scenario: Navigating between pages

- **WHEN** the operator opens the dashboard root
- **THEN** the `Home` page is shown
- **AND** a navigation link to `News` is visible in the navigation bar

#### Scenario: Page slugs are stable

- **WHEN** the operator requests the `Home` page
- **THEN** it is reachable at the slug `home`
- **AND** the `News` page is reachable at the slug `news`

### Requirement: Home page is a service operations panel

The `Home` page SHALL display widgets that report the health and reachability of hosted services, the host machine, and container state. It MUST NOT display content-aggregation widgets.

#### Scenario: Home page widget inventory

- **WHEN** the operator opens the `Home` page
- **THEN** it contains a `calendar` widget, a `monitor` widget, a `docker-containers` widget, a `server-stats` widget, a `search` widget, and a `weather` widget
- **AND** it contains no `rss`, `reddit`, `hacker-news`, `lobsters`, or `videos` widget

#### Scenario: Home page column arrangement

- **WHEN** the operator opens the `Home` page on a desktop viewport
- **THEN** the `calendar` and `monitor` widgets appear in the first small column
- **AND** the `docker-containers` widget appears in the full column
- **AND** the `server-stats`, `search`, and `weather` widgets appear in the second small column

### Requirement: News page is an information flow

The `News` page SHALL display content-aggregation widgets sourced from external feeds. It MUST NOT display infrastructure monitoring widgets.

#### Scenario: News page widget inventory

- **WHEN** the operator opens the `News` page
- **THEN** it contains a grouped `hacker-news` and `lobsters` widget, a repository-tracking `rss` widget, and a GitHub-activity `rss` widget
- **AND** it contains no `monitor`, `docker-containers`, or `server-stats` widget

#### Scenario: Hacker News and Lobsters are grouped

- **WHEN** the operator views the `News` page
- **THEN** `hacker-news` and `lobsters` render inside a single `group` widget occupying one card

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

### Requirement: Container inventory is declared in configuration

The `docker-containers` widget SHALL report the state of the containers running on the host. Container display names, icons, and links MUST be declared in the dashboard configuration rather than sourced from container labels.

#### Scenario: Configured containers are listed

- **WHEN** the operator views the `docker-containers` widget
- **THEN** every configured container appears with its configured display name and icon
- **AND** each container's current state is shown

#### Scenario: Unconfigured containers are hidden

- **WHEN** a container runs on the host but has no entry in the dashboard configuration
- **THEN** it does not appear in the widget

### Requirement: Server statistics read from the host

The `server-stats` widget SHALL report CPU load, memory usage, swap usage, and disk usage for the machine hosting the dashboard, without requiring a separate statistics agent.

#### Scenario: Local statistics are displayed

- **WHEN** the operator views the `server-stats` widget
- **THEN** it shows CPU load, memory usage, swap usage, and root filesystem usage
- **AND** the reported memory total matches the host's physical memory

### Requirement: Search uses Brave with bang shortcuts

The `search` widget SHALL use Brave Search as its default engine and MUST provide bang shortcuts for GitHub, YouTube, npm, and Wikipedia.

#### Scenario: Default search goes to Brave

- **WHEN** the operator enters a query and submits
- **THEN** the request is sent to Brave Search with the query included

#### Scenario: Bang shortcut routes to the named site

- **WHEN** the operator enters `!gh` followed by a query
- **THEN** the request is sent to GitHub search rather than Brave

### Requirement: Weather reflects the operator's location

The `weather` widget SHALL report weather for Shantou, Guangdong, China, using metric units and a 24-hour clock.

#### Scenario: Weather shows the configured location

- **WHEN** the operator views the `weather` widget
- **THEN** the reported location resolves to Shantou, Guangdong, China
- **AND** temperatures are shown in Celsius
- **AND** hour labels use 24-hour format

### Requirement: Repository tracking covers releases and commits

The repository-tracking `rss` widget SHALL surface releases for each tracked repository and commit feeds for repositories where per-commit detail is wanted. Commit feeds MUST target each repository's actual default branch.

#### Scenario: Releases are shown for every tracked repository

- **WHEN** the operator views the repository-tracking widget
- **THEN** release entries appear for the opencode-slim, ponytail, omarchy, and bun repositories

#### Scenario: Commit feeds use the correct branch

- **WHEN** a repository's default branch is not `main`
- **THEN** its commit feed targets that default branch
- **AND** commit entries appear in the widget

### Requirement: Personal GitHub activity is displayed unfiltered

The GitHub-activity `rss` widget SHALL display the operator's public activity feed without truncating the entry count.

#### Scenario: Full activity stream is shown

- **WHEN** the operator views the GitHub-activity widget
- **THEN** every entry present in the public activity feed is rendered
- **AND** the entry count is not artificially limited

### Requirement: Removed default widgets stay removed

The deployment MUST NOT include widgets from the upstream example configuration that do not apply to this host.

#### Scenario: Inapplicable widgets are absent

- **WHEN** the operator inspects both pages
- **THEN** no `twitch-channels`, `videos`, `markets`, or `reddit` widget is present
- **AND** no feed sourced from the upstream example blog list is present
