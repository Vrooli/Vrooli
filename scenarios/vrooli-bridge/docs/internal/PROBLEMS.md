# Problems — Vrooli Bridge

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

## Work ladder

- Rung: W0
- Evidence: the deterministic `swarm-manager goals list --json` search found no goal whose name, title, or description names `vrooli-bridge`; the requested Plan Manager slug `vrooli-onboarding-consent-true-apply-one-wizard-across` is absent from the live plan store.
- Blocker: contract-to-goal comparison is unverifiable until the authoritative goal/plan record is restored or linked.
- Measured: 2026-08-11

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-11 — Remote terminal evidence and native PTY remain open

**Symptom:** Web-console can now create a server-side Bridge target session and
translate browser terminal frames to the Bridge binary session wire. Native PTY
allocation and resize are implemented, but the deployed `minimouse` agent has
not been refreshed with that handler, so a live session records open/input/close
without proving output, resize, or remote process termination. The final Plan
Manager cross-scenario validation also reports `scenario-to-cloud` and
`web-console` as not-comparable against the captured baseline.

**Root cause:** The session wire and backend seams are implemented in Bridge and
the web-console federation adapter is present. The connected Mac predates the
session-frame handler. The supported working-tree re-onboarding attempt
(`f6fc959b-8fca-4816-987c-f2a94dbf01df`) stopped at SSH first touch because the
Bridge machine key is not authorized and no password credential was supplied.
The cross-scenario anchor has inherited failures outside the Bridge run's proof
surface, and shared-package changes are not part of Test Genie's default
scenario freshness digest.

**Workaround:** Use Bridge's owner-authenticated session API and the focused
web-console proxy tests for transport-contract evidence. Treat a target as
operational only when its server-side readiness facts are complete. Supply a
credential or install the Bridge machine key on `minimouse`, then repeat the
working-tree onboarding before using the Mac as live-session evidence.

**Real fix:** Re-onboard the Mac with the current working tree, then prove
output/resize/process termination, reconnect/scrollback, network-drop behavior,
real VPS/device evidence, and BAS video artifacts. Add a reviewed Windows
ConPTY backend when Windows interactive terminal support is required, and repair
or explicitly re-baseline the affected scenario anchors through Plan Manager.
Refresh the cached proto phase after shared-package edits before interpreting
its findings.

**Owner:** next implementation pass for plan phases 10–11.

**Refs:** `packages/session-core/`, `scenarios/vrooli-bridge/api/internal/session/`,
`scenarios/web-console/ui/src/hooks/useSessionManager.ts`,
Plan Manager validation `df0bd1a5-5a9e-4ccc-8e36-8557e9a0add2`, onboarding
operation `f6fc959b-8fca-4816-987c-f2a94dbf01df`.

### 2026-06-18 — Phase 6 COMPLETE (cross-OS deployment gate + consumer/P2 seams). PRODUCT SURFACE DONE.

The `gate` domain (OT-P1-002) is built end-to-end: `gate.proto`
(`GateService` RunGate/GetGate/WaitGate/ListGates), `internal/gate` (select one
eligible node per target OS → delegate each validation run to the SHARED dispatch
service + runs lifecycle via the `Runner` seam → live-recompute the aggregate
verdict), `handlers/gate`, `cli/domains/gate`, `gates`+`gate_os_results` schema.
`dispatch.Module` was split into `NewService`+`Module` so one dispatch instance
backs both the dispatch handler and the gate runner (every gate validation run
flows through the same allowlist + scopes + audit). All tests `[REQ:BRG-P1-002]`;
deployment-manager wired as the first consumer (`crossosgate`, Connect/JSON, no
proto-module dep, additive + inert until `VROOLI_BRIDGE_URL` is set).

**P2 future seams — recorded, deliberately NOT built (gated on separate tracks).**
Bridge is written cross-platform from day one so it is never the blocker, but
these are out of this plan's scope:

| OT | Capability | Why deferred | Revisit trigger | Seam it reuses |
|---|---|---|---|---|
| OT-P2-001 | Control plane on macOS/Windows | Gated on Vrooli-the-platform becoming installable on Mac/Win (a platform-level track, not a bridge feature). Bridge code is already `CGO_ENABLED=0` + cross-compiles for the matrix. | Platform installable on Mac/Win. | (whole scenario — verified portable). |
| OT-P2-002 | remote-desktop integration seam | A separate FUTURE scenario (screen/input control) reuses bridge node identity/reach; not a bridge domain. | Real-time remote control is built. | `registry` identity/reach. |
| OT-P2-003 | Cloud-runner / ephemeral nodes | Extends `registry` (node kind is metadata) + a provider integration; no on-demand capacity need yet. | On-demand VM/cloud capacity required. | `registry` + `provision`. |
| OT-P2-004 | Self-healing re-provisioning | Extends `provision` with drift detection + auto-reprovision; manual `fleet roll` suffices at current fleet size. | Fleet size makes manual re-provisioning costly. | `provision` + `compat`/`presence` gating. |

The contracts these consume (`registry`/`dispatch`/`runs`/`gate`) are the
deliverable seams — documented in [`SEAMS.md`](SEAMS.md) ("exposed (not built)
consumer seams"); the integrations are built by their own initiatives, not here.

### 2026-06-18 — Phase 4 COMPLETE (privileged provisioning tier + cross-platform agent). P0 DONE.

**Status:** RESOLVED for OT-P0-006/007. Built the `provision` control-plane
domain (`api/internal/provision` + `handlers/provision` + `provision.proto`),
the structurally separate privileged node-agent helper
(`agent/internal/privsep`), the cross-platform service-install adapters
(`agent/internal/service`), the `provision` CLI group, and the
`agent/build/crosscompile_test.sh` matrix gate. Privilege separation is proven
by an AST import-graph test (`internal/exec` ⊥ `internal/privsep`). All three Go
modules pass build/vet/gofumpt/golangci-lint/test; agent cross-compiles
CGO_ENABLED=0 for all 6 targets; go.mod tidy clean; `.vrooli/endpoints.json`
regenerated; API↔CLI parity green.

**Suite (`vrooli scenario test`, run `20260618-135225-b4213f37`): 14/18 phases
passed.** Phase 4's own surfaces are green — `unit` (all new tests),
`business`/requirements, `dependencies`, `security`, `integration`,
`architecture`, `quality`, `docs`, `performance`, `structure`, `contracts`,
`ui-health`, `measures`, `playbooks`. The 4 reds are ALL pre-existing /
environmental carry-over, **none introduced by Phase 4**:
- `standards` — the security-headers template campaign (`httpx`, SSE handler);
  pre-existing across every prior run.
- `smoke` — UI iframe-bridge handshake timeout (browser-substrate env issue,
  `project_browser_substrate_unification`), not app-boot breakage.
- `tidiness` — template `internal/testutil/modeltest` complexity +
  `no_prod_import_test` duplication (template debt).
- `proto` — the SAME 2 blocking `proto.shared_type_misplaced` errors as Phase 3
  (`channel.Heartbeat` reused by presence; `channel.RunEvent` reused by runs).
  **Verified `provision.proto` added ZERO new proto errors** (`proto-health
  validate scenario vrooli-bridge` → errors=2, both the channel ones): provision
  deliberately uses its OWN `ProvisionEvent` type rather than reusing a channel
  type. `channel.ProvisionCommand` is now CONSUMED (provision adapter + agent),
  so its prior `possibly_unused` INFO is the only INFO left.

**The deferred proto-layout pass is now READY (reuse set is FINAL).** Phases 3–4
revealed the complete cross-domain reuse set: `channel.Heartbeat`
(+HealthSnapshot/CompatibilityStatus) → presence, and `channel.RunEvent` → runs.
Provision did NOT add to it. The single deliberate layout pass (move those shared
types to a new `v1/shared/` package and re-point channel/presence/runs imports +
the agent) can now be executed once without re-churning. It is orthogonal to
OT-P0-006/007 and tracked as the next proto cleanup, not a Phase-4 gap.

**Environmental note (build-api flake):** mid-Phase-4 a concurrent repo process
deleted vrooli-bridge's *untracked* generated proto packages (audit/dispatch/
pairing/presence/runs/provision survive only as committed gen for registry/
channel/health/errors), breaking `build-api` with "no required module provides
package …/v1/audit". Fix = re-run `make generate` in `packages/proto` (regenerates
from the on-disk schemas). If a future agent sees a sudden module-resolution
failure for ALL vrooli-bridge proto packages, this is the cause — regenerate, do
not edit go.mod.

---

### 2026-06-18 — Phase 1 COMPLETE (node-agent live dial + fleet UI landed)

**Status:** RESOLVED. The node-agent now holds a real dial-out channel
(`agent/internal/channel/channel.go::Dial`: SSE hold + heartbeat loop with
health snapshots + exponential-backoff reconnect; `agent/internal/health/` real
cross-platform sampler, CGO_ENABLED=0 for all 6 targets). The fleet UI shipped
(`ui/src/features/fleet/` + `ui/src/api/nodes.ts` over the generated Connect-Web
client) and is wired into the dashboard. The agent dial tests are tagged
`[REQ:BRG-P0-003]` and registered in `requirements/03-presence-health`.

Per-module dev gates are GREEN: api/cli/agent `go build/vet/gofumpt/golangci-lint/
test`; agent cross-compiles CGO_ENABLED=0 for linux/darwin/windows ×
amd64/arm64; UI `type-check`/`lint` (0 errors)/`test:coverage`/`build` all green.

Template-debt fixes were required to get the UI green (the react-vite copy in
this scenario shipped these committed-red, fixed mirroring device-sync-hub):
router v7 future flags in `ui/src/app/routes.tsx`; distinct sidebar/bottom-nav
aria-labels (landmark-unique); removed genuinely-unused template locale keys
(`app.eyebrow`/`app.description`/`pages.dashboard.statPlaceholderLabel`); theme
choice labels moved to a static map so the no-unused-keys gate can see them.

**Open (full `vrooli scenario test` suite, run `20260618-042142`): the scenario
is a greenfield build in progress — most phase reds are EXPECTED until later
phases build the remaining domains. Triage:**
- `standards`, `structure`, `tidiness`, `dependencies`, `unit` — pre-existing
  greenfield-incompleteness (also red in the §6a anchor `20260618-024949`); the
  later phases + a standards/tidiness cleanup pass drive these green.
- `proto` — REST-exception payload-intent declarations for `channel_events` and
  `health` are now DECLARED (`ProtoPayloads` in their endpoint descriptors;
  `.vrooli/endpoints.json` regenerated — the regen also backfilled the registry
  domain the prior agent never committed). Two reds remain: (1) `ui`
  proto-adoption proof + `endpoint_proof` are blocked by the
  **typescript-code-graph sidecar `STATUS_PERMANENTLY_UNHEALTHY`**
  (environmental, like the covdata toolchain gap) — the fleet UI *does* consume
  generated registry proto but the analyzer can't verify it while the TS sidecar
  is down; (2) `proto.shared_type_misplaced` — `channel.Heartbeat` (and the
  HealthSnapshot/CompatibilityStatus it pulls in) are reused by presence.proto,
  so the analyzer wants them in `v1/shared/`. DEFERRED to a deliberate proto-
  layout pass once Phases 3–4 add the other cross-domain reuse (runs reuses
  RunEvent, dispatch reuses JobPush) — doing it once, after the reuse set is
  known, avoids churning the agent/api imports twice. The `channel.proto`
  message-only contract is correctly flagged `possibly_unused` (INFO; consumed
  by the agent + SSE push, not a served RPC).
- `security` — was the lone blocking `gosec.G101` false positive on the agent
  credential *filename* constant; FIXED (reviewed `//nolint:gosec`). Remaining
  warnings are stdlib `govulncheck` toolchain advisories.
- `smoke` — UI iframe-bridge handshake timeout (browser-substrate env issue,
  see `project_browser_substrate_unification`), not app-boot breakage.

---

### 2026-06-18 — Phase 1 spine landed; node-agent live dial + fleet UI still stubbed (SUPERSEDED — see entry above)

**Symptom:** The control-plane side of OT-P0-003 (dial-out presence) is complete and tested, but no real node can yet appear online: the node-agent's `Dial` is still the Phase-0 stub (it constructs the handshake and returns without holding the SSE stream), and there is no fleet UI. `vrooli scenario test vrooli-bridge` will not pass the scaffold-health gate until the UI feature exists, so `requirements sync` keeps BRG-P0-001/003 at `planned` (the `[REQ:...]` tags and test refs are in place and detected — `exists: true` — so sync flips them the moment a full green run records evidence).

**Root cause:** Phase 1 was built control-plane-first (proto → api → cli → tests) to land a green, regression-anchored spine before the harder real-time agent loop and the UI. The agent live dial (open SSE to `/api/v1/channel/events?node=`, run the heartbeat loop calling `PresenceService.ReportHeartbeat`, reconnect with backoff) and `ui/src/features/fleet/` (node list with presence dots, register/revoke) are the remaining vertical-slice surfaces.

**Workaround:** The API spine is exercisable today via the CLI (`nodes register/list/get/revoke`) and `curl`; the dial-out flip is proven by `api/internal/presence/dialout_test.go`. The Phase-1 stub is clearly marked in `agent/internal/channel/channel.go::Dial`.

**Real fix:** **RESOLVED.** The live SSE hold + heartbeat loop, mutual-auth enforcement, fleet UI, and requirement evidence now exist. The remaining Mac refresh and cross-scenario evidence are tracked by the newer remote-terminal entry above rather than this superseded Phase-1 diagnosis.

**Owner:** unassigned (next Phase 1 increment).

**Refs:** `agent/internal/channel/channel.go`, `api/handlers/channel/`, `api/internal/presence/`, `api/internal/registry/`, `cli/domains/nodes/`, requirements `01-node-registry` / `03-presence-health`.

### 2026-06-18 — Phase 2 landed: one-touch bootstrap + mutual auth + atomic revocation (OT-P0-002)

**Status:** Control-plane + live enforcement COMPLETE and green. As-built:
- **Pairing domain** (`api/internal/pairing/` + `handlers/pairing/`): single-use,
  short-TTL, SHA-256-hashed pairing codes (`IssuePairingCode` owner-gated /
  `RedeemPairingCode` open); request/approve fallback; `node_credentials` storing
  each node's Ed25519 public key; `RevokeCredential`. Mirrors the registry domain
  shape (schema/repo/sqlite/service/handler/endpoints + parity).
- **Mutual auth** (`api/internal/cpkeys` + `api/internal/nodeauth` + agent
  `internal/nodecred`): per-node Ed25519 keypair pinned both directions. The
  control plane has a stable persisted identity key (cpkeys); the node signs every
  heartbeat (X-Bridge-* headers) and binds the dial-out SSE token to its key
  (nodecred); the channel handlers VERIFY both (nodeauth) — enforcement is wired
  live in `main.go`. The agent `--print-public-key` bootstrap helper emits the key
  for `pair redeem --public-key`.
- **Atomic revocation**: `RevokeNode` now revokes the durable record AND severs the
  credential AND force-drops the live channel (`presence.Hub.Disconnect` →
  `Conn.Done()`) in one operation.
- **CLI**: `pair issue/redeem/request/approve/list` + the existing `nodes revoke`.
- **Tests `[REQ:BRG-P0-002]`**: pairing single-use/TTL/unknown-code; rogue-node &
  impostor-control-plane & cross-node-replay & stale-proof & tampered-sig
  rejection; atomic-revocation; live heartbeat enforcement; agent wire-format
  cross-check. api/cli/agent all build/vet/gofumpt/golangci-lint/test green; agent
  still cross-compiles CGO_ENABLED=0 for all 6 targets.

**Reconciliation (docs vs as-built):** DATA.md/SECURITY.md say node credentials are
"hashed at rest / secret material." With the *chosen* asymmetric mechanism the only
secret is the node's PRIVATE key, which never leaves the node — so the control plane
stores the node's PUBLIC key in the clear (it must stay verifiable; a hash can't
verify a signature). This is strictly stronger than hashing a shared secret. The
schema.sql + pairing.proto comments record this; treat the older "hashed" wording as
pre-dating the 2026-06-18 Ed25519 decision.

**Deferred (not a bug):**
- **Windows machine-wide helper validation** remains deferred to the real Windows
  host gate. Linux now has a machine-wide `vrooli-bridge-provisioner` unit and
  typed local IPC; macOS has the system LaunchDaemon renderer, but the current
  environment has no second principal for live owner/process proof. The ordinary
  agent remains a user-scoped service by design.
- **Benign double-redeem race** (single-owner v1): `Redeem` validates → registers →
  stores credential → atomic `BurnCode`. The atomic burn is the single-use gate
  (sequential reuse is rejected at validation + burn). A truly *concurrent* double
  redeem of one code could register two nodes before either burns; impossible in the
  single-owner/one-installer-per-code model, documented in service.go. Tighten to
  burn-before-register if multi-tenant ever lands.

### 2026-06-18 — Phase 3 deferred: live workspace-sandbox audit wiring (implementation RESOLVED; host evidence open)

**Symptom:** The audit trail (OT-P0-008) has a workspace-sandbox HTTP sink, but
the current environment has not completed a live write/read round trip against
that external substrate; SQLite remains the fallback.

**Root cause:** workspace-sandbox is a separate scenario and live availability
is an environment concern. The sink is now wired behind the existing audit
interface, with bounded HTTP writes and SQLite fallback.

**Workaround:** The audit domain is built around the narrow `audit.Sink` seam.
The SQLite store remains the safe fallback; `internal/audit/sandbox_integration_test.go`
proves the workspace-sandbox sink is swappable and the focused HTTP-store test
proves its workspace shape.

**Real fix:** **IMPLEMENTED.** Run one live write/read probe when
workspace-sandbox is available and record its durable evidence in the plan
ledger; do not treat the local fallback or a unit fake as external readback.

**Owner:** unassigned.

**Refs:** `internal/audit/sink.go`, `internal/audit/sqlite.go`, `docs/internal/SECURITY.md#audit`.

### 2026-06-18 — Phase 5: per-node queue built (bounded concurrency); offline-redelivery still deferred

**Done (OT-P1-004):** The per-node scheduler (`internal/queue`) now sits on the
dispatch → push path with **bounded concurrency + fair FIFO**: a busy node's
extra jobs queue (their durable run stays QUEUED) and are promoted as slots
free; the QueueService surfaces running-vs-queued per node. The scheduler
satisfies dispatch's existing JobPusher seam, so dispatch is unchanged.

**Still deferred:** The scheduler is **in-memory** and still requires the node to
be ONLINE at dispatch time. If a node drops between the online-check and the
push, dispatch aborts the run (fail-closed) rather than holding the job for
redelivery on reconnect; and queued state does not survive a control-plane
restart. Durable, offline-tolerant redelivery (persist the queue; promote on a
presence-online event) is the remaining work — a presence→scheduler hook + a
queue persistence layer.

**Workaround:** Online nodes — the overwhelmingly common case — queue correctly;
a node that drops mid-dispatch is re-dispatched by the operator.

**Refs:** `internal/queue/scheduler.go`, `internal/dispatch/service.go` (online
gate), `handlers/queue/`.

### 2026-06-18 — Phase 3 deferred: AbortRun does not yet signal the node — RESOLVED (Phase 5)

**Symptom (was):** `runs abort` marked the run ABORTED on the control plane but
did not push a cancel frame to the node, so a job already executing ran to
completion (its late EXIT ignored as a stale completion).

**Resolution (Phase 5, OT-P1-004):** Added the `channel.AbortJob` ServerFrame
(field 5 of the ServerFrame oneof). `runs.Service` gained a `WithCanceller`
option; `runs.Abort` now pushes an AbortJob via the channel-canceller
(`handlers/queue.channelCanceller` → presence-hub push). The node-agent
registers a per-run cancelable execution context (`internal/channel.runningJobs`)
and cancels it on the AbortJob frame, killing the job's `exec.CommandContext`
process. Proven by `internal/runs/cancel_test.go` (Abort pushes cancel + fires
the terminal hook; a natural EXIT fires the hook WITHOUT a cancel push) and
`agent/internal/channel/abort_test.go` (the AbortJob frame routes to cancel).

**Refs:** `internal/runs/service.go::Abort`, `handlers/queue/adapter.go`,
`agent/internal/channel/channel.go`, `packages/proto/.../v1/channel/channel.proto`.

### 2026-06-18 — proto `shared_type_misplaced` — RESOLVED 2026-08-11

**Symptom (resolved):** The `proto` test-genie phase previously failed on
`channel.Heartbeat` and `channel.RunEvent` being reused across domains while
living in `channel`.

**Root cause:** The versioned wire types live in `channel.proto`; the presence
and (Phase 3) runs domains reuse them so the agent speaks one vocabulary. Phase 3
now reveals the **complete cross-domain reuse set** the Phase 2 handoff said to
wait for: `Heartbeat` + `HealthSnapshot` (presence) and `RunEvent` +
`RunEventKind` (runs/dispatch). proto-health wants shared types in a package that
signals sharing rather than a feature-named one.

**Resolution:** Moved `CompatibilityStatus`, `HealthSnapshot`, `Heartbeat`,
`RunEventKind`, and `RunEvent` into `vrooli-bridge/v1/shared`, regenerated all
typed artifacts, and updated channel/presence/runs plus API, CLI, agent, and UI
consumers. The authoritative run `20260811-101743-3b096c82` passed all 20
phases, including proto and unit; `proto-health validate scenario vrooli-bridge`
passes with only non-blocking warnings.

**Real fix:** Complete. Keep future cross-domain wire types in `v1/shared`
and regenerate through `packages/proto/Makefile` rather than reintroducing
feature-owned duplicates.

**Owner:** resolved by the onboarding/bridge integration work.

**Refs:** `packages/proto/schemas/vrooli-bridge/v1/channel/channel.proto`,
`.../presence/presence.proto`, `.../runs/runs.proto`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| handler adapters (all domains) | `architecture-cartographer` flags `layering/handler-imports-sibling-domain` because each `api/handlers/<domain>/adapter.go` imports SIBLING domains to bind its domain's proto-free seams (registry/presence/audit/provision/runs/dispatch). Phase 6's `handlers/gate/adapter.go` adds the same pattern (it imports dispatch + registry + runs to bind the gate `Runner`/`NodeLister` seams). | This is the codebase's deliberate **"single translation point"** pattern (every domain since Phase 1 — audit/channel/dispatch/registry/provision/fleet/queue/artifacts/gate all do it; SEAMS.md + each adapter.go document it: *the domain never imports a sibling domain or proto; these adapters do*). It was previously ABSTAINED by the auditor; Phase 5 documenting the new domains in DOMAINS.md raised the domain-map authority to high, which turned the abstentions into blocking findings. Not a new code smell — gate follows the identical established pattern. | If the auditor's rule is to be honored ecosystem-wide, move every handler's seam-binding adapter out of the handler package (e.g. a per-domain `wiring/` constructed in main.go) — a cross-cutting refactor across all 12 domains, out of Phase 6 scope. Otherwise teach the test-genie architecture phase to allow handler→sibling-domain imports via `adapter.go` (a documented exception), since the pattern is intentional. |
| convergence/glossary drift (warn) | `convergence_drift` (per-domain) + `glossary_drift` (heuristic, on seams/mocks/types files) warnings. | Heuristic naming-consistency nudges across pre-existing and new domains; non-blocking on their own (the blocking outcome is the layering errors above). | Reconcile domain vocabulary in DOMAINS.md as the glossary stabilizes; low priority. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

Current identity/delegation work remains on the typed Mode-A ladder:

1. Keep the control-plane manifest derived from CLI governance and enforce the
   registry-owned execution scopes before run creation.
2. Keep presence-only equivalent to an empty execution-scope grant; the agent
   flag is only the bootstrap alias and must be serialized as an explicit
   boolean value.
3. Use the existing SSH onboarding path to converge a node when its agent is
   presence-only; do not weaken the signed-frame gate or add a shell path.
4. Treat macOS screenshot proof as hardware evidence only when a real GUI
   login/window-server session exists. An SSH-only launchd domain is a recorded
   environment finding, not a reason to fabricate an artifact or broaden Mode A.
5. The `vrooli-bridge:session` seam is now implemented as an authenticated,
   bounded binary WebSocket transport. PTY/backend selection remains a later
   domain concern; this transport deliberately relays bytes without parsing
   them.
