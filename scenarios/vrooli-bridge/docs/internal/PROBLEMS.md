# Problems — Vrooli Bridge

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

## Work ladder

- Rung: W0
- Evidence: goal `contribution-verification-isolated` directs “vrooli-bridge integration so the inbound triage team can safely run incoming PR patches and scenario proposals on an isolated install before approving them upstream”; no P0 operational target in `PRD.md` names isolated contribution verification (OT-P0-004 covers typed dispatch generally, not this goal capability).
- Blocker: decide whether isolated contribution verification is a Bridge P0 capability or only a future consumer integration, then align the goal or PRD before walking W1–W3.
- Measured: 2026-08-18

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

### 2026-09-09 — Clean Linux lifecycle evidence remains pending

**Symptom:** The Linux node-agent lifecycle is covered by static rendering,
service-manager argv, privilege-boundary, bootstrap convergence, and recovery
fixtures, but this execution does not yet contain the required clean-host
install, reboot, update/rollback, unauthorized-IPC, and ownership-safe removal
receipts.

**Root cause:** The current workspace host is a shared development machine, not
a disposable certification target. Its user systemd manager and linger state
are observable, but using the host for destructive lifecycle certification
would not provide clean restoration evidence.

**Workaround:** Treat the focused agent tests, `bootstrap_test.sh`, static
Linux build, and Bridge API validation as implementation evidence only. Keep
Phase 12 unfinished and do not promote these fixtures to native M01 evidence.

**Real fix:** Run the supported bootstrap and service lifecycle on a recorded,
disposable Linux target, capture process principals before/after reboot,
exercise failed readiness and rollback, test a different local IPC caller, and
verify that removal leaves unrelated data intact.

**Owner:** Bridge deployment validation / operator-provided Linux target.

**Refs:** `agent/internal/service/`, `agent/internal/privsep/`,
`bootstrap/bootstrap.sh`, Phase 12 execution `7c154c06-2692-4679-b5dd-a1e0d4155ff5`,
Bridge API run `20260910-023332-26fa3be5`.

### 2026-09-09 — Certification collection is incomplete and traceability is provisional

**Symptom:** The prospective Plan Manager collection baseline has 11 ready
members, 2 failed members, and 4 pending members. The Bridge member is retained
as a ready baseline artifact, but the collection is not complete.

**Root cause:** Android baseline execution exceeded its deadline, desktop
admission was saturated, and autoheal, emulator, onboarding, web-console, and
workspace-sandbox had not terminalized when the collection was read.

**Workaround:** Keep those cells failed or pending in the evidence index. Use
the Bridge-owned focused receipts and the bounded qualification program for
controlled admission work; do not use the collection as certification evidence.

**Real fix:** Reconcile each failed or pending member through Git Control Tower
after its owner is available, then run the final captured collection again
against the exact release candidate.

**Owner:** Plan Manager / Git Control Tower with each affected scenario owner.

**Refs:** Collection `vrooli-bridge-deployment-foundation-certification-baseline`,
generation 1; Bridge run `20260909-201241-f694f841`.

### 2026-09-09 — Requirements traceability remains below findings-clean

**Symptom:** The requirements registry is structurally valid and the latest
focused business/docs run passes, but full business sync and live evidence do
not yet prove all P0 requirements.

**Root cause:** Several existing requirements intentionally retain planned or
failing validation references while native, security, and release work is
unfinished. The new selective-footprint requirement is planned until its
closure and activation tests land.

**Workaround:** Treat the ledger as an honest skeleton/evidence map. Do not
flip requirement statuses or PRD checkboxes from the focused run.

**Real fix:** Add desired-behavior tests at the owning seams, tag them with
their requirement IDs, run the complete requirements sync, and re-grade the
business dimension.

**Owner:** Bridge requirement owners.

**Refs:** `requirements/22-selective-footprint/module.json`,
`docs/internal/TESTING.md`, focused run `20260909-200637-268743d5`.

### 2026-09-01 — Reference-node end-to-end proof remains pending

**Symptom:** The repository now exposes target-aware configuration, credential,
and re-apply surfaces, but the recorded `minimouse` proof still ends at
credential and safeguard blockers.

**Root cause:** The remeasurement harness and Web Console flow now exist, but
the connected node still needs its outstanding credential and safeguard inputs
before a fresh Web Console-only run can be completed.

**Workaround:** Use the focused Bridge/Web Console tests and the durable
onboarding run status while treating the live node as unverified.

**Real fix:** Re-run the live flow after supplying the node's credential and
safeguard inputs, then record operation id, answer delivery, final readiness,
empty drift, and a deliberate re-apply in the evidence record.

**Owner:** next live validation pass.

**Refs:** `<runtime-home>/plan-artifacts/docs-history-20260907/docs/reference/cross-platform-effort/evidence/node-setup-seam-measurements-2026-08-31.md`,
`scenarios/web-console/ui/src/components/fleet/ConfigurationPanel.tsx`.

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
proto-module dep, additive + inert until its shared Bridge client is wired).

**P2 future seams — recorded, deliberately NOT built (gated on separate tracks).**
Bridge is written cross-platform from day one so it is never the blocker, but
these are out of this plan's scope:

| OT | Capability | Why deferred | Revisit trigger | Seam it reuses |
|---|---|---|---|---|
| OT-P2-001 | Control plane on macOS/Windows | Gated on Vrooli-the-platform becoming installable on Mac/Win (a platform-level track, not a bridge feature). Bridge code is already `CGO_ENABLED=0` + cross-compiles for the matrix. | Platform installable on Mac/Win. | (whole scenario — verified portable). |
| OT-P2-002 | remote-desktop integration seam | Bridge now provides a bounded desktop session relay to Device Control's private owner RPC when `BRIDGE_DESKTOP_OWNER_SOCKET` is explicitly configured; native screen/input semantics remain owned by Device Control. | A separate remote-desktop consumer needs presentation or richer streaming beyond the typed relay. | `registry` identity/reach + authenticated `session` transport. |
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

- Rung: W3
- Evidence: the Bridge session contract and node agent now implement a typed
  desktop relay to Device Control's private owner RPC, with Unix-owner and
  node-session fixture tests; the required live paired-node desktop fixture has
  not yet been executed.
- Blocker: Phase 11 live remote-fixture, network-fault, and revocation evidence
  remains incomplete; the Bridge Test Genie unit receipt is also retained as a
  misconfiguration finding.
- Measured: 2026-09-08

## Work ladder

### 2026-09-07 — Monetization prerequisite #2 readiness assessment

- Rung: W0 — contract does not cover the current requested deployment scope.
- Evidence: the operator requests a "subset using a union of every scenario,
  resource, tool, and safeguard" selected through vrooli-onboarding. The PRD
  node-side dependency instead requires "a full Vrooli install" and its P0
  targets contain no onboarding-owned union shipment or remote configuration
  acceptance target. Cross-OS release evidence is only OT-P1-002, although it
  is load-bearing for the requested desktop deployment preparation.
- Goal lookup: the prescribed named-mention query returned only archived
  `contribution-verification-isolated`, about isolated incoming-PR validation.
  It does not establish a current monetization readiness contract. This
  assessment uses the current operator request as the requested scope.
- Blocker: reconcile the contract through authorized PRD work before running
  W1–W3 certification gates. This review does not modify product scope or
  implementation. No new suite or real-device run was performed.
- Measured: 2026-09-07, local working tree; not a released-revision certificate.

Source-inspection findings for the follow-up review:

| Area | Evidence | Implication |
|---|---|---|
| Remote configuration | `api/internal/onboard/orchestrator.go` calls onboarding handoff and `ApplyAndReadiness`; `docs/concepts/ONBOARDING-BOUNDARY.md` assigns one write authority. | Preserve the existing seam; prove remote answers, apply, readiness, drift, and re-apply on real targets. |
| Selected source shipment | `api/internal/onboard/worktree.go` enumerates tracked and non-ignored untracked repository files; onboarding has `/api/v2/union`. | The inspected shipment path is whole-tree. A union-to-shipment integration and clean-target proof remain necessary. |
| Windows service lifecycle | `agent/internal/service/service_install.go` returns `errRenderOnly("windows")` for install, status, and uninstall. | Native Windows lifecycle is incomplete; cross-compilation does not prove installability. |
| Privilege boundary | `agent/internal/privsep/peer_uid_other.go` refuses peer identity outside Linux/Darwin; SECURITY.md retains G8 live-principal proof. | Require real per-OS runner/helper identity and unauthorized-caller rejection evidence. |
| Browser journeys | Seven entries in `bas/registry.json` describe observer checks. `complete-onboarding-wizard.json` only navigates and asserts page/heading visibility. | These do not certify completed remote operations. Add controlled action journeys with remote postconditions and evidence. |
| Durable recovery | `api/internal/queue/reconcile.go`, scheduler durable-store option, and handler runs-store adapter exist. | The old memory-only queue entry is stale. Measure restart/reconnect/acknowledgment semantics before prescribing more implementation. |
| Pairing concurrency | Uncorrelated `pairing.Service.Redeem` registers a node and stores its credential before `BurnCode`. | Investigate concurrent redemption and partial failures; prove exactly one authorized identity and no orphan credential. This is source-level risk, not a demonstrated exploit. |
| External proofs | Existing ledger retains minimouse configuration/session proof gaps; SECURITY.md retains external audit readback and supply-chain/retention hardening leads. | Re-measure against the release candidate; historical completion labels do not close these proofs. |

Recommended completion sequence: reconcile target kinds and P0 acceptance;
complete per-platform lifecycle and union shipment; close trust/provisioning
and recovery proofs; exercise UI/CLI/API journeys on controlled hosts; then
collect revision- and artifact-bound evidence through Bridge for
deployment-manager. Mobile devices should have explicit capability-based
adapters through build/test hosts rather than inherit a full-Vrooli-node
assumption. Scenario-to-cloud should retain VPS deployment lifecycle;
Bridge should retain machine identity, reach, and transport; onboarding
should retain configuration policy and delegate host mutations to the
control plane.

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
5. The interactive session seam is implemented as an authenticated, bounded
   binary WebSocket transport using the shared `vrooli-bridge:write` effect.
   PTY/backend selection remains a domain concern; this transport deliberately
   relays terminal bytes without parsing them. Remote desktop binding metadata
   carries surface, owner, lease epoch, and policy identity, with bounded
   digest-only evidence. Bridge now has a typed node adapter that relays
   desktop-bound observe/act/stop calls to Device Control's private owner RPC;
   live paired-node and network-fault evidence remains outstanding. Updated
   2026-09-08.

### 2026-09-09 — Deployment-foundation W0 reconciliation

The contract mismatch identified on 2026-09-07 is resolved at the PRD layer:
OT-P0-010 now makes complete and selective footprint activation an explicit
must-ship outcome, and the deployment document defines both modes against the
same onboarding-owned closure. This resolves the W0 omission; it does not claim
that the selective shipment, clean-target native matrix, or release evidence
has passed. Those remain the implementation and certification work in phases
10–29 of the deployment-foundation plan.

The current shared worktree also contains a separate root CLI-manifest drift:
two commands use `process:spawn`, while `.vrooli/schemas/cli-manifest.schema.json`
does not admit that permission. Bridge dispatch tests therefore fail during
catalog construction before exercising their intended assertions. This is
retained as an external/concurrent finding; it is not silently “fixed” by
weakening Bridge admission or by replacing the operator's manifest changes.

### 2026-09-09 — Phase 3 target identity and typed-contract receipt

Phase 3 completed with a fresh Plan Manager validation receipt:
`60e1b951-58fa-43c0-9ccb-7895dbf59f49` (`PASS`, `FRESH`). The required owner
children succeeded for `vrooli-bridge` (`20260909-223624-e11da5b8`) and
`web-console` (`20260909-223659-dbd4fe85`) under the narrowed `contracts`
scope. Direct evidence also includes `go run -mod=mod ./cmd/protogen verify`,
generated Bridge package tests, api-core target-model tests, focused Bridge
projection/attached/gate/onboarding tests, focused Web Console target tests,
and Bridge unit-health static validation with zero blocking findings.

The implementation now uses durable target identity for exact selection,
rejects ambiguous duplicate IDs, preserves unavailable exact targets for
discovery, and places shared channel response/receipt messages in the
canonical proto shared domain. This is source and focused-contract evidence;
it is not native target/device deployment proof.

The Web Console proto owner run `20260909-222351-414b6da4` remains a retained
limitation because its eight REST payload declarations and API-to-CLI endpoint
registry drift are outside the target projection change. Its focused target
tests and scoped contracts receipt pass. Bridge UI coverage also retains three
baseline failures in untouched session, locale, and onboarding-form tests
(`177/180` passing). Neither limitation is upgraded to a Bridge defect or
silently removed from the evidence ledger.

### 2026-09-09 — Phase 4 operation-specific readiness receipt

Phase 4 completed with fresh Plan Manager validation receipt
`03198d19-38a4-4bc5-bb33-75bb3644cf5c` (`PASS`, `FRESH`, execution
`7c154c06-2692-4679-b5dd-a1e0d4155ff5`, scope generation `1`). The required
owner API receipts also passed: Bridge
`20260909-230051-ef807921`, Onboarding `20260909-230058-9e6c5dcf`, and Web
Console `20260909-230110-d4aa5b53`. Direct evidence covers the shared
operation evaluator, headless-versus-visual prerequisite separation, stale
heartbeat freshness windows, revoked-grant refusal, operation-aware selection,
proto generation, Web Console target projection, and remote-session admission.

The maintained owner phase is `api`; the historical `integration` phase is no
longer in the live Test Genie catalog. Broad `unit api` runs retain unrelated
unit/UI/API-workspace and unit-policy baseline failures, while their `api`
phases pass. Full proto verification also retains pre-existing generated drift
in `gen/descriptor/image.binpb` and untracked `gen/manifests/zzprobe.lock.json`.
The first Phase 4 validation attempt was stale because scoped test artifacts
changed during execution; it is retained as historical evidence, and the fresh
retry above is the authoritative receipt. No native real-target or physical
device proof is claimed here.

### 2026-09-09 — Phase 5 pairing and identity-recovery receipt

Phase 5 completed with fresh Plan Manager validation receipt
`2c02219b-57a4-4cd4-9f8c-aac29c8ea922` (`PASS`, `FRESH`, execution
`7c154c06-2692-4679-b5dd-a1e0d4155ff5`). The owner child
`20260909-232448-b06073b8` passed the maintained `api` phase. Focused direct
tests pass for correlated concurrent replay, single-use redemption, failed
credential-write recovery through the durable enrollment saga, immutable
attempt retry lineage, stable machine identity resolution, conflict-safe node
lineage, and explicit merge behavior.

The broader owner run `20260909-232019-567eb2ed` passed `api` but failed the
generic security-health phase with four errors and 351 warnings. The findings
are retained because they cover artifact delivery, SSH/toolchain dependency,
and other surfaces outside this phase's pairing/machine boundary; they are not
promoted to a Phase 5 defect. Its failed validation attempt
`81801106-28f5-449b-b390-6fa544e8d612` is likewise retained; the corrected
`api`-only receipt above is authoritative. The full pairing package also
encounters concurrent `proto-health` CLI manifest permission drift before its
intended assertions. No native target proof is claimed.

### 2026-09-09 — Phase 6 authorization, replay, and revocation receipt

Phase 6 completed with Plan Manager validation operation
`34adf641-0c82-497e-ac43-b58f14e68634` (`PASS`, terminal, fresh scoped
validation). The required owner receipts were Bridge
`20260909-233232-855de38a` and scenario-authenticator
`20260909-233427-49fae3cd`. The maintained `api` owner phase passed for both
scenarios; scenario-authenticator's `security` phase also passed.

Focused direct evidence covers grant denial without a durable run or push,
fresh Ed25519 node proof validation, cross-node and stale-proof replay
rejection, default presence-only agent posture, procedure-based proxy
admission, owner break-glass scope ceilings, durable revocation ordering, and
rejection of a delayed relay result after node revocation. The new delayed
result regression is in `api/handlers/channel/heartbeat_handler_test.go`.
Focused agent channel/cpverify/config/credential-grant tests, api-core
trustposture tests, and scenario-authenticator account/auth/authorization
tests pass.

The retained broad Bridge receipt failed its generic security and proto phases.
Its security findings are artifact-delivery hardcoded-secret findings and SSH
dependency vulnerabilities; its proto findings are descriptor/domain/health
baseline drift. These are outside the Phase 6 authorization boundary and are
retained as attribution evidence. Direct broad Bridge package tests also
encounter concurrent proto-health manifest permission drift. No native
real-target, physical-device, or independent adversarial-review proof is
claimed by this phase.

### 2026-09-09 — Phase 7 privileged host-action boundary receipt

Phase 7 completed with Plan Manager validation operation
`9a49e731-5973-48b8-91b7-24aebfb9f6f2` (`PASS`, terminal, targeted). Its fresh
Bridge owner child `20260909-235034-5d93a96d` passed the maintained `api`
phase. Direct package evidence also passes for the typed privileged IPC,
kernel peer-UID admission, closed helper-operation vocabulary, fixed-argv
no-shell execution, native service definitions, configuration guards,
fail-closed audit ordering, rollback, and unavailable-helper reporting.

Two initial Bridge owner attempts (`20260909-234450-030928be` and
`20260909-234722-4629d6cc`) could not acquire `api-health` ownership and are
retained as provider-unavailable infrastructure receipts. The successful
producer validation and fresh child are authoritative. Phase 7 made no
implementation changes; the live two-principal runner/helper process-owner
proof remains an explicit real-host obligation for phases 12–14. No native
process-owner proof is claimed here.

### 2026-09-10 — Phase 8 target-addressed resumable onboarding receipt

Phase 8 implementation evidence covers the typed onboarding path across CLI,
Web Console, and API. Operator-input answers now carry the target and the
configuration revision observed by the caller; stale submissions fail with
Connect `aborted` before capability application, preserving newer desired
state. CLI selection, session, operator-state, readiness, apply-plan/review,
consent, run recovery, and support export paths preserve the same target axis.
Web Console re-apply now obtains and reviews the target's current plan before
starting a consented durable run with stable idempotency.

Focused native tests pass:

* `cd scenarios/vrooli-onboarding/api && go test ./internal/operatorinputs ./handlers/operatorinputs ./internal/targetproxy ./handlers/readiness ./handlers/apply`
* `cd scenarios/vrooli-onboarding/cli && go test ./domains/wizard ./domains/selection`
* `cd scenarios/web-console/api && go test . -run '^$'`

Fresh maintained owner API receipts pass for vrooli-onboarding
`20260910-001356-772ea770`, vrooli-bridge `20260910-001411-5f8d542c`, and
web-console `20260910-001419-4e2c74f1`. Bridge and Web Console retain their
existing API-health warning findings; those are not Phase 8 regressions.
Plan Manager validation operations `57224871-3a96-4e5d-a431-c494bfa750be`
and `95e16508-9e9b-4d47-a2e6-fa8f50e22d2a` are retained as stale/unknown
because the shared worktree's relevant source identity changed during
producer validation. No real-target or physical-device proof is claimed.

### 2026-09-10 — Minimouse configuration RPC compatibility remains open

**Symptom:** Web Console lists the registered `minimouse` node as reachable,
but its Configuration tab cannot load operator questions. The live path ends
with Web Console HTTP 503 and Bridge HTTP 502.

**Root cause:** The target is reachable and dispatchable, but its remote
`vrooli-onboarding` service returns HTTP 404 for the current
`OperatorInputsService/ListOperatorInputs` route. The local onboarding service
answers the same request with HTTP 200, so the local Bridge/Web Console route
is not the failing layer.

**Workaround:** Inspect the technical detail and target onboarding health. Do
not treat Re-apply as a repair for a missing remote route; use the local
focused tests and preserve the target as unqualified.

**Real fix:** Refresh or redeploy the target-side onboarding service/profile
through an authorized target operation, then repeat the configuration,
readiness, answer, and re-apply journey with independent remote evidence.

**Owner:** Bridge deployment validation / authorized minimouse target owner.

**Refs:** `local:web-console/minimouse-get-configuration-20260910`,
`vrooli-bridge://minimouse/6a43fa2a-5749-4c79-a4b2-c464cb8bfc02`,
scenario-qa bug `knw-1789036207323889110`, and the certification index under
`/home/matthalloran8/.vrooli/plan-artifacts/vrooli-bridge-deployment-foundation-certification/evidence/`.

### 2026-09-10 — Scenario proxy now preserves target compatibility failures

The target-aware raw-protobuf proxy now classifies node-reported scenario
failures before they leave Bridge. A missing target procedure is surfaced as
Connect `failed_precondition` with `target_incompatible`, while timeouts,
authorization failures, and target 5xx responses retain distinct retry and
recovery semantics. The Web Console maps those classifications to an
operator-facing onboarding incompatibility state. This prevents a target API
contract mismatch from appearing as an undifferentiated Bridge 502.

Focused evidence: `go test ./handlers/scenario ./internal/scenario` in
`scenarios/vrooli-bridge/api`, plus the Web Console configuration and error
mapping regressions. The target-side minimouse route remains unresolved until
an authorized refresh or redeployment supplies the missing procedure.
