## Problem
Standards report critical UI dependency and tooling gaps (missing required packages, missing tsconfig/eslint config, and non-pnpm install guidance). These violations contribute to standards and test phase failures.

## Top Violations
- scenarios/vrooli-bridge/ui/package.json: Missing @vrooli/api-base (critical)
- scenarios/vrooli-bridge/ui/package.json: Missing @vrooli/iframe-bridge (critical)
- scenarios/vrooli-bridge/ui/tsconfig.json: tsconfig.json not found (high)
- scenarios/vrooli-bridge/ui/eslint.config.js: ESLint config not found (high)
- scenarios/vrooli-bridge/.vrooli/service.json: npm install guidance in pnpm workspace (high)
- scenarios/vrooli-bridge/ui/node_modules: unexpected node_modules presence (low)

## Impact
Standards failures block readiness and leave UI builds without safety-critical lint and TypeScript guardrails. Missing deps and config also drive lint/test failures.

## Reproduction
git-control-tower review-run vrooli-bridge --json

## Success Criteria
- Standards blockingViolations = 0
- ui/package.json includes required @vrooli packages
- ui/tsconfig.json and ui/eslint.config.js exist with required safety rules
- .vrooli/service.json uses pnpm install guidance
- No stray ui/node_modules in repo state

## Proposed Action
1. Add @vrooli/api-base and @vrooli/iframe-bridge to ui/package.json.
2. Create ui/tsconfig.json with strict/noUncheckedIndexedAccess settings and required comment block.
3. Create ui/eslint.config.js with the required safety-critical ruleset.
4. Update .vrooli/service.json to use pnpm install command for UI dependencies.
5. Clean or ignore ui/node_modules if it is being tracked.
