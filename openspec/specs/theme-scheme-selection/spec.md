# theme-scheme-selection Specification

## Purpose
Defines which color schemes the dashboard offers and how the automatic scheme follows the operating system preference.

## Requirements

### Requirement: Three selectable color schemes

The dashboard SHALL offer three selectable schemes: a dark scheme, a light scheme, and an automatic scheme. The theme picker MUST present all three as distinct choices.

#### Scenario: Picker lists all schemes

- **WHEN** the operator opens the theme picker
- **THEN** dark, light, and automatic entries are each selectable

#### Scenario: Explicit selection is remembered

- **WHEN** the operator selects the dark or light scheme
- **THEN** that scheme is applied on subsequent page loads until changed

### Requirement: Automatic scheme follows the OS preference

When the automatic scheme is selected, the dashboard SHALL render using the scheme matching the operating system's color-scheme preference. If the preference is unavailable or indeterminate, the dark scheme MUST be used.

#### Scenario: OS prefers light

- **WHEN** the automatic scheme is selected and the operating system reports a light preference
- **THEN** the dashboard renders with the light scheme

#### Scenario: OS prefers dark

- **WHEN** the automatic scheme is selected and the operating system reports a dark preference
- **THEN** the dashboard renders with the dark scheme

#### Scenario: Preference unknown falls back to dark

- **WHEN** the automatic scheme is selected and the operating system preference cannot be determined
- **THEN** the dashboard renders with the dark scheme

### Requirement: Automatic scheme resolves without a visible flash

The dashboard MUST resolve the automatic scheme before first paint so that the operator does not observe a transition from one scheme to another on load.

#### Scenario: No scheme flash on load

- **WHEN** the operator loads the dashboard with the automatic scheme selected
- **THEN** the first rendered frame already uses the correct scheme
- **AND** no visible dark-to-light or light-to-dark switch occurs

### Requirement: Automatic resolution does not require an extra round trip

Resolving the automatic scheme MUST NOT depend on a second request to the server, and MUST NOT require the reverse proxy to forward additional client-hint headers.

#### Scenario: First visit resolves correctly

- **WHEN** the operator visits the dashboard for the first time with a system light preference
- **THEN** the first response already renders the light scheme
