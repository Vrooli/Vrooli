## Problem
Code quality score is 64.93 with 17 violations (lint issues, complex functions, long files). This keeps the scenario below the >=70 quality target.

## Top Violations
- scenarios/vrooli-bridge/api/main_test.go: line_count 1421, complexity_max 27 (long file + complex functions)
- scenarios/vrooli-bridge/api/test_helpers.go: line_count 374, complexity_max 12
- scenarios/vrooli-bridge/api/main.go: line_count 488, complexity_max 11
- scenarios/vrooli-bridge/ui/src/app.js: line_count 325, function_count 62
- scenarios/vrooli-bridge/ui/dist/app.js: line_count 325 (generated artifact, should be excluded from refactor scope)

## Impact
Large and complex files increase change risk, slow onboarding, and make tests harder to maintain. Lint issues also contribute to test failures and reduce confidence in release readiness.

## Reproduction
git-control-tower review-run vrooli-bridge --json
tidiness-manager recommend-refactors vrooli-bridge --limit 10 --format json

## Success Criteria
- Code quality score >=70
- complex_functions <20
- long_files <10
- lint_issues = 0

## Proposed Action
1. Split api/main_test.go into smaller focused test files and extract shared helpers.
2. Refactor api/test_helpers.go and api/main.go to reduce complexity and file size.
3. Reduce UI entrypoint size by splitting ui/src/app.js into smaller modules.
4. Remove or ignore generated ui/dist/app.js from refactor scope if it is build output.
5. Re-run tidiness-manager recommendations and GCT to validate improvements.
