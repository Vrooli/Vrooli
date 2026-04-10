## Problem
Standards dimension reports 10 warnings (0 blocking). High severity issues in api/eventlog_setup.go and cli/go.mod are causing standards failures.

## Top Violations
- scenarios/swarm-manager/api/eventlog_setup.go: Direct sql.Open() without api-core database.Connect (high)
- scenarios/swarm-manager/api/internal/dispatch: missing test file (medium)
- scenarios/swarm-manager/api/internal/logging: missing test file (medium)
- scenarios/swarm-manager/cli/cmd: missing test file (medium)
- scenarios/swarm-manager/cli/go.mod: Go CLI builds without workspace mode (high)

## Impact
Standards warnings (high severity) block readiness and cause test failures. Missing tests reduce coverage for core dispatch and logging paths.

## Reproduction
git-control-tower review-run swarm-manager --json

## Success Criteria
- High severity standards violations = 0
- Total standards warnings reduced to <=5
- Tests no longer fail due to standards

## Proposed Action
1. Replace sql.Open with api-core database.Connect in api/eventlog_setup.go.
2. Add *_test.go files in api/internal/dispatch, api/internal/logging, and cli/cmd.
3. Update cli/go.mod to enable workspace mode per standards guidance.
