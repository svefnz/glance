# Deployment notes

## Monitor uptime history

The dashboard records every monitor check as a per-minute aggregate and shows
availability as a bar of 60 buckets under each service. History persists across
restarts.

### Database

| Item | Value |
|---|---|
| File | `/root/glance/data/uptime.db` on the host |
| Container path | `/app/data/uptime.db` |
| Mount | `./data:/app/data` in `/root/glance/compose.yml` |
| Format | SQLite, one row per service per minute |
| Growth | one row per service per minute, pruned to the longest configured window |

The file is the only copy of history. Back it up together with
`config/glance.yml`. Deleting it discards the bar's history, not the dashboard.

### Retention

Rows older than the longest configured `uptime.range` are pruned hourly in a
background goroutine, so a slow delete cannot stall a widget refresh. When no
widget enables uptime, a 30-day window governs retention.

### Configuration

```yaml
- type: monitor
  uptime:
    enabled: true
    range: 30d      # 1h to 90d; defaults to 30d when omitted
```

Recording happens whether or not the bar is shown, so enabling `uptime` later
reveals the history accumulated while it was off. A range outside 1h to 90d is
rejected at startup with an error naming the bounds.

The bar always renders 60 buckets; the bucket width is the range divided by 60.
Each bucket is coloured by success rate: green when every check succeeded,
yellow at 90 percent or above, red below 90 percent, grey when the bucket holds
no checks. An empty bar therefore reads as no data rather than as a perfect
record.

### First deployment

A freshly deployed instance shows mostly grey buckets. They fill in as checks
accumulate, so a 30-day bar is fully populated only after 30 days.

## Agent authentication

Every statistics agent requires an `Authorization: Bearer <token>` header. An
agent with no configured token refuses to start, so a missing token surfaces as
a container that exits rather than one that serves data openly.

Each host has its own token. Tokens live in each host's
`/root/glance-agent/docker-compose.yml` under the `TOKEN` environment variable,
and the matching value appears in `/root/glance/config/glance.yml`. Do not copy
a token between hosts: a leak should identify a single host.

To rotate a token, edit the agent's compose file, recreate the container, then
update the matching `token:` entry in the dashboard config and restart it. The
two sides must agree; a mismatch blanks the affected widget without an error.

## Per-host exposure

| Host | Agent address | Reachable from | Notes |
|---|---|---|---|
| rn1 (dashboard host) | `172.23.0.1:52763` | containers on the bridge | Bound to the Docker bridge gateway. The dashboard container runs on that bridge, so its view of loopback is itself, not the host. |
| rn2 | `107.173.39.135:53847` | `192.3.247.41` only | ufw rule `glance-agent from rn1`. |
| hdz | `23.80.83.100:52763` | `192.3.247.41` only | ufw rule `glance-agent from rn1`. |

The dashboard reaches every agent over plaintext HTTP. Authentication plus a
source restriction is the control in place; there is no TLS on these ports.

## Container inventory permissions

The dashboard reads the Docker socket, which is owned by `root` with group
`docker` (gid 989 on this host). The container therefore needs that group:

```yaml
services:
  glance:
    group_add:
      - "989"
```

Without it the container starts normally but the container-inventory widget
renders empty, with no error in the log.

## Files on the host

| Path | Purpose |
|---|---|
| `/root/glance/compose.yml` | Container definition, mounts, socket group |
| `/root/glance/config/glance.yml` | Dashboard configuration with agent tokens |
| `/root/glance/data/uptime.db` | Monitor availability history |
| `/root/glance-agent/docker-compose.yml` | Agent definition and its token |
| `*.bak.<timestamp>` | Previous copies kept for rollback |

## Rollback

Revert both sides together. The agent change fails closed, so restoring only the
dashboard leaves the agents rejecting every request and blanks all agent-backed
widgets. Restore each `docker-compose.yml` backup, recreate the agents, then
restore the dashboard config backup and restart it.

Reverting to an image built before uptime history means the database file is
inert; it can be deleted or left in place.
