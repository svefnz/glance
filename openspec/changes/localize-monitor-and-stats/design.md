# Design

## Context

See proposal.md - Why.

The wording to change lives in three places, and which place depends on whether the text is static or computed:

| Text | Location | Kind |
|---|---|---|
| Banner sentences, empty states, metric labels, tooltips | `internal/glance/templates/*.html` | static template text |
| Per-service state (`Operational` / `Degraded` / `Down`) | `classifySiteState` in `widget-monitor.go` | computed, returned alongside a state key |

The calendar localization already established the project's pattern: the Chinese strings are written directly where they are rendered, with no lookup table and no locale switch. `calendar.js` holds `MONTH_NAMES` and `WEEKDAY_ABBRSS` as plain Chinese arrays and the tooltips as literal Chinese attributes.

## Goals / Non-Goals

**Goals:**

- Every human-readable label in the monitor, server-statistics, and container-inventory widgets reads in Chinese.
- Technical abbreviations and HTTP status descriptions stay English so values stay comparable with logs and documentation.
- The change is mechanical and reviewable: no new abstraction, no configuration surface.

**Non-Goals:**

- An i18n framework, message catalog, or locale switching. The deployment serves one language.
- Localizing widgets outside this scope (weather, rss, hacker-news, search, clock, releases, bookmarks).
- Localizing the login page or the widget error/loading chrome in `widget-base.html`.
- Making the wording operator-configurable.

## Decisions

### Write the Chinese strings inline, following the calendar pattern

The calendar widget already does this. Introducing a message catalog for one language would add a lookup indirection with no second caller, and would make the templates harder to read for the one person maintaining them.

Alternative considered: a `localization` package with a `T(key)` function. Rejected — it would need keys, a catalog file, and a fallback path, all to serve a single locale.

### State text changes in Go, not in the template

`classifySiteState` returns a state key (`ok` / `degraded` / `failing`) and a display text. The key drives CSS classes and filtering; the text is what the operator reads. Only the text changes, so the key vocabulary and every CSS rule built on it stay untouched.

This also keeps the compact variant correct without editing it, since both variants consume the same returned text.

Alternative considered: map the key to Chinese in the template. Rejected — it would duplicate the mapping across two templates that both render state text.

### Abbreviations stay English

`CPU`, `ms`, `MB`, `GB`, `%`, and `n/a` are units, and `Not Found` / `Server Error` and their siblings are HTTP status descriptions. An operator correlating a dashboard reading with a log line or an upstream response compares those tokens directly; translating them would make the comparison harder, not easier.

This is the one place the change deliberately leaves English in place, so it is stated as its own requirement rather than left implicit.

### Localize the memory tooltip, keep the inline abbreviation

The container widget shows `1690M` inline and `Memory: 1690 MB / 2474 MB (68%)` in the hover popover. Only the word `Memory` is prose; `MB`, the numbers, and the percentage are not. The same split applies to the monitor widget's memory tooltip.

## Risks / Trade-offs

- **Mixed-language widgets if the scope is incomplete** → The three widgets in scope are the ones on the Server page and the container inventory. Other widgets remain English, so the dashboard stays partly English until a follow-up change. Accepted and stated in the proposal's non-goals rather than silently expanded.
- **Chinese text is wider than the English it replaces** → `All systems operational` becomes a shorter Chinese phrase, so the banner has more room, not less. The metric labels (`PLATFORM` → `平台`) also shorten. No layout regression expected; verified during implementation.
- **Template text is not covered by tests** → The existing tests assert on CSS class names and state keys, not on visible words. Verification therefore relies on rendering the page and reading it, plus a test asserting the state text returned by the classification function.
- **A future locale request would need this redone** → Accepted. Adding a second language would mean introducing the catalog this change deliberately avoids; that is the right time to pay for it.
