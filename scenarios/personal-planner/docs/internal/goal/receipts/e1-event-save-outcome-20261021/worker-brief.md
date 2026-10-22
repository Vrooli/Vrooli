# E1 outcome: tell a verified-owner rejection from an uncertain create

## Authority and parent

This is a bounded implementation outcome inside approved Planner I18/E1 and the existing Q1 calendar journey. It does not change destination, product boundary, legacy owner mapping, source-data custody or exit gates. Follow `path:scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md` and `path:scenarios/personal-planner/docs/internal/goal/epochs/E1.md`; parent run `5d07e651-eaa6-48ec-89b4-5dc359059193`, task `790cb264-1a63-47eb-89e6-3b9ee3e16c23`, workload `planner-I18/E1`. Read the six-file resume set and this brief. The worker owns implementation and focused checks; parent retains independent review and epoch acceptance.

## New diagnostic precedence (supervisor feedback seq65)

Before any source write, establish the actual cause from the exact retained owner artifact and supported current-instance logs. Exact `workflow-health validate get` for `5e52fa19-f470-40d8-9b18-ba911adc3ae5` confirms target `personal-planner@shadow`, parent/key, 20 test-pool requests and zero primary requests/writes. The retained `latest.json` is 891 bytes and `timeline.json` is 4,280,422 bytes with 14 step entries; neither records HTTP network events, POST status or response body. The first failed assertion says the uniquely named synthetic event lacks a durable ID; later editor-open and stale-save steps fail. `vrooli scenario logs personal-planner --runtime --tail 80` and the corresponding `--instance shadow` command both return no-such-directory. Seq66 clarifies runtime and lifecycle are distinct log kinds and rejects a control-plane routing repair. Current lifecycle-log reads by both `personal-planner --instance shadow --tail 50` and canonical `personal-planner@shadow --tail 1` returned “no lifecycle log found”; this current result conflicts with the supervisor-reported earlier read, so preserve both without generalizing. No current log result exposes the POST status/body. Exact evidence note: `seq65-worker-directive.md`. Therefore `ErrEventOwnerRequired` remains a hypothesis; the planned mapping/UI behavior may be implemented only if the actual failed create is shown to return that typed owner error. Otherwise diagnose the actual API/fixture/adapter boundary and do not patch the hypothesized handler. A purposeful new diagnostic run is permitted only under seq65's conditions: changed/current capture, proven Planner shadow target, fresh managed lease, exact returned run ID/key and terminal result, response/body and containment. Never replay the stale case or repair shared control-plane infrastructure.

## Correct target syntax (supervisor feedback seq67)

The diagnostic target is positional `personal-planner@shadow` through the reachable normal Workflow Health service; `--instance shadow` selects the Workflow Health instance, not the Planner target. No `workflow-health@shadow` repair or restart is needed. If one purposeful run remains necessary after retained artifacts and current logs, use a current registered diagnostic case (not the stale-revision case), changed/current capture and a new intentional idempotency key:

```text
workflow-health validate start-selected personal-planner@shadow --path /home/matthalloran8/Vrooli/scenarios/personal-planner --case-paths <current registered diagnostic case> --idempotency-key <new intentional diagnostic key> --parent-run-id 5d07e651-eaa6-48ec-89b4-5dc359059193 --json
```

Verify actual returned target `personal-planner@shadow`, exact run ID/key, fresh managed lease and primary containment. Wait once for that exact run; retain POST status/body. Never execute against live. Prior live-target refusals and the shadow-service build interlock were target-axis errors, not a route outage. Do not launch an unchanged case, repeat the stale case, or infer success from a request or older lifecycle result.

## Evidence boundary (supervisor feedback seq68)

Current read-only inspection found no supported route pairing selected-run managed LeaseStats with POST response/body capture: the durable `start-selected` adapter does not set network/HAR capture options, while direct BAS capture lacks provider-owned lease counters. Workflow Health paths are outside this Planner task's admitted six-file allowlist. Do not launch an incomplete-evidence diagnostic or widen into Workflow Health. Return the exact route checks and preserve the unknown create cause; do not infer a Planner owner-error patch.

## User outcome

When REST rejects a new calendar event because a verified calendar owner is absent, report a definitive, actionable rejection and keep the draft usable. Do not label this as an uncertain save. Keep transport failures and server errors uncertain; preserve the idempotency key and the existing lookup-first recovery path, including same-key retry only after the status route confirms the event is absent. A successful status lookup still recovers the durable event ID/revision. Preserve verified-principal checks and keep ownerless legacy rows untouched.

## Evidence and measured boundary

Current source: `api/internal/calendar/events.go` defines `ErrEventOwnerRequired`; `api/internal/calendar/events_sqlite.go` returns it before event insertion when no owner subject exists. `api/handlers/calendar/events_http.go` currently maps invalid input, not-found and conflicts, then turns owner-required into generic HTTP 500. `ui/src/api/calendar.ts` discards HTTP status in a generic `Error`. `ui/src/components/CalendarMonth.tsx` marks every create error uncertain; its status lookup/recovery already preserves the key and checks before retry. The existing UI test covers uncertain transport failure and lookup-first retry.

Measured six-file source/test boundary: 913 lines. Exact pre-admission hashes:
- `api/handlers/calendar/events_http.go` 127 lines `11e2c71439e43c7a2496e9cfd1c5884a8159e2d09952ea923c666fb31955f81a`
- `api/handlers/calendar/events_http_test.go` 119 lines `b7fe1658ae387d0901d1912f0ae3e64660445719f3d78fee35915a579e75980a`
- `ui/src/api/calendar.ts` 120 lines `16a08942829a698db47e3f89cafbd58a1797d1da6a421b53edd4696e87bdc6dd`
- `ui/src/api/calendar.test.ts` 142 lines `2bb6ad4331c68287db0238d1687788c41fe5c0fee4d6a709e73be6449f9e3464`
- `ui/src/components/CalendarMonth.tsx` 214 lines `ab92d58eb2736b149bad3ea45dd182141f3436954c3438a004190be5ca571cb9`
- `ui/src/components/CalendarMonth.test.tsx` 191 lines `3cb94b2ebf67e917ba17ae49068a6b0b1149a4e50b20424640b52d8e75fdf668`

These hashes define observed pre-admission bytes, not sole authorship. Before any write, follow the worker card: use the available owner snapshot route; if it is unavailable and the card permits local fallback, snapshot exactly these six files and record the manifest identity and time in the E1 slice log. Stop on any changed baseline or attribution conflict; preserve concurrent work.

## Scope and estimate

Allowed product paths are exactly the six paths above. Keep this focused on one user-facing save outcome across REST and UI. Map only the typed `ErrEventOwnerRequired` to the existing API's appropriate definitive authentication/owner response (expected 401); do not reclassify generic 4xx, 5xx, timeout, decode or transport errors as proof of no commit. Preserve error status in the UI API boundary and classify only the known owner-required response as definitive. Keep the draft, avoid the uncertain-state lock for this definitive rejection, and retain the same idempotency key if the unchanged draft is retried. Any changed payload may use a new key. Do not change the existing uncertain flow: check status first, recover by durable ID/revision, retry the unchanged payload with its original key only after authoritative not-found.

Estimate: +25 runtime lines and +60 test lines. This is an estimate for reforecasting, not a human authorization cap. Current E1 cumulative growth is +1,930 runtime/+2,698 test; the 2,100 runtime value is the existing internal step-back/reforecast trigger, not an access boundary. Do not pad changes or reset attribution. If source evidence requires a materially wider behavior or runtime growth approaches that trigger, stop and return the exact measurement and proposed coherent adjustment to the parent before widening scope.

## Focused gate

- Add API tests proving the typed owner-required rejection has the explicit status and does not create an event; preserve existing conflict and successful durable readback behavior.
- Add UI API tests for status-preserving errors and component tests proving owner-required rejection leaves an editable draft and is not shown as uncertain.
- Preserve and rerun focused uncertain-save behavior: transport/server failure remains uncertain; no fresh key or duplicate submit; lookup success returns durable identity; confirmed absent permits only same-key retry.
- Run only focused API calendar handler tests and focused UI calendar API/component tests. No broad unchanged Test Genie phases, live provider, ordinary product-data writes, BAS selected-case run or stale-case replay. Synthetic/focused tests must use isolated fixtures.
- Report exact command outputs, source identities/diff, line deltas, attribution limits and remaining full E1/Q1 gates. Do not claim actor parity, full journey acceptance or E1 acceptance.

## Non-goals and retained safety

No Connect/CLI contract change, identity-provider configuration, authorization policy change, schema/migration, provider access, legacy-row assignment, product database mutation, pool reset/clear, stale-case replay, workflow-health shadow repair, broad lifecycle operation, dependency installation or change to original E1 acceptance gates. Preserve the separate selected-case evidence and its exact run IDs. Read no credentials or secrets.
