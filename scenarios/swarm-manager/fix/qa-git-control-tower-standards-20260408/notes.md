## Problem
Standards dimension reports 17 warnings (0 blocking). High severity findings in ui/tsconfig.json are causing standards to fail the test suite.

## Top Violations
- scenarios/git-control-tower/ui/tsconfig.json: strict mode not enabled (high)
- scenarios/git-control-tower/ui/tsconfig.json: noUncheckedIndexedAccess not enabled (high)
- scenarios/git-control-tower/ui/tsconfig.json: missing protective comment block (high)
- scenarios/git-control-tower/ui/eslint.config.js: missing per-rule CRITICAL comments (medium)
- scenarios/git-control-tower/api/filerelations: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/languages/golang: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/languages/python: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/languages/typescript: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/resolver: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/scanner: missing test file (medium)

## Impact
Standards warnings (especially high severity TypeScript safety settings) block readiness and cause test failures. Missing test files reduce coverage and increase regression risk.

## Reproduction
git-control-tower review-run git-control-tower --json

## Success Criteria
- High severity standards violations = 0
- Total standards warnings reduced to <=5
- Tests no longer fail due to standards

## Proposed Action
1. Enable strict and noUncheckedIndexedAccess in ui/tsconfig.json and add required protective comment block.
2. Add CRITICAL comments for safety rules in ui/eslint.config.js.
3. Add *_test.go files under api/filerelations and language/resolver/scanner subdirectories.
