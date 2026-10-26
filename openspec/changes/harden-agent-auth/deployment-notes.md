# Deployment notes

## Agent authentication

Every statistics agent now requires an `Authorization: Bearer <token>` header.
An agent with no configured token refuses to start, so a missing token surfaces
as a container that exits rather than one that serves data openly.

Each host has its own token. Tokens live in each host's
`/root/glance-agent/docker-compose.yml` under the `TOKEN` environment variable,
and the matching value appears in `/root/glance/config/glance.yml` on the
dashboard host. Do not copy a token between hosts: a leak should identify a
single host.

To rotate a token, edit the agent's compose file, recreate the container, then
update the matching `token:` entry in the dashboard config and restart it. The
two sides must agree; a mismatch blanks the affected widget without an error.

## Per-host exposure

| Host | Agent address | Reachable from | Notes |
|---|---|---|---|
| rn1 (dashboard host) | `172.23.0.1:52763` | the dashboard container only | Bound to the Docker bridge gateway. The dashboard container runs on that bridge, so its view of loopback is itself, not the host. |
| rn2 | `107.173.39.135:53847` | `192.3.247.41` only | ufw rule `glance-agent from rn1`. |
| hdz | `23.80.83.100:52763` | `192.3.247.41` only | ufw rule `glance-agent from rn1`. |

The dashboard reaches every agent over plaintext HTTP. Authentication plus a
source restriction is the control in place; there is no TLS on these ports.

## Why rn1 is not loopback

The dashboard container runs on a Docker bridge network, so `127.0.0.1` inside
it refers to the container. Binding the agent to `127.0.0.1` on the host would
make it unreachable from the dashboard. It binds `172.23.0.1` instead, which is
the bridge gateway and therefore not reachable from outside the host.

## Files on the host

| Path | Purpose |
|---|---|
| `/root/glance-agent/docker-compose.yml` | Agent definition and its token |
| `/root/glance-agent/docker-compose.yml.bak.<timestamp>` | Previous agent definition |
| `/root/glance/config/glance.yml` | Dashboard configuration with matching tokens |
| `/root/glance/config/glance.yml.bak.<timestamp>` | Previous dashboard configuration |

## Rollback

Revert both sides together. The agent change fails closed, so restoring only the
dashboard leaves the agents rejecting every request and blanks all agent-backed
widgets. Restore each `docker-compose.yml` backup, recreate the agents, then
restore the dashboard config backup and restart it.
