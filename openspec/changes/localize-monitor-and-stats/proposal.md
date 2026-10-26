# Proposal

## Why

The calendar widget was localized to Chinese, but the widgets the operator actually watches were not. The Server page and the container inventory render their labels and state text in English, so the same dashboard mixes two languages: a Chinese month header sits above an English `All systems operational` banner, and the host metrics read `RAM` / `SWAP` / `DISK`.

This is most visible on the Server page, which is the page built specifically for this deployment and is read most often.

## What Changes

- Localize the monitor widget's aggregate banner: the healthy, degraded, and failing summaries become Chinese.
- Localize the monitor widget's per-service state text: `Operational`, `Degraded`, and `Down` become Chinese.
- Localize the monitor widget's empty state shown when every site is filtered out.
- Localize the monitor widget's memory tooltip wording.
- Localize the server-statistics widget's labels: platform, memory, swap, disk, temperature, load averages, uptime, and the unreachable state.
- Localize the container-inventory widget's memory tooltip and its empty state.

Technical abbreviations stay in English: `CPU`, `ms`, `MB`, `GB`, `%`, `n/a`, and the HTTP status descriptions such as `Not Found` and `Server Error`. They are read as units and codes rather than as prose, and keeping them makes log comparison straightforward.

No i18n framework is introduced. The deployment serves one language, and a translation layer would be machinery without a second caller.

## Capabilities

### New Capabilities

- `widget-localization`: which widgets render their human-readable labels and state text in Chinese, and which technical abbreviations deliberately stay in English.

### Modified Capabilities

None. The existing `calendar-localization` capability is scoped to the calendar's month names, weekday abbreviations, and navigation labels; extending it to unrelated widgets would make its name inaccurate.

## Impact

- `internal/glance/templates/monitor.html` — banner text, empty state, memory tooltip.
- `internal/glance/templates/monitor-compact.html` — empty state.
- `internal/glance/widget-monitor.go` — the state text returned by the classification function.
- `internal/glance/templates/server-stats.html` — metric labels, uptime suffix, unreachable and unknown states.
- `internal/glance/templates/docker-containers.html` — memory tooltip and empty state.
- No configuration changes; the wording is not operator-configurable.
- No new dependencies.
