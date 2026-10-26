# Proposal

## Why

Three `glance-agent` instances expose host statistics and full container inventories over plaintext HTTP on public addresses, and none of them authenticate. The agent's authorization check is gated on `config.Server.Token != ""`, so an unset token silently disables it: anyone who can reach the port receives hostname, platform, boot time, disk usage, every container name, and every image tag without credentials. That inventory is reconnaissance-grade — it names GitLab, a Grok proxy, a subscription API, and their databases.

The dashboard cannot simply turn the token on, because two of the three consumers never send one. Only `server-stats` has a `Token` field and emits an `Authorization` header; the `monitor` widget's container-usage lookup and the `docker-containers` widget's agent path both take a bare URL. Enabling authentication today would silently blank those widgets rather than fail loudly.

The third host, `hdz`, is also missing its agent entirely — it was removed by accident, so that column reports nothing at all.

## What Changes

- Make the agent's token mandatory rather than optional: a missing or empty token becomes a startup error instead of an open door.
- Add token support to the two dashboard code paths that lack it, so every agent request carries `Authorization: Bearer`.
- Configure a distinct token on each of the three agents and the matching token on every `agent-url` in the dashboard configuration.
- Bind each agent so that only its intended client reaches it: loopback on the dashboard host, and a firewall-restricted public port on the two remote hosts.
- Reinstall the agent on `hdz` and add it back to the dashboard configuration alongside `rn1` and `rn2`.
- Correct the existing deployment requirement that forbids an agent on the dashboard host, since one is deployed there deliberately.

**BREAKING**: after this change an agent started without a token refuses to serve. Any client, script, or widget that reaches an agent without the correct `Authorization` header receives 401.

## Capabilities

### New Capabilities

None. All requirements belong to the existing deployment-operations capability.

### Modified Capabilities

- `deployment-operations`: adds requirements for agent authentication, agent network exposure, and per-host agent coverage; replaces the requirement that forbids an agent on the dashboard host.

## Impact

- `internal/glance/widget-monitor.go` — `fetchAgentContainerUsage` and the widget's `agent-url` field gain token support.
- `internal/glance/widget-docker-containers.go` — `fetchContainersFromAgent` and the widget's `agent-url` field gain token support.
- `internal/glance/widget-server-stats.go` — unchanged; it already sends the header.
- Agent source (`github.com/glanceapp/agent`, forked as `svefnz/agent`) — token becomes required; configuration template must always emit one.
- Host `rn1` (dashboard host) — agent token configured, agent bound to loopback.
- Host `rn2` — agent token configured, public port restricted to the dashboard host's address.
- Host `hdz` — agent reinstalled, token configured, public port restricted to the dashboard host's address.
- VPS `/root/glance/config/glance.yml` — `token` added under each `agent-url` consumer.
- Operational: each host's agent requires its own restart; a token mismatch blanks a widget silently, so verification must check rendered values rather than absence of errors.
