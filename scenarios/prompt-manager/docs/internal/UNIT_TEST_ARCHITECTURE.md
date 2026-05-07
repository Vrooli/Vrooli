# Prompt-Manager Unit Testing Architecture

## Last Updated
2026-05-02

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
| `assertx` | Focused assertions for contract fragments and domain expectations that report broken contracts clearly. |

Production code must not import `prompt-manager/internal/testutil/...`. Tests may import it directly, except tests in the `store` package should avoid fixture imports that create an import cycle back through `prompt-manager/store`.

`httpx` is now used by handler tests in `agents`, `teams`, `heartbeat`, and `worldscale`. New handler tests should use it for recorder construction, JSON requests, route variables, response decoding, and status assertions instead of repeating raw `httptest` and mux setup. Use `assertx.Contains` when a test protects a named error-body or prompt-fragment contract; keep generic string checks local when they are incidental.

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

## Decision Boundary Coverage

High-risk behavior should be covered where the decision is made, not only through UI or scenario workflows. Current examples include heartbeat decision/defer/pending behavior, graph health and scoring, AI search fallback paths, file-backed world configuration persistence/malformed data handling, and `/skills/read` experiment-aware variant selection.

The `/skills/read` variant-selection tests protect three contracts: running experiments may override the returned content with a selected variant, control selections keep the original skill content, and non-running experiments fail before returning content. These tests use deterministic experiment weights instead of random assertions.

## Requirement Traceability

Use requirement IDs from `requirements/*/module.json` in test names, comments, or this table when a test protects a business-critical contract. Prefer file-level mapping here when a package already has several focused tests for the same requirement.

| Requirement | Coverage Point |
|-------------|----------------|
| `REQ-P0-001` REST API for Skill CRUD | `api/skills/handlers_test.go` covers create conflict/default ID behavior, read resolution, strict missing handling, variable extraction/substitution, sync variables, and experiment-aware read selection. |
| `REQ-P0-003` Pack-based Skill Structure | `api/skills/handlers_test.go` covers folder/file resolution paths; `api/store/*_test.go` covers file-backed stores and pack/folder persistence. |
| `REQ-P0-004` Full-text Search Implementation | `api/search/*_test.go`, `api/aisearch/aisearch_test.go`, and CLI `search` tests cover text search and AI fallback behavior. |
| `REQ-P0-005` CLI Basic Operations | `cli/internal/testutil` plus command tests in `skills`, `search`, `discover`, `graph`, `topics`, `agents`, `members`, `testing`, `experiments`, and `tags` cover request contracts, validation-before-API, and user-facing output. |
| `REQ-P1-001` Qdrant Vector Search | `api/aisearch/*_test.go` covers embedder/vector-store success, error fallback, entity search, and reconciler (plan/apply/sync-loop) behavior. |
| `REQ-P1-005` Tag Management System | `api/tags/handlers_test.go` and `cli/tags/tags_test.go` cover API and CLI tag behavior. |
| `REQ-P2-001` Import/Export System | `api/teams/handlers_export_test.go`, `api/teams/handlers_import_test.go`, and BAS export/import-adjacent team flows cover team payload import/export contracts. |
| `REQ-P0-016` REST API for Agent CRUD | `api/agents/handlers_test.go`, `cli/agents/agents_test.go`, and agent-related BAS workflows cover API and CLI agent contracts. |
| `REQ-P0-018` REST API for Team CRUD | `api/teams/*_test.go`, `api/heartbeat/*team*test.go`, and BAS team/member workflows cover team API contracts and runtime usage. |
| `REQ-P1-023` Team-Member Relations API | `api/teams/handlers_cleanup_test.go`, heartbeat member-context tests, and member CLI tests cover relation cleanup, context inclusion, and CLI request behavior. |
| `REQ-P2-027` 3D World Canvas | `ui/src/test` R3F harness users and BAS world UI workflows cover core render contracts without accidental network calls. |
| World configuration API contract | `api/worldscale/handlers_test.go` and `api/worldseats/handlers_test.go` cover default reads, malformed persisted data, validation errors, and persistence for file-backed world settings. |

## Issues Found
1. Inline mock definitions in API tests outside the graph package (example: `scenarios/prompt-manager/api/teams/handlers_test.go`).
2. The priority low-coverage CLI domains now use the shared harness for contract coverage, but future CLI tests should continue replacing one-off fakes when they touch the API context seam.
3. Some UI component tests still carry local provider wrappers and warning-prone R3F setup.
4. Some package-local API helpers remain where direct shared-fixture imports would create an import cycle.

## Priority Improvements
1. Fill out `scenarios/prompt-manager/api/internal/testutil/httpx`, `mocks`, and `assertx` while migrating package-local handlers.
2. Add deeper CLI edge-case coverage only where it protects a command contract; the initial low-coverage priority pass now covers `skills`, `search`, `discover`, `graph`, `topics`, `agents`, `members`, `testing`, `experiments`, and `tags`.
3. Continue migrating UI component tests to `@/test`, prioritizing tests with local provider wrappers, accidental fetches, or noisy R3F warnings.
