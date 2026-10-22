# E1 current-source event-save and conflict journey correction

Worker card: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.
Authoritative epoch: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`.
Orchestrator parent run: `5d07e651-eaa6-48ec-89b4-5dc359059193`; task: `790cb264-1a63-47eb-89e6-3b9ee3e16c23`.
Applicable feedback: `PP-EXISTING-SELECTED-CASE-ISOLATION-ROUTE-20261021` (seq61), `PP-MEASURE-REMAINING-JOURNEY-NOT-STALE-SLICE-STOP-20261021` (seq62), `PP-CORRECT-ESTIMATE-AUTHORITY-NO-NEW-SCOPE-VOTE-20261021` (seq63).

## Outcome

Repair or accurately re-specify the ordinary Planner Month event-save journey exposed by current-source Workflow Health selected run `75bb202a-a62c-49c5-83ea-50dfa5c861de`. The user flow must create one synthetic event with a durable ID, reopen that same event, permit a competing supported API revision, then show the stale UI save conflict while retaining the newer durable winner. Preserve the existing uncertain-save recovery contract: never assume success, keep the draft/key when status is unknown, and resolve by idempotency-key read before retrying.

The selected case failed because the UI reported “The save outcome could not be confirmed. Check its idempotency key before retrying.” The event ID was therefore absent and subsequent reopen/conflict/readback steps were not reached. Treat this as a real journey defect to resolve at its owning product or test boundary. First distinguish a production behavior gap from a stale/incorrect fixture assertion using current source and retained step evidence. Do not weaken the accepted journey or turn an uncertain result into a success claim.

## Authority and evidence

- Approved destination and acceptance: `scenarios/personal-planner/docs/internal/goal/GOAL.md`, `CONTRACT.md`, `ACCEPTANCE.md`, E1/Q1.
- Current evidence: `receipts/e1-current-selected-lease-20261021/receipt.md` and Workflow Health run/execution above.
- Source hashes at the pre-dispatch GCT anchor `gcta-d872358fccab` (head `34515f0f501d511f65aa06c8e323620bf6d05891`):
  - `ui/src/components/CalendarMonth.tsx`: `ab92d58eb2736b149bad3ea45dd182141f3436954c3438a004190be5ca571cb9`
  - `ui/src/api/calendar.ts`: `16a08942829a698db47e3f89cafbd58a1797d1da6a421b53edd4696e87bdc6dd`
  - `ui/src/components/CalendarMonth.test.tsx`: re-read and record exact current hash before writing.
  - `bas/cases/calendar/stale-revision-conflict.json`: `645f5048c614766cef48aef0493d1cad9853939bcf3984e52fcb4f2b47c4eb92`
  - `bas/cases/calendar/uncertain-save-recovery.json`: re-read and record exact current hash before writing.

## Scope and measure

- Allowed paths: only the two listed UI source files, their relevant component/API tests, and the two existing BAS case JSON files. Add no framework, counter, identity, or data-migration infrastructure.
- Expected net growth: about +80 runtime / +120 test lines if product code is required; a fixture-only correction should be near zero runtime. This is a planning estimate, not a new authorization cap. Preserve the +1,930 measured cumulative E1 runtime; if source inspection shows a materially larger correction or remaining destination estimate, stop before writes and report exact paths, estimates and options for the parent to reforecast. Do not treat the historical +2,031 brief stop as a global cap or reset cumulative attribution.
- Preserve current actor ownership, event IDs/revisions, idempotency semantics, conflict behavior, private-data boundaries, and all unrelated authored work. Never write live/product records, manufacture an actor, clear a test pool or change legacy rows.
- A retained foreign Workflow Health lease currently prevents a fresh mutating selected BAS run. Do not retry that route or clean up the pool. Use focused component/API tests with isolated fixtures where justified; report that current integrated validation remains blocked by the owner lease.
- Deletion list: none. This is a user-journey correction; do not delete established API behavior or BAS assertions merely to turn the suite green.

## Exit gate

- Identify and explain the exact cause of the current uncertain-create result using current code and retained execution evidence; do not rely on unavailable shadow lifecycle logs.
- Make the smallest justified product or test-boundary correction, if one is evidenced. Desired tests assert expected behavior even when current implementation is wrong.
- Run only focused affected UI/API tests; retain command, exit and output. Do not run broad Test Genie phases, and do not retry mutating BAS until the Workflow Health owner reconciles the foreign lease.
- Preserve current stale-save conflict protection and prove the independent durable readback remains at the newer API revision when the current selected-case route becomes available.
- Report exact changed hashes, net runtime/test growth, focused results, unresolved integrated gate and why any planned correction was not made.

## Non-goals

No other product area, legacy mapping, cross-surface identity fabrication, Proto/generation work, dependency work, provider writeback, broad redesign, baseline reset, E1/Q1 acceptance, or decision to waive original gates.
