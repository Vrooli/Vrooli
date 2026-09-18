# Flows — Nutrition Planner

This document is the canonical workflow and state-transition map for the scenario. Use it
when behavior depends on ordered states, retries, cancellation, stale completion,
background jobs, polling, or mutually exclusive UI modes.

**Current status: no formal workflow model exists yet.** The temporal layers described in
[`Maturity Ladder`](#maturity-ladder) are at levels 1–2: the journeys and state machines
below are documented, but no `*.flow.json` contract, generated Quint model, or replay test
has been authored. This section names where that work will attach.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a workflow model.

## Flow Inventory

`health` is a stateless reporting domain and ships no workflows. The rows below are the
seven product journeys (specification §4) and the stateful system flows that support them.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| A. A first useful plan | workspace-and-profile, planning-engine | A new user completes or skips setup. | Active rules plus a draft week, or a clearly labeled empty collection. | Resumable setup draft; generate/apply separation; stale-input checks. | L1 |
| B. Capture an existing routine | recipe-library, dietary-rules | The user pastes a routine or enters meal names. | A review queue and, after review, accepted recurring anchor templates. | Review-before-write; unresolved quantities preserved; no invented facts. | L1 |
| C. A tired evening | planning-engine, intake-and-feedback | The user opens Today and asks for a lower-effort alternative. | A scoped one-meal override with recalculated groceries and batch dependencies. | Quote/apply; undo only when revisions still match; locks preserved. | L1 |
| D. Shopping and changes | shopping-plan, inventory-and-batches | The user reviews a week and derives groceries. | A checked shopping plan; purchases remain a separate explicit action. | Check state preserves unchanged lines; overrides retained; idempotent purchase. | L1 |
| E. Cooking and logging | recipe-method-graph, inventory-and-batches, intake-and-feedback | The user finishes cooking or eats. | A prepared batch and/or a consumption event with correction linkage. | Cook and eat are distinct; undo/correction reverses stock exactly once. | L1 |
| F. A new week | planning-engine | The user requests the next date range (R2: a scheduled job). | A draft for the range that preserves anchors, locks, and accepted preferences. | Generate/apply separation; a scheduled run never replaces an accepted plan. | L1 |
| G. Moving data | transfer-and-documents | The user exports, imports, restores, or prints. | A portable envelope or a rendered document; staged import applies atomically. | Staged → applied; recoverable restore checkpoint; stale preview rebuild. | L1 |
| Generate plan draft | planning-engine | User request or scheduled run. | A draft with occurrences, evaluations, costs, reasons, and unresolved slots. | Seed-reproducible; bounded search; timeout is not infeasibility. | L1 |
| Apply plan | planning-engine, application-services | User confirms a preview. | An accepted plan revision or a stale/conflict error. | Expected-revision checks; all-or-nothing; preserves locks. | L1 |
| Record / correct intake | intake-and-feedback, inventory-and-batches | User logs or corrects a consumption event. | An idempotent event and refreshed actual totals; batch adjusted once. | Idempotency key; document the same payload/conflict rule. | L1 |
| Confirm preparation | inventory-and-batches | User confirms a cooking session with actual yield. | A prepared batch; raw ingredients consumed once; reservation released. | All-or-nothing; leftover portions reference the batch. | L1 |
| Record purchase | inventory-and-batches, shopping-plan | User confirms shopping lines as bought. | An idempotent purchase event and one inventory application. | Durable source identity prevents double-apply. | L1 |
| Stage / apply import | transfer-and-documents | User selects a file or pasted JSON. | A validation report, then an atomic apply with counts. | Parse/preview never mutates; stale proposal rebuilds. | L1 |
| Run background job | provider-and-job-adapters | Schedule, manual trigger, or an adapter call. | A terminal job record and, if extraction succeeded, an unapplied proposal. | Bounded retries; cancellation; provider failure never blocks manual paths. | L1 |

## Flow Details

### Journey A — A first useful plan

Owner: `workspace-and-profile` with `planning-engine`. Trigger: a new or edited profile.

1. New user enters the brief four-step setup (**Your food**, **Your kitchen**,
   **Your rhythm**, **Ready**) or chooses Explore first. Setup is held as a resumable draft
   separate from active rules.
2. They choose a diet approach and exclusions; they select appliances (zero is valid).
3. Three questions establish initial cost, effort, and variety settings.
4. The final step summarizes the rules and reports how many meals can currently be
   evaluated as fitting — counts for fits, needs review, and excluded, computed by the same
   eligibility service the planner uses.
5. Applying the profile is atomic and previews conflicts with existing plans. With
   sufficient recipes, a draft week is generated; without them, the user can add a meal or
   explore clearly labeled sample content.
6. The user sees one recommended meal with a calculation-backed reason and can start, swap,
   edit setup, or inspect the week.
7. If the profile is applied against an existing plan, locked future conflicts are surfaced
   and historical intake is untouched.

Failure modes: no eligible meals produces a rule-conflict summary and offers Add a meal or
Adjust setup — never an automatic relaxation. A stale profile revision invalidates a
generated draft rather than applying it.

### Journey B — Capture an existing routine

Owner: `recipe-library` with `dietary-rules`. Trigger: pasted text or a sequence of names.

1. The user pastes a rough routine or adds meal names.
2. The application creates a review queue with detected meals, recurring slots, foods, and
   supplements.
3. It asks only material questions: quantity/basis, exact product where fortification
   matters, and whether a repeated item is a separate consumption or the same item described
   twice.
4. The source text is preserved. The system must not invent a recipe, brand, serving
   weight, dose, or daily target.
5. Reviewed recurring items become baseline anchor templates; flexible slots stay variable
   in later weeks.

Failure modes: an ambiguous extraction stays unresolved with its original text. A missing
quantity is unknown, not zero.

### Journey C — A tired evening

Owner: `planning-engine` with `intake-and-feedback`. Trigger: Open Today → next meal.

1. The user sees the next meal and chooses "Not a cooking kind of night?".
2. Eligible alternatives are ranked for minimal active effort; blocked alternatives are
   shown separately with reasons and are not selectable as compliant.
3. The user compares time, cost, and material target effects.
4. Selecting one opens a scoped preview; grocery requirements and future dependent batch
   uses are recalculated.
5. Apply affects only the selected future occurrence and its planned servings; other
   occurrences of the same recipe are unchanged unless explicitly broadened.
6. Giving a reason is optional. This can be a one-meal override without changing permanent
   priorities.

Failure modes: undo restores the prior occurrence and derived state only if the affected
revisions still match; otherwise it offers a new preview. Consumed occurrences are never
changed.

### Journey D — Shopping and changes

Owner: `shopping-plan` with `inventory-and-batches`. Trigger: review a week.

1. The user inspects derived groceries: total need, stock considered, amount missing,
   estimated package count, price source/date, and unknowns — each separately.
2. They check off items as shopping progresses. A checkmark is checklist state only.
3. If a meal changes, checked states are preserved for materially unchanged line
   requirements; increased or changed requirements are marked for review, and a compact
   summary explains the difference.
4. Inventory changes only through an explicit confirm-purchases action that previews
   quantities and prices and creates idempotent purchase/stock events. Partial fulfillment
   is supported; dismissing it keeps approximate stock.

Failure modes: a missing package size or price leaves the row visibly unresolved. Replanning
cannot delete a manual item, and an overridden quantity carries a mismatch badge if plan need
changes.

### Journey E — Cooking and logging

Owner: `recipe-method-graph`, `inventory-and-batches`, `intake-and-feedback`. Trigger: open
a recipe.

1. The user chooses map, read, or focused mode; all three use the same pinned revision.
2. They adjust servings if needed. Scaling changes displayed quantities and totals but not
   the stored recipe or the planned occurrence until explicitly applied.
3. They follow steps or use a timer. Previewing or marking a step complete does not mark
   food eaten; a Next button navigates by default.
4. "Finished cooking" confirms a preparation with actual yield: raw ingredients are
   consumed once and a prepared batch is created. "I ate this" is a separate action.
5. Confirming a default consumed portion is one action, optionally adjusted, or left
   unrecorded. A portion consumed from a batch reduces batch stock, not the raw ingredients
   again.
6. Undo creates a correction and reverses linked inventory effects transactionally.

Failure modes: undo cooking is valid only if no downstream batch portion is unreversable;
otherwise a correction workflow explains the affected records. A timer ending never implies
food is safely cooked.

### Journey F — A new week

Owner: `planning-engine`. Trigger: request the next date range (R2: scheduled draft).

1. The system carries forward accepted routine anchors, current preferences, credible
   stock, and actual feedback.
2. It generates a draft for the next range, preserves locked commitments, and explains
   meaningful changes.
3. The user previews apply scope (default: unlocked future occurrences) and may apply,
   keep the current plan, or try another draft.
4. R2's scheduled job creates a reviewable draft and never silently replaces an accepted
   plan.

Failure modes: stale inputs invalidate the preview and require regeneration. Regeneration
may vary the deterministic seed but must preserve constraints and explain what changed.

### Journey G — Moving data

Owner: `transfer-and-documents`. Trigger: export, import, restore, or print.

1. Export produces a complete backup, meal collection, weekly PDF, recipe PDF, or grocery
   CSV from an immutable input snapshot.
2. Import detects format and version before parsing into the live model, enforces configured
   limits, and shows counts, duplicates, conflicts, missing dependencies, and unsupported
   fields.
3. The default is Add meals only. Full restore offers Restore everything with a precise
   impact summary and a recoverable pre-restore checkpoint.
4. No live data mutates during parse or preview. Apply writes all selected valid changes in
   one transaction against an expected revision.
5. A recipe has its own PDF action. A text-only export that omits source bytes says so.

Failure modes: a multi-record import with invalid entries rejects the full import by
default; an explicit "Import valid records only" shows exactly what is omitted and preserves
rejected content. A stale preview must be rebuilt before apply. A failed restore leaves the
previous workspace intact.

## State Machines

The following lifecycle models are specified but not yet formally encoded. The
`Enforcement` column names the planned mechanism.

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| recipe-library / authoring | `draft → ready → archived` | Recommending a `draft`; resolving an `archived` revision into new plans; any transition out of `archived` except explicit unarchive by the owner. | Authoring status is a field on stable identity, separate from the immutable content revision. |
| dietary-rules / eligibility | `eligible → { ineligible, needs_information }` and back when rules or evidence change | Treating `needs_information` as `eligible`; overriding `ineligible` with a preference weight. | Eligibility is recomputed per profile revision and evaluator version; required exclusions are hard constraints. |
| planning-engine / planned occurrence | `planned → locked → cooked → consumed`, with `planned → skipped`, `locked → planned` (unlock), and any non-terminal → `unrecorded` after its local date passes | Editing a consumed or skipped occurrence; planning a locked occurrence away without an explicit unlock; deriving `consumed` from the plan. | Occurrence status checks at apply time; locks preserved by replan; consumption requires an event. |
| inventory-and-batches / prepared batch | `planned → created → depleting → { consumed, wasted }` | Consuming or wasting more than the remaining amount; creating a batch without a preparation confirmation; reverting a batch after downstream portions. | Batch quantity arithmetic plus idempotent inventory events; undo only when atomically reversible. |
| provider-and-job-adapters / job | `queued → running → { waiting_for_review, succeeded, failed, canceled }`; `waiting_for_review → { succeeded, failed, canceled }` | Applying a proposal from a `failed` or `canceled` job; marking a job `succeeded` when its proposal is unapplied; retrying a non-retryable schema/permission failure indefinitely. | Job record with attempt count, dedup key, budget, and separate application status. |
| transfer-and-documents / import | `staged → { applied, canceled }`, with `staged → staged` only via revalidation | Writing live data while `staged`; applying a stale stage; applying after `canceled`. | Staged proposal id/hash plus expected-revision check inside the apply transaction. |
| inventory-and-batches / purchase | `proposed → recorded` (idempotent); `recorded → corrected` | Double-applying the same purchase; marking a shopping check as a purchase. | `(source transaction identity, normalized line)` dedup key; idempotency key on the operation. |
| inventory-and-batches / preparation | `reserved → confirmed → released` | Double-consumption of raw ingredients; confirming without an explicit yield basis. | All-or-nothing confirmation; one inventory application per operation key. |
| intake-and-feedback / intake | `unrecorded → recorded → corrected` | Editing a recorded event in place; deleting it in place; recording the same logical intake twice under one key. | Idempotency key with key/payload conflict detection; corrections are linked new events. |

### Why The Unknown And Stale Rules Matter

Two transitions are easy to get wrong and expensive when wrong:

- **Unknown is not a pass.** A missing allergen tag or price produces
  `needs_information` or an unresolved shopping row. Treating unknown as false would let an
  ineligible meal through under a green badge.
- **Stale completion cannot apply.** A draft or import preview captured against an input
  revision must be revalidated before apply. A plan update with a stale expected revision
  returns a conflict and preserves the newer revision plus the stale client's proposals.

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers to add a formal
document that no test reads.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

The flows above are at level 1. The first flows to promote to level 2–3 are planned
occurrence, prepared batch, job, and import, because they carry the riskiest idempotency and
stale-completion behavior.

## Production Shape

Three (Go) or four (UI) files per flow at the top of the feature folder, plus one
`generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with conventional file names.
API domains that own durable lifecycle state use:

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

The `flow/` directory is the unit; the contract declares no output paths or module names.
The workflow owns state/status values, events, `Transition`, and `CheckInvariants`, and stays
pure or nearly pure. Effects live outside it behind seams: repositories, BlobStore, clocks,
timers, HTTP clients, and UI API modules.

The `*.flow.json` contract is the source of truth. Level 5 generated artifacts are
checked-in source artifacts, refreshed and checked by the `flow-verifier` scenario CLI; the
scenario lifecycle runs `make temporal-models` (which calls `flow-verifier verify check`)
before the normal test suite. A Quint file alone is not accepted: the model must typecheck,
test, verify named invariants, emit deterministic artifacts, and those artifacts must replay
against the production transition functions.

To scaffold a new flow:

```bash
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
```

To add or rename a state/event: edit the owning `*.flow.json`, regenerate with
`flow-verifier verify run --flow <flow-id>`, update only payload-specific wrapper branches,
update the UI replay fixtures, then run `make temporal-models` and the scenario tests.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Scheduled draft generation | A scheduled run must create one draft per intended local period and never replace an accepted plan or mark food eaten. | Model the recurrence dedup key and the `draft_created` outcome at L2 when OT-P1-005 is implemented. |
| Provider research job | Cancellation can still receive a late external response; a late proposal must not apply over newer edits. | Model the job and proposal application split at L2/L3 alongside the first adapter. |
| Receipt ingestion | Duplicate or out-of-order receipts must not double-apply purchases or imply current stock. | Model source-identity dedup at L2 before any receipt adapter ships. |
| Preference learning | Learned preferences must stay inspectable, resettable, and weaker than explicit dislikes. | Model the explicit/inferred separation and cooldown at L2 when OT-P1-006 adds inference. |
| Household collaboration | Roles, profile-to-person mapping, and shared shopping introduce per-person targets. | Deferred until OT-P2-003; do not model prematurely. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — state ownership and invalidation
- [`EXPERIENCE.md`](EXPERIENCE.md) — the surfaces these flows run on
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../reference/product-specification.md`](../reference/product-specification.md) — §4, §8–§11, §14–§18, §20
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md#temporal-workflow-tests) — matrix and trace testing
