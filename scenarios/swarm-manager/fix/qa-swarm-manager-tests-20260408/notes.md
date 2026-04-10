## Problem
Tests failed 4 of 11. Failures reported by GCT:
- standards: standards violations exceed fail_on=high (highest=high)
- lint: type checking failed with 5 error(s)
- docs: docs validation failed
- smoke: ui smoke blocked due to newer source file ui/src/components/command-post/SnoozePopover.tsx

## Top Violations
- scenarios/swarm-manager/api/eventlog_setup.go: Direct sql.Open() without api-core database.Connect (high)
- scenarios/swarm-manager/api/internal/dispatch: missing test file (medium)
- scenarios/swarm-manager/api/internal/logging: missing test file (medium)
- scenarios/swarm-manager/cli/cmd: missing test file (medium)
- scenarios/swarm-manager/cli/go.mod: Go CLI builds without workspace mode (high)

## Impact
Failing tests and smoke checks block readiness and mask regressions. Type errors and docs validation failures reduce confidence in release readiness.

## Reproduction
git-control-tower review-run swarm-manager --json

## Success Criteria
- Tests pass 11/11
- Type errors = 0
- Docs validation passes
- UI smoke passes after restart
- Standards high violations = 0

## Proposed Action
1. Replace sql.Open with api-core database.Connect in api/eventlog_setup.go.
2. Add missing *_test.go files in api/internal/dispatch, api/internal/logging, and cli/cmd.
3. Fix Go CLI workspace mode in cli/go.mod.
4. Resolve the 5 type errors and repair docs validation.
5. Restart scenario and re-run ui smoke to clear stale bundle warning.
