# Prompt-Manager Unit Testing Architecture

## Last Updated
2026-02-02

## Test Organization Status
- [x] Go tests are co-located with source files (`scenarios/prompt-manager/api/**/_test.go`).
- [x] TypeScript tests are co-located with source files (`scenarios/prompt-manager/ui/src/**/*.test.ts(x)`).
- [x] Naming conventions are consistent (`*_test.go`, `*.test.ts`, `*.test.tsx`).
- [x] Vitest configuration present (`scenarios/prompt-manager/ui/vitest.config.ts`).
- [ ] Shared test utilities package exists (no `api/internal/testutil/` or `ui/src/test-utils/` found).

## Mock Organization Status
- [ ] Centralized mock packages (mocks are defined inline per test file).
- [ ] Mock factory/builder patterns (not standardized across packages).

## Infrastructure Status
- [ ] Testcontainers configured for database tests (none found).

## Issues Found
1. Inline mock definitions in API tests (example: `scenarios/prompt-manager/api/teams/handlers_test.go`).
2. No shared test utility directories for API or UI.

## Priority Improvements
1. Add `scenarios/prompt-manager/api/internal/testutil/` for shared HTTP helpers and mock builders.
2. Add `scenarios/prompt-manager/ui/src/test-utils/` for shared render helpers and mock factories.
