## Problem
Code quality score is 0/100 with 88 violations. Top categories: complex_functions (45), long_files (43), tech_debt_markers (2). File-level evidence comes from tidiness-manager recommendations.

## Top Violations
- scenarios/git-control-tower/api/gitignore_health_service_test.go: line_count 392, complexity_max 8, duplication_pct 85.2
- scenarios/git-control-tower/api/tidiness_manager_client.go: line_count 156, complexity_max 7, duplication_pct 100
- scenarios/git-control-tower/api/test_genie_client.go: line_count 123, complexity_max 7, duplication_pct 100
- scenarios/git-control-tower/api/ssh/service_test.go: line_count 137, complexity_max 5, duplication_pct 100
- scenarios/git-control-tower/api/filerelations/languages/golang/scanner.go: line_count 49, complexity_max 4, duplication_pct 100
- scenarios/git-control-tower/api/filerelations/languages/python/scanner.go: line_count 48, complexity_max 4, duplication_pct 100
- scenarios/git-control-tower/api/tidiness_manager_handler_test.go: line_count 51, complexity_max 2, duplication_pct 100

## Impact
Low code quality increases defect risk and slows iteration. It also keeps readiness yellow and blocks QA confidence in new work.

## Reproduction
git-control-tower review-run git-control-tower --json
tidiness-manager recommend-refactors git-control-tower --limit 10 --format json

## Success Criteria
- Code quality score >=70
- complex_functions <10
- long_files <10
- tech_debt_markers = 0

## Proposed Action
1. Refactor the longest/most complex files listed above (split into smaller helpers, reduce duplication).
2. Remove tech debt markers during refactor work and track progress via tidiness-manager visit notes.
3. Re-run GCT review and tidiness recommendations to confirm score improvement.
