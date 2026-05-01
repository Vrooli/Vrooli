# Agent Manager Test Architecture Improvement Plan

## Purpose

Define a concrete, handoff-ready plan for improving Agent Manager's test architecture using the recent Workspace Sandbox testing improvements as the reference pattern. This plan focuses on test utilities, mock/fake unification, seam enforcement, UI test modernization, and coverage quality. It does not implement the changes.

## Required Reading

Future implementers should start with:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement utils-unification test unit-testing-architecture-steer decision-boundary-extraction boundary-of-responsibility-enforcement
```

Scenario context:

```bash
sed -n '1,260p' scenarios/agent-manager/docs/internal/SEAMS.md
sed -n '1,220p' scenarios/agent-manager/docs/internal/UTILS_UNIFICATION_NOTES.md
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/testutil.go
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/no_prod_import_test.go
find scenarios/agent-manager/api/internal/testutil -maxdepth 3 -type f | sort
find scenarios/workspace-sandbox/api/internal/testutil -maxdepth 4 -type f | sort
```

## Problem Statement

Agent Manager has substantial backend test coverage, but the test architecture has grown by accretion. The scenario currently has 95 Go test files across `api` and `cli`, plus 10 UI test files. The backend has useful seams documented in `docs/internal/SEAMS.md`, but many tests still define local mocks, fixtures, and assertions inline instead of using canonical test utilities. A search found 27 inline `mock/stub/fake` struct definitions in Agent Manager Go tests, while Workspace Sandbox has centralized most of that surface under `api/internal/testutil/mocks` and only one inline test double definition remains in `_test.go` files.

The UI tests are also narrower than the app surface suggests. Agent Manager's UI test runner compiles TypeScript with `tsc` and executes Node's built-in test runner against mostly pure utility tests. It does not use Vitest, jsdom, Testing Library, a shared setup file, or colocated component tests. Workspace Sandbox does, and its UI tests exercise React behavior directly.

## Scope

In scope:

- Add a structured Agent Manager API test utility package modeled on Workspace Sandbox's `testutil` pattern.
- Consolidate duplicated backend mocks/fakes for stable seams such as sandbox provider, event store, broadcaster, stats repository, tool provider, model/pricing providers, runners, and orchestration adapters.
- Add domain fixture factories for `Run`, `Task`, `AgentProfile`, sandbox config, run events, and stats objects.
- Add assertion helpers for run status/phase, emitted events, sandbox lifecycle calls, HTTP responses, and WebSocket filtering.
- Modernize UI tests so component and hook behavior can be covered with Vitest, jsdom, Testing Library, and a shared setup file.
- Document testing architecture decisions in `scenarios/agent-manager/docs/internal/UNIT_TEST_ARCHITECTURE.md`.

Out of scope:

- Product behavior changes.
- Rewriting all tests in one pass.
- Introducing generated mocks, gomock, mockery, or heavyweight dependency injection frameworks.
- Installing new dependencies without explicit approval. Vitest/jsdom/Testing Library adoption requires a dependency approval step if they are not already acceptable for this scenario.

## Current Technical Context

Current Agent Manager evidence:

- Backend shared test utilities are limited to SQLite setup in `scenarios/agent-manager/api/internal/testutil/testdb.go`.
- Inline backend test doubles appear in files such as:
  - `api/internal/orchestration/run_executor_test.go`: local `testBroadcaster`, `testFixtures`, and `mockSandboxProvider`.
  - `api/internal/orchestration/phases/finalize_test.go`: local `memEventStore` and `stubSandbox`.
  - `api/internal/handlers/stats_test.go`: local `stubStatsRepo`.
  - `api/internal/toolregistry/registry_test.go` and `api/internal/handlers/tools_test.go`: duplicated `mockToolProvider`.
- Existing production seams are good raw material: `runner.Runner`, `sandbox.Provider`, `event.Store`, repository interfaces, `toolregistry.ToolProvider`, `pricing.Provider`, `modelregistry.ModelProber`, and phase `Deps`.
- UI tests live under `ui/tests/**` and run via `pnpm test`, which expands to `tsc -p tsconfig.test.json` plus `node --test .test-dist/tests/**/*.test.js`.
- `ui/tsconfig.test.json` explicitly includes selected production utility files and `tests/**/*.test.ts`, which reinforces pure TypeScript utility testing but makes React component/hook tests awkward.

Workspace Sandbox reference pattern:

- `api/internal/testutil/testutil.go` documents test-only ownership, subpackages, and fake conventions.
- `api/internal/testutil/mocks/*` provides canonical fakes with constructors, sane defaults, inspection fields, and error knobs.
- `api/internal/testutil/fixtures/*` centralizes domain object creation.
- `api/internal/testutil/assertx/*` centralizes domain-aware assertions.
- `api/internal/testutil/no_prod_import_test.go` enforces that production code cannot import test utilities.
- UI tests are colocated under `ui/src/**` and use Vitest plus Testing Library.

Validation already run during this investigation:

```bash
cd scenarios/agent-manager/api && GOWORK=off go test ./... -run '^$'
cd scenarios/agent-manager/ui && pnpm test
```

Observed result: backend compile-only package check passed, and UI Node tests passed 58 tests.

## Target End State

Agent Manager should have a clear test architecture:

- `api/internal/testutil/` is the only backend home for shared test-only fixtures, fakes, assertions, and DB helpers.
- Production code cannot import `agent-manager/internal/testutil`.
- Tests use canonical fakes for major seams unless a tiny one-off fake is genuinely local to one test.
- Shared fixtures make default test data consistent and override-friendly.
- UI tests can cover pure utilities, hooks, and React components without custom DOM bootstrapping in every file.
- Test names and organization make responsibility boundaries obvious: domain, orchestration, adapters, handlers, UI components, UI reducers, and protocol utilities.
- Documentation explains where new test helpers belong and which patterns are prohibited.

## Implementation Strategy

### Phase 1: Baseline Audit and Documentation

Create `scenarios/agent-manager/docs/internal/UNIT_TEST_ARCHITECTURE.md` with:

- Test organization status for Go, CLI, and UI.
- Current helper/mocking status.
- High-value seam inventory.
- Red flags with file references.
- Prioritized migration order.

Run and record:

```bash
find scenarios/agent-manager/api scenarios/agent-manager/cli -name '*_test.go' | wc -l
find scenarios/agent-manager/ui \( -path '*/node_modules/*' -o -path '*/.test-dist/*' \) -prune -o \( -name '*.test.ts' -o -name '*.test.tsx' \) -print | wc -l
find scenarios/agent-manager/api -name '*_test.go' -print0 | xargs -0 rg -n "type (mock|stub|fake)\w+ struct"
```

Acceptance criteria:

- The architecture doc exists and points to concrete current files.
- It distinguishes one-off test doubles from consolidation candidates.
- No production or test behavior has changed.

### Phase 2: Establish Backend Testutil Structure

Extend `api/internal/testutil/` from DB-only to a documented package:

```text
api/internal/testutil/
  testutil.go
  no_prod_import_test.go
  db/sqlite.go or existing testdb.go kept with compatibility wrappers
  fixtures/
  mocks/
  assertx/
  httpx/
```

Keep existing `SetupTestDB`, `SetupTestRepos`, and `SetupTestReposWithDB` available initially so this phase does not force broad test rewrites. If moving to `db/`, leave thin wrappers until all call sites migrate in a later phase.

Acceptance criteria:

- `go test ./internal/testutil ./... -run TestNoProductionImports` passes.
- `no_prod_import_test.go` rejects non-test imports beginning with `agent-manager/internal/testutil`.
- Package docs state constructor/default/error-knob conventions for fakes.

### Phase 3: Canonical Backend Fakes for High-Churn Seams

Add hand-written fakes with constructors and compile-time interface guards:

- `mocks.FakeSandboxProvider` for `sandbox.Provider`.
- `mocks.FakeEventStore` for `event.Store`.
- `mocks.FakeBroadcaster` for orchestration/phase event broadcasting surfaces.
- `mocks.FakeStatsRepository` for `repository.StatsRepository`.
- `mocks.FakeToolProvider` for `toolregistry.ToolProvider`.
- `mocks.FakeRunner` only if existing `runner.MockRunner` does not cover the needed assertion/error knobs.

Initial migration targets:

- Replace `mockSandboxProvider` in `api/internal/orchestration/run_executor_test.go`.
- Replace `stubSandbox` and `memEventStore` in `api/internal/orchestration/phases/finalize_test.go`.
- Replace `stubStatsRepo` in `api/internal/handlers/stats_test.go`.
- Replace duplicated `mockToolProvider` in `api/internal/toolregistry/registry_test.go` and `api/internal/handlers/tools_test.go`.

Acceptance criteria:

- Inline mock/stub/fake struct count materially drops from the current 27.
- Migrated tests remain behavior-equivalent.
- Canonical fakes expose state copies under locks where concurrent tests inspect calls.
- Each fake has at least focused tests for defaults, error knobs, and interface compliance.

### Phase 4: Fixture and Assertion Unification

Add fixtures:

- `fixtures.NewAgentProfile(opts...)`
- `fixtures.NewTask(opts...)`
- `fixtures.NewRun(opts...)`
- `fixtures.NewSandboxConfig(opts...)`
- `fixtures.NewRunEvent(opts...)`
- `fixtures.NewStatsSnapshot(opts...)`

Add assertions:

- `assertx.RunStatus(t, run, want)`
- `assertx.RunPhase(t, run, want)`
- `assertx.EventMessageContains(t, events, substring)`
- `assertx.SandboxApplyRequest(t, got, want)`
- `assertx.HTTPStatus(t, recorder, want)`

Migrate the repeated fixture setup in orchestration and handler tests first, especially `newTestFixtures`, `newInPlaceFixtures`, and local HTTP response checks.

Acceptance criteria:

- Repeated default `Run`, `Task`, and `AgentProfile` literals are reduced in high-churn tests.
- Assertions call `t.Helper()`.
- Tests become shorter without hiding the behavior being asserted.

### Phase 5: UI Test Runner Modernization

Adopt the Workspace Sandbox UI testing shape for Agent Manager, subject to dependency approval:

- Add Vitest/jsdom/Testing Library dependencies if approved.
- Add `ui/src/test-setup.ts`.
- Move or add React behavior tests under `ui/src/**/*.test.tsx`.
- Keep pure utility tests either under `ui/tests/lib` temporarily or migrate them to colocated `ui/src/lib/*.test.ts`.
- Replace the custom `tsc && node --test` path with `vitest run`, or keep a temporary `test:node` script during migration.

Priority UI coverage:

- `useWebSocket` adapter behavior around reconnect/subscription replay using mocked WebSocket.
- `useRunEventStore` integration with REST gap-fill controller behavior.
- `RunTimeline` filtering/grouping UI using the already-tested pure timeline helpers.
- `QuickRunDialog` persistence and submit/reset flow, including localStorage cleanup.
- Stats/dashboard breakdown components that consume shared display/date/currency utilities.

Acceptance criteria:

- `pnpm test` runs UI tests through the chosen runner.
- React component tests can render with shared providers and cleanup.
- LocalStorage/WebSocket/window state is reset between tests.
- Existing 58 Node-test assertions are preserved or migrated with equivalent coverage.

### Phase 6: Boundary Coverage and Scenario-Level Gates

Add coverage around decision and responsibility boundaries that are currently high-risk:

- Sandbox mode routing: protected vs tracking vs fallback visibility.
- Apply-at-run-end lifecycle: success, failure, manual review, partial approval, cancelled context, and workspace-sandbox unreachable.
- Runner codec contracts: transcript parsing, pricing extraction, exit/error classification.
- Realtime event contract: backend filter plus UI reducer/protocol behavior remain symmetric.
- Recommendation worker: extractor, allowlist, repository, and event outputs are tested through seams rather than real Ollama.

Evaluate whether `scenarios/agent-manager/.vrooli/service.json` test lifecycle should keep starting the full scenario before `test-genie` or split fast unit checks from comprehensive live checks. Do not change lifecycle behavior until the unit architecture is stable.

Acceptance criteria:

- Fast local checks exist for unit-level behavior without requiring scenario startup.
- Comprehensive scenario test remains available through `cd scenarios/agent-manager && make test`.
- Boundary tests assert desired behavior, not current implementation quirks.

## Contract Decisions

- Test utilities are test-only. Production imports from `agent-manager/internal/testutil` are prohibited.
- Fakes are hand-written and local to Agent Manager. Do not introduce generated mocks in this pass.
- Fakes should model seam contracts, not internal implementation details.
- Shared fixtures should use option functions or explicit override structs, not global mutable defaults.
- UI tests should prefer user-visible behavior and stable data-testid hooks already present; avoid asserting Tailwind implementation details except where the styling rule is itself the contract.
- Keep Workspace Sandbox as an architecture reference, not a source of cross-scenario imports.

## Testing Plan

During implementation, run progressively:

```bash
cd scenarios/agent-manager/api && GOWORK=off go test ./internal/testutil ./internal/orchestration/... ./internal/handlers/... ./internal/toolregistry/...
cd scenarios/agent-manager/api && GOWORK=off go test ./...
cd scenarios/agent-manager/ui && pnpm test
cd scenarios/agent-manager && make test
```

For longer comprehensive runs, use the repository timeout guidance; `make test` may start services and execute `test-genie`.

## Rollout and Validation Checklist

- [ ] `UNIT_TEST_ARCHITECTURE.md` documents the baseline and target.
- [ ] Backend `testutil` package has docs and no-production-import enforcement.
- [ ] Canonical fakes cover sandbox, event store, broadcaster, stats repository, and tool provider.
- [ ] High-duplication tests migrate to canonical fakes and fixtures.
- [ ] UI runner decision is made with dependency approval if needed.
- [ ] React component/hook test setup exists before adding broad UI behavior tests.
- [ ] `GOWORK=off go test ./...` passes in `scenarios/agent-manager/api`.
- [ ] `pnpm test` passes in `scenarios/agent-manager/ui`.
- [ ] `cd scenarios/agent-manager && make test` passes or any environmental blocker is documented with exact output.

## Risks and Mitigations

- Risk: over-centralized fakes become too configurable and obscure test intent.
  Mitigation: one canonical fake per seam, with sane defaults and narrow error knobs.

- Risk: moving existing DB helpers breaks many tests at once.
  Mitigation: keep compatibility wrappers until call sites are migrated.

- Risk: UI dependency changes create package-manager churn.
  Mitigation: get explicit dependency approval before adding Vitest/jsdom/Testing Library to Agent Manager.

- Risk: shared fixtures hide important scenario-specific setup.
  Mitigation: fixtures provide defaults only; tests must still set behavior-relevant fields explicitly.

- Risk: lifecycle `make test` remains slow even after unit improvements.
  Mitigation: split fast unit validation from comprehensive scenario validation in docs first, then adjust lifecycle only after consensus.

## Non-goals and Prohibited Patterns

- Do not rewrite all tests purely for style.
- Do not remove meaningful coverage to reduce duplication counts.
- Do not import Workspace Sandbox test utilities from Agent Manager.
- Do not add broad abstraction layers to production code solely for tests.
- Do not add dependencies without explicit permission.
- Do not use direct scenario execution; use `make test` or `vrooli scenario test agent-manager` for scenario lifecycle tests.

## Definition of Done

- Agent Manager has documented test architecture in `docs/internal/UNIT_TEST_ARCHITECTURE.md`.
- Shared backend test utilities cover the main seams and are enforced as test-only.
- Inline mock/fake/stub duplication is substantially reduced in the initial high-value files.
- UI test infrastructure can exercise React behavior, not only pure TypeScript utilities.
- Existing behavior is preserved.
- Fast unit checks and comprehensive scenario checks are both documented and passing, or blocked only by documented environment issues.
