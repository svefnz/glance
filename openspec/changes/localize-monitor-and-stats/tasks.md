# Tasks

## 1. Monitor widget state text

- [x] 1.1 Change the display text returned by the site state classification to Chinese for the healthy, degraded, and failing cases, leaving the state keys unchanged. Verify with a test asserting each of the three returned texts, and confirm the existing tests that assert on state keys still pass.
- [x] 1.2 Confirm the compact variant renders no state text at all, so the state-text localization does not reach it: it conveys state through an icon and a status-code tooltip only. Verify by rendering a compact widget and confirming it shows the title, latency, and icon, with no state word in either language.

## 2. Monitor widget template

- [x] 2.1 Localize the aggregate banner for the healthy, degraded, and failing cases. Verify by rendering a widget in each of the three states and confirming the banner text is Chinese.
- [x] 2.2 Localize the empty state shown when a failing-only filter matches nothing, in both the standard and compact templates. Verify by rendering a widget with `show-failing-only` and no failing services and confirming the text is Chinese.
- [x] 2.3 Localize the memory wording in the container usage tooltip while leaving the numbers, `MB`, and the percentage unchanged. Verify by rendering a site with usage figures and reading the tooltip text.

## 3. Server statistics widget

- [x] 3.1 Localize the metric labels: platform, memory, swap, disk, and temperature. Verify by rendering the widget and confirming no Latin-script label remains except `CPU`.
- [x] 3.2 Localize the one-minute and fifteen-minute load average labels. Verify by rendering a host whose load is available and reading both labels.
- [x] 3.3 Localize the uptime suffix, the unreachable state, and the unknown-uptime state. Verify by rendering one reachable host and one unreachable host and reading both.

## 4. Container inventory widget

- [x] 4.1 Localize the memory label in the usage tooltip while leaving the values, `MB`, and the percentage unchanged. Verify by rendering a container with usage figures and reading the tooltip.
- [x] 4.2 Localize the empty state shown when no container matches the filter. Verify by rendering the widget with a filter that matches nothing and reading the text.

## 5. Verification

- [x] 5.1 Confirm the abbreviations and HTTP status descriptions are unchanged: `CPU`, `ms`, `MB`, `GB`, `%`, `n/a`, and the status phrases. Verify by rendering a widget containing a failing service and reading its status description.
- [x] 5.2 Run the full test suite and confirm it passes, since the classification text change touches tested code.
- [ ] 5.3 Deploy and verify on the running instance that the Server page and the container widget show Chinese labels, and that no layout breaks from the wider or narrower text.

## Workflow follow-up

- Archive this change once every task above is complete, so the `widget-localization` capability is created.
