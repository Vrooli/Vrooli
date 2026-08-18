# Flows — Token Economy

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

`health` is a stateless reporting domain and ships no workflows. Plain CRUD in
`mints`, `catalog` and `holders` — create a token type, add a catalog entry,
add a holder — has no ordering constraints and deliberately does not appear
here.

Six flows carry real ordering constraints. **Redemption approval is the one
that must be modeled first and hardest**: it is the only flow where an ordering
mistake spends a balance twice, and the append-only journal has no repair verb
by design.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Redemption approval | redemption | A holder redeems a catalog entry. | The entry is received and the balance is debited exactly once, or the request is denied and the reservation released. | Reservation held while pending; approval or denial is terminal; stale approval after denial is illegal; settlement is idempotent under a key. | Level 5 — matrix, traces, declarative contract, checked Quint model, production replay. |
| Earning submission | earning | An adapter or the operator reports work done. | A grant is issued once, or the replay is a successful no-op. | Dedup precedes grant; replay must not re-grant; provenance is captured before the journal write. | Level 5. |
| Savings reservation | redemption | A holder reserves balance toward a catalog entry (P1). | Balance becomes unspendable until released or consumed by its target redemption. | Release and consume are mutually exclusive terminal transitions; double-release is illegal. | Level 4 initially, Level 5 with the redemption model it shares. |
| Recurring grant issuance | grants | A schedule reaches its next-issue time (P1). | Exactly one grant per window, with a declared catch-up policy for missed windows. | Issue-once-per-window; cancel stops future issuance without reversing past ones; an offline gap must not multiply or silently skip. | Level 5 — the offline-gap case is precisely what a formal model catches. |
| Grant expiry and decay | grants | A read observes a grant past its expiry or decay point (P1). | The reduction is materialized as a journal event, never applied silently. | Lazy evaluation must be idempotent — two concurrent reads must not double-expire. | Level 4. |
| Holder-to-holder transfer | holders | A holder sends tokens to another holder (P1). | Sender debited and recipient credited, or neither. | Credit-before-debit is illegal; partial application is illegal; the sender learns success only, never the recipient's balance. | Level 5. |
| Projection rebuild | journal | Operator or startup integrity check. | The balance projection is reconstructed from events. | Rebuild must be safe to interrupt and safe to repeat; no mutation may interleave with a rebuild in a way that loses an event. | Level 3 — matrix and traces; a declarative contract is likely overkill for a maintenance verb. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships an `Attachment upload` flow on the `notes` domain as a
worked Level 5 temporal-workflow vertical slice. Copy its shape for your
own stateful flows, then remove it.

Add this row to the Flow Inventory above:

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Attachment upload | notes | User/CLI uploads a file for a note. | Blob is stored and metadata is persisted. | Stateful upload request with validation and failure paths. | Level 5 workflow tests: matrix, traces, declarative spec, checked Quint model, generated artifacts, and production replay. |

#### Attachment upload

- Owner domain: notes.
- Trigger: multipart upload request from UI or CLI.
- Inputs: note id, file key/name, file bytes, content type, file size.
- Steps:
  1. Parse multipart request.
  2. Validate note id and file metadata.
  3. Store opaque bytes through BlobStore.
  4. Persist attachment metadata through notes repository seam.
  5. Return proto-typed metadata response.
- Outputs: uploaded attachment metadata or typed error response.
- Failure modes: missing note id, missing file, invalid metadata, blob
  write failure, metadata persistence failure.
- Retry/cancel behavior: caller may retry after transport/storage
  failure; duplicate handling belongs to the owning real domain when
  product requirements demand it.
- Tests: `api/handlers/notes/attachments_handler_test.go`,
  `api/internal/notes/attachments_service_test.go`,
  `api/internal/notes/flow/flow_test.go`,
  `ui/src/features/notes/AttachmentUpload.test.tsx`, and
  `ui/src/features/notes/flow/flow.test.ts`.
- Generated subpackages: `api/internal/notes/flow/generated/`
  (`model.qnt`, `artifact.json`, `runtime.go`, `replay.go`) and
  `ui/src/features/notes/flow/generated/` (`model.qnt`, `artifact.json`,
  `runtime.ts`, `replay.helper.ts`).
- Requirements: template starter only.

These example state machines belong in the State Machines table below:

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| notes / attachment upload API | received, bytes_stored, metadata_recorded, failed | metadata before bytes, terminal-state escape, duplicate terminal events | `*.flow.json` contract, generated Quint model, generated formal artifact replay, side-effect cleanup tests |
| notes / attachment upload UI | idle, selected, uploading, succeeded, failed | start before select, stale completion after reset/reselect, retry without file context | `*.flow.json` contract, generated Quint model, generated formal artifact replay, attempt-id stale completion tests |
<!-- EXAMPLE-DOMAIN:notes END -->

### Redemption approval

- Owner domain: redemption.
- Trigger: a holder redeems a catalog entry through the holder service.
- Inputs: holder id, catalog entry id, caller-supplied idempotency key.
- Steps:
  1. Resolve the catalog entry and its approval posture; refuse with a typed
     reason if unavailable or outside its window (`TKE-P0-008`).
  2. Evaluate the holder's grants and rules server-side; refuse naming the rule
     that refused (`TKE-P0-003`).
  3. Reserve the cost so it cannot be spent twice while the request is open.
  4. If the posture is immediate, settle. If gated, enqueue for the minter.
  5. On settle: debit and append the journal event **in one transaction under a
     row lock**, keyed by the idempotency key (`TKE-P0-009`).
  6. On deny: release the reservation and append an event recording the reason.
- Outputs: a settled redemption, a pending redemption, or a typed refusal.
- Failure modes: unavailable entry, rule refusal, insufficient spendable
  balance, reservation conflict, storage failure between debit and event write.
- Retry/cancel behavior: a repeated idempotency key returns the first result and
  does not debit again. A holder may cancel a pending request, which releases
  the reservation. A minter decision on an already-decided request is rejected
  as a stale transition rather than silently overwriting.
- Tests: `api/internal/redemption/settlement_test.go`,
  `api/internal/redemption/approval_test.go`,
  `api/internal/redemption/failure_injection_test.go`,
  `api/handlers/redemption/approval_test.go`,
  `bas/cases/redemption-approval.json`, and the generated flow replay.
- Requirements: `TKE-P0-009`, `TKE-P0-013`, and `TKE-P1-005` once
  reservations serve savings goals too.

### Earning submission

- Owner domain: earning.
- Trigger: an adapter — a chore scenario, a habit tracker, a webhook, or the
  operator pressing a button in the console — reports work done.
- Inputs: adapter identity, dedup key, holder id, token type, what was done.
- Steps:
  1. Authenticate the adapter; capture actor provenance through the shared
     `packages/cli-core` verifier (`TKE-P0-011`).
  2. Check the dedup key. A replay short-circuits to the original result.
  3. Construct a grant request through the one grant contract — never a direct
     journal write (`TKE-P0-002`).
  4. Issue the grant and append its event.
- Outputs: a grant, or an idempotent no-op on replay.
- Failure modes: unknown token type, unknown holder, adapter not authorized for
  the type, dedup storage failure.
- Retry/cancel behavior: adapters are expected to retry; replay safety is the
  whole point of the dedup key. There is no cancel — a mistaken earn is
  corrected by a compensating event (`TKE-P0-010`), not by deletion.
- Tests: `api/internal/earning/service_test.go`,
  `api/handlers/earning/handler_test.go`,
  `api/handlers/earning/provenance_test.go`, and the generated flow replay.
- Requirements: `TKE-P0-007`, `TKE-P0-011`; later `TKE-P1-009`.

### Recurring grant issuance (P1)

- Owner domain: grants.
- Trigger: a schedule reaches its next-issue time, or a read observes that one
  or more windows elapsed while the instance was off.
- Inputs: schedule id, current time, catch-up policy.
- Steps: compute elapsed windows → apply the declared catch-up policy (issue
  once, issue all, or skip) → issue grants → advance the next-issue pointer.
- Outputs: zero or more grants and an advanced pointer.
- Failure modes: **the offline gap** — an instance off for three weeks must not
  silently skip three allowances or silently issue three at once without the
  policy saying so. This is the specific case the formal model exists to catch.
- Retry/cancel behavior: issuance is idempotent per window key. Cancelling stops
  future issuance and never reverses past grants.
- Tests: `api/internal/grants/schedule_test.go` and the generated flow replay.
- Requirements: `TKE-P1-003`.

### Holder-to-holder transfer (P1)

- Owner domain: holders.
- Trigger: a holder sends tokens to another holder where type policy permits.
- Steps: check type transfer policy → reserve on the sender → debit sender and
  credit recipient in one transaction → append both events.
- Failure modes: policy forbids, insufficient balance, recipient unknown.
- Retry/cancel behavior: idempotent under a key; no partial application is
  reachable. The sender's response reports success only and never discloses the
  recipient's resulting balance, preserving the isolation boundary
  (`TKE-P0-006`).
- Tests: `api/internal/holders/transfer_test.go` and the generated flow replay.
- Requirements: `TKE-P1-004`.

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| redemption / approval (API) | `requested`, `reserved`, `pending_approval`, `approved`, `settled`, `denied`, `released`, `refused` | Settle before approval on a gated entry; settle twice under one key; decide an already-decided request; release after settle; reserve without an available entry | `flow.json` contract, generated Quint model, replay against production transition, failure-injection tests |
| redemption / redeem (UI) | `idle`, `confirming`, `submitting`, `pending`, `succeeded`, `refused` | Submit without confirmation; stale success arriving after a cancel or a new selection; retry without the original entry context | `flow.json` contract, generated Quint model, attempt-id stale-completion tests |
| earning / submission (API) | `received`, `deduped`, `granted`, `journaled`, `noop_replay`, `rejected` | Grant before dedup check; journal before grant; re-grant on a seen key; journal without provenance resolved | `flow.json` contract, generated Quint model, replay tests |
| grants / recurring issuance (API) | `scheduled`, `due`, `catching_up`, `issued`, `cancelled` | Issue twice for one window key; issue after cancel; advance the pointer without issuing under an issue-all policy | `flow.json` contract, generated Quint model, offline-gap named traces |
| grants / expiry and decay (API) | `active`, `expiring`, `expired` | Reduce a balance without appending an event; double-expire under concurrent reads | `flow.json` contract, generated Quint model, concurrent-read tests |
| holders / transfer (API) | `initiated`, `reserved`, `applied`, `failed` | Credit before debit; apply partially; disclose recipient balance to the sender | `flow.json` contract, generated Quint model, replay tests |
| journal / projection rebuild | `idle`, `rebuilding`, `complete`, `interrupted` | Serve a projection read as authoritative mid-rebuild; lose an event appended during a rebuild | Matrix and trace tests; rebuild is interruptible and repeatable by construction |

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
| Rule-program evaluation (`TKE-P1-002`) | Medium. A declared condition that mints automatically is a state machine whose trigger is journal state rather than a user action, so its ordering risk is real but its surface is small — the condition vocabulary is closed. | Model at Level 4 once the condition vocabulary is fixed. Do not model earlier; the states depend on which conditions exist. |
| Minter approval delegation (`TKE-P2-005`) | Low today, high if built naively. Delegation that can widen authority is the classic failure, and the ecosystem already solved it once in agent-manager's `Attenuate()`. | Reuse the existing one-way attenuation semantics rather than modeling a new flow. If that reuse turns out not to fit, model at Level 5 before writing the handler. |
| Real-value settlement (`TKE-P2-001`) | High, and deliberately not modeled. A chain-backed rail has irreversible terminal states, which is a different risk class from everything above. | Blocked on a recorded custody and regulatory decision. Model at Level 5 as the first implementation step, not the last. |
| Cross-instance token recognition (`TKE-P2-002`) | High. Two journals that must agree without a shared transaction is a distributed-consensus problem wearing a household costume. | Do not model until `vrooli-bridge` relay semantics are settled; the flow's states depend on what the bridge guarantees. |
| Marketplace listing and offer (`TKE-P2-003`) | Medium. Listings reserve balance, and reservation ordering is already modeled for redemption. | Extend the redemption reservation model rather than adding a parallel one. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
