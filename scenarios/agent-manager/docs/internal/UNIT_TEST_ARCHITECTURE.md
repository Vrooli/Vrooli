# Agent Manager Unit Test Architecture

## Purpose

This document records the current Agent Manager test architecture and the first migration path toward shared test utilities. It complements `docs/internal/SEAMS.md` and `docs/internal/UTILS_UNIFICATION_NOTES.md`.

## Current Test Organization

- Go API and CLI tests: 95 `*_test.go` files under `api` and `cli`.
- UI tests: 22 TypeScript test files outside `node_modules` and `.test-dist`.
- Backend shared utilities currently live in `api/internal/testutil`, which started as a SQLite helper package.
- UI tests now run through Vitest with jsdom and Testing Library setup configured in `ui/vite.config.ts`.
- UI test-only helpers live under `ui/tests/testutil` for the current pure TypeScript tests and `ui/src/test-utils` for React component/hook tests. `runEvents.ts`, `runs.ts`, `tasks.ts`, and `stats.ts` are the canonical fixture factories for reducer/timeline/controller/stats tests, including summary, breakdown, time-series, and error-pattern stats responses.

## Helper And Mocking Status

The API has a meaningful set of interfaces, but some tests still define seam doubles inline. The initial audit found 27 inline `mock`, `stub`, or `fake` struct definitions in Agent Manager Go tests. After the shared-fake and utility migrations, the count is 8. Three prior audit hits in the OpenRouter provider tests were JSON response fixture DTOs rather than seam doubles, and the sandbox launcher httptest server was renamed from `mockSandbox` to `sandboxTestServer` because it is a protocol simulator rather than a reusable seam double.

High-value consolidation candidates:

- Pricing and runner codec tests: local same-package seam doubles that need case-by-case review before promotion because shared fakes importing the package under test can create Go import cycles. The pricing service provider double remains local for that reason.
- Runner core tests intentionally keep very local process/codec fakes because those model private runner-core contracts rather than broad scenario seams.

## Seam Inventory

Stable seams worth canonical test doubles:

- `sandbox.Provider`: workspace-sandbox lifecycle, diff, apply, validation, and exec behavior.
- `event.Store`: append/get/stream/count/delete for run event assertions.
- `repository.StatsRepository`: aggregation inputs and canned analytics output for HTTP and orchestration tests.
- `toolregistry.ToolProvider`: manifest and tool lookup tests.
- `runner.Runner`: already has production test helpers in the runner package; only add a shared fake if existing runner helpers cannot express a boundary test.
- Model registry and pricing providers: small seam-specific fakes are useful when tests need error knobs and call inspection.
- `toolexecution.Orchestrator`: shared fake for server executor tests, kept in a narrow `mocks/toolexecution` subpackage so the root mocks package does not import orchestration and create cycles.
- `runner.Launcher` and `runner.SandboxLauncherFactory`: shared fakes for protected-mode routing boundary tests.
- `runner.TranscriptParser`: shared transcript replay runner for recovery and restart-resume tests.
- `phases.ModelChainResolver`: shared fixed-chain resolver for model fallback tests.

## Shared Test Utility Contract

`api/internal/testutil` is the Agent Manager home for shared test-only helpers:

- `mocks/`: hand-written fakes for stable seams.
- `fixtures/`: domain object factories with explicit overrides.
- `assertx/`: domain-aware assertions that call `t.Helper()`.
- `httpx/`: HTTP doer and response helpers for package-neutral HTTP client tests.
- Existing DB helpers remain available through `SetupTestDB`, `SetupTestRepos`, and `SetupTestReposWithDB` while tests migrate incrementally.

Production code must not import `agent-manager/internal/testutil` or any child package. The `TestNoProductionImports` meta-test enforces that boundary.

## Prioritized Migration Order

1. Done: Establish package documentation and the production-import guard.
2. Done: Move duplicated tool provider test doubles to `testutil/mocks`.
3. Done: Add a fake stats repository and migrate handler-level stats tests.
4. Done: Add event store and sandbox provider fakes, then migrate finalize tests.
5. Done: Migrate the run executor sandbox fake after finalize coverage proves the shared fake has the needed call inspection and error knobs.
6. In progress: Add fixtures/assertions for repeated `Run`, `Task`, `AgentProfile`, sandbox config, run event, stats, HTTP status, and event-message assertions.
7. Done: Move recommendation allowlist and broadcaster test doubles to shared fakes and reuse the shared broadcaster in recommendation orchestration tests.
8. Done: Add a shared model prober fake and migrate model health tests.
9. Done: Migrate the sandbox CWD integration contract to the shared sandbox fake and add workspace-path call inspection.
10. Done: Add package-neutral `httpx` helpers and migrate prompt-manager HTTP client tests away from a local doer stub.
11. Done: Add a package-neutral fake orchestrator and migrate tool execution tests.
12. Done: Add shared runner launcher/factory fakes and migrate launcher selector tests to an external test package.
13. Done: Add shared transcript replay runner and model-chain resolver fakes, then migrate recovery, restart-resume, and execute fallback tests.
14. Done: Add `ui/tests/testutil/runEvents.ts` and migrate repeated `RunEvent` builders in run event store and timeline tests.
15. Done: Switch `pnpm test` from `tsc && node --test` to `vitest run`, add jsdom/Testing Library setup, add `src/test-utils/renderWithProviders.tsx`, and add initial `DiffViewer` render coverage.
16. In progress: Review pricing and runner codec doubles for consolidation boundaries. The pricing service provider double remains local because moving it to `testutil/mocks` would create an import cycle with `package pricing` tests. Be careful with same-package tests; moving those fakes into `testutil/mocks` can create import cycles when the fake must import the package under test.
17. Done: Add first `RunTimeline` React behavior tests for filter UI behavior and persisted filter restoration.
18. Done: Add `QuickRunDialog` React behavior tests for default custom-run submit/reset behavior and persisted task draft restoration.
19. Done: Add hook-level `useWebSocket` tests for queued subscription replay, reconnect subscription replay, and normalized server message delivery.
20. Done: Add `KPISummary` React behavior tests for formatted stats metrics, selected-window throughput, loading state, and error state.
21. Done: Add `ModelUsageBreakdown` and `ToolUsageAnalytics` React behavior tests for chart-to-detail selection, run/model detail rendering, empty states, and error states.
22. Done: Add `RunnerPerformanceTable` and `ProfileActivityTable` React behavior tests for formatted metrics, visible sorting controls, profile links, loading states, empty states, and error states.
23. Done: Add `RunStatusTrends` and `CostDurationTrends` React behavior tests for populated chart series, empty states, loading states, and error states using lightweight chart mocks.
24. Done: Add `ErrorAnalysisSection` and `ExportButton` React behavior tests for error totals, sample-run links, generated CSV content, disabled state, loading states, empty states, and error states.
25. Done: Add `useSelectedRunController` hook coverage for REST event gap-fill into `useRunEventStore`, including sorted event exposure and WebSocket subscribe/unsubscribe wiring.
26. Next slices: move to broader boundary coverage from Phase 6, especially realtime event contract symmetry and sandbox mode routing tests.

## Prohibited Patterns

- Do not import test utilities from production Go files.
- Do not introduce generated mocks for this pass.
- Do not hide relevant behavior behind broad assertion helpers; assertions should name domain expectations.
- Do not move private one-off fakes into shared utilities unless at least two call sites need the same seam.
- Do not add UI test dependencies without explicit approval.
