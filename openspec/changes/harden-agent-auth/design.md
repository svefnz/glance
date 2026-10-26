# Design

## Context

See proposal.md - Why.

The agent is `github.com/glanceapp/agent`, forked locally as `svefnz/agent`. Its entire HTTP surface is three handlers in `internal/agent/server.go`:

| Endpoint | Data | Source |
|---|---|---|
| `/api/sysinfo/all` | hostname, platform, boot time, CPU load, memory, swap, mountpoints | `sysinfo.Collect`, cached 1s |
| `/api/containers` | id, name, image, state, status, cpu_percent, memory_*, per running container | Docker API via socket or TCP, 10s timeout per stats call |
| `/api/healthz` | empty body | constant |

All three call a shared `isAuthorized` closure, which short-circuits when `config.Server.Token == ""`. The comparison itself already uses `subtle.ConstantTimeCompare`, so the primitive is sound — only its activation is conditional.

On the dashboard side the three consumers differ in what they support today:

| Consumer | YAML field | Sends `Authorization`? |
|---|---|---|
| `server-stats` (`widget-server-stats.go:107`) | `token` on each `servers[]` entry | yes, when non-empty |
| `monitor` container usage (`widget-monitor.go:258`) | none | no — function takes only `agentURL` |
| `docker-containers` (`widget-docker-containers.go:601`) | none | no — function takes only `agentURL` |

Deployment topology, established by inspection rather than assumption:

- `start.lll.rest` resolves to Cloudflare, forwards through Caddy to the `glance` container on `192.3.247.41` (rn1). The response carries `via: 1.1 Caddy`.
- rn1 also runs `glance-agent` on `:52763`. So the "local" host is reached over the same agent HTTP interface as the remote ones — there is no local `sysinfo.Collect` path in use, because the widget is configured with `type: remote`.
- rn2 (`107.173.39.135:53847`) runs a reachable agent.
- hdz (`23.80.83.100:52763`) has no reachable agent; all of its ports are filtered from an outside client, while rn1 can reach it over SSH. The agent was removed from it accidentally.

The existing main spec forbids an agent on the dashboard host ("Unused statistics agent is not deployed"), which contradicts the deployed reality. That requirement is modified rather than deleted, so its original scenario name is preserved.

## Goals / Non-Goals

**Goals:**

- A token-less agent cannot serve data: fail at startup rather than fall open.
- Every dashboard→agent request carries its token, including the two paths that currently do not.
- Each agent is reachable only from the dashboard host, enforced at the network layer rather than relying on the token alone.
- hdz rejoins the dashboard as a normal remote agent source, configured exactly like rn2.

**Non-Goals:**

- TLS between dashboard and agents. The ports stay plaintext; authentication plus source restriction is the chosen control, and adding TLS would require certificate distribution across three hosts for a channel that never leaves the operator's own machines.
- Replacing the agent with a different collection mechanism.
- Rotating the dashboard's own auth secret or user password.
- Rate limiting or audit logging on the agent.

## Decisions

### Token becomes mandatory, and startup fails without it

`isAuthorized` currently guards on `config.Server.Token != ""`. Removing that guard is not enough — an empty token would then compare against `"Bearer "` and reject everything, which fails closed but produces a confusing 401 rather than a clear operational error. Instead, config validation rejects an empty token before the listener starts.

Alternative considered: keep the token optional and rely on the firewall. Rejected — a firewall misconfiguration silently reopens the endpoint, whereas a mandatory token makes the insecure state impossible to reach by omission.

### Token is passed explicitly to the two functions that lack it

`fetchAgentContainerUsage` and `fetchContainersFromAgent` each gain a `token string` parameter, mirroring the pattern already used in `fetchRemoteServerInfo`. Their widgets gain a `token` YAML field beside `agent-url`.

Alternative considered: a shared `agentClient` struct holding URL plus token. Rejected as premature — there are three call sites and no behavior to share beyond setting one header. A struct would be a refactor the change does not need.

### Header is set only when the token is non-empty on the dashboard side

The agent enforces the token; the dashboard tolerates an absent one. This keeps the existing loopback/local development path working without requiring a token file, while production configuration always supplies one. The asymmetry is deliberate: the enforcement boundary is the agent.

### Binding and exposure per host

| Host | Agent bind | Reachable by |
|---|---|---|
| rn1 (dashboard host) | loopback | the dashboard container on the same host |
| rn2 | public interface, firewall-limited to rn1's address | rn1 only |
| hdz | public interface, firewall-limited to rn1's address | rn1 only |

The operator's rule — local host uses `127.0.0.1`, remote hosts use the public address — maps directly onto this. hdz is treated as an ordinary remote host, exactly like rn2; the SSH jump used for administrative access plays no part in the data path.

Note that rn1's agent being loopback-bound requires the dashboard container to share the host network namespace, or the agent must be reachable from the container's view of loopback. This is verified during implementation rather than assumed.

### Distinct token per host

Each host gets its own generated token. A shared token would mean one leak compromises all three, and per-host tokens make it possible to identify which host a leaked credential belongs to.

## Risks / Trade-offs

- **A token mismatch blanks a widget silently** → The failure mode of `monitor` and `docker-containers` on a bad response is an empty value, not a visible error. Verification must assert on rendered container counts and CPU/memory figures, not merely on the absence of error styling.
- **rn1's agent on loopback may be unreachable from the dashboard container** → If the container runs on a bridge network, `127.0.0.1` inside it is the container, not the host. Mitigation: verify the connection during implementation; if it fails, either move the dashboard to host networking or bind the agent to the docker bridge gateway and firewall it to that interface.
- **Firewall rules can lock out the operator** → Apply rules that allow rn1's address and the operator's administrative path before restricting, and verify the dashboard still reads each host before considering the host done.
- **hdz's agent has to be reinstalled without interactive access to its ports** → Access is via the SSH jump; the install is performed over that channel. If the host cannot reach the image registry, the image must be transferred over the same channel.
- **Removing the token's optionality is a breaking change for any other consumer** → The agent is single-purpose in this deployment; the only known clients are the three dashboard paths. Acceptable, and stated as breaking in the proposal.

## Migration Plan

1. Ship the agent change first (token mandatory), then the dashboard change (token on all three paths). Deploying the dashboard change first would leave it sending tokens to an agent that ignores them — harmless but pointless.
2. Bring up each agent with its token and its binding, one host at a time, verifying that host's widget data before moving on. Start with rn1 since it also proves the loopback question.
3. Reinstall hdz's agent last, since it depends on the SSH jump and may need an image transfer.
4. Update `/root/glance/config/glance.yml` with the three tokens and restart the dashboard.
5. Rollback: revert `glance.yml` to its timestamped backup and restart the dashboard; revert each agent's image tag to the previous build. Because the agent change fails closed, a rollback that restores only the dashboard leaves agents rejecting requests — so roll back agents and dashboard together.

## Open Questions

None that affect the specs, approach, or task breakdown. The loopback reachability question is resolved during implementation and has a documented fallback in Risks.
