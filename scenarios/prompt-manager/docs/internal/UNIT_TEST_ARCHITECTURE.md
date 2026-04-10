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
- [x] `api/graph/helpers_test.go` — Centralized shared mock file for the graph package.
- [ ] Other packages still define mocks inline per test file.
- [ ] Mock factory/builder patterns (not standardized across packages).

## Graph Package Shared Mocks

The `api/graph/helpers_test.go` file consolidates all test doubles for the graph package into a single location. In Go, all `_test.go` files in the same package share scope, so these types are available everywhere without import.

| Mock | Purpose | Used By |
|------|---------|---------|
| `mockAgentLister` | Stubs agent store for scanner | `scanner_test.go` |
| `mockTeamLister` | Stubs team store for scanner | `scanner_test.go` |
| `mockSkillLister` | Stubs skill store for scanner | `scanner_test.go` |
| `mockRelationStore` | Stubs relation store for membership scanning | `scanner_test.go` |
| `mockAgentNodeSource` | Stubs agent listing for builder | `builder_test.go` |
| `mockTeamNodeSource` | Stubs team listing for builder | `builder_test.go` |
| `mockSkillNodeSource` | Stubs skill listing for builder | `builder_test.go` |
| `mockGraphScanner` | Stubs edge scanning for builder | `builder_test.go` |
| `mockGraphIndexProvider` | Stubs index provider for handlers | `handlers_test.go` |
| `testIndex()` | Creates a `GraphIndex` with predetermined data | `handlers_test.go` |
| `mockGraphBuilder` | Stubs graph building for index store | `index_test.go` |
| `stubCodeDetector` | Stubs the `codeDetector` interface | `scanner_test.go` |

## Infrastructure Status
- [ ] Testcontainers configured for database tests (none found).

## Issues Found
1. Inline mock definitions in API tests outside the graph package (example: `scenarios/prompt-manager/api/teams/handlers_test.go`).
2. No shared test utility directories for API or UI.

## Priority Improvements
1. Add `scenarios/prompt-manager/api/internal/testutil/` for shared HTTP helpers and mock builders.
2. Add `scenarios/prompt-manager/ui/src/test-utils/` for shared render helpers and mock factories.
