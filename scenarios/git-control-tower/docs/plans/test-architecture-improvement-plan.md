# Git Control Tower Test Architecture Improvement Plan

## Purpose

Improve the `git-control-tower` scenario's test quality, maintainability, and coverage using the same architectural direction recently applied to `workspace-sandbox`: shared test utilities, explicit seams, responsibility-focused tests, and scenario-level validation that catches lifecycle regressions.

This is a plan only. It records the current state, evidence, target end state, and phased execution path for a future implementation loop.

## Required Reading

Future agents should load the same planning and test architecture guidance before implementation:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement utils-unification test unit-testing-architecture-steer decision-boundary-extraction boundary-of-responsibility-enforcement react-coherence
```

Also read the current seam docs and workspace-sandbox reference utilities:

```bash
sed -n '1,260p' scenarios/git-control-tower/docs/internal/SEAMS.md
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/testutil.go
find scenarios/workspace-sandbox/api/internal/testutil -maxdepth 3 -type f | sort
```

## Problem Statement

`git-control-tower` has many useful tests and its direct API, CLI, and UI unit gates currently pass, but the test architecture has drifted:

- API test helpers and fakes are scattered through root-level `_test.go` files instead of a canonical `internal/testutil` package.
- HTTP/client test setup repeats `httptest.NewServer`, JSON decoding, response assertions, and ad hoc client construction across multiple scenario-integration clients.
- UI tests cover selected components and pure utilities, but the app shell, settings/review panels, file list flows, mobile surfaces, and React Query hooks are mostly uncovered.
- Vitest has no shared setup file, so test globals and noisy `api-base` resolution behavior are not centrally controlled.
- Scenario-level `make test` currently fails even though direct unit tests pass, which means test architecture work must include lifecycle/smoke reliability.

## Current Technical Context

Key `git-control-tower` files:

- API root package: `scenarios/git-control-tower/api/*.go`
- API test helper root: `scenarios/git-control-tower/api/testutil_test.go`
- API fakes: `api/git_runner_fake_test.go`, `api/fileio_fake_test.go`, `api/audit_logger_fake_test.go`, `api/db_checker_fake_test.go`, `api/workspace_sandbox_fake_test.go`
- UI config: `scenarios/git-control-tower/ui/vite.config.ts`
- UI tests: `scenarios/git-control-tower/ui/src/**/*.test.{ts,tsx}`
- Scenario lifecycle test config: `scenarios/git-control-tower/.vrooli/service.json`, `.vrooli/testing.json`
- Current seam doc: `scenarios/git-control-tower/docs/internal/SEAMS.md`

Workspace-sandbox reference pattern:

- `scenarios/workspace-sandbox/api/internal/testutil/testutil.go`
- `api/internal/testutil/mocks/`
- `api/internal/testutil/fixtures/`
- `api/internal/testutil/db/`
- `api/internal/testutil/httpx/`
- `api/internal/testutil/assertx/`
- `api/internal/testutil/no_prod_import_test.go`
- `ui/src/test-setup.ts`

## Evidence

Discovery commands run on 2026-05-01:

```bash
vrooli help
find scenarios/git-control-tower -path '*/node_modules' -prune -o \( -name '*_test.go' -o -name '*.test.ts' -o -name '*.test.tsx' \) -print | sort | wc -l
find scenarios/workspace-sandbox -path '*/node_modules' -prune -o \( -name '*_test.go' -o -name '*.test.ts' -o -name '*.test.tsx' \) -print | sort | wc -l
go test ./...                         # in scenarios/git-control-tower/api
go test -cover ./...                  # in scenarios/git-control-tower/api
pnpm test                             # in scenarios/git-control-tower/ui
pnpm vitest run --coverage            # in scenarios/git-control-tower/ui
go test ./...                         # in scenarios/git-control-tower/cli
go test ./...                         # in scenarios/git-control-tower/platforms/electron/renderer
make test                             # in scenarios/git-control-tower
```

Observed results:

- `git-control-tower` has 85 first-party test files excluding `node_modules`; `workspace-sandbox` has 104.
- API `go test ./...` passes.
- API `go test -cover ./...` reports `git-control-tower` root package at 38.3% statement coverage and `ssh` at 18.6%; `filerelations` subpackages report 0.0% despite having tests, suggesting tests are narrow and do not execute package statements much.
- CLI `go test ./...` passes; `go test -cover ./...` reports 25.0% in the root CLI package and 0.0% in command-domain packages.
- UI `pnpm test` passes: 14 files, 268 tests.
- UI coverage is low overall: 15.62% statements, 30.94% functions. `App.tsx`, panels, settings tabs, file list surfaces, review panels, many hooks, and API modules are at or near 0%.
- `platforms/electron/renderer go test ./...` fails because `scenario-to-desktop-runtime/manifest` is not resolvable from that package.
- `make test` fails after 1m26s with 7 passed and 3 failed phases: standards, lint, smoke.
- UI smoke failure is `ERR_CONNECTION_REFUSED` at `http://localhost:21400`; the iframe bridge never signaled ready because the UI endpoint was not reachable.
- Standards reported 86 findings: 1 critical required-layout finding for `Makefile` and 85 low focus-visible findings.
- Lint phase says Go lint and TypeScript pass, but reports one ESLint issue plus unmatched `initialization` and `platforms` code-bearing components.

## Target End State

- API tests use a canonical test-only utility structure modeled after workspace-sandbox, with one fake per seam and no production imports of test utilities.
- HTTP/client tests use shared server/client harnesses and assertions so adding tests for `agent-manager`, `test-genie`, `tidiness-manager`, `auditor`, `browser-automation-studio`, and `workspace-sandbox` clients is straightforward.
- Domain tests are grouped by responsibility: pure parsing, Git side effects, repo registry, cross-scenario clients, review orchestration, visual capture storage, HTTP handlers.
- UI tests have a shared setup file, stable render helpers, QueryClient helpers, API/fetch mock helpers, and coverage for the app shell and high-value workflows.
- The full scenario `make test` gate passes or has documented, intentionally skipped unsupported components.
- `docs/internal/SEAMS.md` describes test seams and responsibility boundaries at the same level of specificity as workspace-sandbox.

## Scope

In scope:

- Refactor test-only helpers and fakes.
- Add shared API and UI test utilities.
- Add missing tests for existing behavior.
- Fix or configure scenario-level test/lint/smoke gates when the failures are test-infrastructure issues.
- Document seam and responsibility decisions.

Out of scope:

- Changing product behavior.
- Introducing new dependencies without explicit approval.
- Broad production package restructuring unless required to expose a clean seam.
- Raising coverage thresholds before reliable coverage collection and high-value test targets exist.

## Implementation Strategy

### Phase 1: Stabilize Scenario Test Gates

Goal: make the system-level signal trustworthy before expanding tests.

Actions:

1. Re-run `make test` and inspect fresh logs under `coverage/logs/<run_id>/`.
2. Fix UI smoke reachability. The current artifact shows `ERR_CONNECTION_REFUSED` on port `21400`; verify whether `test-genie` should start the scenario before smoke or whether `service.json` needs lifecycle/test configuration that ensures the UI server is running.
3. Investigate the standards critical `Makefile` finding. The Makefile appears to contain standard targets; confirm whether the auditor expects additional target names, formatting, or metadata.
4. Resolve the ESLint warning surfaced by the lint phase and decide whether `initialization` and `platforms` should be explicitly lint-configured or ignored as non-target components.
5. Decide how `platforms/electron/renderer` should be tested. Either wire its module dependency so `go test ./...` works, add the missing runtime module path, or explicitly exclude it from scenario unit scans with documented rationale.

Acceptance gate:

```bash
cd scenarios/git-control-tower && make test
```

### Phase 2: Introduce API `internal/testutil`

Goal: consolidate test helpers without changing production behavior.

Actions:

1. Create `api/internal/testutil/` with subpackages mirroring the workspace-sandbox shape where useful:
   - `mocks/` for canonical fakes.
   - `fixtures/` for repo, branch, file-change, commit, review, and visual-capture data builders.
   - `httpx/` for `httptest.Server` setup, JSON response helpers, request assertion helpers, and scenario-client base URL wiring.
   - `db/` for SQLite repo/audit/credentials stores using `t.TempDir()`.
   - `assertx/` for domain-specific assertions.
2. Move or wrap existing root helpers from `api/testutil_test.go`, `git_runner_fake_test.go`, `fileio_fake_test.go`, `audit_logger_fake_test.go`, `db_checker_fake_test.go`, and `workspace_sandbox_fake_test.go`.
3. Keep migration incremental: update one test family at a time, starting with high-duplication client tests and Git service tests.
4. Add `api/internal/testutil/no_prod_import_test.go` equivalent, adapted for module path `git-control-tower/internal/testutil`.

Acceptance gate:

```bash
cd scenarios/git-control-tower/api && go test ./...
```

### Phase 3: Normalize HTTP and Cross-Scenario Client Tests

Goal: reduce repeated server/client boilerplate and make failure-mode coverage cheap.

Actions:

1. Build `httpx` helpers for JSON handlers, request body capture, path/method assertions, status/content-type assertions, and timeout-safe clients.
2. Migrate tests for:
   - `agent_manager_client_test.go`
   - `test_genie_client_test.go`
   - `tidiness_manager_client_test.go`
   - `auditor_client_test.go`
   - `browser_automation_client_test.go`
   - `capabilities_checkers_test.go`
3. Add tests for shared failure contracts: non-JSON body, missing fields, non-2xx response, request timeout/cancelation where supported.
4. Verify all cross-scenario HTTP calls remain behind their documented client seams.

Acceptance gate:

```bash
cd scenarios/git-control-tower/api && go test ./... -run 'Client|Capabilities'
```

### Phase 4: Strengthen Domain and Integration Boundary Coverage

Goal: raise API coverage by testing responsibilities, not line counts.

Priority targets:

1. `filerelations/*`: coverage currently reports 0.0%; add table-driven tests that exercise scanner/resolver behavior through public package APIs.
2. `ssh/*`: expand tests around key validation, platform differences, and service error handling.
3. `repo_store.go`, `credentials_store.go`, `review_store.go`, and audit persistence: use `db` helpers to test schema-backed behavior and edge cases.
4. Git operation workflows: stage/unstage/discard/commit/push/pull/branch should use `mocks.FakeGitRunner` for unit tests and `SetupTestRepo` only for explicitly labeled integration tests.
5. Visual capture storage/service: introduce fixed clock/test fixtures so time-sensitive assertions do not depend on `time.Now()` spread across tests.

Acceptance gates:

```bash
cd scenarios/git-control-tower/api && go test ./... -cover
cd scenarios/git-control-tower/api && go test ./... -run 'Filerelations|SSH|Store|Visual'
```

### Phase 5: Add UI Test Setup and Shared React Test Utilities

Goal: make UI tests easier to write and less noisy.

Actions:

1. Add `ui/src/test-setup.ts` and wire it in `vite.config.ts` using `setupFiles`.
2. Import `@testing-library/jest-dom` centrally rather than relying on individual test behavior.
3. Add shared helpers under `ui/src/test-utils/` or `ui/src/test/`:
   - `renderWithQueryClient`
   - `createTestQueryClient`
   - `mockFetchJson`
   - viewport/media helpers for mobile/desktop tests
   - localStorage reset helpers
4. Centralize suppression or mocking for noisy `@vrooli/api-base` resolution logs in tests that do not assert API base behavior.
5. Convert existing tests to the helper only when touching them for related coverage, to avoid churn.

Acceptance gate:

```bash
cd scenarios/git-control-tower/ui && pnpm test
```

### Phase 6: Cover High-Value UI Workflows

Goal: cover behavior users rely on, especially surfaces with current 0% coverage.

Priority targets:

1. `App.tsx`, `AppPanels.tsx`, `AppMobilePanels.tsx`: app shell renders, desktop/mobile branching, initial loading/error states, selected-file flow, empty repo state.
2. `FileList.tsx`, `FileRow.tsx`, grouped/flat/project tree views: file selection, grouping, action availability, keyboard/focus affordances.
3. `SettingsModal` and settings tabs: credentials, SSH, grouping, layout, health, storage validation and save/error flows.
4. `ScenarioReviewPanel*`: overview/tests/screenshots/rules/workflows tabs, loading/error/empty states, action buttons routed to callbacks.
5. API hooks in `lib/hooks-*.ts`: QueryClient behavior, cache keys, error propagation, mutation invalidation.
6. Mobile header/nav equivalents, using workspace-sandbox's mobile layout tests as a pattern.

Acceptance gates:

```bash
cd scenarios/git-control-tower/ui && pnpm test
cd scenarios/git-control-tower/ui && pnpm vitest run --coverage
```

Do not set hard coverage thresholds until this phase has landed. Use coverage as a prioritization map first.

### Phase 7: Document and Enforce Seams

Goal: make future test work converge instead of drifting.

Actions:

1. Expand `docs/internal/SEAMS.md` with responsibility zones like workspace-sandbox:
   - HTTP/presentation
   - service/orchestration
   - Git integration
   - persistence
   - cross-scenario clients
   - UI app shell and feature surfaces
2. Add explicit test seam contracts:
   - no direct `exec.Command("git", ...)` outside production `ExecGitRunner` and explicitly labeled integration tests.
   - no production import of `api/internal/testutil`.
   - cross-scenario calls route through client interfaces.
   - React Query tests use a local QueryClient, never a shared global cache.
3. Document remaining weak seams if they are too broad to fix in the first pass.

Acceptance gate:

```bash
rg "internal/testutil" scenarios/git-control-tower/api -g '*.go'
cd scenarios/git-control-tower/api && go test ./...
```

## Contract Decisions

- Test utilities are test-only and may depend on `testing`; production code must not import them.
- Fakes should be hand-written, small, and behavior-oriented. Do not add mock generation unless explicitly approved.
- Keep integration tests explicit. Tests that shell out to real `git`, touch SQLite, or run HTTP servers should be named/described as such and use shared harnesses.
- Coverage improvement should follow behavior risk, not simple file percentage.
- UI tests should prefer public user-facing queries and stable `data-testid` selectors already defined in `ui/src/consts/selectors.ts`.

## Testing Plan

Run focused gates during each phase, then the full scenario gate:

```bash
cd scenarios/git-control-tower/api && go test ./...
cd scenarios/git-control-tower/api && go test -cover ./...
cd scenarios/git-control-tower/cli && go test ./...
cd scenarios/git-control-tower/ui && pnpm test
cd scenarios/git-control-tower/ui && pnpm vitest run --coverage
cd scenarios/git-control-tower && make test
```

If renderer/electron is intentionally in scope:

```bash
cd scenarios/git-control-tower/platforms/electron/renderer && go test ./...
```

## Rollout Checklist

- [ ] `make test` failures are fixed or intentionally documented.
- [ ] API `internal/testutil` package exists with no production import leaks.
- [ ] Existing root-level fake files are migrated or reduced to compatibility wrappers for tests only.
- [ ] HTTP/client tests use shared `httpx` utilities.
- [ ] UI has `test-setup.ts` wired into Vitest.
- [ ] App shell and high-risk UI surfaces have focused tests.
- [ ] Coverage reports show meaningful movement in API, CLI, and UI hotspots.
- [ ] `docs/internal/SEAMS.md` documents the updated test seams and remaining gaps.

## Risks and Mitigations

- Risk: moving fakes out of the root package could require exported interfaces or constructors that production does not need.
  - Mitigation: migrate one fake at a time; if a fake depends on unexported production details, keep it package-local temporarily and document the required seam.

- Risk: coverage work becomes shallow snapshot-style UI testing.
  - Mitigation: prioritize workflows, state transitions, error states, and accessibility/focus contracts.

- Risk: lifecycle smoke failures are caused by environment state rather than code.
  - Mitigation: preserve artifacts and verify with `make start`, `make status`, `make logs`, and `make stop` through the scenario lifecycle system only.

- Risk: fixing the standards Makefile finding could mask an auditor bug.
  - Mitigation: inspect the auditor rule and add a narrow regression test or documented exemption if the Makefile is already compliant.

## Non-Goals / Prohibited Patterns

- Do not install packages without explicit permission.
- Do not directly execute scenario binaries or scripts; use `make start`, `make test`, `make logs`, `make stop`, or `vrooli scenario ...`.
- Do not duplicate workspace-sandbox utilities blindly; copy the architecture, not irrelevant domain fixtures.
- Do not add permanent compatibility layers just to move tests faster.
- Do not introduce global mutable test state shared across UI tests.

## Definition of Done

- `cd scenarios/git-control-tower && make test` passes.
- Direct API, CLI, and UI unit tests pass.
- API test utilities are centralized and protected from production imports.
- UI tests have a shared setup and reusable render/API helpers.
- New tests cover previously weak high-value surfaces, especially app shell, file workflows, settings, review panels, cross-scenario clients, and persistence seams.
- `docs/internal/SEAMS.md` reflects the current seam and responsibility model.
