# Planner I18 E1 worker brief — calibration regression fixture correction

Read the worker card `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`, the active correction section and the preceding calibration section in `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`, accepted `CONTRACT.md` item 2 and `ACCEPTANCE.md`, and the retained reviewer receipt `scenarios/personal-planner/docs/internal/goal/receipts/e1-review-calibration-20261020/independent-review.md`. Do not read FEEDBACK.md. Parent/orchestrator run: `5d07e651-eaa6-48ec-89b4-5dc359059193`. Directive: `ORCH-E1-REVIEW-AGGREGATION-CORRECT-20261020`.

Scoped parent anchor: `gcta-0fdd0a0d82cf` covers `scenarios/personal-planner/**` at head `52097258290d1bff0191319242d4948e214af9e3`.

## Authority and finding

This is a bounded correction inside the approved I18 E1 outcome. The existing implementation in `api/internal/review/service.go` groups and sums manual actuals by allocation. The independent review failed because the live test no longer exercises the accepted split 30+30-minute actuals against one 60-minute allocation. The accepted calibration worker originally reported +1 runtime/+16 test lines. Later goal-variance changes (+1/+11) and trend-order changes (+0/+6) were admitted and independently accepted as separate E1 slices. Preserve those changes. The reviewer’s baseline-to-current aggregate line count is reconcilable as +2 runtime/+33 tests; it is not a reason to revert either accepted slice.

## Exact scope

Only edit `scenarios/personal-planner/api/internal/review/service_test.go`, specifically the calibration fixture/assertions. `service.go` is protected context; do not edit it. Preserve goal-variance tests and the trend-order behavior. No schema, UI, provider, product data, unrelated tests, dependencies, profile/permission changes, or E1 gates. Task scope is `scenarios/personal-planner`; the allowed product write set is the one test file. Write the exact focused output and handoff evidence only under `scenarios/personal-planner/docs/internal/goal/receipts/e1-review-calibration-20261020/`.

## Baseline and stop rule

The orchestrator retained exact current bytes at `receipts/e1-review-calibration-20261020/baseline-correction/manifest.json` (manifest SHA-256 `98d0351c210de1bec7ef8f55d08666d43b502596ea0c40bef0f527ed07d22846`, captured 2026-10-20T23:36:45Z). Protected `service.go` SHA-256 `15b5b300263c55a6424d9765ebe1469e278a467ad6537c581b1a2421b2fd7753` (557 lines). Allowed `service_test.go` SHA-256 `28436a2ba040a3454723f493d5c23a32430a90acb02126f297c46ff9902645e3` (338 lines). Recheck both immediately before writing; if either differs, stop without edits and return the exact observed mismatch. Preserve the snapshot unchanged.

## Required regression

Use two distinct allocations, each planned for 60 minutes. Link two 30-minute actual rows on different dates to allocation A, summing to 60; link one 90-minute actual row to allocation B. The test must verify one matched 0% allocation sample plus one +50% sample: total sample size 2, average error 25%, over ratio 0.5, under ratio 0. Chronological trend order is by each allocation’s latest contributing actual date: put B (+50%) latest date before A (0%) and assert response and persisted trend `[50, 0]`. Verify persisted sample count, average/ratios, category projection, trend and timestamp parity with the response. Keep existing separate accepted goal-variance assertions intact.

## Estimate, validation and handoff

One work unit, expected +0 runtime/+0 test lines (literal and expected-value edits only), no deletions. Current E1 cumulative growth is +1,731 runtime/+2,225 tests against the +2,100 runtime ceiling. Stop and return if required runtime changes or net test growth exceeds the brief; do not exceed the ceiling. Run only `cd scenarios/personal-planner/api && go test ./internal/review -count=1`; retain exact command/output and before/after hashes and line counts. Append one slice-log line and run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your run ID> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`. Do not write ACCEPTED. Independent review of the corrected candidate and orchestrator mechanical acceptance remain separate; E1/Q1 stays NOT ACCEPTED. End with changed, verified, remaining, and unverified.
