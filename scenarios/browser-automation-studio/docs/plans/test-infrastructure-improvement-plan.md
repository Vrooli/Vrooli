# Browser Automation Studio Test Infrastructure Improvement Plan

## Purpose

Improve the browser-automation-studio scenario's test quality, maintainability, and coverage by applying the same testing-infrastructure lessons recently used in workspace-sandbox: canonical test utilities, explicit seams, stronger responsibility boundaries, and coverage growth focused on business-critical behavior instead of raw test count.

Implementation began on 2026-05-01. This file now tracks both the original plan and the current execution state.

## Required Reading

Before implementing, run:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement utils-unification test unit-testing-architecture-steer decision-boundary-extraction boundary-of-responsibility-enforcement react-coherence
```

Also inspect the adjacent reference implementation:

```bash
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/testutil.go
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/no_prod_import_test.go
find scenarios/workspace-sandbox/api/internal/testutil -maxdepth 3 -type f | sort
```

## Problem Statement

Browser Automation Studio has a large number of tests, but the infrastructure is uneven across its four major surfaces:

- Go API: 127 `_test.go` files, plus package-local mocks and a top-level `api/testutil` package.
- Playwright driver: 99 Jest test files and a `tests/helpers` directory.
- UI: 73 Vitest test files and a small `src/test-utils` layer.
- CLI: 7 Go test files.

The issue is not lack of tests. The issue is that test seams and utilities are not yet canonical enough to keep future tests small, reliable, and intent-focused. BAS has several local mock implementations, long test files, duplicated setup patterns, low UI coverage despite many tests, and mixed test runner contracts. Workspace-sandbox shows a stronger target shape: documented testutil ownership, canonical fakes per seam, subpackages for fixtures/assertions/http/db/mocks, and a meta-test that prevents production code from importing test-only helpers.

## Scope

In scope:

- Consolidate BAS test utilities into clearer, canonical layers.
- Add meta-tests that enforce test-only boundaries.
- Migrate duplicate or file-local mocks into shared fakes where the same seam recurs.
- Improve runner configuration, noisy test output, and coverage reporting.
- Add focused coverage around high-value gaps in API, driver, and UI.
- Update seam documentation after test seams are clarified.

Out of scope:

- Product feature changes.
- Broad rewrites of production code solely to make tests easier.
- Dependency installation without explicit approval.
- Replacing Jest/Vitest/Go test frameworks in this pass.
- Direct scenario execution outside lifecycle commands.

## Current Technical Context

Test entry points:

- Scenario lifecycle: `cd scenarios/browser-automation-studio && make test`
- Direct API package discovery ran successfully with `cd scenarios/browser-automation-studio/api && go test ./... -run '^$'`
- Playwright-driver unit suite ran successfully when probed: 89 suites and 1172 tests passed.
- UI tests are sharded through `scenarios/browser-automation-studio/ui/scripts/run-vitest.sh`, but the script currently runs only `stores`, `features-core`, `workflow-palette`, and `workflow-builder` even though `vite.config.ts` defines more projects.

Observed counts from discovery:

- API: 127 Go test files.
- Playwright driver: 99 Jest test files.
- UI: 73 Vitest test files.
- CLI: 7 Go test files.
- Large test files exist in API and UI, including `api/workflow/validator/validator_test.go` at 2265 lines, `api/services/export/render/video_encoder_test.go` at 1638 lines, `ui/src/stores/__tests__/entitlementStore.test.ts` at 866 lines, and `ui/src/stores/workflowStore.test.ts` at 833 lines.

Current coverage signals:

- UI existing coverage summary: 10.01% lines, 5.88% functions, 37.32% branches.
- Playwright-driver existing coverage summary: 66.64% lines, 52.5% functions, 47.74% branches.
- `.vrooli/testing.json` still uses low Go thresholds: warning 40%, error 30%.

Implemented on 2026-05-01:

- Added documented `api/internal/testutil` package shell and Go production-import boundary meta-test.
- Added Playwright-driver production-import boundary test for `tests/helpers`.
- Added UI production-import boundary test for `src/test-utils` under `ui/vitest/boundaries`.
- Added a `boundaries` Vitest project that runs in the default UI smoke suite.
- Updated `ui/scripts/run-vitest.sh` to make the default smoke suite explicit and added a full-suite path through `BAS_VITEST_SUITE=full` / `pnpm test:full`.
- Moved Playwright-driver `isolatedModules` from deprecated `ts-jest` config into `tsconfig.json`.

Validation on 2026-05-01:

- `cd scenarios/browser-automation-studio/api && go test ./...` passed.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm test:unit -- --runInBand` passed: 90 suites, 1173 tests.
- `cd scenarios/browser-automation-studio/ui && bash ./scripts/run-vitest.sh --coverage=false` passed the default smoke suite: `boundaries`, `stores`, `features-core`, `workflow-palette`, and `workflow-builder`.
- `cd scenarios/browser-automation-studio && make test` still failed after unit stages on existing lifecycle gates: standards violations, playbook `bas/cases/01-foundation/01-projects/new-project-create.json` at `assert-workflows-tab-visible`, and mobile-dashboard Lighthouse performance at 52% versus the 70% threshold.

Implemented in the next slice:

- Added canonical import usecase fakes under `api/internal/testutil/mocks`: `ImportDirectoryScanner`, `ImportWorkflowIndexer`, and `ImportProjectIndexer`.
- Migrated the duplicated scan/routines import handler test fakes to the shared `internal/testutil/mocks` seam.
- Added contract tests for the new import fakes.
- Added a Playwright-driver Jest setup file that installs a silent test logger and suppresses routine console output by default; verbose console output can be restored with `BAS_JEST_VERBOSE_LOGS=1`.
- Made the driver default logger silent under `NODE_ENV=test` so tests that call `jest.resetModules()` do not recreate a noisy default logger.
- Documented `playwright-driver/tests/helpers` as the driver testutil boundary with file-level responsibilities for Playwright fakes, HTTP harnesses, typed instruction builders, config fixtures, and the compatibility barrel.

Validation for this slice:

- `cd scenarios/browser-automation-studio/api && go test ./usecases/import/... ./internal/testutil/...` passed.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit/handlers/upload.test.ts tests/unit/idempotency/upload-idempotency.test.ts --runInBand --coverage=false` passed with no structured logger output.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm test:unit -- --runInBand` passed: 90 suites, 1173 tests. The suite still reports an existing Jest worker open-handle warning after completion.

Implemented in the UI utility slice:

- Split `ui/src/test-utils` into focused canonical entry points for `render`, `hooks`, `mocks`, and `stores`.
- Kept compatibility re-exports in `testHelpers.tsx` and `renderHook.tsx` so existing tests can migrate gradually.
- Added browser API shim installation under `mocks/browser`.
- Added fetch mock helpers under `mocks/fetch`: `installFetchMock`, `fetchJsonResponse`, `fetchTextResponse`, and `fetchEmptyResponse`.
- Migrated representative repeated fetch setup in workflow, execution, project, and replay-style API adapter tests to the shared fetch helpers.

Implemented in the workflow-builder UI test slice:

- Added workflow fixtures under `ui/src/test-utils/fixtures/workflow.ts` for workflow nodes, edges, ReactFlow viewports, validation responses, and workflow-builder store state.
- Added canonical DOM, Monaco, and ReactFlow shims under `ui/src/test-utils/mocks`.
- Migrated `WorkflowBuilder.test.tsx`, `WorkflowToolbar.test.tsx`, and `useReactFlowReady.test.ts` to the shared workflow fixtures and shims.

Validation for this slice:

- `cd scenarios/browser-automation-studio/ui && pnpm vitest run src/domains/workflows/services/workflowApi.test.ts src/domains/executions/services/executionApi.test.ts src/domains/projects/services/projectApi.test.ts src/domains/replay-style/__tests__/api.test.ts --coverage=false` passed: 10 tests.
- `cd scenarios/browser-automation-studio/ui && bash ./scripts/run-vitest.sh --coverage=false` passed the default smoke suite: `boundaries`, `stores`, `features-core`, `workflow-palette`, and `workflow-builder`.
- `cd scenarios/browser-automation-studio/ui && pnpm vitest run src/domains/workflows/builder/WorkflowBuilder.test.tsx src/domains/workflows/builder/WorkflowToolbar.test.tsx src/hooks/useReactFlowReady.test.ts --coverage=false` passed: 37 tests across `features-core` and `workflow-builder`.

Implemented in the UI store fetch-helper migration slice:

- Migrated `ui/src/stores/projectStore.test.ts`, `ui/src/stores/__tests__/scenarioStore.test.ts`, and `ui/src/stores/workflowStore.test.ts` from file-local fetch-response builders and direct `global.fetch` assignment to the shared `installFetchMock` and `fetchJsonResponse` / `fetchTextResponse` helpers from `@/test-utils`.
- This extends the fetch/API seam from service-adapter tests into the store layer while preserving existing assertions and requirement coverage.

Validation for this slice:

- `cd scenarios/browser-automation-studio/ui && pnpm vitest run src/stores/__tests__/scenarioStore.test.ts src/stores/projectStore.test.ts src/stores/workflowStore.test.ts --coverage=false` passed: 42 tests in the `stores` project.

Implemented in the entitlement store fetch-helper migration slice:

- Migrated `ui/src/stores/__tests__/entitlementStore.test.ts` from direct `global.fetch` assignment and hand-built response objects to `installFetchMock`, `fetchJsonResponse`, and `fetchEmptyResponse` from `@/test-utils`.
- This covers the largest remaining store test file with the shared fetch seam and keeps async loading-state tests on explicit deferred `Response` promises.

Validation for this slice:

- `cd scenarios/browser-automation-studio/ui && pnpm vitest run src/stores/__tests__/entitlementStore.test.ts --coverage=false` passed: 40 tests in the `stores` project.

Implemented in the remaining UI fetch-helper migration slice:

- Migrated `ui/src/domains/recording/context/ViewportProvider.test.tsx`,
  `ui/src/domains/recording/utils/ViewportSyncManager.test.ts`, and
  `ui/src/domains/projects/ProjectDetail.test.tsx` from direct `global.fetch` / `window.fetch`
  assignment to `installFetchMock`, `fetchJsonResponse`, and `fetchEmptyResponse` from `@/test-utils`.
- A source scan now finds no remaining direct `global.fetch =`, `window.fetch =`, or `new Response`
  patterns in UI test files under `ui/src`.

Validation for this slice:

- `cd scenarios/browser-automation-studio/ui && pnpm vitest run src/domains/recording/context/ViewportProvider.test.tsx src/domains/recording/utils/ViewportSyncManager.test.ts src/domains/projects/ProjectDetail.test.tsx --coverage=false` passed: 50 tests across `features-core` and `record-mode`.

Implemented in the API integration skip-helper slice:

- Added `api/internal/testutil/integration` with shared gates for short mode, required environment variables, required local commands, and HTTP health checks.
- Migrated representative optional integration checks to the shared helpers:
  - Playwright driver checks in API engine and export capture integration tests.
  - Playwright, browser extraction, and Ollama checks in AI element analysis tests.
  - FFmpeg/ffprobe command checks in export renderer integration-style tests.
  - MinIO testcontainer short-mode checks in storage tests.

Validation for this slice:

- `cd scenarios/browser-automation-studio/api && go test ./internal/testutil/...` passed.
- `cd scenarios/browser-automation-studio/api && go test ./handlers/ai ./automation/engine ./services/export/render` passed.
- `cd scenarios/browser-automation-studio/api && go test -tags testcontainers ./storage -run '^$'` passed and compiled the testcontainer-only storage test file without starting containers.

Implemented in the driver low-coverage utility slice:

- Added focused unit coverage for `playwright-driver/src/utils/timing.ts`, covering immediate resolution for non-positive sleeps and timer-gated resolution for positive sleeps.
- Added focused unit coverage for `playwright-driver/src/utils/metrics-server.ts`, covering `/metrics`, non-metrics 404s, metrics collection failures, and duplicate-port startup errors.

Validation for this slice:

- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit/utils/timing.test.ts --runInBand --coverage=false` passed.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit/utils/metrics-server.test.ts --runInBand --coverage=false` passed.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit --runInBand --coverage=false` passed: 92 suites and 1179 tests.

Implemented in the driver fetch-helper slice:

- Added `playwright-driver/tests/helpers/fetch-mocks.ts` with canonical global fetch installation, JSON/text response builders, and request-body/header inspection helpers.
- Migrated vision-client and record-mode callback route tests away from direct `global.fetch` assignment and file-local `new Response` builders:
  - `tests/unit/ai/vision-client/openrouter.test.ts`
  - `tests/unit/ai/vision-client/claude-computer-use.test.ts`
  - `tests/unit/routes/page-events.test.ts`
  - `tests/unit/routes/recording-pages.test.ts`
- A unit-test source scan now finds no remaining direct `global.fetch =`, `globalThis.fetch =`, or `new Response` patterns under `playwright-driver/tests/unit`.

Validation for this slice:

- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit/ai/vision-client/openrouter.test.ts tests/unit/ai/vision-client/claude-computer-use.test.ts tests/unit/routes/page-events.test.ts tests/unit/routes/recording-pages.test.ts --runInBand --coverage=false` passed: 4 suites and 44 tests.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit --runInBand --coverage=false` passed: 92 suites and 1179 tests.

Implemented in the driver session state-machine coverage slice:

- Added focused contract coverage for `playwright-driver/src/session/state-machine.ts` in `tests/unit/session/state-machine.test.ts`.
- The test locks the full valid-transition table, closeability contract, fail-safe invalid-transition behavior, strict invalid-transition errors, and phase predicates for operational/busy/terminal/instruction-accepting states.

Validation for this slice:

- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit/session/state-machine.test.ts --runInBand --coverage=false` passed: 1 suite and 6 tests.

Implemented in the driver record-mode route coverage slice:

- Added focused HTTP contract coverage for `playwright-driver/src/routes/record-mode/recording-validation.ts` in `tests/unit/routes/recording-validation.test.ts`.
- The validation tests cover selector request validation, selector-result response mapping, replay-preview request validation, default replay option mapping, explicit replay option overrides, and snake_case replay failure responses.
- Added focused lifecycle coverage for `playwright-driver/src/routes/record-mode/recording-lifecycle.ts` in `tests/unit/routes/recording-lifecycle.test.ts`.
- The lifecycle tests cover status response mapping, idle status without a pipeline manager, idempotent stop no-ops, active stop cleanup, and session phase reset behavior.

Validation for this slice:

- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit/routes/recording-validation.test.ts tests/unit/routes/recording-lifecycle.test.ts --runInBand --coverage=false` passed: 2 suites and 9 tests.
- `cd scenarios/browser-automation-studio/playwright-driver && pnpm jest tests/unit --runInBand --coverage=false` passed: 95 suites and 1194 tests.

Implemented in the API recording fixture slice:

- Added `api/internal/testutil/fixtures` with canonical builders for recording sessions, timeline entries, recording actions, page events, and session profiles.
- Added focused fixture contract tests so defaults and functional options stay deterministic.
- Migrated representative recording service warm-cache and page-event tests away from repeated domain struct literals.
- Extended the API testutil boundary guard to prevent new imports of the legacy top-level `github.com/vrooli/browser-automation-studio/testutil` package while useful helpers move under `internal/testutil`.
- Left `services/recording/persistence` tests package-local because importing the fixture package there would create a Go import cycle through `services/recording/persistence`.

Validation for this slice:

- `cd scenarios/browser-automation-studio/api && go test ./internal/testutil/... ./services/recording` passed.

Implemented in the API workflow-service fake slice:

- Added canonical function-backed fakes for `workflow.CatalogService` and `workflow.ExecutionService` under `api/internal/testutil/mocks`.
- Added focused fake contract tests so delegated functions, deterministic defaults, and explicit unarranged-method errors remain stable.
- Migrated `api/internal/toolexecution/executor_test.go` from package-local catalog/execution service mocks to the shared fakes, removing the largest remaining local mock block from that package.
- Left handler-local `MockRepository`, `MockCatalogService`, and `MockExecutionService` in place for now because they are broad in-memory handler harnesses with handler-specific call tracking. They should move only if another package needs the same stateful behavior, or if the handler tests are first split around smaller role interfaces.

Validation for this slice:

- `cd scenarios/browser-automation-studio/api && go test ./internal/testutil/... ./internal/toolexecution` passed.

Relevant current files:

- `scenarios/browser-automation-studio/.vrooli/testing.json`
- `scenarios/browser-automation-studio/Makefile`
- `scenarios/browser-automation-studio/api/testutil/*.go`
- `scenarios/browser-automation-studio/api/internal/testutil/fixtures/*.go`
- `scenarios/browser-automation-studio/api/internal/testutil/mocks/*.go`
- `scenarios/browser-automation-studio/api/handlers/testutil_mock_repository.go`
- `scenarios/browser-automation-studio/api/handlers/testutil_mock_services.go`
- `scenarios/browser-automation-studio/playwright-driver/tests/helpers/*.ts`
- `scenarios/browser-automation-studio/playwright-driver/jest.config.js`
- `scenarios/browser-automation-studio/playwright-driver/playwright.config.ts`
- `scenarios/browser-automation-studio/playwright-driver/docs/internal/SEAMS.md`
- `scenarios/browser-automation-studio/ui/vite.config.ts`
- `scenarios/browser-automation-studio/ui/scripts/run-vitest.sh`
- `scenarios/browser-automation-studio/ui/src/test-utils/**/*.ts*`
- `scenarios/browser-automation-studio/ui/docs/SEAMS.md`

## Target End State

BAS should have an explicit, enforced test architecture:

- Go API has an `internal/testutil` package with documented subpackages for fakes, fixtures, db/http harnesses, and domain assertions.
- Production Go code cannot import test utilities, enforced by a meta-test.
- Repeated API mocks are canonicalized by seam, not duplicated in each package.
- Playwright-driver helpers provide typed builders and focused fakes for browser/session/recording/telemetry seams, with noisy global logging suppressed in tests.
- UI tests use one shared render/hook/provider harness, canonical fetch/API mocks, store reset utilities, and stable ReactFlow/Monaco/browser API shims.
- Test project selection in Vitest matches either all defined projects or an explicitly documented stable subset.
- Coverage thresholds are meaningful per surface and ratcheted only after the utility consolidation reduces test friction.

## Implementation Strategy

### Phase 1: Baseline and classify the existing suite

1. Run the supported lifecycle command:

   ```bash
   cd scenarios/browser-automation-studio && make test
   ```

2. Capture direct surface baselines:

   ```bash
   cd scenarios/browser-automation-studio/api && go test ./...
   cd scenarios/browser-automation-studio/playwright-driver && pnpm test:unit
   cd scenarios/browser-automation-studio/ui && pnpm test
   cd scenarios/browser-automation-studio/cli && go test ./...
   ```

3. Generate a short inventory table in the implementation PR description:

   - surface
   - command
   - pass/fail status
   - test count
   - coverage if emitted
   - known skips
   - noisy warnings/output

4. Do not start by raising coverage thresholds. First remove friction that makes adding tests expensive.

### Phase 2: Canonicalize Go API test utilities

1. Create or migrate toward `api/internal/testutil/` with package docs modeled after workspace-sandbox.

2. Split responsibilities into focused subpackages:

   - `fixtures/`: domain factories for projects, workflows, executions, recordings, session profiles, timeline entries, export specs.
   - `mocks/`: one fake per stable seam, such as repository, catalog service, execution service, storage, hub, clock, driver, AI/vision client, scenario-port resolver, filesystem scanner/indexer.
   - `db/`: SQLite/temp database setup where appropriate.
   - `httpx/`: handler harnesses and response assertions.
   - `assertx/`: BAS-specific assertions for workflow definitions, proto responses, event ordering, exports, and recording timelines.

3. Move currently handler-local fakes into shared testutil only when they recur across packages. Keep one-off fakes local.

4. Add a meta-test equivalent to workspace-sandbox's `no_prod_import_test.go`, adjusted for the BAS module path:

   - production `.go` files must not import `github.com/vrooli/browser-automation-studio/internal/testutil`.
   - test files may import it.

5. Resolve the current ambiguity around `api/testutil`:

   - `api/testutil/helpers.go` has a `//go:build testing` tag and includes old helper patterns.
   - `api/testutil/factories.go` and assertions are not under an internal test-only package.
   - Decide whether to migrate useful pieces into `api/internal/testutil` and retire the old package.

### Phase 3: Consolidate high-duplication API mock seams

Prioritize seams where discovery already shows repeated local mocks:

1. Import usecases:

   - `usecases/import/scan/handler_test.go`
   - `usecases/import/routines/handler_test.go`
   - Both define similar `mockDirectoryScanner`, workflow indexer, and project indexer fakes.

2. Tool execution:

   - `api/internal/toolexecution/executor_test.go` defines very large local catalog/execution service mocks.
   - Extract only if the same seam is used elsewhere or if the local fake is large enough to obscure the tests.

3. Handlers:

   - `api/handlers/testutil_mock_repository.go`
   - `api/handlers/testutil_mock_services.go`
   - Treat these as candidates for `internal/testutil/mocks` if they are useful outside `handlers`.

4. Integration skips:

   - AI, Playwright driver, Browserless, Ollama, MinIO, and ffmpeg tests skip based on env/tool availability.
   - Add small shared skip helpers with explicit skip reasons so optional integration coverage is visible and consistent.

### Phase 4: Strengthen Playwright-driver test seams

1. Document `tests/helpers` as the driver testutil layer, or move toward `tests/testutil` if that name is clearer locally.

2. Split helper responsibilities:

   - Playwright object fakes.
   - HTTP route/request/response harnesses.
   - typed instruction/action builders.
   - session/context builders.
   - recording event/timeline builders.
   - logger suppression and inspection helpers.

3. Add a Jest setup file to suppress or capture expected structured logger output. The current driver unit suite passes but emits many console logs, which makes failure triage harder.

4. Fix the repeated ts-jest deprecation warning by moving `isolatedModules: true` into the driver tsconfig path recommended by the warning, or by adjusting Jest config if that is the local preferred pattern.

5. Add focused coverage for low-coverage high-value files reported by the latest driver coverage summary:

   - `src/recording/orchestration/pipeline-manager.ts`
   - `src/recording/testing/self-test.ts`
   - `src/routes/record-mode/recording-lifecycle.ts`
   - `src/routes/record-mode/recording-validation.ts`
   - `src/session/state-machine.ts`
   - `src/utils/metrics-server.ts`
   - `src/utils/timing.ts`

6. Keep real-browser tests separate from mock-based unit tests. The package already has Jest integration tests and Playwright selector tests; preserve that distinction.

### Phase 5: Normalize UI testing architecture

1. Expand `ui/src/test-utils` into a real test architecture:

   - `render/`: `renderWithProviders`, router wrappers, query client factory.
   - `hooks/`: use official Testing Library hook support where possible, or improve the existing local `renderHook` helper.
   - `mocks/`: fetch/API response builders, desktop bridge mocks, logger mocks, ReactFlow/Monaco shims, observer shims.
   - `stores/`: reset helpers and typed store fixtures.
   - `fixtures/`: projects, workflows, executions, recordings, export data, session profiles.
   - `assertions/`: accessible dialog/form assertions and domain-specific UI expectations.

2. Replace repeated `global.fetch` setup and service mocking in store/API tests with shared fetch helpers.

3. Replace repeated ReactFlow/Monaco mocks in workflow-builder tests with canonical shims.

4. Use UI seam documentation as a guide:

   - `ui/docs/SEAMS.md` already documents the export domain well.
   - Extend it after adding test utilities for recording, workflow builder, stores, and service/API boundaries.

5. Fix `scripts/run-vitest.sh` project drift:

   - The script currently runs only four projects.
   - `vite.config.ts` defines many more projects: stores, features-core, workflow-palette, workflow-builder, utils, record-mode, session-manager, subscription, components, export-domain, exports-domain, shared.
   - Either update the script to run the intended full unit set or document and encode a stable smoke/full split, for example `pnpm test` for stable shards and `pnpm test:full` for every project.

6. After utility consolidation, target high-value UI coverage gaps first:

   - workflow builder orchestration and toolbar interactions
   - recording session state and frame streaming
   - execution export and replay paths
   - store/network error handling
   - settings/session-profile flows

### Phase 6: Add contract and boundary tests

1. Add contract tests for production/test boundary rules:

   - no Go production imports of `internal/testutil`
   - no UI production imports from `src/test-utils`
   - no Playwright-driver production imports from `tests/helpers`

2. Add seam contract tests where behavior crosses process or protocol boundaries:

   - Go API action/proto conversion parity.
   - Go API to Playwright-driver request/response conversion.
   - recording timeline event ordering.
   - WebSocket event contracts.
   - export manifest/render contract.
   - scenario-port and registry behavior.

3. Add requirement linkage where tests directly validate requirements. BAS already uses `@vrooli/vitest-requirement-reporter`; use it consistently for UI tests and keep requirement evidence visible in coverage artifacts.

### Phase 7: Ratchet coverage and reliability gates

Only after phases 2-6:

1. Re-run all baselines.

2. Set realistic ratchets:

   - API: raise warning/error thresholds only if full `go test ./... -coverprofile` confirms stable gains.
- Driver: preserve or improve the current 66% line coverage; add file-level attention to recording/session/router low points. `utils/timing`, `utils/metrics-server`, `session/state-machine`, `routes/record-mode/recording-validation`, and `routes/record-mode/recording-lifecycle` now have focused unit coverage.
   - UI: raise from the current 10% line coverage in small increments after the test runner actually covers all intended projects.

3. Add explicit smoke/full distinction if full UI tests are too expensive for default scenario tests.

4. Keep `.vrooli/testing.json` aligned with the commands actually run by lifecycle.

## Contract Decisions

- Test utilities are test-only infrastructure, not production helpers.
- Canonical fakes should be hand-written, small, inspectable, and have deterministic defaults.
- Factories should use functional options so tests express only the fields relevant to the behavior under test.
- Shared assertions should describe BAS domain behavior, not generic equality wrappers.
- Integration tests that require external services must skip with consistent, searchable reasons and should have unit seam coverage nearby.
- Runner scripts must make it obvious whether they are running a smoke subset or the full suite.

## Testing Plan

Validation commands for the implementation loop:

```bash
cd scenarios/browser-automation-studio/api && go test ./...
cd scenarios/browser-automation-studio/cli && go test ./...
cd scenarios/browser-automation-studio/playwright-driver && pnpm test:unit
cd scenarios/browser-automation-studio/playwright-driver && pnpm test:integration
cd scenarios/browser-automation-studio/ui && pnpm test
cd scenarios/browser-automation-studio && make test
```

Optional/expensive gates:

```bash
cd scenarios/browser-automation-studio/playwright-driver && pnpm test:selectors
cd scenarios/browser-automation-studio/ui && pnpm test:coverage
cd scenarios/browser-automation-studio && vrooli scenario requirements report browser-automation-studio --format markdown
```

## Rollout / Validation Checklist

- [x] Baseline commands and coverage are captured before changes.
- [x] Go `api/internal/testutil` package is documented.
- [x] Go production import boundary meta-test exists and passes.
- [ ] Repeated API mocks and fixtures are migrated or intentionally left local with rationale. Import scan/routines fakes are migrated; tool-execution now uses shared workflow service fakes; recording/session fixtures now exist for service-level tests; persistence recording tests remain local to avoid an import cycle; handler fakes remain local because they are broad stateful harnesses coupled to handler tests. Optional integration skip helpers now cover Playwright, Ollama, MinIO, and FFmpeg gates.
- [x] Playwright-driver test helpers are documented and split by responsibility.
- [x] Repeated Playwright-driver fetch/API test setup is centralized. Vision-client and record-mode callback route unit tests now use `tests/helpers/fetch-mocks.ts`; no unit tests directly assign global fetch or construct ad hoc `Response` objects.
- [x] Driver ts-jest warnings are addressed.
- [x] Driver logger noise is addressed or explicitly documented.
- [ ] UI `src/test-utils` has canonical render, mock, store, and fixture helpers. Render, hook, browser mock, fetch mock, workflow fixture, ReactFlow/Monaco shim, and store mock layers exist; project/scenario/workflow/entitlement store tests, `ProjectDetail`, and recording viewport sync tests now use the shared fetch seam; assertion helpers and wider fixture/assertion migration remain.
- [x] UI Vitest script project selection matches the intended test contract.
- [x] Seam docs are updated for API, Playwright-driver, and UI testing seams.
- [ ] Lifecycle `make test` still passes or documented known failures are reduced.

## Risks and Mitigations

- Risk: Over-consolidating one-off fakes into a generic mock layer.
  Mitigation: Extract only recurring seams or large fakes that obscure test intent.

- Risk: Moving helpers breaks package import constraints.
  Mitigation: Add boundary meta-tests early and move incrementally.

- Risk: UI full-suite runtime or memory usage increases.
  Mitigation: Preserve a smoke/full split and document which command lifecycle runs.

- Risk: Coverage thresholds become performative.
  Mitigation: Ratchet thresholds only after verified, stable coverage gains.

- Risk: Driver integration tests become flaky due to real browser/network behavior.
  Mitigation: Keep deterministic unit seam tests adjacent to each optional integration test.

## Non-goals / Prohibited Patterns

- Do not install new packages without explicit permission.
- Do not rewrite production architecture just to satisfy test utility preferences.
- Do not delete meaningful tests to simplify migration.
- Do not hide integration gaps by lowering thresholds without documenting why.
- Do not run scenarios directly; use `make start`, `make test`, `make logs`, `make stop`, or `vrooli scenario ...`.
- Do not add generic `helpers` dumping grounds with mixed responsibilities.

## Definition of Done

The improvement loop is complete when:

- New tests can be written against canonical fixtures/fakes without copying large setup blocks.
- Test-only import boundaries are enforced for Go, UI, and driver surfaces.
- Duplicate BAS mocks are either consolidated or documented as intentionally local.
- Driver and UI test output is quiet enough that failures are easy to find.
- `make test` remains the primary scenario validation path.
- Coverage and requirement reports reflect the tests that actually run.
- Seam documentation explains the test substitution points future agents should use.
