# Domains — Vrooli Bridge

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `vrooli scenario detemplate <scenario>` removes every
fenced example once your real domains are green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/vrooli-bridge/v1/health/` |
| registry | Paired Node identity and explicitly approved scopes. | CRUD / entity | `nodes`. | API, CLI, UI | OT-P0-001 | `api/internal/registry/`, `cli/domains/nodes/`, `ui/src/features/fleet/` |
| pairing | One-touch bootstrap, single-use pairing codes/tokens, mutual-auth credentialing, atomic revocation. | Workflow / security | `pairing_codes`, node credentials. | API, CLI, UI | OT-P0-002 | `api/internal/pairing/`, `api/internal/auth/` |
| presence | Dial-out channel management; online/offline presence and self-reported node health. | Realtime / reporting | Ephemeral presence + health snapshots (optionally Redis). | API, UI | OT-P0-003 | `api/internal/presence/` |
| dispatch | Validate `{scenario, verb, args}` against the CLI manifest + per-node scopes; dispatch typed jobs. | Policy / command | Job definitions, allowlist decisions. | API, CLI, UI | OT-P0-004 | `api/internal/dispatch/` |
| runs | Durable server-owned remote runs; stream exit/logs/artifacts back; re-attach by id. | Workflow / lifecycle | `runs`, logs, artifact refs. | API, CLI, UI | OT-P0-005 | `api/internal/runs/` |
| provisioning | Privileged tier: sync a node to revision R (`vrooli setup`), update, version-pin, rollback. | Workflow / privileged | `provisioning_ops`, `provision_events`, `node_versions`. | API, CLI | OT-P0-006, OT-P1-001 | `api/internal/provision/`, `api/handlers/provision/`, `cli/domains/provision/`, `agent/internal/privsep/`, `packages/proto/schemas/vrooli-bridge/v1/provision/` |
| gate | Cross-OS deployment gate: select one eligible node per target OS, dispatch native validation to each (delegating to dispatch+runs), aggregate per-OS verdicts into one cross-OS deployment-readiness result. | Aggregation / workflow | `gates`, `gate_os_results`. | API, CLI | OT-P1-002 | `api/internal/gate/`, `api/handlers/gate/`, `cli/domains/gate/`, `packages/proto/schemas/vrooli-bridge/v1/gate/` |
| audit | Append-only trail of every dispatch and provisioning op (actor/node/verb/args/outcome). | Reporting / security | Audit records (via workspace-sandbox). | API, UI | OT-P0-008 | `api/internal/audit/` |
| fleet | Fleet-wide version roll: pin every (or named) node to a target revision by fanning provisioning out across the fleet; per-node rollout ledger + protocol-compat gating. | Aggregation / workflow | `rollouts`, `rollout_results`. | API, CLI, UI | OT-P1-001 | `api/internal/fleet/`, `api/handlers/fleet/`, `cli/domains/fleet/` |
| queue | Per-node bounded-concurrency + fair-FIFO scheduler on the dispatch→push path; read-only control-plane view of running-vs-queued. | Policy / realtime | In-memory scheduler state (the run is the durable record). | API, CLI, UI | OT-P1-004 | `api/internal/queue/`, `api/handlers/queue/`, `cli/domains/queue/` |
| artifacts | Distribute non-git artifacts (installers, fixtures) to nodes via device-sync-hub directed delivery, and accept bounded typed outputs produced by authenticated runs for owner retrieval. | Integration / workflow | `distributions` plus run-produced artifact bytes and references. | API, CLI, agent | OT-P1-003 | `api/internal/artifacts/`, `api/handlers/artifacts/`, `cli/domains/artifacts/` |
| machines | Durable operator intent for a managed machine: stable identity, ordered connection locators, lifecycle, desired policy reference, trust references, and immutable Node lineage. | Entity / lifecycle | `machines`, `machine_locators`, `machine_node_lineage`, migration-review records. | API, CLI, UI | BRG-MEC-001, BRG-MEC-002 | `api/internal/machines/`, `packages/proto/schemas/vrooli-bridge/v1/machines/` |
| enrollment | Immutable attempts to enroll a Machine, including checkpoints, typed pairing correlation, retry lineage, and recovery diagnostics. | Workflow / recovery | `enrollment_attempts`, checkpoint and reconciliation records. | API, CLI, UI | BRG-MEC-003 | `api/internal/onboard/attempts.go`, `packages/proto/schemas/vrooli-bridge/v1/onboard/onboard.proto`, and the Machine detail projection |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### registry

- Purpose: register, list, inspect, and revoke paired Vrooli Nodes; hold the paired identity and explicitly approved scopes.
- Primary archetype: CRUD / entity. Distinct from device-sync-hub's ephemeral sync-peer grants — nodes are persistent infrastructure.
- Owns: the `nodes` table and Node revocation. Does not own: durable operator Machine intent/lifecycle or locators (machines), pairing/credential issuance (pairing), execution (dispatch/runs), or Presence.
- Related docs: [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md).

### pairing

- Purpose: the one-touch bootstrap and trust establishment — issue/redeem single-use pairing codes/tokens, mint mutual-auth credentials, and revoke them atomically (killing both job and provisioning rights).
- Primary archetype: workflow / security. Models on device-sync-hub's proven pairing mechanics, with a richer, longer-lived node record.
- Owns: `pairing_codes` and per-node credentials. Does not own: the node record itself (registry).
- Related docs: [`../internal/SECURITY.md`](../internal/SECURITY.md), [`FLOWS.md`](FLOWS.md).

### machines

- Purpose: own durable operator intent before any network contact. A Machine is
  not a paired Node, a current health snapshot, an SSH key, or an enrollment
  attempt.
- Owns: stable UUID identity; normalized, ordered connection locators;
  archive/removal lifecycle; desired-policy reference; opaque trust references;
  and immutable references to Node lineage.
- Does not own: Registry Node identity and approved scopes; Presence liveness;
  private key bytes or host keys; pairing-code lifecycle; capability
  observations; or external workload truth.
- Invariant: locator equality is advisory only. Host, IP, username, display
  name, and Node name never establish Machine identity or cause automatic
  historic-data adoption.

### enrollment

- Purpose: record one immutable enrollment attempt for one Machine. It carries
  the input snapshot, correlation id, typed checkpoints, diagnostics, terminal
  outcome, and optional `retry_of_attempt_id`.
- Owns: attempt lifecycle and reconciliation state. A retry always creates a
  new attempt; a terminal attempt is never reopened, deleted by retry, or
  mutated into success.
- Does not own: the Machine lifecycle, Registry Node record, Presence, or SSH
  secret material. It references each owner instead of copying their mutable
  fields.
- Related docs: [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md), and
  [`../internal/SECURITY.md`](../internal/SECURITY.md).

### presence

- Purpose: manage each node's persistent dial-out channel; track online/offline presence and self-reported readiness (toolchain, disk, container runtime) so dispatch only targets capable nodes.
- Primary archetype: realtime / reporting. Presence is in-memory single-instance, optionally Redis-backed for scale-out (device-sync-hub's pattern).
- Owns: ephemeral presence + health state. Does not own: durable run state (runs).
- Related docs: [`FLOWS.md`](FLOWS.md), [`../internal/SEAMS.md`](../internal/SEAMS.md).

### dispatch

- Purpose: the safety gate for remote execution — validate every `{scenario, verb, args}` job against the scenario-CLI manifest plus the target node's transport and `<namespace>:<effect>` catalog grants before anything runs; never construct raw shell.
- Primary archetype: policy / command. The allowlist surface is the manifest-declared verb set.
- Owns: job definitions and allowlist decisions. Does not own: the running of the job (runs) or privileged setup (provisioning).
- Related docs: [`../internal/SECURITY.md`](../internal/SECURITY.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### runs

- Purpose: durable, server-owned remote execution — a dispatched job survives client/agent disconnect, is re-attachable by id with a block-once wait, and streams exit status, logs, and artifacts back for aggregation. Reuses test-genie's run-lifecycle philosophy (no polling).
- Primary archetype: workflow / lifecycle.
- Owns: the `runs` table, log/artifact references. Produced artifact bytes are owned by the artifacts domain and remain owner-scoped to their run. Does not own: cross-OS verdict aggregation (gate).
- Related docs: [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### provisioning

- Purpose: the privileged tier — bring a node to a target project revision R (fetch source + idempotent `vrooli setup`), update, version-pin, and roll back on failed setup. Structurally separated from the non-privileged runner so everyday jobs can never escalate privilege.
- Primary archetype: workflow / privileged.
- Owns: provisioning operations and per-node version history. Does not own: allowlisted job execution (dispatch/runs).
- Related docs: [`../internal/SECURITY.md`](../internal/SECURITY.md), [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

### gate

- Purpose: the headline capability — validate a scenario natively on one node per target OS (Ubuntu/macOS/Windows) at a revision and aggregate the per-OS verdicts into a single "production-ready on every OS" result that deployment-manager gates promotion on. Verbs: `RunGate` (fan out + durable record), `GetGate` (live verdict), `WaitGate` (block-once for the terminal verdict, no polling), `ListGates`.
- Primary archetype: aggregation / workflow. Bridge supplies the capability; deployment-manager owns the verdict.
- Delegates, never reimplements: each per-OS validation is dispatched through the SHARED dispatch service (allowlist + per-node scopes + audit) and tracked as a durable run (runs domain) via the `Runner` seam. Aggregation rule: ANY failing OS — a non-zero/aborted run OR a target OS with no eligible node — fails the gate; the verdict is recomputed live from the per-OS runs on read.
- Owns: the `gates` + `gate_os_results` ledger. Does not own: the individual node runs (runs), the allowlist gate (dispatch), or artifact transport (device-sync-hub).
- Consumer: deployment-manager's `crossosgate` package speaks `GateService` over Connect/JSON and owns the production-readiness decision (`POST /api/v1/cross-os-gate/evaluate`, additive + inert until `VROOLI_BRIDGE_URL` is set).
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md), [`FLOWS.md`](FLOWS.md), [`../internal/SEAMS.md`](../internal/SEAMS.md).

### audit

- Purpose: an append-only, immutable record of every dispatch and provisioning operation (actor, node, verb/args, outcome), because remote code execution and remote provisioning must be fully reconstructable after the fact.
- Primary archetype: reporting / security. Routed to workspace-sandbox as the accountability substrate rather than a bespoke store.
- Owns: audit record construction/query. Does not own: the operations themselves.
- Related docs: [`../internal/SECURITY.md`](../internal/SECURITY.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### Cross-cutting: the node-agent

The cross-compiled **node-agent** (OT-P0-007) is not a control-plane domain — it is the thin client installed on each node. It holds the dial-out channel (presence) in **presence-only posture by default**; verified job and provisioning frames are dropped until an explicit policy-approved control-action configuration opts in. When enabled, it accepts typed jobs (dispatch), runs them durably via the local `vrooli` CLI (runs), and executes privileged provisioning when explicitly invoked (provisioning). It installs itself as a platform-native service (systemd / launchd / Windows Service). It is bridge-owned so the root `vrooli` CLI stays scenario-agnostic. See [`ARCHITECTURE.md`](ARCHITECTURE.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Machine | Durable operator intent and lifecycle. | machines domain. |
| EnrollmentAttempt | Immutable execution history for an effort to enroll one Machine. | enrollment domain. |
| Node | A paired Registry identity. It can be absent for a managed SSH connection. | registry domain. |
| Presence | A live observation from the existing restricted agent, never durable Machine state. | presence domain. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

The remaining P2 capabilities are genuinely future. (The P1 set — fleet roll,
artifact distribution, per-node queue, fleet UI, mDNS — shipped in Phase 5: the
first three became their own domains above rather than extending an existing
one, the queue is in-memory scheduling over the durable runs, and mDNS is a
node-agent concern.)

| Candidate Domain / Capability | Why Deferred | Revisit Trigger |
|---|---|---|
| Control-plane portability to macOS/Windows (OT-P2-001) | Cross-cutting; gated on Vrooli-the-platform running on those OSes. | When the platform becomes installable on Mac/Win. |
| remote-desktop seam (OT-P2-002) | A *separate future scenario* (screen/input control) that reuses bridge's identity/reach, not a bridge domain. | When real-time remote control is built. |
| Cloud-runner / ephemeral nodes (OT-P2-003) | Extends the registry (node kind is metadata). | When on-demand VM/cloud capacity is needed. |
| Self-healing re-provisioning (OT-P2-004) | Extends provisioning with drift detection. | When fleet size makes manual re-provisioning costly. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
