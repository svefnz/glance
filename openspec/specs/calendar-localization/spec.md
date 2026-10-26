# calendar-localization Specification

## Purpose
Defines how the calendar widget renders its month names, weekday abbreviations, and navigation labels in Chinese for this deployment.

## Requirements

### Requirement: Month names render in Chinese

The calendar widget SHALL display month names in Chinese rather than English.

#### Scenario: Current month header is Chinese

- **WHEN** the operator views the calendar widget in January
- **THEN** the month header reads a Chinese month name rather than `January`

#### Scenario: Month navigation stays Chinese

- **WHEN** the operator navigates to the next or previous month
- **THEN** the month header still reads a Chinese month name

### Requirement: Weekday abbreviations render in Chinese

The calendar widget SHALL display single-character Chinese weekday abbreviations in the day-of-week header row, ordered according to the configured first day of week.

#### Scenario: Weekday header row is Chinese

- **WHEN** the operator views the calendar widget with Monday as the first day of the week
- **THEN** the header row begins with the Chinese abbreviation for Monday
- **AND** every label in the row is a Chinese character rather than a Latin abbreviation

### Requirement: Calendar navigation labels render in Chinese

The calendar widget SHALL localize the accessible labels attached to its navigation controls.

#### Scenario: Month navigation tooltips are Chinese

- **WHEN** the operator hovers the previous-month or next-month control
- **THEN** the tooltip text is Chinese

#### Scenario: Return-to-current-month tooltip is Chinese

- **WHEN** the operator views a month other than the current one and hovers the undo control
- **THEN** the tooltip text is Chinese

### Requirement: Calendar structure is unchanged by localization

Localization MUST NOT alter the calendar's layout, date math, first-day-of-week handling, or month navigation behavior.

#### Scenario: Date grid still aligns correctly

- **WHEN** the operator views any month with the localized calendar
- **THEN** the first day of the month falls under the correct weekday column
- **AND** the grid still contains six week rows

#### Scenario: Navigation still works

- **WHEN** the operator navigates forward and backward through months
- **THEN** the displayed dates remain correct for each month
