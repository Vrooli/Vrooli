## Problem
Standards report P0 targets missing requirement links in requirements/index.json. This blocks standards and business validation for the scenario.

## Top Violations
- scenarios/vrooli-bridge/requirements/index.json: P0 target missing requirements (critical)
- scenarios/vrooli-bridge/ui/package.json: Missing @vrooli/api-base (critical)
- scenarios/vrooli-bridge/ui/package.json: Missing @vrooli/iframe-bridge (critical)
- scenarios/vrooli-bridge/ui/server.js: Missing proxyToApi in custom server (high)
- scenarios/vrooli-bridge/ui/tsconfig.json: tsconfig.json not found (high)

## Impact
Without requirement traceability, the scenario fails standards gates and business validation cannot confirm P0/P1 coverage. This blocks readiness and undermines compliance expectations.

## Reproduction
git-control-tower review-run vrooli-bridge --json

## Success Criteria
- Standards blockingViolations = 0
- Every P0/P1 operational target links to at least one requirement in requirements/index.json
- Business phase detects requirement modules successfully

## Proposed Action
1. Create missing requirement modules (if absent) using `vrooli scenario requirements init`.
2. Link each P0/P1 target to at least one requirement in requirements/index.json.
3. Re-run standards and business phases to confirm clearance.
