# Tasks

## 1. Agent: token becomes mandatory

- [ ] 1.1 Remove the `config.Server.Token != ""` guard from `isAuthorized` in `internal/agent/server.go` so the comparison always runs. Verify by starting an agent with a token and confirming an unauthenticated request returns 401 rather than data.
- [ ] 1.2 Add startup validation in `internal/agent/config.go` that exits with a clear error when the token is empty or absent, for both the file and environment-variable configuration paths. Verify by starting an agent with no token and no `TOKEN` environment variable, and confirming the process exits non-zero without binding a port.
- [ ] 1.3 Update `internal/install/templates/agent.yml` so a token is always emitted rather than only when `RandomAuthToken` is set, and confirm the rendered template contains a `token:` line with a non-empty value.
- [ ] 1.4 Add a test covering the three authorization outcomes (no header, wrong token, correct token) and verify it passes with `go test ./internal/agent/`.

## 2. Dashboard: every agent consumer sends its token

- [x] 2.1 Add a `token` YAML field beside `agent-url` on the monitor widget and pass it into `fetchAgentContainerUsage` in `internal/glance/widget-monitor.go`, setting `Authorization: Bearer` when non-empty. Verify with a test that asserts the outgoing request carries the header.
- [x] 2.2 Add a `token` YAML field beside `agent-url` on the container-inventory widget and thread it through `fetchContainersFromAgent` in `internal/glance/widget-docker-containers.go`, setting `Authorization: Bearer` when non-empty. Verify with a test that asserts the outgoing request carries the header.
- [x] 2.3 Confirm `server-stats` already sends the header and needs no change; verify by reading `fetchRemoteServerInfo` and noting the existing `Token` field and header assignment.
- [x] 2.4 Run `go build ./...` and `go test ./internal/glance/` and verify both succeed.

## 3. Host rn1 (dashboard host)

- [ ] 3.1 Generate a distinct token for rn1 and write it into the agent's configuration on the host. Verify by restarting the agent and confirming it stays up.
- [ ] 3.2 Bind the rn1 agent to loopback. Verify that the dashboard container can still reach it at `127.0.0.1:52763`; if it cannot, apply the documented fallback (host networking, or bridge-gateway binding plus firewall) and record which was used.
- [ ] 3.3 Confirm the rn1 agent rejects an unauthenticated request and accepts one carrying its token. Verify with two curl calls against the agent's address.

## 4. Host rn2

- [ ] 4.1 Generate a distinct token for rn2 and write it into the agent's configuration on the host. Verify by restarting the agent and confirming it stays up.
- [ ] 4.2 Restrict the agent's public port so only rn1's address reaches it, leaving the operator's administrative path open. Verify that a request from a different source is refused while rn1's succeeds.
- [ ] 4.3 Confirm the rn2 agent rejects an unauthenticated request and accepts one carrying its token. Verify with two curl calls.

## 5. Host hdz

- [ ] 5.1 Reinstall the agent on hdz over the SSH jump, using the image that includes the mandatory-token change. Verify the container is running and the port is listening.
- [ ] 5.2 Generate a distinct token for hdz and write it into the agent's configuration. Verify by restarting the agent and confirming it stays up.
- [ ] 5.3 Restrict the agent's public port so only rn1's address reaches it. Verify that a request from a different source is refused while rn1's succeeds.
- [ ] 5.4 Confirm the hdz agent rejects an unauthenticated request and accepts one carrying its token. Verify with two curl calls.

## 6. Dashboard configuration

- [ ] 6.1 Back up `/root/glance/config/glance.yml`, then add the matching token under every `agent-url` consumer: the `server-stats` servers, the three `monitor` widgets, and any container-inventory widget using an agent. Verify no `agent-url` is left without a token.
- [ ] 6.2 Restart the dashboard and verify it starts without configuration errors.

## 7. Integration verification

- [ ] 7.1 Verify each of the three hosts reports CPU and memory figures in the server-statistics widget, not blank values.
- [ ] 7.2 Verify each of the three hosts reports container CPU and memory alongside its monitor sites, confirming the monitor widget's token path works.
- [ ] 7.3 Verify the container-inventory widget lists containers with live state, confirming its token path works.
- [ ] 7.4 Verify that an unauthenticated request to each agent port from a source other than rn1 is refused, and that each agent returns 401 without a token.
- [ ] 7.5 Update the deployment notes with the token locations and the per-host exposure model, and verify the notes name all three hosts.

## Workflow follow-up

- Archive this change once every task above is complete, so the deployment-operations delta is merged into the main spec.
