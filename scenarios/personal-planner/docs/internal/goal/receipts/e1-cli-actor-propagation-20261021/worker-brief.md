# Planner I18 E1 bounded slice — trace routed CLI actor propagation

- Directive: `ORCH-E1-CLI-ACTOR-PROPAGATION-20261021`.
- Epoch: `scenarios/personal-planner/docs/internal/goal/epochs/E1.md`.
- Exact parent run: `5d07e651-eaa6-48ec-89b4-5dc359059193`; parent task: `790cb264-1a63-47eb-89e6-3b9ee3e16c23`.
- Worker card: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`.
- Effort: `effort:personal-planner-I18`; workload `planner-I18/E1`; scope `scenarios/personal-planner`.

## Resume set and evidence

Read this brief, the worker card, E1 modules/budgets/exit gate and current directive, the last 20 E1 slice-log entries, `GOAL.md`, `CONTRACT.md` actor/workspace and calendar requirements, `AUTHORIZATION.md`, and these retained receipts:

- `receipts/e1-cli-actor-propagation-20261021/baseline/manifest.json` (SHA-256 `5a18e4b7557efa37bd8f1e3476ff789f6714891c575b1f83cfd61096604ff28d`; anchor `gcta-6e54cfd42e78`; checkout head `52097258290d1bff0191319242d4948e214af9e3`)
- `receipts/e1-goal-cli-progress-compile-20261021/focused-calendar.stdout` (SHA-256 `e044446dea0a858b663d28f1ee0b6455c36d30953f154d51163af40df5be10a`, exit 1)
- `receipts/e1-goal-cli-progress-compile-20261021/independent-review.md`
- `receipts/e1-authenticated-calendar-owner-20261020/independent-review.md`

Baseline exact sources: `cli/app.go` (68 lines; SHA-256 `50c7e21369b561930b2ad2e41595634e454db0b1628ae5698c8a5d72c9b237d6`), `cli/domains/calendar/handlers.go` (270 lines; SHA-256 `d3881da2cc0af8d2c6e145b1479ac5e99f27e48a4d9226897ae371791b479bbd`), and `api/handlers/calendar/connect_events_test.go` (506 lines; SHA-256 `db239514eebd5da62df59d3e77d49d88ec5ee9db895f50712594f139adbd18c4`). Confirm every identity before any write; preserve the baseline.

## Outcome

Trace why the existing routed CLI requests in `TestCalendarRESTConnectAndWorkspaceShareRoutedTestPoolIdentityAndRevisions` reach the synthetic API without a verified actor after the prior pointer compile fix. Establish the concrete request/header/config path and root cause. If a Planner-local cause is proven, make the smallest correction in the allowed files and rerun the required focused command. A bounded trace with no source change is acceptable if the cause is outside this scope; return the exact evidence and needed owner amendment.

## Allowed write set

Only these existing Planner files:

- `scenarios/personal-planner/cli/app.go`
- `scenarios/personal-planner/cli/domains/calendar/handlers.go`
- `scenarios/personal-planner/api/handlers/calendar/connect_events_test.go`

No new files, no shared packages, no API auth middleware, no product data, Proto/generated outputs, provider calls, credential minting, identity substitution, anonymous retry, or gate weakening. Never print, copy, or log credential values; presence/absence and redacted header names only. If root cause requires shared CLI/auth core, API middleware, permission/profile policy or an operator-only credential, stop and report a scope-amendment/owner dependency without changing it.

## Estimate, gate, and limits

Estimate at most +40 runtime lines / +30 test lines; prefer no line growth for a trace-only result. Stop and return before exceeding either cap, before any product-data write, or if the exact source identity drifts. Do not touch the existing newly added owner-binding implementation or unrelated dirty files. Required focused command: `cd scenarios/personal-planner/api && go test ./internal/calendar ./handlers/calendar -count=1`; preserve raw stdout/stderr, exit status, hashes, line counts, and the native epoch-check result. The preceding run passed `internal/calendar` and failed the routed handler test with HTTP 401 `verified actor missing` / `unauthenticated`. Do not mark that prior gate green. Do not claim acceptance of owner binding, E1, or Q1.

Complete the bounded assignment, append one slice-log line, then run `agent-manager effort epoch-check scenarios/personal-planner/docs/internal/goal/epochs/E1.md --runs <your-run-id> --wake-key 5d07e651-eaa6-48ec-89b4-5dc359059193`. Preserve all original E1 gates and historical evidence.
