# E1 outcome: distinguish rejected event creation from uncertain commit

## Authority and parent

This is an approved I18/E1 implementation slice under the existing calendar journey. It does not change product boundaries, legacy ownership decisions, source custody, or original exit gates. Follow `path:scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`, the full E1 ledger at `path:scenarios/personal-planner/docs/internal/goal/epochs/E1.md`, and the exact resume set named here. Parent run `5d07e651-eaa6-48ec-89b4-5dc359059193`, task `790cb264-1a63-47eb-89e6-3b9ee3e16c23`, workload `planner-I18/E1`. This worker owns implementation, focused checks, and evidence handoff; independent review and mechanical acceptance remain with the parent.

Applicable supervisor directives: `PP-ENGINEERING-DIAGNOSIS-IS-NOT-A-HUMAN-BLOCK-20261021` (seq65), `PP-CANONICAL-LIFECYCLE-LOG-READ-AVAILABLE-20261021` (seq66), `PP-SELECT-PLANNER-SHADOW-POSITIONALLY-NOT-WH-INSTANCE-20261021` (seq67), and `PP-EXISTING-EVALUATE-RESPONSE-DIAGNOSTIC-20261021` (seq69). Seq69 supersedes the seq68 assumption that no supported paired evidence route exists: use the current registered fixture's existing evaluate provider while retaining Workflow Health managed lease/counters.

## User outcome

The ordinary calendar create flow must distinguish an observed definitive rejection from a request whose commit outcome is uncertain. Rejections must preserve the editable draft and avoid claiming a commit. Transport loss after a successful commit must retain the existing idempotency status lookup, durable ID/revision recovery, and same-key retry only after authoritative not-found. Never infer an actual or durable event from estimates or the UI alone.

## First action: add sanitized evidence to the current fixture

The current registered `bas/cases/calendar/uncertain-save-recovery.json` wraps `window.fetch` for one POST to `/api/v1/calendar/events`. Amend that current case only as needed by seq69, preserving its current synthetic lost-response behavior on successful POST.

On a non-OK POST response, use `response.clone().json()` only to derive the HTTP status and a strictly allowlisted known `code` enum value. Use `other` when JSON parsing fails, the code is absent, or it is outside the allowlist. Store only `{method:"POST", status, code}` in a fixture-local window slot; return the original Response unchanged. Never store raw body, message, headers, URL, token, title, or other payload. Immediately after the submit node and before any UI assertion that can fail, add an evaluate that waits with a short bound for wrapper-owned completion and returns only that sanitized object or `no-post-observed`. BAS `extracted_data.result` is retained in the validation timeline. Do not add a network/HAR adapter, new scenario, or new capture infrastructure.

Preserve the source and gate limitation from earlier evidence: the retained old runs had no POST response and do not establish the cause. `ErrEventOwnerRequired` is one possible API branch, not the established explanation. Do not patch an owner-required hypothesis. Use the newly captured status/code and other exact supported evidence to determine whether a branch is established.

## One purposeful isolated selected run

After the case change, use the normal Workflow Health service with positional Planner target `personal-planner@shadow` and this current registered case:

```text
workflow-health validate start-selected personal-planner@shadow --path /home/matthalloran8/Vrooli/scenarios/personal-planner --case-paths /home/matthalloran8/Vrooli/scenarios/personal-planner/bas/cases/calendar/uncertain-save-recovery.json --idempotency-key <new deliberate key> --parent-run-id 5d07e651-eaa6-48ec-89b4-5dc359059193 --json
```

Use a unique new key; do not replay any previous/stale case key. Verify the returned actual target is exactly `personal-planner@shadow`, capture its exact run ID and key, require a fresh managed lease and zero primary requests/writes, then consume the exact run's terminal result once. Never execute against Planner LIVE, clear/reset pools, or proceed on a refused run. A failed selected case can still provide its sanitized POST observation and managed counters; report its actual verdict without laundering it into a pass.

## Conditional product correction and focused gate

If and only if the captured response or other exact current evidence identifies the definitive rejection branch, finish the approved original outcome in the same coherent assignment: map only that typed branch to a definitive API response; preserve status at the UI API boundary; keep the draft editable and show an actionable rejection without entering uncertain state. Preserve generic server/transport failures as uncertain. Keep the original idempotency key for an unchanged draft and do not weaken identity checks. If the evidence does not identify the branch, do not edit product behavior; return the sanitized observation and exact cause boundary for the parent to choose the next independently measured E1 slice.

Where the evidence supports source changes, add/retain focused API tests for the typed rejection and absence of insertion, UI API status preservation, and component behavior for definitive rejection versus uncertain transport failure. Preserve the existing success/lost-response recovery test: successful lookup recovers the durable identity; confirmed absence permits only same-key retry. Run only the focused calendar handler and UI calendar API/component tests, plus scoped diff checks. Do not run a broad unchanged suite or create product data outside managed test isolation.

## Scope, prewrite bytes, and estimate

Only these seven product paths are admitted:

- `scenarios/personal-planner/api/handlers/calendar/events_http.go`
- `scenarios/personal-planner/api/handlers/calendar/events_http_test.go`
- `scenarios/personal-planner/ui/src/api/calendar.ts`
- `scenarios/personal-planner/ui/src/api/calendar.test.ts`
- `scenarios/personal-planner/ui/src/components/CalendarMonth.tsx`
- `scenarios/personal-planner/ui/src/components/CalendarMonth.test.tsx`
- `scenarios/personal-planner/bas/cases/calendar/uncertain-save-recovery.json`

Exact preserved prewrite bytes and manifest: `path:scenarios/personal-planner/docs/internal/goal/receipts/e1-event-save-outcome-20261021/prewrite-v2/manifest.json`; capture metadata is adjacent in `capture.json`. The UI component test already had unrelated dirty content at admission; preserve it. Stop before writes if any admitted current hash differs from the manifest. This is a scoped byte baseline, not sole-authorship proof.

Estimate: +35 runtime lines and +100 test/fixture lines. E1 forecast remains cumulative +1,930 runtime lines, with the existing +2,100 line value an internal reforecast/step-back trigger rather than a new authorization boundary. Do not pad or reset accounting. Report exact deltas, identities, fixture output, test output, managed counters, and any uncertainty.

## Non-goals and handoff

No Workflow Health/BAS adapter work, Proto/Connect/CLI change, identity-provider/configuration change, schema/migration, provider access, ownerless-row assignment, product DB write outside managed test storage, pool clear/reset, stale-case replay, broad lifecycle repair, dependency install, or change to original E1/Q1 criteria. Preserve earlier failed evidence. Do not claim cross-surface actor parity, full E1/Q1 acceptance, or epoch acceptance. Append one truthful E1 slice-log entry acknowledging seq69, run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your exact run ID> --effort effort:personal-planner-I18 --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`, and hand off the exact selected final and source/validation evidence.
