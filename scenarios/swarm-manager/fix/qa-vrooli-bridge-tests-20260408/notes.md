## Problem
GCT tests failed 8 of 11 phases for vrooli-bridge: structure (docs dir missing), standards (critical violations), lint (type error), smoke (stale UI bundle), unit (go test failure), playbooks (registry missing), business (no requirement modules), performance (go build failure).

## Top Violations
- scenarios/vrooli-bridge/docs (structure): required directory missing
- scenarios/vrooli-bridge/ui/package.json (smoke): source newer than bundle
- scenarios/vrooli-bridge/api/main.go (lint): undefined http type errors during typecheck
- scenarios/vrooli-bridge/api (unit/performance): go test ./... failed and go build failed
- scenarios/vrooli-bridge/bas/registry.json (playbooks): registry not found
- scenarios/vrooli-bridge/requirements/index.json (business/standards): P0 target missing requirements

## Impact
Readiness is blocked and multiple validations cannot run. Missing docs and requirements also prevent downstream playbooks and business validation from executing reliably.

## Reproduction
git-control-tower review-run vrooli-bridge --json

## Success Criteria
- Tests pass 11/11
- structure phase passes with required docs directory present
- standards phase passes (0 high/critical)
- lint phase passes with 0 type errors
- smoke passes after scenario restart
- unit and performance phases pass for Go tests/build
- playbooks registry present and business requirements modules detected

## Proposed Action
1. Create scenarios/vrooli-bridge/docs with required structure.
2. Regenerate bas/registry.json for playbooks.
3. Run `vrooli scenario requirements init` and link P0/P1 targets in requirements/index.json.
4. Resolve standards violations (UI deps/config + server compliance).
5. Fix Go type errors in api/main.go and make go test ./... pass.
6. Restart scenario to clear UI bundle staleness, then re-run ui smoke.
7. Re-run GCT tests to confirm 11/11.
