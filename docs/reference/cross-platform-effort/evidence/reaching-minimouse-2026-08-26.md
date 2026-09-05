# Reaching minimouse baseline and delivery evidence

> Historical evidence snapshot. The `vrooli-bridge:session` references in this file describe the pre-2026-08-29 transport contract.

The screenshots named below are preserved under their original
`docs/reference/cross-platform-effort/evidence/` paths in the
[documentation preservation archive](../../../internal/PROGRESS.md#documentation-capture-preservation--2026-09-05).
They are historical captures, not live machine-state claims.

Date: 2026-08-27 00:36 UTC

Fleet under test: Linux control plane (`swarminator`) and paired Darwin/amd64
agent (`minimouse`). The `minimouse` enrollment is production data and was not
re-paired or re-onboarded.

## Pairing anchor

`minimouse` node id: `25c7e426-c76c-421a-8351-aaf964589802`

This identifier is the invariant for every later probe.

## Baseline live probes

Command: `vrooli scenario status minimouse/system-monitor --json`

Result: exit 0. The root verb returned a successful relay response for
`system-monitor` with status `stopped` on the node.

Command: `vrooli-bridge relay call --json --node-id 25c7e426-c76c-421a-8351-aaf964589802 --scenario system-monitor --command "system-monitor metrics current" --args "--json"`

Correlation id: `03eccc45-b3c1-4a7e-b171-facdb2c09710`

Result: the relay returned `RELAY_CALL_OUTCOME_FAILED`, exit code `127`, with
`exec: "system-monitor": executable file not found in $PATH`.

The complete node list output is preserved here from the baseline command:

```json
{
  "nodes": [
    {
      "id": "25c7e426-c76c-421a-8351-aaf964589802",
      "name": "minimouse",
      "os": "darwin",
      "arch": "amd64",
      "revision": "3e5bccc5cd98db3d99d4b9fcb22d26707f3c55a1+dirty",
      "scopes": ["vrooli-bridge:read", "vrooli-bridge:write", "*:read", "*:write"],
      "status": "NODE_STATUS_ONLINE",
      "online": true,
      "created_at": "2026-08-17T15:39:06.606107059Z",
      "updated_at": "2026-08-26T03:29:25.277213815Z",
      "last_seen_at": "2026-08-26T20:28:26.496469Z",
      "registry_record_present": true,
      "heartbeat_fresh": true,
      "heartbeat_age_seconds": "7",
      "channel_held": true,
      "protocol_compatible": true,
      "dispatchable": true,
      "kind": "NODE_KIND_AGENT"
    },
    {
      "id": "697b6224-6283-4a31-90e2-73724e424c05",
      "name": "swarminator",
      "os": "linux",
      "arch": "amd64",
      "status": "NODE_STATUS_ONLINE",
      "online": true,
      "created_at": "2026-08-10T15:55:49.348005106Z",
      "updated_at": "2026-08-25T01:46:52.819226024Z",
      "last_seen_at": "2026-08-19T18:42:21.589036906Z",
      "registry_record_present": true,
      "heartbeat_fresh": true,
      "protocol_compatible": true,
      "dispatchable": true,
      "kind": "NODE_KIND_CONTROL_PLANE"
    }
  ]
}
```

Bridge health at the baseline was healthy on `127.0.0.1:18767`.

After the governed Bridge restart, the stale control-plane record was no
longer treated as live. The same `swarminator` record remained present with
its last heartbeat at `2026-08-19T18:42:21.589036906Z`, `heartbeat_fresh=false`,
and no dispatchable status; `minimouse` remained the same node id and returned
`online=true`, `heartbeat_fresh=true`, `channel_held=true`, and
`dispatchable=true`. The registry regression tests also cover stale and
missing-heartbeat control-plane records.

The restarted Web Console catalog exposed the same truth through its public
target projection: `minimouse` was `TARGET_STATE_DISPATCHABLE` with Darwin/
amd64 facts, a current `lastSeenAt`, and all five readiness facts passed;
`swarminator` was `TARGET_STATE_OFFLINE` with its old `lastSeenAt`, no online
or dispatchable value, and failed heartbeat/channel/dispatch readiness facts.

Relevant catalog output:

```text
minimouse  nodeId=25c7e426-c76c-421a-8351-aaf964589802  os=darwin arch=amd64
  online=true dispatchable=true state=TARGET_STATE_DISPATCHABLE
swarminator nodeId=697b6224-6283-4a31-90e2-73724e424c05 os=linux arch=amd64
  online=null dispatchable=null state=TARGET_STATE_OFFLINE
  lastSeenAt=2026-08-19T18:42:21.589036906Z
```

## Baseline suite status

The immutable Git Control Tower collection was created under
`reaching-minimouse-one-node-client-a-resolvable-node-cli-baseline`. Generation
7 was explicitly re-anchored after the earlier capture was invalidated. It
finished with five members ready, one member pending, and one failed:
`scenario-to-ios` failed before any phase result because its lifecycle start
encountered unrelated stale builds and dependency-lock contention in the
pre-existing dirty worktree. This is a pre-existing failure, not a result of
this plan:

`start target scenario scenario-to-ios: exit status 1; build component ui: exit status 1`

The detailed server error names stale `browser-automation-studio`,
`backdrop-studio`, `scenario-authenticator`, and `vrooli-bridge` build inputs.
The failed collection is retained as degraded baseline evidence; it is not
represented as a passing suite. Generation 7 collection details: deployment-
manager, notification-hub, program-runtime, system-monitor, and web-console
ready; scenario-to-ios failed; vrooli-bridge remained queued when the
collection became terminal.

Plan Manager's explicit re-anchor produced generation 8 under the same
collection name. It terminally recorded deployment-manager, notification-hub,
and program-runtime as ready; scenario-to-ios failed before a Test Genie phase
result; and system-monitor, vrooli-bridge, and web-console remained pending when
the collection became terminal. The collection is therefore still a partial
baseline and is not presented as a clean full-suite comparison.

## Validation record

- `packages/cliresolve`: `go test ./...` passed.
- `packages/nodeclient`: `go test ./...` passed.
- The root CLI owner-session regression is fixed: `packages/nodeclient` now
  accepts a token provider and preserves the `LocalSession` scheme, while
  `internal/cli/vroolicli` resolves the enrolled local operator session. The
  focused root validation and `go build ./cmd/vrooli` passed before the later
  unrelated dirty-worktree import cycles; the current root package rerun is
  recorded below as setup-blocked.
- `scenarios/vrooli-bridge/agent`: full tests, `go vet`, and the six-target
  Linux/Darwin/Windows matrix passed before the final qualifier normalization;
  focused channel and executor tests pass after it, including the regression
  that keeps root `scenario ...` verbs on the native `vrooli` binary.
- `scenarios/vrooli-bridge/cli`: `go test ./...` passed; the comprehensive
  scenario run passed the contracts phase after the manifest repair.
- `scenarios/vrooli-bridge/api`: focused pairing, registry, gate, and handler
  tests passed; the broad suite was externally terminated after exercised
  packages passed.
- `scenarios/web-console/api`: `go test ./...` passed. Focused UI machine,
  launcher, session, and wrapper tests passed (73 tests); the full UI run has
  one pre-existing locale-catalog parity failure. The new launcher-focused
  run passes 37 tests and UI type-check/strings-check passes. The opt-in live
  session probe had previously passed against the running service; a later
  source rerun was blocked during package setup by the pre-existing root
  module import cycle, before the test body. The running-service proof remains
  a production Connect session plus browser-facing WebSocket session with
  `uname -a` containing Darwin, resize `100x30`, and `stty size` reporting
  `30 100`, followed by a successful Delete response.
- `scenarios/system-monitor/api`: `go test ./...` passed. Focused UI monitor
  and API-fetch tests passed (11 tests); UI type-check passed.
- The system-monitor remote-metrics regression test now asserts that the shared
  node client sends the fully qualified `system-monitor metrics current` verb,
  and the focused handler/services suites pass.
- Test Genie run `20260826-232942-a034eeef` reached a terminal `FAIL` verdict
  with 18/22 phases passed. The four failures were dependencies, unit,
  workflow, and experience standing checks; the run did not report a
  nodeclient or qualified-verb contract failure.
- delivery-ramp, notification-hub, deployment-manager, scenario-to-ios, and
  secrets-manager targeted Go suites passed.

The Bridge comprehensive run `20260826-225938-fcf3b5ab` passed contracts and
returned 14/20 phase passes. Its six failed phase results expose existing
maturity or environment findings across architecture metadata, PWA offline,
API hygiene, dependencies, quality, docs, unit, storage, workflow, business,
tidiness, security, measures, proto, and branding standing; none is a
nodeclient contract failure.

The current local reruns of `go test ./internal/cli/vroolicli` and the Web
Console API package setup are blocked by unrelated root-module import cycles
in the pre-existing dirty worktree. `packages/nodeclient` and the Bridge
registry tests still pass independently.

## Before/after mechanical checks

The baseline counts are intentionally recorded after the first source scan and
the final counts are filled only after implementation is complete. This keeps
absence claims auditable rather than inferred from a partial scan.

| Check | Baseline | Final |
| --- | ---: | ---: |
| active Bridge endpoint reads by consumers | 6 private wirings | 0; nodeclient owns resolution (program-runtime private sidecars are documented separately) |
| `bridgeNodeList` definitions | 1 | 0 |
| `bridgeRelayEnvelope` definitions | 1 | 0 |
| `shellMetachars` definitions | 2 | 1 shared validation implementation (the remaining hit is a bootstrap comment) |
| `remoteTerminalTarget` definitions | 1 | 0; targetmodel owns the model |
| mDNS browser/responder implementations | 1 hand-rolled browser | 2 shared mdns-go types |
| relay argument `strings.Join` calls | 2 competing encoders | 0 |
| system-monitor machine/node integration | 0 | API + UI machine axis and remote polling |

## Final live validation status

The latest governed redeploy used the existing Machine identity and retained
the node id. It installed the prebuilt Darwin/amd64 agent and left the node
ONLINE. It also proved the native CLI and scenario resolver paths were active.
The root CLI owner-session fix then made a live addressed status call return
valid remote JSON for `minimouse/system-monitor`, with the expected stopped
state; the prior unauthenticated transport failure is no longer reproduced.

The latest root status call returned exit 0 with `success=true`,
`scenario.name=system-monitor`, and `scenario.status=stopped` before the
recovery retry. The rebuilt root CLI was then installed over SSH with the
existing user-scoped binary path; the local and remote SHA-256 was
`5cc3cd0214806c6e570e0f9feba7b9d6dcc7595611265be8bf7d559f544cf356`.

The bounded remote start subsequently returned `success=true` and the
intended degraded result: `system-monitor` became healthy while optional
`agent-manager` and `ollama` remained failed. Operation id:
`startop-29be81f1649407c3bdd4f2900accac0a`.

The direct metrics relay then returned correlation
`4a26a43a-d59d-4271-bf30-b0ed4a90f399`, outcome
`RELAY_CALL_OUTCOME_COMPLETED`, and a 1047-byte typed JSON payload. The
payload is Darwin-owned and honest about first-sample state: CPU, memory,
connections, GPU, and disk each carry `not_yet_sampled_reason`; swap traffic,
major faults, and fragmentation index carry measured zero values.

The production Web Console endpoint was also exercised without the Go test
binary: the browser-facing payload capture reported
`status=200; browser_payload_credentials=none; uname=Darwin; resize=100x30;
stty=30 100`, and the cleanup request returned `cleanup_status=200`.

```text
LIVE_SESSION status=200; browser_payload_credentials=none; uname=Darwin; resize=100x30; stty=30 100
LIVE_SESSION cleanup_status=200
```

A live scenario-owned metrics relay first returned the typed stopped-state
diagnostic (exit code 1) rather than an exit-127 or unknown-command error. After
the bounded lifecycle recovery it returned the completed Darwin payload:

```text
correlation=4a26a43a-d59d-4271-bf30-b0ed4a90f399
outcome=RELAY_CALL_OUTCOME_COMPLETED
cpu.not_yet_sampled_reason=CPU has not been sampled yet
memory.not_yet_sampled_reason=memory has not been sampled yet
connections.not_yet_sampled_reason=network has not been sampled yet
gpu.not_yet_sampled_reason=GPU has not been sampled yet
disk.not_yet_sampled_reason=disk has not been sampled yet
swap_traffic.measured=0
major_faults.measured=0
fragmentation_index.measured=0
```

The lifecycle repair is in `internal/lifecycle`: bounded optional dependency
starts now carry their context through freshness probes, Go dependency
resolution, and source walks. The focused lifecycle regression passed, and the
live operation completed without re-pairing or changing the node id.

The Web Console live session prerequisite was repaired through the governed
Bridge node update, adding the required `vrooli-bridge:session` scope while
retaining the same node id. This was an in-scope permission convergence, not
a re-pair. The live production session then created, streamed, resized, and
deleted successfully.

The earlier corrected-agent redeploy also proved the new absolute-path error
contract:

```text
resolve scenario CLI "system-monitor": installed CLI "system-monitor" was not found; searched: /Users/matthalloran8/.vrooli/.vrooli/bin
```

The search-path and qualified-verb defects are fixed locally and covered by
regression tests. The node id remained
`25c7e426-c76c-421a-8351-aaf964589802` throughout, and no new identity or
re-pair was performed. The onboarding operation did reuse the existing
identity's redeem/convergence step, as required for installation convergence.

## Scope and uncovered paths

Windows node agents and a scratch-machine terminal-free join require hardware or
packaging environments not present in this control-plane workspace. They remain
explicitly uncovered until the implementation and its static/build evidence are
complete. No test in this file authorizes re-pairing `minimouse`.

## Validation runs

Focused module validation on the Linux control plane passed for root lifecycle,
nodeclient, cliresolve, mdns-go, delivery validation, API discovery, and the
Bridge API, CLI, and agent modules. The edited Go files pass `gofumpt -l` and
`git diff --check`; the lifecycle package test took 102.384 seconds and passed.

Test Genie run `20260827-011754-7835c322` for system-monitor reached terminal
FAIL. Its three error findings were recorded as baseline/non-scope carry-over:
`dependency.go.tidy` module metadata drift, the UI `pnpm run test:coverage`
execution failure, and the pre-existing experience tap-target-size failure.
The run otherwise completed all 22 phases.

Test Genie run `20260827-012435-fbc3ca5b` (web-console) reached terminal FAIL
with pre-existing runtime-surface, visual-status-bar, module-tidy, docs,
workflow, performance, test-coverage, storage, and dependency-audit findings;
the production Web Console capture above remains the authoritative Outcome A
probe. The first Bridge run `20260827-012559-7876f3f4` reached terminal FAIL
and identified stale mDNS requirement references. Those references were
corrected to `agent/internal/discovery/mdns_adapter_test.go`; the current
Bridge rerun `20260827-013325-9ebf9892` reached terminal FAIL with stale
metadata/UI, storage, and workflow findings only. The deleted-file finding did
not recur.

Outcome C remains uncovered: no scratch machine with only the installed
application was available in the Linux/Darwin fleet. The implementation has
bundle/bootstrap and pairing tests, but no live terminal-free join is claimed.
Windows node-agent and physical hardware paths are likewise not claimed.

## Desktop package implementation and latest validation

The Web Console Electron package now owns the installed-app join bootstrap.
Release builds cross-compile the Bridge agent with `CGO_ENABLED=0` and
`GOWORK=off` for Linux/Darwin x64/arm64, copy the binaries into the package's
`bridge-agent` resource, and start the matching binary from Electron's main
process. The agent state is kept under the Electron user-data directory, mDNS
discovery is enabled, pairing words are shown in a native dialog, and the
application registers login auto-start where Electron supports it. Windows is
explicitly not a node-agent target. Go is a release-build prerequisite only;
the installed application does not invoke Go or require a repository.

Commands and output:

```text
cd scenarios/web-console/platforms/electron
git diff --check -- scenarios/web-console/platforms/electron
npm run build
> web-console-desktop@0.0.1 build
> tsc

node --check scripts/build-bridge-agent.js
npm run build:bridge-agent
Bridge agent packaged: bridge-agent/vrooli-bridge-agent-linux-x64
Bridge agent packaged: bridge-agent/vrooli-bridge-agent-linux-arm64
Bridge agent packaged: bridge-agent/vrooli-bridge-agent-darwin-x64
Bridge agent packaged: bridge-agent/vrooli-bridge-agent-darwin-arm64
```

`file bridge-agent/*` identified ELF x86-64, ELF aarch64, Mach-O x86_64, and
Mach-O arm64 executables. This is static package/build evidence only; it does
not substitute for the missing scratch-machine join.

The compiled Electron helper checks also passed:

```text
bridge-agent target and resource-resolution checks passed
```

Test Genie run `20260827-014841-fbe86dfa` for web-console reached terminal
`FAIL` (`success=false`, `phaseSummary: 23 total, 13 passed, 10 failed`). The
run completed its phases but retained existing UI runtime-surface and CLI,
module, documentation, workflow, storage, performance, dependency, and
coverage findings. It did not provide a packaged-agent or terminal-free join
probe. The production Web Console session capture above remains the
authoritative Outcome A evidence.

## Outcome B live device graph and recovery evidence

The corrected system-monitor API was redeployed through the lifecycle manager
after the Darwin network collector's nil-map panic was fixed. The lifecycle
operation `startop-10ef6c6635747bab9e9998286a42f2bd` completed healthy; its only
degraded dependency was optional `ollama`. The existing minimouse agent was
not re-paired.

The live REST projection now returns 200 for both remote surfaces:

```text
GET /api/v1/metrics/current?node=25c7e426-c76c-421a-8351-aaf964589802&fresh=1
cpu.failedError=kern.cp_time: no such file or directory
memory.unsupportedReason=host_statistics64 memory pressure binding unavailable
disk.measured=22.194270946269494; provenance=native filesystem statistics
connections.failedError=darwin interface counters unavailable

GET /api/v1/metrics/devices?node=25c7e426-c76c-421a-8351-aaf964589802
platform=darwin; available=true; devices=8
device=APPLE SSD AP0256M; capacity_bytes=251000193024
```

The dashboard browser probe selected `minimouse` and rendered the same Darwin
identity, SSD model/capacity, and explicit unavailable CPU/memory/network
states. Screenshot: `system-monitor-minimouse-device-graph-live.png`.

For the disconnect proof, the enrolled agent process was temporarily
suspended and then resumed; no pairing or persistent host change occurred. The
browser observed three successful baseline/recovery responses and four 503
responses during the interruption. It retained the graph and capacity reading,
rendered `Showing the last reading`, `Retrying every 15 seconds (attempt 3)`,
and `reconnecting automatically`, then cleared the banner after recovery
without operator action. Screenshots:
`system-monitor-minimouse-network-interrupted.png` and
`system-monitor-minimouse-recovered.png`.

The focused regression coverage for this addition passes:

```text
go test ./internal/handlers ./internal/server ./internal/httputil     PASS
go vet ./internal/handlers ./internal/server ./internal/httputil      PASS
pnpm exec vitest run ...DeviceGraphPanel... ...useSystemMonitor...    13 tests PASS
pnpm run build                                                         PASS
```

The latest authoritative Test Genie run was
`20260827-030226-903d7954`: 18/22 phases passed. Its four failures are
pre-existing or broader scenario standing issues (UI interop/experience,
module tidy, coverage/quality and other maturity findings); the new focused
device-graph and retry tests pass. The run remains a suite-level FAIL and is
not relabeled as green.

## Latest audit additions

The final implementation audit added explicit authorization visibility to both
machine controls. system-monitor now returns the operator-facing grant sentence
and the raw scope list for each remote node; its header displays the sentence,
offers an in-app `Add machine` entry that opens Bridge pairing, and disables the
local-only system-output action when a remote machine is selected. Web Console's
typed target projection now carries the same grant sentence plus sorted,
deduplicated concrete scopes in the Bridge-scope readiness detail. Focused API
and UI tests cover these facts, including the remote action's disabled reason.

The agent mDNS adapter now honors the URL advertised in DNS-SD TXT data and
falls back to HTTP for generic records. The Bridge responder advertises the
same canonical configured/tunnel/derived endpoint used by onboarding. Focused
adapter, responder, Bridge API, and mdns-go tests pass; no second mDNS parser
was reintroduced.

Bridge was restarted through the lifecycle manager with operation
`startop-df4a6eb9bec6469b21724f0174745910`, and reported healthy on port 18767.
The protected node remained `25c7e426-c76c-421a-8351-aaf964589802`, online,
fresh, and dispatchable. An initial local `avahi-browse -rt
_vrooli-bridge._tcp` probe did not render the service, but the actual agent
browser path was then exercised with
`MDNS_BRIDGE_LIVE_CHECK=1 GOWORK=off go test ./internal/discovery
-run TestLiveBrowseBridgeAdvertisementWhenRequested -count=1 -v`: it passed,
finding the live record on the LAN interface with port 18767 and an advertised
URL. This proves the production shared-browser adapter against the running
responder; discovery from a second machine remains unclaimed.

The post-fix `packages/mdns-go` test suite passes, and the scoped agent lint
suite passes. The mdns-go lint command reports only pre-existing findings in
`mdns.go` and `dual_packet_conn.go`; the new responder has no remaining lint
findings.

The latest required scenario-owned runs also completed as suite-level failures,
with the changed behavior still green in focused tests:

- vrooli-bridge run `20260827-035853-93fdcfda`: 16/20 phases passed. The
  terminal FAIL is from standing findings in CLI primitive metadata, PWA/API
  hygiene, dependency/quality/docs, unit/storage/workflow, duplication,
  security, measures, proto, and branding. The portability, structure,
  performance, and business phases passed; the responder and agent focused
  tests remain green.

- system-monitor run `20260827-032829-d6e5fd3d`: 19/22 phases passed; failures
  were existing CLI primitive metadata, UI interop/visual, and coverage/
  maturity checks.
- web-console run `20260827-033316-ba3c0a9c`: 13/23 phases passed; failures
  were existing CLI metadata, UI surface/visual, documentation, storage,
  dependency, and coverage findings. The focused target-catalog and launcher
  tests passed independently.

`GOWORK=off go mod tidy -diff` is clean for system-monitor after the governed
nodeclient dependency addition. The broad suite results remain recorded as
failures and are not presented as full-suite proof.

## Final acceptance boundary

Outcomes A and B are now live-proven on the unchanged minimouse identity, with
focused implementation and package validation passing. Outcome C remains
explicitly uncovered: no scratch machine with only the installed application
was available to prove terminal-free discovery, matching-word approval,
ONLINE transition, and a relayed read verb. Windows node-agent and physical
hardware paths are likewise not claimed. The plan is therefore not complete.
