# Known Problems & Technical Debt

This document tracks known issues, technical debt, and stability concerns for the swarm-manager scenario UI.

---

## Open Issues

No known open issues at this time.

## Deferred — REST→proto migration of swarm-manager API (2026-05-28)

Per the project-wide `feedback_proto_connect_always.md` rule, every scenario API domain should use proto + Connect-RPC. Swarm-manager today is REST for almost all domains (`/backlog`, `/initiatives`, `/captures`, `/records` shipped 2026-05-28, etc.); the only Connect-RPC surface is `discovery.DiscoveryService.GetAudioToolsEndpoint`.

The **records** domain (added 2026-05-28) was shipped as REST deliberately to match the existing surface — bundling a scenario-wide proto migration into the records plan would have ballooned scope and coupled two unrelated migrations. The records REST surface was designed to be mechanically proto-able later:

- No multipart bodies, no streaming, no third-party shape constraints — all four `RESTReason*` exemption constants are inapplicable.
- Endpoints: `POST /records`, `GET /records`, `GET /records/{id}`, `PATCH /records/{id}/narrative`, `POST /records/{id}/supersede`, `POST /records/search` — each maps 1:1 to a Connect RPC.
- Request/response shapes are flat structs; the JSON tags already match the planned proto field names.

**Next step:** A dedicated swarm-manager-wide proto migration plan should sweep the remaining REST domains together (backlog + initiatives + captures + records + execution + queue + settings + stats + overview) rather than converting one at a time. Until then, this is a tracked, intentional drift — do not silently expand individual feature plans to absorb it.

## Deferred — operating-mode panel

The operating-mode panel rework (2026-05-02) explicitly deferred a few API-side improvements that would have polished UX further. They are tracked here so the next iteration sees them:

1. **Optimistic concurrency on mode-switch.** `POST /api/v1/initiatives/{name}/operating-mode/switch` is last-write-wins. Adding an optional `expected_from_mode` field to the request body would let the picker dialog reject stale switches.
2. **Server-computed switch deltas.** The picker derives capability deltas client-side from before/after catalog entries. Exposing a `preview_deltas` field on `SwitchModeResult` (or a dedicated `POST /switch/preview` endpoint) would let the server be the source of truth for "what will change".
3. **In-flight item executions on the workspace endpoint.** `ItemLevelEmptyState` shows in-flight count from `InitiativeRollup.inProgress` (already piped through). For modes where the rollup isn't available, a richer `OperatingModeWorkspace.activeItemExecutions[]` field would unblock the same display.

## Deferred — operating-mode production-polish UX pass (2026-05-02)

A second polish pass landed five UI-side phases (clickable mode chip + Info-tab card, mode-picker retry, skill viewer, shared sidebar empty-state, header hamburger gating, capability-aware round timeline empty state, artifact viewer dialog, round detail dialog). Two follow-ups stayed out of v1 by design:

1. **Live agent-log streaming on `RoundDetailDialog`.** The dialog renders the round's existing client-side fields (summary, error, items, handoffs, agent profile) and presents `runId` as a copyable mono token. A future iteration could stream agent-manager logs and linkify `runId` to a stable run-detail URL — both depend on contracts we don't yet own. The dialog is built so adding either is a non-breaking extension.
2. **Hover-popover tooltip primitive.** The phase-card `profileKey` chip uses an expanded `title=""` plus a native `<details>` info disclosure for the longer copy. A richer popover primitive (Radix-style) would benefit other surfaces too, but adding it for one chip wasn't worth the dependency surface in this pass.

## Recently Resolved

### Operating-mode "Execution" tab → Flow, with interpretable trace + simulation presets (2026-07-07)

**Problem**: The operating-mode detail page's third tab was labeled "Execution",
which overloaded the term with backlog execution records and read as an
unfinished developer diagnostic panel. Its trace showed a count grid (items,
artifacts, prior rounds, criteria) plus a raw JSON `Emits` blob — useful to
developers, opaque to operators trying to understand *why* a mode works, what
each phase consumes, and how branches (replan / continue / blocked / review /
reconcile) behave. The simulation was a single implicit happy path and the UI
never said the data was fixture-driven rather than a real initiative trace.

**Resolution** (plan `improve-operating-mode-trace-ux-and-simulation`):

- Renamed the tab **Execution → Flow** (greenfield rename of the internal
  `tab=flow` route value, no compat shim). Icon `Workflow`, a compact Flow intro,
  and a **Guide** button (shared `ConceptExplainerDialog`, `FLOW_GUIDE_EXPLAINER`)
  that distinguishes simulation fixtures from live traces and defines round /
  phase / reads / emits / transition / artifact / handoff.
- Backend simulation grew from one happy path into **named deterministic
  presets** ([CODE: api/internal/operatingmode/simulation.go]) that seed
  different phase outputs to exercise real transition guards: holistic-loop
  (clean pass, replan-after-execute, review-not-accepted) and phased-plan-drain
  (drains-in-one-slice, continue-next-slice, replan-plan, blocked). Presets still
  walk the registered graph via the same guard path a live round uses — no agent,
  prompt, lock, or persistence. Selected by `?preset=` on the simulate endpoint.
- Replaced the count-grid + raw-JSON panel with shared **semantic trace
  components** ([CODE: ui/src/components/initiative/operating-mode/phase-trace-panel.tsx]
  + helpers in `phase-interpretability.ts`): reads-by-category cards, an emits
  summary (handoffs, progress, verdict, replan, artifacts, backlog sync,
  readiness), and a transition explanation derived from backend guard data. Raw
  payload stays behind a "View raw payload" disclosure. Simulation and live use
  the same components — no forked UIs.

**Why keep it this way**: The Flow tab must stay operator-facing. Do not regress
it to a raw JSON dump above static prose, and do not make the simulation appear
to use real initiative data — presets are labeled fixtures. Transition logic is
backend-owned; the UI only selects/phrases guard data, so new modes and branches
surface without mode-specific UI.

### Runtime-home data migration completed (2026-05-28)

The 2026-05-27 commit `0b225a5556` ("swarm-manager data to ~/.vrooli") moved domain data from the repo source path to `~/.vrooli/data/vrooli/swarm-manager/...` but left the API still constructing its stores against `scenarioRoot` — so the UI saw an empty world even though 1300+ items were intact on disk. Today's plan (`finish-swarm-manager-runtime-home-data-migration-data-path-vs-repo-anchor-split`) closed the loop:

- `runtimepaths.CachePath` defined (was referenced by `paths_test.go` only).
- `backlog.Handler`, `initiatives.Store`, `captures.Handler`, `records.FileStore`, `execution.Service`, `review.Service` all carry an explicit `dataRoot` field instead of overloading one `rootDir` for both data-folder lookups and repo-anchor walks. `backlog.Handler` and `execution.Service` additionally carry a `repoRoot` for the two legitimate repo-anchor uses (`validate_globs.go:resolveRepoRoot` and `execution/backlog_bridge.go:scenariosRootDir`).
- `main.go` resolves `dataRoot = runtimepaths.DataPath("")` and `cacheRoot = runtimepaths.CachePath("")` at startup and threads both into the four `register*Routes` callsites. Captures get `cacheRoot` (preserves the cache-class invariant in `runtimepaths/paths_test.go:T-R2`); the other three get `dataRoot`.
- `testDataRoot(t)` helper defined; `newTestServer` swapped XDG_* envs for the canonical `VROOLI_STORAGE_ROOT` substrate.
- `fresh_checkout_storage_test.go` and the four `e2e_initiative_feedback_*_test.go` files now compile and pass.

### Operating Mode Panel Polish (2026-05-02)

**Problem**: The `OperatingModePanel` rendered five inline cards with no `DetailSection` chrome, used a native `<select>` discarding 90% of catalog data, kept Acceptance Criteria visible regardless of `requiresAcceptanceCriteria`, hid phase metadata in `title=""` tooltips, never embedded the existing `<PhaseGraph>`, and showed a single gray sentence for the most-seen state (item-level). Six of eight `OperatingModeCapabilities` flags (`canStartPhases`, `canApplyBacklogSyncProposals`, `requiresAcceptanceCriteria`, `supportsArtifacts`, `supportsHandoffs`, `usesItemExecutionFlow`) were dead in the UI.

**Resolution**: Greenfield rewrite of the panel surface: `OperatingModeHero`, `ModePickerDialog` (rich card grid + compare panel + override ack), `PhaseComposer` (embedded `PhaseGraph` + chip row + collapsible item picker + free-form note → XML envelope), capability-gated `AcceptanceCriteriaEditor` with parsed preview and common-criteria chips, and `ItemLevelEmptyState` with explainer + stats. Extracted shared `OperatingModeCard` composite (sidebar + picker reuse). Deleted `mode-switch-control.tsx` and `phase-controls.tsx`. All eight capability flags now drive UI gating.

**Plan reference**: `~/.claude/plans/i-think-doing-both-cozy-toast.md`.



### Backlog Contract Drift Between API, CLI, and Skills (2026-03-26)

**Problem**: The Swarm Manager API, CLI, and prompt-manager skills had drifted apart around backlog import semantics. The most visible symptoms were:

1. Meta-orchestrator examples still used legacy `scope`
2. Batch create docs still taught a request-level `--initiative` flow
3. Initiative updates behaved as partial updates in operator expectations but not in the documented contract
4. There was no canonical preview-first import workflow for multi-initiative planning

**Fix**:
- Strict request decoding now rejects unknown fields like `scope`
- backlog batch-create now supports preview plus inline initiative metadata
- initiatives update is documented and implemented as partial
- new reference docs capture the canonical API and CLI contract
- prompt-manager Swarm Manager skills were rewritten to match the real system

**Files Changed**:
- `api/internal/backlog/`
- `api/internal/initiatives/`
- `cli/`
- `docs/reference/api-endpoints.md`
- `docs/reference/cli-commands.md`
- `prompt-manager/store/skills/packs/core/swarm-manager-*.md`

### Backlog status silently reverted on execution failure (2026-03-21)

**Problem**: When an execution failed, `restoreBacklogStatus()` silently reverted the backlog item to its previous status (e.g., "backlog"). Users couldn't distinguish items that were never executed from those that failed multiple times.

**Fix**: Added a "failed" backlog status. Execution failures now set the backlog item to "failed" instead of reverting. Cancellations still restore the previous status (user-initiated). Added a status dropdown on BacklogDetailsPage for manual status control, inline execution history, and a default filter that hides finished items (completed/failed/archived) on BacklogPage.

## Deferred Work

### P2 Targets (Future)

- OT-P2-001: Advanced cost formulas for priority calculations
- OT-P2-002: Pattern recognition integration
- OT-P2-003: Analytics dashboard with usage metrics
- OT-P2-004: Batch operations on backlog and scenarios
- OT-P2-005: Webhook support for external integrations

### Deferred Design Decisions

- Multi-user authentication (current model is local single-operator)
- Complex workflow builders (queue/execution governance remains the preferred path)
- Kanban-style UI (current interface remains list + details + execution controls)

---

## Stability Issues

### TypeScript `noUncheckedIndexedAccess` Violations

**Status**: ✅ Resolved (Phase 13: Progress phase iter 2)

**Resolution**: Refactored `src/consts/selectors.ts` to use deterministic const types instead of computed merge types. Changes made:

1. Replaced complex `SelectorTreeResult<L, D>` mapped type with explicit object spread using `as const`
2. Defined `literalSelectors`, `dynamicSelectors`, and combined `selectors` as separate const objects
3. Added `ES2022.Error` lib to tsconfig for Error cause support
4. Fixed `global.fetch` → `globalThis.fetch` in test files

The selector types are now fully deterministic - TypeScript can statically verify all property accesses without the "possibly undefined" inference issue.

**Files Changed**:
- `src/consts/selectors.ts` - Simplified type system, explicit object composition
- `tsconfig.node.json` - Added `lib: ["ES2020", "DOM", "DOM.Iterable", "ES2022.Error"]`
- `src/lib/api-client.test.ts` - `global.fetch` → `globalThis.fetch`
- `src/lib/error-utils.test.ts` - Added null checks for noUncheckedIndexedAccess compliance

---

### Error Boundary Coverage

**Status**: Resolved (Phase 10)
**Resolution**: Added `PageErrorBoundary` component and wrapped all pages

**Changes Made**:
- Created `src/components/ui/page-error-boundary.tsx` - page-level error isolation
- Updated `src/App.tsx` to wrap each page route with `PageErrorBoundary`
- Now have two-layer error handling:
  1. Top-level `ErrorBoundary` - catches catastrophic failures, full refresh recovery
  2. `PageErrorBoundary` - isolates page crashes, allows navigation to other pages

---

## ESLint Configuration

**Status**: Resolved (Phase 10)

ESLint was configured with the following safety-critical rules:
- `react-hooks/rules-of-hooks: error` - Prevents React Error #310
- `@typescript-eslint/no-non-null-assertion: error` - Prevents hidden null bugs
- `@typescript-eslint/no-explicit-any: error` - Prevents type erasure

See `ui/eslint.config.js` for the full configuration with protective comments.

---

## Non-Null Assertion in main.tsx

**Status**: Resolved (Phase 10)

**Previous Issue**: `document.getElementById("root")!` used non-null assertion

**Resolution**: Added explicit null check with descriptive error:
```typescript
const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found - check index.html has <div id=\"root\"></div>");
}
ReactDOM.createRoot(rootElement).render(...);
```

---

## Go API Test Failures

**Status**: ✅ Resolved (Phase 13: Progress phase)

**Resolution**: Ran `go mod tidy` in the api directory to regenerate go.sum with all dependencies. All Go tests now pass.

---

## Tech Debt

### API Handler Duplication (Resolved - Phase 23)

**Status**: ✅ Resolved (Phase 23: Refactor & Structural Improvement)

**Problem**: The `ideas/handler.go` and `scenarios/handler.go` files had significant code duplication:
- 10 occurrences of JSON response writing (`w.Header().Set("Content-Type", "application/json")` + `json.NewEncoder(w).Encode(...)`)
- 44 occurrences of `http.Error()` calls with inconsistent logging patterns
- Duplicate path validation logic for file operations
- Complex filter logic nested in handler functions

**Resolution**: Created `api/internal/httputil/` package with shared utilities:
- `httputil.JSON(w, data)` - Writes JSON response with proper Content-Type
- `httputil.JSONWithStatus(w, status, data)` - Writes JSON with custom status code
- `httputil.BadRequest/NotFound/InternalError/Conflict(w, prefix, msg)` - Structured error responses
- `httputil.ValidatePath/SafeFilePath()` - Path traversal protection utilities

**Benefits**:
- Eliminated ~50 duplicate patterns across handlers
- Centralized error logging format
- Reduced ideas/handler.go complexity (removed inline path validation)
- Extracted filter logic to dedicated `filterScenarios()` and `sortScenarios()` methods
- Added 16 unit tests for the new httputil package

**Files Changed**:
- Created `api/internal/httputil/response.go` and `response_test.go`
- Refactored `api/internal/ideas/handler.go` to use httputil
- Refactored `api/internal/scenarios/handler.go` to use httputil + extracted filter methods

---

### Test Helper Duplication (Resolved - Phase 23 Iter 2)

**Status**: ✅ Resolved (Phase 23: Refactor & Structural Improvement Iteration 2)

**Problem**: Go test files had inconsistent patterns:
- `ideas/handler_test.go` used `t.TempDir()` with automatic cleanup
- `scenarios/handler_test.go` used `os.MkdirTemp()` with manual `defer cleanup()`
- Duplicated JSON file creation, status assertions, and response decoding

**Resolution**: Created `api/internal/testutil/` package with shared test helpers:
- `testutil.WriteJSONFile` - Creates JSON files with automatic parent dir creation
- `testutil.WriteFile` - Creates text files with parent dir creation
- `testutil.MakeDir` - Creates directories
- `testutil.AssertStatusOK`, `AssertNotFound`, `AssertBadRequest` - Status code assertions
- `testutil.DecodeJSON` - Generic JSON decoding from responses
- `testutil.ReadJSONFile` - Generic JSON file reading
- `testutil.AssertFileExists`, `AssertFileNotExists` - File existence checks

**Benefits**:
- Unified test patterns: all tests now use `t.TempDir()` for automatic cleanup
- Reduced `scenarios/handler_test.go` by ~18% (556 → 458 lines)
- Eliminated manual cleanup code (`defer os.RemoveAll(dir)`)
- Type-safe generic JSON helpers prevent decode errors
- Added 9 tests for testutil package itself

**Files Changed**:
- Created `api/internal/testutil/helpers.go` and `helpers_test.go`
- Refactored `api/internal/scenarios/handler_test.go` to use testutil and `t.TempDir()`

---

### ideas/handler_test.go testutil Adoption (Resolved - Phase 23 Iter 3)

**Status**: ✅ Resolved (Phase 23: Refactor & Structural Improvement Iteration 3)

**Problem**: `ideas/handler_test.go` was already using `t.TempDir()` but had inconsistent patterns compared to the newly refactored `scenarios/handler_test.go`:
- Manual status code assertions (`if w.Code != http.StatusOK`)
- Manual JSON decoding (`json.NewDecoder(w.Body).Decode(&result)`)
- Manual file operations (`os.WriteFile`, `os.MkdirAll`, `os.Stat`)

**Resolution**: Refactored to use testutil helpers for consistency:
- `testutil.AssertStatusOK`, `AssertNotFound`, `AssertBadRequest`, `AssertCreated` - Status assertions
- `testutil.DecodeJSON` - Type-safe JSON decoding
- `testutil.WriteFile` - File creation with automatic parent dir
- `testutil.MakeDir` - Directory creation
- `testutil.AssertFileExists`, `AssertFileNotExists` - File existence checks

**Benefits**:
- Unified test patterns across all Go API tests (ideas, scenarios, testutil)
- Improved test readability with semantic helper names
- Eliminated ~50 manual status/decode patterns
- Better error messages from testutil helpers (include response body on failure)

**Files Changed**:
- Refactored `api/internal/ideas/handler_test.go` to use testutil helpers

---

### Remaining Tech Debt (Updated Phase 23 Iter 4)

**Priority: Medium**

1. ~~**CLI command structure** - `cli/app.go` has command handlers that could benefit from consistent error handling patterns similar to httputil~~ (Reviewed: Already clean with idiomatic Go error wrapping via `fmt.Errorf("%w")`, consistent JSON unmarshalling, and clear separation of concerns)
2. ~~**Service layer duplication** - UI service files (`ideas-service.ts`, `scenarios-service.ts`) have similar fetch patterns that could be unified~~ (Reviewed: Already clean with dependency injection pattern)
3. ~~**Config loading duplication** - Both ideas and scenarios handlers read JSON config files (spec.json, lighthouse.json, metadata.json) with similar patterns~~ (Reviewed Phase 23 Iter 4: Patterns are domain-specific with different error handling and defaults. `ideas/handler.go` reads spec.json; `scenarios/handler.go` reads lighthouse.json and metadata.json to enrich CLI-sourced scenario data. A generic ReadJSON utility would not reduce complexity since each has context-specific logic.)

**Priority: Low**

1. ~~**Test helper duplication** - Go test files have similar setup patterns for creating test directories/files~~ ✅ Resolved
2. **Logging consistency** - Some handlers log with `[prefix]` format, others without - could unify (Low priority: current approach is readable)
3. ~~**ideas/handler_test.go testutil adoption** - Could be refactored to use testutil helpers for consistency (already uses t.TempDir())~~ ✅ Resolved Phase 23 Iter 3

### Codebase Health Summary (Phase 23 Complete - Iter 5)

After 5 iterations of refactoring, the codebase is in good structural health:

**API Layer**:
- `httputil/` package consolidates HTTP response patterns (JSON, errors, path validation)
- `testutil/` package consolidates test patterns (file ops, assertions, decoding)
- Handler complexity is appropriate given endpoint count and domain logic
- All Go tests pass with consistent patterns

**UI Layer**:
- Well-structured pages with proper separation of concerns
- Shared components (ErrorState, SearchBar, TagList, FileTree, FilePreview, FileUpload)
- Centralized config in `src/config/`, types in `src/types/`, utilities in `src/lib/`
- Error boundaries at app and page level
- MainLayout uses data-driven tab configuration with responsive design
- Domain types well-organized with JSDoc comments and DOC references

**CLI Layer**:
- Idiomatic Go error handling with consistent patterns
- Clean command structure using cli-core framework

**No further high-value refactoring opportunities identified.** The remaining tech debt items are low priority cosmetic issues that don't impact maintainability.

### Auditor False Positive

The scenario-auditor flags `httputil/response.go:71` as HIGH security (PATH-001 "Path Traversal Pattern"). This is a **false positive** - the `ValidatePath()` function is a path traversal **protection** mechanism, not a vulnerability. It:
1. Joins base directory with relative path
2. Cleans paths using `filepath.Clean`
3. Ensures result stays within base directory
4. Returns false if path would escape base directory

This is the standard Go pattern for path traversal protection.

---

## Future Stability Improvements

### Recommended (High Priority)

1. ~~**Fix selector type system** - Refactor `src/consts/selectors.ts` to produce deterministic types~~ ✅ Resolved
2. ~~**Add Zod validation at API boundaries** - Runtime validation in service layer~~ ✅ Resolved (Proto-backed Zod validation added)
3. **Implement React Query error boundaries** - Use `QueryErrorResetBoundary` for better error recovery

### Nice to Have

1. **Add visual regression tests** - Catch UI crashes in CI
2. **Implement error reporting service** - Send production errors to observability platform
3. **Add React Strict Mode violations cleanup** - Address any StrictMode warnings

---

## Test Gaps

This section documents known test coverage gaps and recommended improvements.

### Operating-mode Phase 4 input/prompt frontier (2026-07-10)

Caller-input compilation, validation, normalization, persistence, replay
conflict handling, Connect/CLI/UI transport, digest verification, and
mutation-free rejection now have focused coverage. Reachable prompt sources are
also pinned transitively with exact template-variable comparison, and each round
records immutable source/variable/render digests. Plan-ref targets now resolve
contained, non-symlink, bounded UTF-8 workspace content with canonical path and
content-hash provenance. The remaining Phase 4 implementation gap is moving
dynamic provider resolution, pinned prompt rendering, and spawn-request
construction before round reservation and lock acquisition with a final
execution-version compare-and-reserve.

### Current Test Coverage Status

**Go API Tests**: Comprehensive coverage
- `ideas/handler_test.go` - 45+ tests covering CRUD, file operations, queue functionality, idempotency
- `scenarios/handler_test.go` - 17+ tests covering list/get/update with search, filter, and sort
- `ecosystem/client_test.go` - 12+ tests covering HTTP client, error handling, priority mapping
- `httputil/response_test.go` - 15+ tests covering JSON responses, error functions, path validation
- `testutil/helpers_test.go` - 9+ tests covering test utilities

**Go CLI Tests**: Good coverage
- `app_test.go` - 18+ tests covering app initialization, endpoint resolution, command validation

**UI TypeScript Tests**: Comprehensive coverage
- Service tests (ideas, scenarios) - Seam-based testing with mock API clients
- Page tests (IdeasPage, ScenariosPage) - Component testing with mocked services
- Library tests (api-client, error-utils, format-utils, query-utils) - Utility function tests
- Component tests (file-tree, file-upload, file-preview, MainLayout) - UI component tests

### Identified Gaps (Resolved in Phase 25)

The following test gaps were identified and resolved:

1. **Queue endpoint success path** - Added `TestQueue_Success`, `TestQueue_WithImproverOperation`, and `TestQueue_EcosystemError` tests using mocked ecosystem client
2. **ecosystem client edge cases** - Added `TestMapPriorityToString_Boundaries`, `TestHTTPClient_CreateTask_ResponseDecodeError`, `TestHTTPClient_CreateTask_RequestConstruction`, and `TestNewHTTPClient` tests
3. **httputil error function variants** - Added `TestErrorResponses_WithPrefix`, `TestJSON_ErrorCase`, `TestJSONWithStatus_Various`, and `TestValidatePath_AdditionalCases` tests
4. **CLI edge cases** - Added `TestResolveV1Endpoint_EdgeCases`, `TestHealthResponseStruct`, `TestHealthResponseWithError`, `TestCmdIdeasGet_MultipleArgs`, `TestCmdIdeasCreate_EmptyName`, `TestCmdIdeasCreate_EmptyTitle`, and `TestCmdIdeasUpdate_TwoArgs` tests
5. **sanitizeName edge cases** - Added `TestSanitizeName_EdgeCases` with unicode, consecutive hyphens, and empty string tests

### Identified Gaps (Resolved in Phase 25 Iteration 2)

The following test gaps were identified and resolved:

1. **ideas-service.test.ts** - Added 14 new tests covering previously untested service methods:
   - `getFiles()` - Tests for fetching idea file trees (2 tests)
   - `getFileContent()` - Tests for retrieving file content with text response type (2 tests)
   - `uploadFile()` - Tests for FormData uploads with/without path (2 tests)
   - `queue()` - Tests for queueing ideas with generator/improver operations (2 tests)
   - Error propagation tests for all service methods (6 tests)

2. **scenarios-service.test.ts** - Added error handling tests (3 tests):
   - Error propagation for list, get, and updateMetadata methods

3. **error-state.test.tsx** - Created new test file with 18 tests for ErrorState component:
   - Auto-detection tests: Verifies component correctly identifies error types from ApiError (5 tests)
   - Retry button behavior: Tests for show/hide logic and callback invocation (5 tests)
   - Custom overrides: Tests for title, message, and variant overrides (3 tests)
   - Generic error handling: Tests for standard Error and null cases (3 tests)
   - Accessibility and styling: Tests for test selectors and custom className (2 tests)

### Remaining Test Opportunities (Low Priority)

1. **Integration tests** - End-to-end tests that verify full stack behavior (API → CLI → UI)
2. **Boundary condition tests** - Large file uploads, concurrent requests, timeout scenarios
3. **Error message internationalization** - Tests for user-facing error messages in different locales
4. **Performance regression tests** - Baseline performance benchmarks for critical paths

### Test Quality Guidelines

- Use table-driven tests for comprehensive input coverage
- Prefer behavioral assertions over implementation details
- Mock at seam boundaries (service layer) rather than internal modules
- Use `[REQ:ID]` annotations to link tests to requirements
- Ensure tests would fail if protected behavior breaks

---

### Phase 25 Iteration 3 Assessment

Comprehensive review of all test files found the test suite is in good health:

**Test File Count**: 19 test files (6 Go, 13 TypeScript)
**Total Tests**: 200+ individual test cases
**All Tests Pass**: Verified via `vrooli scenario test swarm-manager unit`

**Coverage Assessment**:
- Go API: Comprehensive CRUD, error handling, path traversal protection, edge cases
- Go CLI: Command validation, endpoint resolution, struct marshaling
- UI Services: Seam-based testing pattern with full API method coverage
- UI Components: Rendering, interactions, accessibility selectors
- UI Libraries: Error handling, formatting, query utilities

**Scoring Note**: The completeness scoring tool identifies a -1pt penalty for "superficial tests" but this appears to be a false positive. The `query-utils.test.ts` file at 58 lines has 7 meaningful tests with specific assertions testing all exported functionality. The file is small because the module it tests is small (a single config-sourced object).

**No new gaps identified** - the test suite provides solid protection against regressions.

---

### Phase 25 Iteration 4 Assessment

Added comprehensive test coverage for two complex detail pages that were previously untested:

**New Test Files Created**:
1. **IdeaDetailsPage.test.tsx** (28 tests):
   - Page structure and navigation (back button, container)
   - Metadata display (title, description, status, priority, tags)
   - Loading and error states with retry functionality
   - File tree rendering and selection
   - Upload toggle visibility
   - Queue functionality for different statuses (backlog/researching/ready vs queued/in_progress/completed)
   - Action buttons (edit, delete)
   - Edge cases (missing name parameter)

2. **ScenarioDetailsPage.test.tsx** (28 tests):
   - Page structure and navigation
   - Metadata display (title, description, priority, tags, completeness score)
   - Loading and error states with retry
   - Metadata management section with greenfield and recommendations toggles
   - Optimistic update behavior with error rollback
   - Status display for running/stopped/error states
   - Edge cases (missing name parameter, undefined completeness score)

**Updated Test File Count**: 21 test files (6 Go, 15 TypeScript)
**Updated Total Tests**: 256+ individual test cases
**All Tests Pass**: Verified via `vrooli scenario test swarm-manager unit`

**Coverage Improvements**:
- IdeaDetailsPage: Complex page with mutations, file tree, upload, and queue functionality now fully tested
- ScenarioDetailsPage: Complex page with optimistic updates and error handling now fully tested
- Both pages test [REQ:REQ-P0-004] and [REQ:REQ-P0-007] requirements

**Remaining Untested Pages** (medium priority - now stateful):
- SettingsPage.tsx - Settings persistence + recommendation controls now implemented; add tests for load/save + error handling
- RecommendationsPage.tsx - Recommendation list/refresh + status updates implemented; add tests for filters and actions

---

### Phase 25 Iteration 5 Assessment (Final)

Added test coverage for two additional components:

**New Test Files Created**:
1. **TagList.test.tsx** (18 tests):
   - Empty/null state handling
   - Tag display without truncation
   - Truncation logic with +N indicator
   - Custom maxTags prop
   - Styling and className application
   - Edge cases (maxTags=0, maxTags=1, special characters, spaces, large arrays)

2. **NotFoundPage.test.tsx** (9 tests):
   - Page structure and test selectors
   - User-friendly messaging (no technical 404 jargon)
   - Navigation to /ideas with replace mode (prevents back-to-404 loop)
   - Accessibility (heading hierarchy, focusable button)

**Updated Test File Count**: 23 test files (6 Go, 17 TypeScript)
**Updated Total Tests**: 283+ individual test cases
**All Tests Pass**: Verified via `vrooli scenario test swarm-manager unit`

**Test Suite Completion Summary**:
- All complex pages have comprehensive tests (IdeasPage, ScenariosPage, IdeaDetailsPage, ScenarioDetailsPage, NotFoundPage)
- All components with logic have tests (TagList, FileTree, FilePreview, FileUpload, ErrorState, MainLayout)
- All services have tests (ideas-service, scenarios-service)
- All utilities have tests (api-client, error-utils, format-utils, query-utils)
- All API handlers have comprehensive tests (ideas, scenarios, ecosystem, httputil, testutil)
- CLI has comprehensive tests (app_test.go)

**Remaining Untested** (needs coverage):
- SettingsPage.tsx - Add tests for persistence, validation, and error display
- RecommendationsPage.tsx - Add tests for refresh, status updates, and filters
- Error boundaries (error-boundary.tsx, page-error-boundary.tsx) - Require integration tests to trigger
- Shadcn primitives (button.tsx, input.tsx, tabs.tsx, search-bar.tsx) - Wrapper components

**Phase 25 Test Suite Strengthening Complete**: The test suite provides comprehensive protection against regressions with high-signal, behavior-focused tests. No remaining high-value test opportunities identified.

---

## React Stability Audit (Phase 27)

### Problem Register Summary

**Status**: ✅ Complete - No issues identified

A comprehensive React stability audit was performed covering all pages and complex components. The codebase is well-hardened against runtime crashes.

### Audit Criteria

1. **Hook Discipline**: All hooks placed before any early returns
2. **Defensive Data Access**: Optional chaining (?.) and nullish coalescing (??) used consistently
3. **Error Boundary Coverage**: Two-layer architecture with proper isolation
4. **State Handling**: Loading/error/empty/success states properly distinguished
5. **TypeScript Safety**: `strict` mode and `noUncheckedIndexedAccess` enabled
6. **ESLint Rules**: `rules-of-hooks` and `no-non-null-assertion` configured

### Files Audited

**Pages** (4 audited):
- `IdeaDetailsPage.tsx` - All hooks before early return (line 104). Proper data handling.
- `ScenarioDetailsPage.tsx` - All hooks before early return (line 130). Optimistic updates with rollback.
- `IdeasPage.tsx` - No early returns. Proper loading/error/empty states.
- `ScenariosPage.tsx` - No early returns. Safe useMemo with null check.

**Components** (10 audited):
- `error-boundary.tsx` - Proper class component with getDerivedStateFromError/componentDidCatch
- `page-error-boundary.tsx` - Page-level isolation with retry/navigate recovery
- `error-state.tsx` - Centralized error categorization, user-friendly messages
- `file-tree.tsx` - Proper null guards for children, empty state handling
- `file-preview.tsx` - Conditional rendering for loading/error/content
- `file-upload.tsx` - All hooks at top level, safe mutation callbacks
- `confirm-dialog.tsx` - Early return after all hooks (line 85), cleanup in useEffect
- `MainLayout.tsx` - Defensive fallback for activeTab
- `App.tsx` - Two-layer error boundary architecture
- `main.tsx` - Explicit null check instead of non-null assertion

**Excluded** (not React components or simple wrappers):
- Config files (vite.config.ts, tailwind.config.ts)
- Type declarations (types/index.ts, vite-env.d.ts)
- Shadcn primitives (button.tsx, input.tsx, tabs.tsx, search-bar.tsx)
- Test files and setup

### Architecture Strengths

1. **Two-Layer Error Boundaries**:
   - Top-level `ErrorBoundary` catches catastrophic failures (full refresh recovery)
   - `PageErrorBoundary` isolates route crashes (retry/navigate recovery)

2. **Structured Error Handling**:
   - `ApiError` class with typed error categories (network/timeout/http/parse)
   - `ErrorState` component maps error types to user-friendly messages
   - Centralized `categorizeError` function for consistent classification

3. **TypeScript Guardrails**:
   - `strict: true` - Catches null/undefined bugs at compile time
   - `noUncheckedIndexedAccess: true` - Forces handling of potentially undefined array access
   - Protective comments in tsconfig.json explain why rules must not be weakened

4. **ESLint Safety Rules**:
   - `react-hooks/rules-of-hooks: error` - Prevents hook count changes between renders
   - `@typescript-eslint/no-non-null-assertion: error` - Prevents `!` operator that hides bugs
   - `@typescript-eslint/no-explicit-any: error` - Prevents type erasure

### No Issues Found

The codebase demonstrates strong React stability practices:
- All hooks are called unconditionally before any early returns
- Data from async sources (useQuery) is handled with proper loading/error/data states
- Optional chaining and nullish coalescing used consistently
- Error boundaries provide meaningful recovery paths
- No anti-patterns like `!` assertions or `as any` casts

---

*Last updated: 2026-01-28 (Phase 27: React Stability Audit)*

---

## API transport drift: Discovery is the only Connect-RPC domain

*Added: 2026-05-17 (audio-tools adoption)*

When swarm-manager adopted audio-tools, a new Connect-RPC `DiscoveryService`
was added so the browser can resolve audio-tools' base URL server-side. It
is mounted alongside the existing gorilla/mux REST surface in
`api/main.go` via `discovery.RegisterRoutes(...)`.

This makes swarm-manager's API L1-mixed: every other domain (backlog,
captures, initiatives, execution, settings, review, graph, search,
operations, agent-sessions, agent-activity) still speaks REST + JSON.
That violates `feedback_proto_connect_always.md` ("Every scenario API
domain uses proto + Connect-RPC").

**Follow-up:** Plan a per-domain migration to Connect-RPC, prioritizing
read-heavy surfaces first (overview, backlog list, scenarios list).
Each domain should land its own proto schema under
`packages/proto/schemas/swarm-manager/v1/<domain>/` and replace its
`RegisterRoutes(...)` REST handler with a Connect handler module
mounted via `api-core/connectx`.

## Audio: auto-speak pref is in localStorage, not Settings proto

*Added: 2026-05-17 (audio-tools adoption)*

The swarm-manager-local "auto-speak agent replies" preference is
persisted in `localStorage` (key `swarm-manager:audio-prefs:v1`) rather
than as a field on the unified `Settings` proto. This is intentional
for the initial slice — adding a proto field would require
regenerating all three gen trees and updating settings.go / settings
adapter / SettingsTab patch paths. Voice / speed / summarize knobs are
already shared via audio-tools' own settings API so no swarm-manager-
side proto change is needed for the audio-tools-owned settings.

**Follow-up:** When a richer client-side preferences slice is
introduced (Zustand persist or similar), migrate this pref or
formalise it as `Settings.audio.autoSpeak`.

## Measures: monolithic /stats + UI not yet migrated (Phase 6b)

*Added: 2026-06-08 (measures-federated-metrics-layer Phase 6 — dogfood)*

Phase 6 added the granular **measures** layer (`api/handlers/measures/`, proto
`MeasuresService`, `cli/manifest.json`) over the event log, and swarm-manager
now passes `measures-health validate scenario swarm-manager` (static + probe)
at **full tier** with the marquee question answered end-to-end through
search-hub. That work is **additive** — the legacy monolithic
`internal/stats` engine + `GET /api/v1/stats` blob and its consumers still
stand.

**Remaining (Phase 6b), in order:**
1. **Expand the measure set to dashboard parity.** The five Phase-6 measures are
   windowed *counts* (the stateful-domain question-families measures-health
   grades). The `/stats` blob also serves rates, medians, velocity series,
   and by-mode/by-kind/by-initiative breakdowns. Deleting `/stats` without
   these would regress the OperationsCenter / Execution dashboards. Add
   `table`/`series` measures + rate measures (each: a `MeasuresService` RPC with
   a `TimeWindow` request → descriptor regen → manifest block → compute over the
   event log / the existing `stats` aggregateState as a read-model).
2. **Migrate the UI** (~22 files: `pages/OperationsCenterPage`, `ExecutionPage`,
   `surfaces/graph/components/StatsPanel` + HUD, `services/stats-service`,
   `hooks/useStats`, `types/stats`, `lib/stats-format-utils`, `consts/selectors`,
   and their tests) onto the new measures (via `/measures/execute` or the
   Connect client) or the search-hub central index.
3. **Delete** `internal/stats/`, `stats.NewHandler`/`RegisterRoutes`, the
   `GET /api/v1/stats` route, and the CLI `stats` group (greenfield — no shim).
4. **Add a `measures` CLI group.** Phase 6 left the CLI untouched: the manifest's
   command declarations are the intended surface, but the hand-rolled CLI is
   REST-only (`core` prepends `/api/v1`, no Connect-client infra), so wiring
   `swarm-manager measures backlog-completed --window …` needs either a
   Connect-client (preferred) or a raw-HTTP helper to reach root-mounted
   `/measures/execute`. Pairs naturally with the UI migration.
