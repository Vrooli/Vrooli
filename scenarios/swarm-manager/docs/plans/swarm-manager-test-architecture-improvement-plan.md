# Swarm Manager Test Architecture Improvement Plan

## Purpose

Improve the Swarm Manager scenario test architecture using the same direction recently applied to `workspace-sandbox`: shared test utilities, clearer seams, lower warning noise, and stronger coverage around behavior boundaries instead of brittle implementation details.

This is a planning artifact only. It does not implement the changes.

## Required Reading

Run these before implementation:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement utils-unification test unit-testing-architecture-steer decision-boundary-extraction boundary-of-responsibility-enforcement react-coherence
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Read these files:

- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/internal/UTILS_UNIFICATION_NOTES.md`
- `scenarios/swarm-manager/docs/internal/COHERENCE-NOTES.md`
- `scenarios/workspace-sandbox/api/internal/testutil/testutil.go`
- `scenarios/workspace-sandbox/api/internal/testutil/no_prod_import_test.go`
- `scenarios/swarm-manager/api/internal/testutil/helpers.go`
- `scenarios/swarm-manager/ui/src/setupTests.ts`
- `scenarios/swarm-manager/ui/vite.config.ts`
- `scenarios/swarm-manager/.vrooli/testing.json`

## Problem Statement

Swarm Manager has a large and currently green test suite, but the support architecture has not kept pace with the scenario's size. The tests cover many surfaces, yet repeated local fakes, local factories, direct browser/API mocks, warning-heavy UI tests, and flat helper files make the suite harder to extend safely.

Baseline from this investigation:

- API: `196` Go test files across `41` internal packages; `go test ./...` passes.
- CLI: `12` Go test files; `go test ./...` passes.
- UI: `163` test files under `ui/src`; `pnpm test` passes `1808` tests with `1` skipped.
- API test support is only `api/internal/testutil/helpers.go` plus `mocks.go`.
- Workspace Sandbox has a richer pattern: `internal/testutil/{assertx,db,fixtures,httpx,mocks,services}` plus a `no_prod_import_test.go` guard.
- UI setup is minimal and only globally mocks Monaco; no canonical render wrapper, QueryClient factory, browser API fixture layer, API mock factory, or console-warning policy.
- UI test output is noisy despite passing: repeated React `act(...)` warnings, React Router future-flag warnings, API-base proxy metadata logs, query functions returning `undefined`, nested button/key warnings, and an expected context-throw test printing uncaught error stacks.

The main risk is not immediate failure. The risk is that future test additions will keep copying local fakes and warning-prone harnesses, making regressions harder to identify and the suite slower to reason about.

## Scope

In scope:

- Shared Go API test utility package structure.
- Shared CLI HTTP/env command harness.
- Shared React test utility layer.
- Incremental migration of high-churn tests to shared utilities.
- Tests that lock seams and decision boundaries for agent identity, agent sessions, graph projections, execution, backlog, feedback, and operating mode behavior.
- Documentation updates to Swarm Manager internal seams, utility notes, and coherence notes.

Out of scope:

- Product behavior changes.
- Adding dependencies without explicit permission.
- Rewriting the entire suite in one pass.
- Migrating tests just for style when no reliability, reuse, or coverage gain exists.
- Running scenario binaries directly; use `make test` or `vrooli scenario test swarm-manager`.

## Current Technical Context

### API

High-test-density packages:

- `api/internal/backlog`: `41` test files.
- `api/internal/execution`: `14` test files.
- `api/internal/feedback`: `11` test files.
- `api/internal/operatingmode`, `api/internal/settings`, `api/internal/graph`: `10` each.

Current shared utilities:

- `api/internal/testutil/helpers.go`: file helpers, JSON decode helpers, status assertions.
- `api/internal/testutil/mocks.go`: dispatch invalidators and an error response writer.

Observed local duplication candidates:

- Local fake/stub/mock types appear across backlog, feedback, execution, graph, initiatives, scenarios, operatingmode, agentactivity, agentmanager, and prompts tests.
- `httptest.NewServer` appears `79` times across API/CLI tests.
- `time.Sleep(` appears `24` times in API tests.
- `testing.T{}` is used in `3` helper self-tests and should be replaced with safer helper validation patterns.

### CLI

The CLI suite passes quickly, but HTTP test servers and `SWARM_MANAGER_API_BASE` setup are repeated across command tests. This is a good target for a small CLI-local test harness before adding more command coverage.

### UI

Test distribution is broad:

- `ui/src/lib`: `23` test files.
- `ui/src/components/backlog`: `23`.
- `ui/src/hooks`: `13`.
- `ui/src/surfaces/graph/lib`: `12`.
- `ui/src/surfaces/graph/components`: `10`.
- `ui/src/services`: `9`.
- `ui/src/pages`: `7`.

Current shared support:

- `ui/src/setupTests.ts`: Jest DOM + Monaco mock.
- `ui/src/surfaces/graph/test-helpers.ts`: graph-specific node/edge helpers.

Observed friction:

- `vi.mock` appears roughly `333` times in UI tests.
- `render(` appears roughly `528` times.
- `fireEvent` appears roughly `329` times, while `userEvent.setup` appears only `28` times.
- `screen.getByText` appears roughly `478` times, often making tests sensitive to copy rather than behavior.
- One skipped test remains in `components/ui/file-preview.test.tsx`.

## Target End State

Swarm Manager has a layered test architecture:

- Go API tests share canonical fixtures, fakes, HTTP helpers, assertion helpers, and service builders.
- CLI tests share one command harness for API server setup, env isolation, stdout/stderr capture, and request assertions.
- UI tests share one render harness with QueryClient, router, store resets, stable browser API mocks, and console warning enforcement.
- Tests exercise behavior at seams: API contracts, service orchestration, persistence boundaries, graph projection mapping, agent-manager integration, identity/provenance enrichment, and UI service/store boundaries.
- Warning-free test output becomes the norm. Expected errors are explicitly silenced inside narrowly scoped helpers.
- Test utility imports are guarded so production code cannot import test-only packages.

## Implementation Strategy

### Phase 0: Baseline and Guardrails

Capture the current state before changing tests:

```bash
cd scenarios/swarm-manager/api && go test ./...
cd scenarios/swarm-manager/cli && go test ./...
cd scenarios/swarm-manager/ui && pnpm test
cd scenarios/swarm-manager && make test
```

Add lightweight governance first:

- Add `api/internal/testutil/no_prod_import_test.go`, modeled on workspace-sandbox, adjusted to reject imports beginning with `swarm-manager/internal/testutil` from non-test Go files.
- Add the same guard for any future CLI `internal/testutil` package if created.
- Document the utility boundaries in `scenarios/swarm-manager/docs/internal/SEAMS.md`.

Acceptance:

- Existing API, CLI, and UI unit gates still pass.
- Production Go files cannot import testutil.

### Phase 1: Build the Go API Testutil Spine

Expand `api/internal/testutil` incrementally rather than creating one large helper file:

```text
api/internal/testutil/
├── assertx/       # domain assertions with useful diffs
├── fixtures/      # BacklogItem, Initiative, ExecutionRecord, AgentActivity, AgentSession builders
├── fsx/           # temp scenario/backlog directory builders
├── httpx/         # request/response helpers and handler harnesses
├── mocks/         # canonical hand-written fakes per seam
└── services/      # service builders that wire common fakes
```

Start with the highest-reuse seams:

- `dispatch.Invalidator` / `NodeDispatcher`.
- Agent Manager service/client fake.
- Agent activity store/service fake.
- Backlog store/list/create fake.
- Initiative store/list fake.
- Execution store/service fake.
- Prompt Manager client fake.
- Graph source fakes.

Do not move every fake at once. Pick one package cluster, migrate its duplicated fakes, and prove the pattern.

Acceptance:

- At least backlog, execution, feedback, and graph tests use shared fakes where those fakes replace duplicate local implementations.
- Local fake names remain only when they encode truly package-specific behavior.
- New fakes expose sane defaults, call recording, and per-method error knobs.

### Phase 2: Replace Timing and Polling Fragility

Target tests using `time.Sleep(` first. Introduce test seams for time, polling, and async drains where production code already has natural variation points.

Candidate areas:

- `api/initiative_review_trigger_test.go`
- `api/e2e_initiative_feedback_merge_test.go`
- `api/e2e_initiative_feedback_test.go`
- `api/graph_materialize_integration_test.go`
- `api/internal/aisearch/*_test.go`
- `api/internal/backlog/handler_create_test.go`

Preferred patterns:

- Inject clock/ticker/poller interfaces where production already coordinates time.
- Use channels or eventual assertions for emitted events instead of fixed sleeps.
- Keep real-time integration tests only where the test is explicitly validating timeout behavior.

Acceptance:

- Fixed sleeps are removed or justified with a comment naming the real-time behavior being validated.
- Polling tests can fail fast with useful diagnostics.

### Phase 3: Add CLI Harness

Create a small CLI test utility layer under `cli/internal/testutil` or `cli/testutil` depending on current package import constraints.

It should provide:

- `NewAPIServer(t, routes)` or similar for route-specific `httptest.Server` setup.
- `WithAPIBase(t, server.URL)` for `SWARM_MANAGER_API_BASE`.
- Request capture helpers for method/path/body/header assertions.
- Command execution helper that captures stdout/stderr and exit errors consistently.

Migrate command tests in small batches:

- `cmd_aisearch_test.go`
- `cmd_initiatives_context_test.go`
- `cmd_initiatives_feedback_test.go`
- `cmd_scenarios_fixes_test.go`
- Larger `app_test.go` sections last.

Acceptance:

- Command tests stop repeating server/env boilerplate.
- Identity-token forwarding tests still assert real headers.
- CLI tests remain fast and deterministic.

### Phase 4: Build the UI Test Utility Layer

Add `ui/src/test-utils/` with focused files:

```text
ui/src/test-utils/
├── render.tsx        # renderWithProviders, router + QueryClient harness
├── query.ts          # test QueryClient defaults: no retries, clean cache
├── factories.ts      # domain object builders
├── services.ts       # typed service mocks
├── browser.ts        # IndexedDB, ResizeObserver, matchMedia, WebSocket helpers
├── console.ts        # fail-on-unexpected-console with narrow allowlist
└── stores.ts         # reset Zustand/localStorage/sessionStorage state
```

Update `setupTests.ts` to own global browser mocks and expected noisy library setup. Avoid hiding real React warnings globally until tests have been migrated; start with a console policy that can run in audit mode, then make it fail on unexpected noise once hot spots are clean.

Render harness requirements:

- Router support with React Router future flags enabled where applicable.
- QueryClient with `retry: false`, deterministic cache cleanup, and no undefined query results.
- Optional route table for pages that navigate to graph/detail routes.
- Store reset and storage cleanup after each test.
- Explicit helpers for expected thrown-hook tests so context invariant tests do not print uncaught jsdom stacks.

Acceptance:

- New UI tests use `renderWithProviders` by default.
- Existing tests migrated from the noisiest files stop emitting `act(...)`, route, query, and expected-error noise.
- Browser API mocks are centralized, not hand-written per test.

### Phase 5: Migrate UI Hot Spots

Migrate high-noise/high-churn files first:

- `ui/src/pages/ScenarioDetailsPage.test.tsx`
- `ui/src/components/initiative/feedback-dialog.test.tsx`
- `ui/src/components/initiative/feedback-panel.test.tsx`
- `ui/src/pages/ScenariosPage.test.tsx`
- `ui/src/components/backlog/clarification-panel.test.tsx`
- `ui/src/components/review/follow-up-sheet.test.tsx`
- `ui/src/hooks/useCapturePolling.test.ts`
- `ui/src/surfaces/graph/components/SettingsDrawer.test.tsx`
- `ui/src/pages/ExecutionPage.test.tsx`
- `ui/src/contexts/BacklogDetailContext.test.tsx`

Migration rules:

- Prefer `userEvent` for user interactions.
- Keep `fireEvent` only for low-level DOM events that `userEvent` does not model well.
- Prefer roles, labels, and accessible names over broad text searches.
- Use factories for large domain objects.
- Mock at service or browser boundary, not arbitrary internal implementation details.

Acceptance:

- The skipped `file-preview` error-state test is either fixed or replaced by an equivalent passing assertion.
- UI suite passes without uncontrolled React warnings.
- Tests become shorter because setup moves to shared utilities, not because assertions are weakened.

### Phase 6: Add Boundary and Decision Tests

After shared utilities exist, add focused tests for durable decision boundaries:

- Agent identity/provenance: verified token, fail-open operator provenance, session enrichment by `run_id`, artifact links.
- Sandbox mode downstream client behavior: `SandboxConfig.Mode` is the single source of truth; do not reintroduce `RequiresSandbox`.
- Graph projection: focus validation, oneof payload mapping, cache key by lens/focus, operation lens behavior.
- Execution lifecycle: queue, retry, cancellation, finalization, review-trigger transitions.
- Feedback/review flows: stuck-round recovery, lock behavior, proposal extraction, apply/cancel idempotency.
- Operating mode: status transitions, mode registry, artifact output contracts.
- Proto/UI contract tests: generated proto JSON parsing and route/service payload compatibility.

Acceptance:

- Coverage improves at seams, not by duplicating implementation tests.
- Decision-boundary tests fail for the specific class of regression they protect.

### Phase 7: Documentation and Maintenance Tracking

Update:

- `scenarios/swarm-manager/docs/internal/SEAMS.md`: add test seams and ownership boundaries.
- `scenarios/swarm-manager/docs/internal/UTILS_UNIFICATION_NOTES.md`: replace stale Ideas-era utility notes with current test utility architecture.
- `scenarios/swarm-manager/docs/internal/COHERENCE-NOTES.md`: record UI test harness and React warning policy.

If this work is tracked as a recurring maintenance task, add/update an `AI_CHECK` marker in the relevant TypeScript test utility file according to `docs/ai-maintenance/README.md`.

## Contract Decisions

- Test utilities are test-only. Production code must not import `internal/testutil` or `ui/src/test-utils`.
- Shared fakes are hand-written. Do not add mock generation or new packages without explicit permission.
- Mock at public seams: service interfaces, HTTP/API client boundaries, browser APIs, Agent Manager/Prompt Manager clients, stores, and graph projection sources.
- Avoid tests that assert incidental class strings or arbitrary text unless the text is the user-facing contract.
- Console warnings/errors are test failures unless explicitly expected and locally silenced.
- Scenario lifecycle validation uses `cd scenarios/swarm-manager && make test` or `vrooli scenario test swarm-manager`; do not run scenario binaries directly.

## Testing Plan

Required validation during implementation:

```bash
cd scenarios/swarm-manager/api && go test ./...
cd scenarios/swarm-manager/cli && go test ./...
cd scenarios/swarm-manager/ui && pnpm test
cd scenarios/swarm-manager && make test
```

Useful audit commands:

```bash
find scenarios/swarm-manager/api -name '*_test.go' -type f | wc -l
find scenarios/swarm-manager/ui/src -name '*test.*' -type f | wc -l
rg 'time\.Sleep\(' scenarios/swarm-manager/api --glob '*_test.go'
rg 'type (fake|mock|stub)|type [A-Za-z0-9_]*(Fake|Mock|Stub)' scenarios/swarm-manager/api scenarios/swarm-manager/cli --glob '*_test.go'
rg 'fireEvent|screen\.getByText|container\.querySelector|it\.skip|vi\.mock' scenarios/swarm-manager/ui/src --glob '*.{test,spec}.{ts,tsx}'
rg 'swarm-manager/internal/testutil' scenarios/swarm-manager/api scenarios/swarm-manager/cli --glob '!**/*_test.go'
```

## Rollout / Validation Checklist

- [x] Baseline commands captured.
- [x] API no-production-testutil-import guard added.
- [x] API testutil package split into focused subpackages.
- [x] First package cluster migrated and reviewed before wider migration.
- [ ] Fixed sleeps removed or justified.
- [x] CLI harness introduced and initial high-duplication command tests migrated.
- [ ] UI test-utils introduced with render/query/browser/store/console helpers.
- [ ] UI high-noise files migrated.
- [ ] Skipped UI test resolved.
- [ ] Boundary tests added for provenance, sandbox mode, graph, execution, feedback, and operating mode.
- [ ] Internal docs updated.
- [ ] `make test` passes from `scenarios/swarm-manager`.

Progress notes:

- 2026-05-01: API testutil now includes reusable dispatch, scheduler, HTTP error writer, and Agent Manager spawner fakes. Backlog service and execution handler tests have started using the shared fakes.
- 2026-05-01: CLI API server/env harness now covers `ai-search`, `initiatives context`, `initiatives feedback`, and `scenarios fixes` command tests.

## Risks and Mitigations

- Risk: broad helper migration creates churn without value.
  Mitigation: migrate by package cluster and require a measurable reduction in duplicate setup or warning noise.

- Risk: shared fakes become too general and hard to understand.
  Mitigation: keep one fake per seam with explicit method knobs; keep package-specific behavior local.

- Risk: console-warning enforcement initially breaks many tests.
  Mitigation: start in audit mode, fix hot spots, then fail on unexpected warnings.

- Risk: UI render helper hides missing providers.
  Mitigation: make providers explicit options; keep low-level component tests free to use raw `render` when no app context is needed.

- Risk: scenario-level `make test` is slower than local unit gates.
  Mitigation: use unit gates during iteration and reserve `make test` for phase completion.

## Non-goals / Prohibited Patterns

- Do not install new packages without explicit permission.
- Do not weaken assertions to make migration easier.
- Do not delete meaningful tests because they are awkward to migrate.
- Do not add global warning suppression that hides real React, router, or query regressions.
- Do not use mass-update scripts to rewrite tests.
- Do not directly execute scenario binaries or bypass the scenario lifecycle system.

## Definition of Done

- API, CLI, UI, and scenario-level gates pass.
- `internal/testutil` and UI `test-utils` are documented and enforced as test-only.
- Duplicate local fakes and fixture builders are materially reduced in the highest-churn packages.
- Fixed sleeps are removed from tests except where explicitly justified.
- UI test output is clean or has a narrow, documented allowlist for expected warnings.
- The skipped UI test count is zero unless a skip has an issue/plan reference and explicit owner.
- New boundary tests protect the main decision seams: identity/provenance, sandbox mode, graph projection, execution lifecycle, feedback/review, and operating modes.
