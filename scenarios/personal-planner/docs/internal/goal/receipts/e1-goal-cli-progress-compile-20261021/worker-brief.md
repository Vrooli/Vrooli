# Planner I18 E1 bounded worker brief — restore goal progress CLI compilation

Worker card: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.
Full epoch: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`.
Parent/orchestrator exact native run: `5d07e651-eaa6-48ec-89b4-5dc359059193`; task `790cb264-1a63-47eb-89e6-3b9ee3e16c23`.
Directive: `ORCH-E1-GOAL-CLI-PROGRESS-COMPILE-20261021`.

## Outcome and authority

Restore compilation of the already accepted Planner measurable-goal CLI handler so the active E1 calendar ownership slice can run its required focused gate. This is a bounded correction inside the approved Planner goal/CLI scope and preserves all current user/agent-authored bytes except the single invalid pointer arithmetic expression. Do not alter the earlier accepted measurable-goal behavior, edit unrelated paths, or infer acceptance of another slice or E1/Q1.

## Exact scope and baseline

Only allowed product path: `scenarios/personal-planner/cli/domains/goals/handlers.go`.
All other product files are protected. The file contains previously accepted measurable-goal work; preserve its current contents and make only the minimal compile correction at `progressReport` around line 180. The target expression currently divides a `*int64` field. Use the generated nil-safe accessor (`GetProgressBasisPoints()`) or another equivalent type-correct expression, preserving the reported integer percentage behavior. Do not refactor or reformat unrelated code.

Anchor: `gcta-c3335d76a3f7`, checkout head `52097258290d1bff0191319242d4948e214af9e3`.
Baseline manifest: `receipts/e1-goal-cli-progress-compile-20261021/baseline/manifest.json`, SHA-256 `553eddd8c3d0c504ccd2b83b6cf6ed15b6ca9a1c5582a4aeb03a24a9af1504ae` (exact original bytes at `baseline/bytes/scenarios/personal-planner/cli/domains/goals/handlers.go`). Baseline SHA-256 `67589bc1bff2c5e36e19dee433e73a80ce82c8bce4394bd7b5d6f88cb4af0959`, 245 lines. Estimate 0 net runtime lines and 0 net test lines; no test additions. The existing cumulative test growth is already above the expected boundary, so add no tests.

## Required validation and stop conditions

Run exactly: `cd scenarios/personal-planner/api && go test ./internal/calendar ./handlers/calendar -count=1`.
Retain raw output and exit code in this receipt. This command is the required focused gate for the independently reviewed new-event ownership slice; it previously failed at the compile error identified above. Do not repair or bypass any subsequent unrelated failure. Do not run broad Test Genie or mutate product data.

Append an honest E1 slice-log line with actual line deltas, then run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your run ID> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`. Do not write acceptance. End with changed, verified, remaining, and unverified. The orchestrator will obtain independent review and rerun the mechanical gate. E1/Q1 stays unaccepted.
