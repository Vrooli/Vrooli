# Prompt-Manager Unit Testing Architecture

## Last Updated
2026-05-01

## Test Organization Status
- [x] Go tests are co-located with source files (`scenarios/prompt-manager/api/**/_test.go`).
- [x] TypeScript tests are co-located with source files (`scenarios/prompt-manager/ui/src/**/*.test.ts(x)`).
- [x] Naming conventions are consistent (`*_test.go`, `*.test.ts`, `*.test.tsx`).
- [x] Vitest configuration present (`scenarios/prompt-manager/ui/vitest.config.ts`).
- [x] Shared API test utility package exists (`api/internal/testutil/`).
- [x] Shared CLI command harness exists (`cli/internal/testutil/`).
- [x] Shared UI routine render/provider utility layer exists (`ui/src/test/`).

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
- [x] `api/internal/testutil/no_prod_import_test.go` prevents production imports of test-only API helpers.
- [x] `cli/internal/testutil/no_prod_import_test.go` prevents production imports of test-only CLI helpers.
- [x] `ui/src/test/network.test.ts` documents and verifies the explicit fetch guard for UI unit tests.

## Shared API Test Utilities

The canonical shared test package is `scenarios/prompt-manager/api/internal/testutil`.

| Package | Purpose |
|---------|---------|
| `fixtures` | Domain object factories with functional options. Team fixtures live here first because heartbeat and teams tests had duplicate builders with undocumented drift. |
| `mocks` | Hand-written fakes, one per production seam, with sane defaults and explicit error knobs. |
| `httpx` | HTTP handler request/response helpers for JSON bodies, route variables, decoding, and status assertions. |
| `assertx` | Domain assertions that report broken contracts clearly. |

Production code must not import `prompt-manager/internal/testutil/...`. Tests may import it directly, except tests in the `store` package should avoid fixture imports that create an import cycle back through `prompt-manager/store`.

## Shared CLI Test Utilities

The canonical shared CLI test package is `scenarios/prompt-manager/cli/internal/testutil`.

| Helper | Purpose |
|--------|---------|
| `NewContext(t)` | Creates an `appctx.Context` fake that records method/path/query/payload and injects typed responses or errors. |
| `Output(t, fn)` | Captures stdout/stderr for command formatting assertions. |
| `IO(t, stdin, fn)` | Captures stdout/stderr while providing stdin for interactive command paths. |

CLI command tests should prefer this harness over package-local fake contexts when they are testing request construction, API error handling, or user-facing output. Package-local fakes remain acceptable only when they model a package-specific seam that the shared harness cannot represent cleanly.

## Shared UI Test Utilities

The canonical shared UI test import is `scenarios/prompt-manager/ui/src/test`, usually via `@/test`.

| Helper | Purpose |
|--------|---------|
| `createTestQueryClient()` | Creates a React Query client with retries and cache retention disabled for deterministic tests. |
| `renderWithProviders(ui, options)` | Wraps components with React Query, MemoryRouter, ThemeProvider, and optional Toaster support. |
| `renderHookWithProviders(callback, options)` | Gives hooks the same provider defaults as component tests. |
| `createTestWrapper(options)` | Exposes the provider wrapper for tests that need direct Testing Library control. |
| `installStorageMock()` / `resetStorageMock()` | Provides a consistent localStorage mock used by global setup and storage-focused tests. |
| `setViewport()` / `restoreViewport()` | Drives viewport-dependent behavior without repeating window property setup. |
| `installFetchGuard()` / `jsonResponse()` | Makes network intent explicit. Unexpected fetches fail with the requested URL instead of silently reaching localhost. |
| `installR3FDOMWarningFilter()` | Suppresses only known React DOM warnings caused by rendering R3F intrinsic tags in jsdom; use it only in R3F suites that intentionally render Three primitives without a Canvas reconciler. |

UI tests should prefer `@/test` for provider setup and browser API mocks. Tests that intentionally exercise a service or API call should either mock the service seam directly or install an explicit fetch guard with an allow-list. Component tests should not import stores that auto-fetch unless that store is mocked or the network call is the behavior under test.

Persisted world stores skip import-time auto-fetches in Vitest and expose explicit fetch actions for tests. Component tests that mount settings, world, or AI status panels should mock service seams directly instead of allowing `localhost` API requests. Tests that intentionally cover error logging should spy on `console.error`/`console.warn` and assert the logged contract instead of emitting expected failures into the shared test output.

## Issues Found
1. Inline mock definitions in API tests outside the graph package (example: `scenarios/prompt-manager/api/teams/handlers_test.go`).
2. Some CLI packages still carry package-local fake contexts that should be migrated to `cli/internal/testutil`.
3. Some UI component tests still carry local provider wrappers and warning-prone R3F setup.
4. Some package-local API helpers remain where direct shared-fixture imports would create an import cycle.

## Priority Improvements
1. Fill out `scenarios/prompt-manager/api/internal/testutil/httpx`, `mocks`, and `assertx` while migrating package-local handlers.
2. Continue migrating CLI command tests to `scenarios/prompt-manager/cli/internal/testutil`, prioritizing the remaining low-coverage domains after the initial `skills`, `search`, `discover`, `graph`, and `topics` slices.
3. Continue migrating UI component tests to `@/test`, prioritizing tests with local provider wrappers, accidental fetches, or noisy R3F warnings.
