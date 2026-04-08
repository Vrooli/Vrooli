## Problem
Tests failed 3 of 11. Failures reported by GCT:
- standards: standards violations exceed fail_on=high (highest=high)
- dependencies: required resources unhealthy: sqlite (running=false healthy=false)
- playbooks: bas/cases/01-health-check-endpoint/api/health.json returns 400 (unknown field "requirements")

## Top Violations
- scenarios/git-control-tower/ui/tsconfig.json: strict/noUncheckedIndexedAccess/protective comment missing (high)
- scenarios/git-control-tower/ui/eslint.config.js: missing per-rule CRITICAL comments (medium)
- scenarios/git-control-tower/api/filerelations: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/languages/golang: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/languages/python: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/languages/typescript: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/resolver: missing test file (medium)
- scenarios/git-control-tower/api/filerelations/scanner: missing test file (medium)

## Impact
Test suite failure blocks readiness and hides regressions. Standards high violations and missing sqlite resource prevent reliable health checks. The playbook failure blocks smoke validation.

## Reproduction
git-control-tower review-run git-control-tower --json

## Success Criteria
- Tests pass 11/11
- Standards high violations = 0 and warnings reduced
- sqlite resource is healthy during tests
- Health-check playbook bas/cases/01-health-check-endpoint/api/health.json executes without 400

## Proposed Action
1. Fix standards high issues in ui/tsconfig.json and ui/eslint.config.js.
2. Add missing *_test.go files under api/filerelations and language subdirectories.
3. Ensure sqlite resource is started before tests (document or enforce in test setup).
4. Update the health-check playbook schema to remove the unknown "requirements" field.
