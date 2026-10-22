# E1 outcome: make event-create rejection truthful and preserve uncertain recovery

## Authority, parent, and current evidence

This coherent implementation and journey-diagnostic slice is within the approved Planner I18/E1 calendar outcome. It does not change destination, product boundary, legacy ownership decision, source-data custody, or the original exit gates. Follow `path:scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md` and the full ledger `path:scenarios/personal-planner/docs/internal/goal/epochs/E1.md`. Parent run `5d07e651-eaa6-48ec-89b4-5dc359059193`, task `790cb264-1a63-47eb-89e6-3b9ee3e16c23`, workload `planner-I18/E1`. Worker owns source, focused tests and selected-run diagnostics; parent retains independent review and mechanical acceptance.

Act on directives `PP-ENGINEERING-DIAGNOSIS-IS-NOT-A-HUMAN-BLOCK-20261021` (seq65), `PP-CANONICAL-LIFECYCLE-LOG-READ-AVAILABLE-20261021` (seq66), `PP-SELECT-PLANNER-SHADOW-POSITIONALLY-NOT-WH-INSTANCE-20261021` (seq67), `PP-EXISTING-EVALUATE-RESPONSE-DIAGNOSTIC-20261021` (seq69), and `PP-PLAINTEXT-DIAGNOSTIC-AND-INDEPENDENT-DEFECT-BOUNDARY-20261021` (seq70). Seq69/70 correct seq68's route assumption: current fixture evaluate output is retained by BAS under the existing managed selected-run lease. Seq70 also clarifies that journey cause must remain unknown when evidence is insufficient, while a focused test may independently establish and authorize correction of a reproducible API defect.

## User outcome

The ordinary calendar create flow distinguishes a definitive owner/authentication rejection from an uncertain commit. A definitive rejection keeps the draft editable and gives an actionable message; it never claims a save or offers uncertain-save recovery. Transport loss/server failure remains uncertain and keeps the lookup-first recovery path, durable ID/revision readback, and same-key retry only after authoritative not-found. Preserve actor isolation and do not infer event actuals or durable state.

## Current fixture evidence

The current registered `bas/cases/calendar/uncertain-save-recovery.json` already wraps `window.fetch` for the event POST. Amend this current case, preserving the existing successful-response-lost simulation. For a non-OK response, retain only a fixture-local sanitized observation:

- If JSON, inspect `response.clone().json()` only for `status` and a strict allowlist of known error `code` values; otherwise use `other`.
- The actual `events_http.go` path also emits plaintext via `http.Error`. Capture the numeric status even when JSON parsing fails. It is permitted to read only a size-capped clone and classify the body by exact equality to the known static text `calendar event operation failed`; export only a category such as `calendar_operation_failed`, never the text. No other body content may be retained.
- Store only `{method:"POST",status,code}` or `{method:"POST",status,category}`. Never store raw body/message/headers/URL/token/title/payload. Return the original Response unchanged.

Immediately after submit and before any assertion likely to fail, add an evaluate that waits for wrapper-owned completion with a short bound and returns only the sanitized observation or `no-post-observed`. BAS stores evaluate `extracted_data.result` in the selected run timeline. Keep the route within the existing registered case; add no adapter, schema, or capture infrastructure.

Old selected runs `5e52fa19-f470-40d8-9b18-ba911adc3ae5`, `be632cb4-0771-4575-ab39-a973305ada50`, and `db0584b7-c4ff-4c98-9519-1ad8d7a907f8` do not establish a POST response cause. `ErrEventOwnerRequired` is a source possibility, not the diagnosis of those runs. Do not claim that owner-required caused a selected-run failure unless the new exact response or other current evidence proves it.

## Expected-behavior correction

First add/run the focused expected-behavior API regression for `d.ErrEventOwnerRequired`: assert the proper typed unauthenticated/owner response and prove no event is inserted. If it demonstrates the current catch-all HTTP 500, that test independently establishes a real handler defect even if the selected journey has another or unknown cause. Correct the typed mapping using Planner's shared typed HTTP error-envelope helper; add the narrow canonical unauthenticated code there only if needed. Add focused coverage for the shared envelope constant and handler mapping.

Preserve HTTP status and safe error code at the UI API boundary. In the create mutation, classify only the explicit unauthenticated/owner response as definitive: keep the draft editable, leave uncertain state off, and show an actionable owner-required message. Keep all other server/transport/decode errors uncertain; retain same-key lookup/retry behavior and never fabricate actor identity. Add UI API and component tests for both the owner rejection and uncertain transport failure.

After the correction, use the selected fixture's captured outcome to observe the changed response. Give each purposeful selected run a new deliberate idempotency key. Keep the successful synthetic lost-response branch and its durable recovery assertions intact. If the current managed route has no legitimate actor and therefore produces a definitive owner rejection, retain that exact result and counter proof; do not fabricate identity or primary writes. Adjust the current case only as needed to assert the definitive branch truthfully while preserving the committed-response-loss branch. Do not weaken the original successful-path assertions merely to get a green status.

## Selected-run proof

Use normal Workflow Health with positional Planner target `personal-planner@shadow` and current registered case:

```text
workflow-health validate start-selected personal-planner@shadow --path /home/matthalloran8/Vrooli/scenarios/personal-planner --case-paths /home/matthalloran8/Vrooli/scenarios/personal-planner/bas/cases/calendar/uncertain-save-recovery.json --idempotency-key <new deliberate key> --parent-run-id 5d07e651-eaa6-48ec-89b4-5dc359059193 --json
```

Use at most two purposeful invocations: first to observe the current rejected POST, and a second only after the fixture/source correction to observe changed behavior. Each uses a distinct new key. For each, verify actual target `personal-planner@shadow`, exact returned run ID/key, fresh managed lease and zero primary requests/writes; consume that exact run's terminal result once. A failed case can still provide useful status/code/category and counters, but report the actual verdict. Never execute against LIVE, replay an old key, clear/reset pools, or continue after a refused run.

## Focused validation

- Focused Go calendar handler tests prove owner-required maps to the explicit typed response without insertion and preserve conflicts/successful readback. If the shared HTTP helper changes, run its focused package tests too.
- Focused UI calendar API/component tests prove definitive owner rejection keeps the draft editable and out of uncertain state, while transport/server errors remain uncertain and key/status recovery is unchanged.
- Run only these focused package tests and scoped diff checks. No broad unchanged suite, provider call, or unisolated product-data write.
- Retain exact command outputs, response category/status, managed counters, all returned IDs/keys, hashes, line deltas, attribution limits, and remaining original gates.

## Scope and exact prewrite evidence

Allowed product paths are exactly:

- `scenarios/personal-planner/api/handlers/calendar/events_http.go`
- `scenarios/personal-planner/api/handlers/calendar/events_http_test.go`
- `scenarios/personal-planner/api/internal/httpx/errors.go`
- `scenarios/personal-planner/api/internal/httpx/errors_test.go`
- `scenarios/personal-planner/ui/src/api/calendar.ts`
- `scenarios/personal-planner/ui/src/api/calendar.test.ts`
- `scenarios/personal-planner/ui/src/components/CalendarMonth.tsx`
- `scenarios/personal-planner/ui/src/components/CalendarMonth.test.tsx`
- `scenarios/personal-planner/bas/cases/calendar/uncertain-save-recovery.json`

Exact captured prewrite bytes/manifest: `path:scenarios/personal-planner/docs/internal/goal/receipts/e1-event-save-outcome-20261021/prewrite-v3/manifest.json`; capture metadata is adjacent in `capture.json`. `CalendarMonth.test.tsx` already contained dirty shared-checkout content; preserve it. Stop before writes if any admitted current hash differs. This is scoped byte identity, not sole-authorship proof.

Estimate: +55 runtime lines, +145 test/fixture lines. Current E1 feature growth remains historically +1,930 runtime lines; +2,100 is an internal reforecast/step-back trigger, not a permission boundary. Do not pad or reset attribution. The estimate is guidance; disclose measured actuals and reforecast transparently.

## Non-goals and handoff

No Proto/Connect/CLI change, identity-provider or authorization-policy weakening, schema/migration, provider access, legacy-row assignment, product DB write outside managed test storage, pool clear/reset, stale-case replay, adapter/lifecycle repair, dependency install, or original-gate change. Do not claim root cause beyond evidence, actor parity, full E1/Q1 acceptance, or epoch acceptance. Append one truthful E1 slice-log line acknowledging seq69/70; run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <exact worker run ID> --effort effort:personal-planner-I18 --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`; return exact final, source evidence, tests and selected-run results.
