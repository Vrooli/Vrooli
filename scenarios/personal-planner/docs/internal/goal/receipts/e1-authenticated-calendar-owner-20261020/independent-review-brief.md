# Independent review brief — authenticated ownership for new native calendar events

Reviewer task only; do not implement or amend files. Parent/orchestrator exact run: `5d07e651-eaa6-48ec-89b4-5dc359059193`; parent task: `790cb264-1a63-47eb-89e6-3b9ee3e16c23`.

## Scope and authority

Review the bounded E1 slice in `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`, section “Bounded continuation — authenticated ownership for new native calendar events.” It changes only these five admitted product paths:

- `scenarios/personal-planner/api/internal/calendar/events.go`
- `scenarios/personal-planner/api/internal/calendar/events_sqlite.go`
- `scenarios/personal-planner/api/internal/calendar/schema.sql`
- `scenarios/personal-planner/api/internal/calendar/events_test.go`
- `scenarios/personal-planner/api/handlers/calendar/connect_events_test.go`

Read the approved Planner workspace/security contract and the qualification receipt `receipts/e1-actor-workspace-qualification-20261020.md`. The intended behavior applies only to events created under a verified principal. Existing ownerless rows must remain unowned, invisible and unchanged. The operator has directed that no legacy record be assigned or exposed and no mapping decision be changed. Do not sync or answer a decision, mutate product data, or infer E1/Q1 acceptance.

## Native identities and evidence

- Worker task/run: `cbad6395-7d01-4c2f-8e2b-ad41de6a471f` / `a7df133f-03ca-4217-b451-aa3dac6c8445`.
- Worker profile: `4c19aef1-36a2-4cc5-8961-42648692883b`; exact native parent is the run above.
- Worker status: COMPLETE, exit 0, native goal terminal class `blocked` because the focused test command exited 1 on a pre-existing build failure in protected `scenarios/personal-planner/cli/domains/goals/handlers.go:180` (`*int64 / 100`).
- Baseline manifest: `receipts/e1-authenticated-calendar-owner-20261020/baseline/manifest.json`, SHA-256 `01f6871225b82ef6eb5f377dda323c52e98c5ada70aba689c75e4960331b6efa`; exact original bytes are in `baseline/bytes/`.
- Current manifest: `receipts/e1-authenticated-calendar-owner-20261020/current-manifest.json`, SHA-256 `08b52d0247314345d319d065529b0b7e76f50f89d1b3c048cb75f8169b090127`.
- Focused command: `cd scenarios/personal-planner/api && go test ./internal/calendar ./handlers/calendar -count=1`; exact output is `focused-gate.log`, SHA-256 `8230703ff6b41e4a523709e10ba4cbe1cafe9b661c267324e9caa7eac2a80e5c`. Calendar package passed; handler owner assertions ran, but the integrated test then failed while building the supported Planner CLI at the unrelated protected pointer-arithmetic compile error.
- Worker handoff: `handoff.md`. Worker epoch-check: `epoch-check.log`, SHA-256 `d41b1dc2675ace7f7316d2e0402a69f482c2b3086f41588252612092f3324d96`; this is not acceptance.

## Review questions

Inspect all five current files against the retained baseline and the brief. Assess verified-principal ownership extraction, owner-scoped list/get/replay/create/update, transaction atomicity, cross-owner idempotency/provider collisions, inaccessible unowned legacy rows, additive schema behavior, test coverage, exact scope and reported growth. Report implementation defects, any behavior the retained test does not establish, and whether the nonzero focused gate prevents slice acceptance. The review must distinguish a source-level PASS from acceptance of the failed gate. Return a clear verdict (PASS, PASS WITH LIMITS, or FAIL), evidence and limits. Do not rerun tests or alter files.

This review concerns only this bounded backend REST/Connect behavior. Live UI/CLI actor propagation, full actor/workspace parity, the calibration full-epoch gate and all original Q1/E1 requirements remain open.
