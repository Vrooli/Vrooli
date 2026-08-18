# Flows — Treasury

This document is the canonical map of stateful workflows in this scenario:
what starts them, what states they move through, which transitions are
forbidden, and how that is enforced.

This scenario is unusually flow-heavy for its size, because almost nothing
here is plain CRUD. A mandate has a lifetime. An authorization reserves
something and must release it. A charge must happen exactly once across a
retry. An approval waits on a human who may never answer. Those ordering
constraints are the product, so they are modeled rather than left inside
handlers.

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

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Mandate lifecycle | mandate | An operator issues a grant. | A mandate that is live, exhausted, expired or revoked. | States; expiry is time-driven and needs no actor; revocation is terminal. | Level 4 target |
| Spend authorization | authorization | An agent proposes a charge. | A verdict, and a hold when approved. | States; holds must release on every terminal path including crash recovery. | Level 4 target |
| Approval resolution | approval | An authorization requires a human. | Approved, declined or expired. | States; retries on relay only; stale completion is real because humans are slow. | Level 4 target |
| Settlement | settlement | An approved authorization is executed. | Settled, failed or abandoned. | States; exactly-once under retry; the hard case is the unknown-outcome window. | Level 5 target |
| Standing mandate recurrence | mandate | A recurrence boundary is reached. | A new authorization, or a stopped obligation. | States; cancellation must beat the next charge. | Level 3 target |
| Ledger emission | ledger | A charge reaches a terminal outcome. | A money event accepted downstream. | Retries; idempotent; must not block settlement. | Level 3 target |
| Freeze propagation | budget | An operator freezes a scope. | Authorization refused at that scope. | Effect must precede the next authorization, not the next settlement. | Level 2 target |

## Flow Details

### Mandate lifecycle

- **Owner:** `mandate`.
- **Trigger:** an operator issues a grant, or a standing mandate reaches a
  recurrence boundary.
- **Inputs:** authorizer, book, budget, cap, counterparty scope, expiry,
  required evidence, optional recurrence.
- **Steps:** validate that every required field is present → persist as
  immutable → serve to the evaluator on request → refuse once expired,
  exhausted or revoked.
- **Outputs:** a mandate record; nothing else in the system may widen it.
- **Failure modes:** issuing a mandate broader than the budget backing it
  (refused at issue); a clock skew that makes expiry ambiguous (expiry is
  evaluated against stored UTC, never a client-supplied time).
- **Retry/cancel:** issue is idempotent on a caller-supplied key.
  Revocation is terminal and cannot be undone — reissue instead.
- **Why expiry needs no job:** expiry is evaluated from the mandate's own
  timestamp at read time, so a stopped scheduler cannot leave a stale
  mandate live. This is a deliberate rejection of the sweep-job pattern,
  which fails open.
- **Tests:** `TRS-P0-001`, `TRS-P0-012`, `TRS-P1-005`, `TRS-P2-006`.

### Spend authorization

- **Owner:** `authorization`.
- **Trigger:** an agent calls the agent-facing surface proposing a charge.
- **Inputs:** mandate reference, amount, counterparty, idempotency key,
  and the caller's verified identity claims.
- **Steps:** verify identity (fail closed) → load mandate → check live,
  unexpired, unrevoked → check counterparty against budget scope → check
  every cap dimension against computed headroom → check freeze state →
  either refuse with the specific violated constraint, or place a hold and
  return a verdict; when the budget requires approval, the verdict is
  `pending` and an approval request is raised.
- **Outputs:** an authorization record with a verdict; a hold when
  approved or pending; an evidence record either way.
- **Failure modes:** identity authority unreachable (refuse); mandate
  expires between load and hold (refuse); concurrent authorizations racing
  for the last headroom (resolved by the hold under row lock, so exactly
  one wins).
- **Retry/cancel:** retry under the same idempotency key returns the
  original verdict. A pending authorization can be cancelled by the caller,
  which releases its hold.
- **Trust boundary:** the evaluator reads only stored state.
  Caller-supplied verdict, allowance or override fields are ignored when
  present rather than rejected, so a probing caller learns nothing from
  the difference in response.
- **Tests:** `TRS-P0-002`, `TRS-P0-003`, `TRS-P0-005`, `TRS-P1-007`.

### Approval resolution

- **Owner:** `approval`.
- **Trigger:** an authorization whose budget declares `requires_approval`.
- **Inputs:** the authorization, its mandate, the amount and counterparty.
- **Steps:** admit to the queue → attempt relay through
  `notification-hub` if configured → wait → an operator approves or
  declines, or the request expires → resolve the parent authorization →
  release or convert the hold.
- **Outputs:** a resolution with resolver identity and timestamp; a
  relay-attempt record independent of the outcome.
- **Failure modes:** relay unreachable (recorded, outcome unaffected); the
  operator never answers (expiry resolves it as declined, and the expiry
  window is part of the budget's configuration, not a global default).
- **Stale completion:** an operator may approve a request whose mandate has
  since expired. The approval succeeds and the *settlement* refuses, so the
  evidence shows a human agreed and the contract still held. Collapsing
  these into one refusal would lose that distinction.
- **Retry/cancel:** relay retries with backoff; resolution is
  single-shot and terminal.
- **Tests:** `TRS-P0-006`.

### Settlement

- **Owner:** `settlement`.
- **Trigger:** an approved authorization is executed against a rail.
- **Inputs:** authorization, mandate, scoped instrument, rail adapter,
  idempotency key.
- **Steps:** lock the idempotency row → if a terminal outcome exists,
  return it unchanged → re-check mandate expiry and freeze → call the rail
  adapter → record the outcome → release the hold → write evidence →
  enqueue ledger emission.
- **Outputs:** a charge record; an evidence record; a queued money event.
- **Failure modes:** the genuinely hard one is the **unknown-outcome
  window** — the rail was called, the response was lost, and whether money
  moved is unknown. That is a first-class state, not an error. It resolves
  by querying the rail rather than by guessing, and it never
  auto-transitions to failed, because retrying an unknown is how double
  charges happen.
- **Retry/cancel:** retry is safe by construction under an unchanged key.
  There is no cancel after the rail is called; there is only resolution of
  the unknown.
- **Why level 5:** exactly-once semantics under concurrent retry with a
  partially-observable external system is exactly the class of property
  where a checked formal model earns its cost. This is the one flow where
  a hand-written test matrix is not convincing on its own.
- **Tests:** `TRS-P0-011`, `TRS-P0-012`, `TRS-P0-007`.

### Standing mandate recurrence

- **Owner:** `mandate`.
- **Trigger:** a recurrence boundary on a standing mandate.
- **Steps:** evaluate whether the obligation is still active → raise a
  fresh authorization for the period → surface the next charge date.
- **Failure modes:** a cancellation racing the boundary. Cancellation must
  win; the test asserts that ordering explicitly rather than assuming it.
- **Tests:** `TRS-P1-005`.

### Ledger emission

- **Owner:** `ledger`.
- **Trigger:** a charge reaches a terminal outcome.
- **Steps:** build a money event with adapter, external id, fetch time,
  amount and basis → emit → record acceptance → retry on failure with its
  own idempotency.
- **Constraint:** emission never blocks settlement. A downstream outage
  must not prevent money that already moved from being recorded locally;
  the emission queue drains later.
- **Basis rule:** `authoritative` when a rail confirmed the movement,
  `operator-asserted` for the manual rail.
- **Tests:** `TRS-P0-008`.

### Freeze propagation

- **Owner:** `budget`.
- **Trigger:** an operator freezes a budget, a book, or everything.
- **Constraint:** the freeze binds before the *next authorization*, not the
  next settlement. An already-authorized but unsettled charge is also
  stopped, which is what makes it a kill switch rather than a policy edit.
- **Tests:** `TRS-P1-006`.

<!-- EXAMPLE-DOMAIN:notes START -->
<!-- EXAMPLE-DOMAIN:notes END -->

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| mandate | `draft` → `live` → (`exhausted` \| `expired` \| `revoked`) | Any transition out of a terminal state; `expired` → `live` (a lapsed grant is never resurrected, only reissued); widening cap or scope after issue. | `mandate/flow/flow.json`, generated model, replay tests. |
| authorization | `evaluating` → (`refused` \| `pending` \| `approved`) → (`settled` \| `released`) | `refused` → anything; `approved` without a hold; `settled` without a charge record. | `authorization/flow/flow.json`, generated model, replay tests. |
| approval | `queued` → (`approved` \| `declined` \| `expired`) | Re-resolving a resolved request; resolving without a resolver identity. | `approval/flow/flow.json`, generated model, replay tests. |
| settlement | `ready` → `calling` → (`settled` \| `failed` \| `unknown`); `unknown` → (`settled` \| `failed`) | `unknown` → `failed` without a rail query (guessing); any transition that moves money twice under one key; `settled` → anything. | `settlement/flow/flow.json`, generated Quint model, replay tests. |
| ledger emission | `queued` → `emitting` → (`accepted` \| `retrying`); `retrying` → `emitting` | Dropping a queued emission; emitting without a terminal charge. | `ledger/flow/flow.json`, generated model, replay tests. |

The illegal transitions are the load-bearing half of this table. Two are
worth naming: **`expired` → `live`** on a mandate, because the most
tempting operator convenience ("just extend it") is exactly the one that
breaks the guarantee that authority has a bound; and **`unknown` →
`failed` without a rail query** on settlement, because that shortcut is how
a system double-charges while believing it retried a failure.

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

**Current level: 1 (inventory) for every flow.** Nothing is implemented
yet, so every flow above is listed and none is modeled. That is the honest
reading and it is recorded here rather than left implied — a maturity table
whose levels are aspirations reads identically to one whose levels are
measurements, which is how these ladders become misleading.

Targets and their justification:

- **Settlement targets level 5.** Exactly-once under concurrent retry
  against a partially-observable external system is the property most
  likely to be subtly wrong and least likely to be caught by a hand-written
  matrix.
- **Mandate, authorization and approval target level 4.** Their illegal
  transitions are the security-relevant ones, so a declarative contract
  that a test replays is worth the cost; a checked formal model is not
  obviously worth it on top.
- **Recurrence and emission target level 3.** Ordering matters but the
  state spaces are small.
- **Freeze targets level 2.** It has two states and one invariant.

## Production Shape

Three (Go) or four (UI) files per flow at the top of the feature folder,
plus one `generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    states.go                   # hand: state and event values
    transition.go               # hand: Transition + CheckInvariants
    generated/                  # codegen: model, matrix fixtures, traces
```

For this scenario that means `flow/` directories under `mandate`,
`authorization`, `approval`, `settlement` and `ledger`.

## Deferred / Unmodeled Flows

| Flow | Why deferred | Revisit trigger |
|---|---|---|
| Reconciliation matching | `TRS-P2-001`. No rail statement exists to match against until an automated rail has settled real charges. | First automated rail has produced a statement. |
| Browser checkout drive | `TRS-P2-003`. The flow is owned by `browser-automation-studio`; this scenario contributes only the scoped instrument and the mandate check. | Card rail is live and a counterparty without an API is worth automating. |
| Instrument revocation at the rail | Revocation is recorded locally at `TRS-P1-003`, but propagating it to each rail is adapter-specific and cannot be modeled generically yet. | Second automated rail exists, so the shared shape becomes visible. |
| Dispute and chargeback | No target yet. A disputed charge has a lifecycle this scenario does not model, and inventing it before a real dispute would be speculative. | A rail reports a dispute, or a P1/P2 target is added for it. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — which domain owns each flow
- [`DATA.md`](DATA.md) — the records these flows create
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — durable choices and their reasons
