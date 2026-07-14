# Flows — Vrooli Bridge

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, polling, or mutually
exclusive UI modes.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a
workflow model.

## Flow Inventory

`health` is a stateless reporting domain and ships no workflows. List
each real stateful flow your domains add below, with its owner, trigger,
outcome, statefulness, and validation level.

These are bridge's intended stateful flows. None is modeled yet — this is the documentation-first foundation, so each is at **Level 1 (inventory)** and listed again under Deferred / Unmodeled Flows with its next step. Several (dispatch+run, provisioning) are genuinely Level-5-worthy because they carry retries, cancellation, rollback, and stale-completion risk across a network boundary.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Node bootstrap & pairing | pairing | Operator runs the one-touch installer with a pairing token. | A paired, mutually-authenticated, online node. | issued → installing → dialed-out → redeemed → trusted; failure/expiry paths. | Target Level 4–5. |
| One-shot node onboarding | onboard | Operator points the control plane at a raw SSH host. | A paired, ONLINE, auto-starting node — driven end-to-end from bare OS. | pending → ssh_setup → pushing_script → bootstrapping → verifying → (succeeded \| failed \| cancelled); durable, cancellable, restart-reconciled. | **Level 3 (built, tested).** |
| Job dispatch & durable run | dispatch + runs | Operator/automation dispatches a typed job to a node. | A terminal run verdict with logs/artifacts, re-attachable by id. | queued → dispatched → running → (passed/failed/aborted); survives disconnect; stale-completion risk on reconnect. | Target Level 5. |
| Provisioning / sync-to-revision | provisioning | Control plane brings a node to revision R (or updates the fleet). | Node at revision R with reported version, or rolled back. | requested → fetching → setup → verify → (ready/rolled-back); idempotent re-runs. | Target Level 5. |
| Cross-OS deployment gate | gate | deployment-manager requests a cross-OS validation. | Aggregated per-OS verdict → single pass/fail. | fan-out → per-node runs → aggregate → terminal; any-OS-fails-fails-gate. | Target Level 4. |
| Node presence | presence | Agent dial-out connect/disconnect/heartbeat. | Accurate online/offline + readiness state. | offline → online → (heartbeat) → offline; ephemeral. | Target Level 2–3. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

### Remote run

The dispatch → durable-run flow (OT-P0-004/005, built in Phase 3). Owner
dispatches a typed `{scenario, verb, args}` job to a node; the control plane
validates it against the allowlist + the node's scopes, creates a server-owned
run, audits the dispatch, and pushes the typed `JobPush` down the node's held
dial-out channel. The node-agent runs it as the non-privileged runner and
streams `RunEvent`s (STATUS/LOG/EXIT/ARTIFACT_REF) back via the node-facing
`RunsService.ReportRunEvent`.

States: `queued → running → (passed | failed | aborted)`. The run is
**server-owned and durable**: it survives the dispatching client disconnecting
and is re-attachable by id (`runs get`), with a block-once `runs wait` that
returns exactly once on the terminal transition (no polling) or `timed_out` when
its wait window elapses. **Stale-completion safety:** a late event for an
already-terminal run is acknowledged but never changes the verdict. Implemented
at Level 1 (inventory) today; the Level-5 Quint model is future work (the
service's `coordinator` already centralises the transition logic for it).

Code: `internal/dispatch`, `internal/runs`, `agent/internal/exec`. Tests:
`internal/runs/durable_test.go`, `internal/runs/results_test.go`,
`handlers/runs/connect_handler_test.go`.

### One-shot node onboarding

The `onboard` domain (phase 5) is the **orchestration tier** that turns a raw,
SSH-reachable host into a paired, ONLINE, auto-starting fleet node in one durable
`OnboardingOp`. It sequences the artifacts the earlier phases built — SSH
first-touch (phase 1), the server-side pairing code (phase 3), and the idempotent
node bootstrap script (phase 4) — so the operator kicks off `onboard start` and
walks away.

**States (persisted):** `pending → ssh_setup → pushing_script → bootstrapping →
verifying → (succeeded | failed | cancelled)`. `succeeded/failed/cancelled` are
terminal; `WaitOnboarding` is the block-once verb that returns exactly once on a
terminal transition (no polling) or `timed_out` when its window elapses.

**Ordered steps (server-side orchestration on a detached context):**

1. **ssh_setup** — SSH first-touch: generate the bridge key, install it with the
   owner's password, retest passwordless. The password is used once and zeroed.
2. **pushing_script** — SCP the bootstrap script to the node.
3. **bootstrapping** — issue a single-use pairing code SERVER-SIDE, then run the
   bootstrap script remotely, injecting the code over **stdin** (env-only, never
   argv/logs). Each `VBOOTSTRAP` stdout marker is parsed into a persisted
   `OnboardingStepEvent` (the 12 bootstrap steps + the run envelope), so the
   progress trail matches the script's actual emitted steps.
4. **verifying** — the node id is captured from the pair-redeem/run-ok marker;
   the orchestrator confirms the node is ONLINE in the fleet (holds a live
   dial-out channel ⇒ its control-plane key is pinned).

**Failure taxonomy** (recorded on the FAILED op, machine-branchable):
`ssh_setup_failed`, `script_push_failed`, `pairing_issue_failed`,
`bootstrap_usage_error` (exit 2), `unsupported_platform` (exit 3),
`pairing_failed` (exit 4), `bootstrap_failed` (exit 1/other),
`verify_online_failed`, `interrupted_by_restart`. Every step is idempotent, so a
FAILED op is always safe to retry.

**Durability & interruption:** the op is server-owned and survives the operator
disconnecting. `CancelOnboarding` cancels the op-scoped context; the orchestrator
(the single writer of op state) observes it at the next boundary and drives the op
to CANCELLED. On a control-plane restart, `ResumeInterrupted` (run once at boot)
reconciles any op left non-terminal to FAILED/`interrupted_by_restart` — never
left dangling — because a mid-flight SSH/bootstrap is not checkpointable, and a
re-run converges (idempotent).

**Secrets never persist:** there is no column or event field for the SSH password
(request-scoped, zeroed after the key install) or the pairing code (issued
server-side, injected over stdin, zeroed after the remote exec). Both are asserted
absent from DB rows, step events, and captured logs in tests.

**Tests:** `internal/onboard/service_test.go` (state machine incl. cancel +
failure at each phase, dry-run, resume, secret-non-persistence, block-once wait),
`internal/onboard/marker_test.go` (VBOOTSTRAP parsing + node-id extraction),
`internal/onboard/sqlite_test.go` (durable round-trip + migrate-never-recreate),
`handlers/onboard/integration_test.go` (full flow through the Connect handler over
a real SSH/SCP/streaming round-trip against an in-process sshd + stub bootstrap;
env-only code delivery + non-leakage; real-SQLite restart-resume). Generated
subpackage: `packages/proto/.../v1/onboard`.

### Cross-OS deployment gate

The cross-OS validation flow (OT-P1-002, built in Phase 6). Owner — or
deployment-manager on the owner's behalf — calls `RunGate(scenario,
target_revision, target_oses[])`. The gate domain:

1. **Selects** one eligible node per target OS — registered + online +
   protocol-compatible + not-revoked + matching OS (lowest node id wins for
   determinism). A target OS with no eligible node is recorded `NO_NODE`.
2. **Fans out** a validation run per selected node by delegating to the SHARED
   dispatch service (the default verb is `scenario test <scenario>`), so each run
   is allowlist-checked, scoped, audited, and tracked as a durable run exactly
   like any other job. Gate reimplements neither dispatch nor run management.
3. **Records** a durable `Gate` + per-OS ledger and returns the gate id
   immediately (the per-OS runs execute concurrently on their nodes).

The aggregate verdict is **recomputed live** from the per-OS runs:
`GetGate` reads the current state without blocking; `WaitGate` blocks once until
every target run is terminal (or the wait window elapses) and returns the final
verdict — no polling. **Aggregation rule:** PASSED only when every OS is green;
ANY failing OS — a non-zero/aborted run OR a `NO_NODE` target — fails the gate
(fail-fast, even while other targets still run), with the offending OS's run id
surfaced for log drill-in. A timed-out gate stays PENDING (`timed_out=true`); it
is durable and re-attachable by id, never assumed green.

deployment-manager consumes this over Connect/JSON (`crossosgate`) and owns the
resulting production-readiness decision (see [`INTEGRATIONS.md`](INTEGRATIONS.md#deployment-manager)).

Code: `internal/gate`, `handlers/gate` (delegates to `internal/dispatch` +
`internal/runs`). Tests: `internal/gate/aggregate_test.go`,
`internal/gate/service_test.go`, `handlers/gate/deployment_manager_test.go`.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| _(your flow)_ | The ordered/terminal states. | Transitions the contract forbids. | `*.flow.json` contract, generated Quint model, replay tests. |

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

## Production Shape

Three (Go) or four (UI) files per flow at the top of the feature folder,
plus one `generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.go               # hand: wrapper (package flow)
    flow_test.go                # hand: thin replay delegation (package flow)
    generated/
      model.qnt
      artifact.json
      runtime.go                # package generated
      replay.go
```

UI features that own client-side modes use:

```text
ui/src/features/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.ts               # hand: wrapper
    fixtures.ts                 # hand: replay fixtures
    flow.test.ts                # hand: thin replay delegation
    generated/
      model.qnt
      artifact.json
      runtime.ts
      replay.helper.ts
```

Every flow uses the same file names. The `flow/` directory IS the unit;
the contract no longer declares any output paths or module names.

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`. It should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, BlobStore, clocks,
timers, HTTP clients, or UI API modules.

The `*.flow.json` contract is the source of truth. Level 5 generated
Quint models, formal artifacts, and Go/TypeScript declarations are
checked-in source artifacts for reviewability, but they are refreshed
and checked by the `flow-verifier` scenario CLI; the
scenario lifecycle runs `make temporal-models` (which calls
`flow-verifier verify check`) before the normal test
suite. A Quint file by itself is not accepted: the model must typecheck,
test, verify named invariants, emit deterministic artifacts, and those
artifacts must replay against the production Go/TypeScript transition
functions.

The generated declarations keep state/event topology and formal
freshness metadata out of hand-maintained test lists. They also provide
pure status-transition helpers generated from the `*.flow.json`
transition matrix. For TypeScript flows, the same declarations can own
the discriminated state/event union shape and replay fixture contract.
Production workflow wrappers call those helpers for abstract validity
and next-status outcomes, while keeping payload validation, side-effect
orchestration, and rich state construction in hand-authored code. API
replay tests get expected paths, hashes, invariants, and generated checks
from `generated/<folder>/runtime.go`; UI replay tests import the same metadata
from `generated/<folder>/runtime.ts`. The generated `replay.{go,helper.ts}`
files own the assertion calls; the hand-authored top-level test simply binds
the wrapper's transition function and the fixtures and invokes
`RunReplay`/`runFormalReplay` once.

Formal artifacts use schema v6 coverage metadata. Matrix completeness,
terminal transition checks, named trace coverage, and generated MBT trace
coverage are separate fields. Do not treat generated trace
`allPairsCovered` as required proof of correctness; replay tests require
the complete transition matrix and named traces, while generated trace
coverage reports how much the model explorer happened to visit.

Schema v6 `flow.json` files carry no path or module information. The
`replay` block declares only `transition.function` (plus
`transition.statusAccessor` for TS or `transition.stateType` /
`transition.statusField` for Go). Everything else is derived from the
flow directory.

Go flows emit `flow/generated/replay.go` and require a hand-authored
`flow/flow_test.go` (package `flow`) that calls `generated.RunReplay`.
TypeScript flows emit `flow/generated/replay.helper.ts` and require a
hand-authored `flow/flow.test.ts` that calls
`runFormalReplay({ transition, fixtures })` at module top level.
`flow-verifier verify check` byte-compares every generated file and runs an
AST-level lint over the hand-authored test, so a silent bypass — missing
import, stubbed transition, or call buried inside a guarded block —
fails the check.

To scaffold a new flow:

```bash
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
```

The scaffold writes the hand-authored files and immediately runs
`generate`, so `check` is green from the moment it returns.

To add or rename a state/event:

1. Edit the owning `*.flow.json`.
2. Regenerate that flow with `flow-verifier verify run --flow <flow-id>`.
3. Update only payload-specific wrapper branches that need new runtime
   data; the abstract transition table is generated.
4. Update the UI replay fixture module. The generated formal replay fixture
   interface should make missing state/event fixtures a type error.
5. Run `make temporal-models` and the scenario tests.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Job dispatch & durable run | High — network boundary + disconnect + cancellation + stale completion; mirrors the risks test-genie's durable runs already solve. | Model at Level 5 once the runs domain exists; reuse test-genie run-lifecycle semantics rather than reinventing. |
| Provisioning / sync-to-revision | High — partial/failed setup must roll back cleanly; idempotency is load-bearing. | Model at Level 5 with explicit rollback states when the provisioning domain exists. |
| Node bootstrap & pairing | Medium — single-use codes, expiry, and mutual-auth handshake have ordering constraints. | Model at Level 4 when the pairing domain exists. |
| Cross-OS deployment gate | Modeled (Level 1 inventory + detail above) as of Phase 6. | Lift to a Level-4/5 Quint model of the fan-out + partial-failure aggregation when the live A/B warrants it. |
| Node presence | Low — simple online/offline with heartbeat. | Model at Level 2–3 when the presence domain exists. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
