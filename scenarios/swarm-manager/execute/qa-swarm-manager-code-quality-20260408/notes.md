## Problem
Code quality score is 0/100 with 379 violations. Top categories: complex_functions (107), long_files (48), tech_debt_markers (3), lint_issues (1). File-level evidence comes from tidiness-manager recommendations.

## Top Violations
- scenarios/swarm-manager/api/internal/backlog/handler_update_test.go: line_count 472, complexity_max 8, duplication_pct 66.7
- scenarios/swarm-manager/api/internal/eventlog/emitter.go: line_count 244, complexity_max 4, duplication_pct 94.7
- scenarios/swarm-manager/api/internal/settings/normalize_test.go: line_count 136, complexity_max 3, duplication_pct 89.0
- scenarios/swarm-manager/api/internal/backlog/validation_test.go: line_count 100, complexity_max 4, duplication_pct 77.0
- scenarios/swarm-manager/api/internal/queue/encode_error_test.go: line_count 39, complexity_max 4, duplication_pct 100
- scenarios/swarm-manager/api/internal/backlog/encode_error_test.go: line_count 39, complexity_max 4, duplication_pct 100

## Impact
Low code quality increases defect risk and slows iteration. It keeps readiness yellow and raises the cost of future changes.

## Reproduction
git-control-tower review-run swarm-manager --json
tidiness-manager recommend-refactors swarm-manager --limit 10 --format json

## Success Criteria
- Code quality score >=70
- complex_functions <20
- long_files <10
- tech_debt_markers = 0

## Proposed Action
1. Refactor the longest/most complex files listed above (split handlers/tests, reduce duplication).
2. Remove tech debt markers during refactor work and record visits via tidiness-manager.
3. Re-run GCT review and tidiness recommendations to confirm improvements.
