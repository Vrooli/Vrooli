# macOS validation attempt — 2026-08-25

## Result

No macOS qualification evidence was produced. The enrolled Intel Mac was
briefly offline during a local Bridge restart, then reconnected. It is not
currently an unregistered or revoked node. The live ladder still could not
start because Bridge refused the cleanup preflight before creating an operation:
the node has no active X25519 encryption credential. This is not a pass or a
skip-budget exception.

## R0 — Bridge reachability and node state

Commands:

```text
vrooli scenario start vrooli-bridge
/home/matthalloran8/.vrooli/bin/vrooli-bridge nodes list --json
```

Bridge started healthy on API port 18767. The node inventory reported:

```text
minimouse  darwin/amd64  NODE_STATUS_OFFLINE (transient during Bridge restart)
last_seen_at: 2026-08-25T14:17:52.066460Z (initial attempt)
registry_record_present: true
protocol_compatible: true
```

The Linux control-plane node `swarminator` was online, but it is not the
darwin/amd64 hardware target. A follow-up read-only check after the Bridge
restart reported `minimouse` online, heartbeat-fresh, channel-held,
protocol-compatible, and dispatchable; `nodes doctor` passed in that stable
window. The target therefore remains reachable, but the live ladder is still
not authorized to begin.

## R1–R9 — Not run

The following remain explicitly recorded as not run because the cleanup
preflight was blocked and no safe clean-host operation was created:

- cleanup tombstone and clean-host reset;
- first-run onboarding and passphrase-prompt count;
- pairing, service installation, and online heartbeat;
- reboot persistence for unattended credential read, Keychain, and LaunchAgent;
- remote `start`, `status`, and `stop` for `minimouse/system-monitor`;
- the Mac scenario suite and platform skip-budget measurement;
- structured qualified evidence attachment.

The current ledger still reports zero qualified cells for `darwin/amd64`.
No `qualified` evidence object was authored from this attempt, and no claim is
made for `darwin/arm64`.

## Follow-up preflight — 2026-08-25

The authoritative current node ID is
`25c7e426-c76c-421a-8351-aaf964589802`. Earlier diagnostic commands used a
transcribed ID ending in `4217`; those targeted the wrong identity and
returned the expected not-found result. The durable Machine lineage, registry
row, and current node list all agree on the `421a` identity.

Bridge startup initially failed after the Bridge UI/API rebuild because the
manifest's 15-second startup grace expired while the API completed schema,
catalog, key, and queue initialization. The Bridge service contract was
increased to a bounded 60-second grace period. Governed restart then completed
healthy in about 31 seconds, and the API log recorded `Starting server on port
18767`.

After restart, one `nodes list --json` observation briefly showed the Mac
offline while the agent reconnected. The next observation showed:

```text
minimouse  NODE_STATUS_ONLINE
registry_record_present: true
heartbeat_fresh: true
channel_held: true
protocol_compatible: true
dispatchable: true
```

The active Machine identity is durable and points to `minimouse.local`; its
lineage contains the current node and older superseded node IDs. The historical
cleanup record explicitly says no remote cleanup was executed. A read-only
cleanup preflight with the active Machine identity and exact target hostname
was refused before creating an operation because the node has no active X25519
encryption credential. The Ed25519 heartbeat credential remains valid; this is
a separate missing key and is why credential-grant synchronization also logged
`no active encryption credential`.

Typed relay itself is healthy: a read-only remote `system-monitor` status
completed and returned the Mac's lifecycle state. The requested scenario was
stopped because its required `agent-manager` dependency was stopped. A typed
start of `agent-manager` reached develop and launched both remote processes,
but its lifecycle operation remained `abandoned` in the `health` step while
waiting on the `workspace-sandbox` dependency; the five-minute relay deadline
was not reached by the caller, so a bounded five-second retry was used for the
diagnostic. A later 60-second typed start returned
`deadline_exceeded`; read-only status showed the operation still `starting`,
with `start-api` and `start-ui` running and no bound ports. The partially
started service was then stopped through the typed lifecycle command, and a
follow-up status showed `stopped` with zero processes. This is a remote
scenario-lifecycle blocker, not node-offline evidence. No cleanup, reboot,
re-onboarding, or qualification claim was made.

## Recovery condition

Resume the destructive C1 ladder only after the active encryption credential
has been registered (normally by restarting or re-onboarding the current
agent with current Bridge agent code), SSH/setup authority is available, the
`agent-manager` dependency can complete setup, and a stable Bridge check
reports `minimouse` online and dispatchable. Do not treat this document as
hardware qualification evidence.

## Recheck after toolchain repair — 2026-08-25 17:36Z

The authoritative Bridge check now reports the current node
`25c7e426-c76c-421a-8351-aaf964589802` as `NODE_STATUS_ONLINE` with a fresh
heartbeat, held channel, compatible protocol, and `dispatchable: true`;
`nodes doctor` passes. This confirms that the earlier offline observation was
a transient reconnect window during local Bridge restart, not an unregister or
revocation.

The fresh C1 producer validation ticket did not fail on the missing local
`vrooli` binary. Its collection diff started, but Test Genie admission became
saturated: five required members were deferred behind active run capacity, and
`structure-health` reported its provider unavailable. The local waiting client
was detached after more than six minutes without a terminal response; the
server-owned operation remains queued. This is validation infrastructure
evidence, not hardware qualification evidence.

## Current presence and local tooling recheck — 2026-08-25 18:59 EDT

The current Bridge readout for node
`25c7e426-c76c-421a-8351-aaf964589802` is online, heartbeat-fresh,
channel-held, protocol-compatible, and dispatchable; `nodes doctor` passes.
One immediate fleet-list read during Bridge restart reported an offline status
with a recent `last_seen_at`; the next doctor/list read updated it to online.
This is a transient Bridge reconnect/read-model window, not evidence that the
node was unregistered. The earlier sustained “offline” diagnosis was also
incorrect because it conflated the stopped remote `system-monitor` scenario
with node presence.

C1 remains unqualified. The active blockers are still the missing X25519
cleanup credential/operator confirmation, unavailable remote cleanup authority,
and the unresolved remote dependency-start path. The local CLI disappearance
that invalidated earlier validation is now attributed to the Storage Manager
retention path and is guarded as documented in the C2 evidence.

## Presence correction and remote lifecycle root cause — 2026-08-25 19:27 EDT

The current authoritative Bridge readout is still online and dispatchable:

```text
node: 25c7e426-c76c-421a-8351-aaf964589802 (minimouse)
status: NODE_STATUS_ONLINE
online: true
heartbeat_fresh: true
channel_held: true
protocol_compatible: true
dispatchable: true
```

This confirms the operator's suspicion. The earlier offline diagnosis was not
evidence that minimouse had been turned off or unregistered. Bridge had a
presence/read-model handshake window, and the status investigation also mixed
the stopped `system-monitor` scenario state into the node-presence conclusion.
The SSE handler now registers presence before acknowledging the stream, and
the focused channel/presence regression tests pass.

The lifecycle retry also separated a second false signal from the hardware
claim. The project CLI previously forwarded `scenario --timeout 600` to the
node while leaving the outer Bridge HTTP transport at its 120-second default.
That made a valid slow remote build look like a relay failure. The project CLI
and Bridge relay client now mirror the requested timeout at both layers, with
focused tests for the propagation. A durable dispatch reached the Mac after
that repair and returned the actual failure:

```text
scenario start failed: build component ui: exit status 1
Could not resolve "./features/logs/pages/LogsPage" from "src/App.tsx"
```

The local checkout contains that tracked file; the Mac checkout used by the
node does not. The remaining lifecycle failure is therefore stale/incomplete
remote source state, not node absence. C1 remains unqualified because the
cleanup/onboarding authority, source synchronization, reboot persistence,
credential-store persistence, and complete lifecycle ladder still need real
evidence. No destructive reset, reboot, or qualification record was performed
from this run.

## Source-ship repair and lifecycle confirmation — 2026-08-25 19:49 EDT

The missing remote UI file was traced to an ignore-rule collision, not to
hardware absence. Both `scenarios/system-monitor/.gitignore` and the nested UI
ignore file used an unanchored `logs` pattern. That pattern ignored
`ui/src/features/logs/pages/LogsPage.tsx`, even though `src/App.tsx` imports it.
The working-tree onboarding shipper honored the ignore rules and therefore
transferred an incomplete source closure to minimouse.

The rules now anchor only the runtime log directories (`/logs/` and `/logs`),
and the previously present `LogsPage.tsx` is included in the source closure.
The local UI build passes and produces the `LogsPage` chunk. A governed
working-tree re-onboarding for the existing machine transferred the corrected
closure, rebuilt the native macOS CLI, reinstalled the agent/helper, and
reached ONLINE. Its final onboarding handoff returned HTTP 404 after the
successful bootstrap; this remains a control-plane handoff defect, not a node
availability defect.

Durable remote lifecycle evidence after the repair:

```text
system-monitor scenario start   RUN_STATUS_PASSED
system-monitor scenario status  RUN_STATUS_PASSED
system-monitor scenario stop    RUN_STATUS_PASSED
```

This closes the previously observed remote build failure and confirms that the
node was online and dispatchable. It does not by itself claim the clean-host,
reboot, credential-store, or full C1 qualification ladder.

## Onboarding contract and headless setup findings — 2026-08-25 20:00 EDT

The first post-repair onboarding retry reached pairing, service installation,
autostart, and online confirmation, then failed only because `--skip-setup`
left the remote `vrooli-onboarding` CLI unavailable for the final selection
step. This was a setup-mode mismatch, not an offline node.

The corrected retry enabled setup with `minimal`, no optional resources, and no
optional scenarios. It reached the real host setup phase, where the
`emergency_watchdog` safeguard attempted to load a user LaunchAgent from an
SSH/headless macOS session and returned `launchctl bootstrap: exit status 5`.
That failure aborts setup before the onboarding CLI is installed. The safeguard
now classifies an unavailable GUI launchd domain as manual/degraded so it cannot
abort unrelated pairing and service installation. Darwin cross-compilation and
the focused safeguard tests pass; a fresh governed onboarding run is still
required to verify the remote result.

Separately, Bridge's onboarding handoff was corrected to append the stable
`POST /api/v2/handoff` route to the discovered onboarding scenario base URL.
The focused client regression test passes. The earlier HTTP 404 was therefore a
real Bridge URL-construction defect, not node unavailability.

C1 remains unqualified. The clean reset, reboot persistence checks, full suite,
and structured qualified evidence record are still outstanding.

## Final governed onboarding convergence attempt — 2026-08-25 20:24 EDT

Operation `7058d519-236f-4e79-b47a-2be7793cc316` transferred the corrected
working tree and passed every bootstrap rung through setup, native CLI build,
pairing, service install, LaunchDaemon autostart, and final ONLINE confirmation.
The terminal Bridge node state was:

```text
readiness: PASS
online=true channel=true heartbeat=true protocol=true dispatchable=true
node: 25c7e426-c76c-421a-8351-aaf964589802 / minimouse / darwin/amd64
```

The onboarding CLI was found through the runtime-home path, auto-started the
remote onboarding API, and reached the real readiness check. The remaining
terminal result was `configuration is not complete: 6 blocking item(s) remain`.
That is an operator/credential readiness blocker, not a transport or liveness
failure. The separate lifecycle ladder still passed start and status with
`health_status=degraded`, then stop passed; the node returned to `stopped`.

C1 is not qualified: no clean-reset tombstone, reboot persistence evidence,
passphrase count, full suite result, or qualified ledger record exists yet.

The remaining remote command gate is also explicit. Bridge derives
`system-monitor metrics devices` from the system-monitor CLI manifest, but a
live relay attempt was refused with missing `system-monitor:read`. This is a
grant/readiness issue after the node was already proven online and
dispatchable; it is not evidence of an offline Mac or a missing allowlist
entry.

## Presence, readiness, and lifecycle correction — 2026-08-26 01:04–01:20 EDT

The current authoritative node check remains a pass:

```text
node: 25c7e426-c76c-421a-8351-aaf964589802 (minimouse)
status: NODE_STATUS_ONLINE
online: true
heartbeat_fresh: true
channel_held: true
protocol_compatible: true
dispatchable: true
```

This directly resolves the repeated offline concern: minimouse was not turned
off, unregistered, or revoked. The false state came from the Bridge
presence/read-model handshake window, and the SSE registration ordering has
been repaired. Remote scenario state is now recorded separately from node
presence.

The latest governed working-tree convergence was operation
`b07ee77f-f1e3-4e7a-bcab-67f61827c4c5`. It passed source sync, native
darwin/amd64 CLI installation, pairing, service installation, LaunchDaemon
autostart, and online confirmation. Its readiness failure now retains safe
metadata instead of only a generic message:

```text
credential vrooli/openrouter:api-key — credential backend could not answer
credential vrooli/postgres:password — credential backend could not answer
host autoheal_watchdog — safeguard missing on this host
host tpm_credential_access — safeguard unsupported on this host
host workspace_sandbox_userns — safeguard missing on this host
```

The same result was reproduced by operation
`8ec93d3f-ceac-41f1-951e-76aaa932970b` after requesting the system-monitor
scenario CLI. These are real onboarding-readiness blockers, not node-offline
evidence. The readiness and wizard CLIs now include only this metadata in
process error text, so Bridge preserves blocker names and reasons without
exposing credential values.

The typed relay investigation found and fixed a separate argv bug: the
allowlisted `system-monitor metrics devices` verb was routed through the root
`vrooli` command even though it is a scenario-owned CLI verb. The node agent
now translates scenario-owned verbs to their scenario CLI directly, with a
regression test. The current Mac still lacks the `system-monitor` executable
on its service PATH because this headless setup did not complete scenario CLI
installation; that is an install/lifecycle prerequisite, not a presence
failure.

The exact remote lifecycle retry remains blocked at dependency orchestration:

```text
vrooli scenario start minimouse/system-monitor --best-effort --timeout 120 --json
status: stopped
operation: abandoned
current_step: dependencies
dependency_current: agent-manager
```

Remote `agent-manager` is present but unhealthy/abandoned while its required
`workspace-sandbox` dependency is unresolved. This is the next lifecycle
repair target. C1 remains unqualified: no cleanup reset, reboot persistence,
full scenario suite, skip-budget measurement, or qualified ledger record has
been claimed.

## Corrected dependency and artifact evidence — 2026-08-26 03:00 EDT

Later remote checks corrected the earlier interpretation of the dependency
state. `agent-manager` is running, healthy, and fresh; `workspace-sandbox` is
running, healthy, and fresh; and `system-monitor` reports fresh artifacts. The
failed `system-monitor` starts are abandoned while waiting at dependency
`agent-manager`, despite a direct `agent-manager` start/reuse succeeding:

```text
system-monitor startop-0fc11e4c64fd4adab546d1ef7f4bdb6d  abandoned at dependencies/agent-manager after 36s
system-monitor startop-e4f9bd51a3675b4322b5ae3d0d66764e  abandoned at dependencies/agent-manager after 152s
agent-manager                                  running / healthy / fresh
workspace-sandbox                              running / healthy / fresh
system-monitor                                 stopped / fresh
```

This is a lifecycle dependency-arbitration or stale ownership/lock blocker,
not evidence that minimouse is offline. The C1 lifecycle gate remains
unqualified until this control-plane path is repaired and one bounded start,
wait, status, and stop sequence completes.
