# Prompt Manager Test Architecture Improvement Plan

## Purpose

Improve the prompt-manager scenario's test architecture using the same direction that recently helped workspace-sandbox: shared test utilities, clearer seams, stronger behavior-focused coverage, and less duplicated local setup.

This is not a failing-test rescue plan. The latest recorded scenario run passed all gates, so the work should preserve the current green state while making future tests easier to write, harder to drift, and better aligned with requirements.

## Required Reading

Future agents should load the same steering context before implementing this plan:

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement utils-unification test unit-testing-architecture-steer decision-boundary-extraction boundary-of-responsibility-enforcement react-coherence
```

Recommended reference scan:

```bash
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/testutil.go
sed -n '1,220p' scenarios/workspace-sandbox/api/internal/testutil/no_prod_import_test.go
sed -n '1,220p' scenarios/prompt-manager/docs/internal/UNIT_TEST_ARCHITECTURE.md
sed -n '1,220p' scenarios/prompt-manager/docs/internal/SEAMS.md
```

## Problem Statement

Prompt-manager has broad test volume, but the test infrastructure has grown unevenly:

- API tests define duplicate fixtures and fakes inside feature packages instead of a canonical test utility layer.
- CLI command tests have very low coverage for many command domains even though the scenario's business contract depends heavily on CLI parity.
- UI tests contain useful specialized utilities for R3F and TipTap, but routine render, provider, API, storage, viewport, and service-mocking patterns are still repeated locally.
- Some unit tests still trigger noisy real-network attempts or framework warnings, which weakens signal even when tests pass.
- Requirements and playbooks exist, but unit/component tests are not consistently traceable to requirement IDs or decision boundaries.

The result is a suite that passes today but is more expensive than necessary to extend safely.

## Evidence

Discovery commands run on 2026-05-01:

```bash
find scenarios/prompt-manager -path '*/dist' -prune -o -path '*/node_modules' -prune -o -type f \( -name '*_test.go' -o -name '*.test.ts' -o -name '*.test.tsx' \) -print
cat scenarios/prompt-manager/coverage/latest/manifest.json
sed -n '1,260p' scenarios/prompt-manager/coverage/logs/20260501-200340-9602b578/unit.log
sed -n '1,220p' scenarios/prompt-manager/coverage/logs/20260501-200340-9602b578/integration.log
sed -n '1,220p' scenarios/prompt-manager/coverage/logs/20260501-200340-9602b578/playbooks.log
```

Observed shape:

- Test file counts: API 87, CLI 23, UI 105, total 215.
- Latest scenario run `20260501-200340-9602b578` passed structure, standards, dependencies, lint, docs, smoke, unit, integration, playbooks, business, and performance.
- Unit phase passed Go API, Go CLI, UI Vitest, and shell syntax checks.
- UI Vitest passed 105 files and 1,615 tests, with 4 skipped.
- Playbooks passed 6/6 workflows.
- Go API package coverage is uneven: strong areas include `validation` 100%, `interop` 96.7%, `worldseats` 94.6%, `graph` 80.3%; weak areas include root `prompt-manager` 4.2%, `metrics` 5.0%, `ogmeta` 4.9%, `testing` 4.5%, `agents` 22.9%, `tags` 26.3%.
- Go CLI coverage is especially uneven: `domains` and `internal/types` are 100%, but `agents` 0.7%, `experiments` 1.0%, `graph` 0.6%, `members` 1.6%, `metadata` 4.5%, `search` 0.3%, `skills` 0.5%, `tags` 3.3%, `testing` 2.9%, `topics` 0.9%.
- UI unit logs include avoidable noise from real fetch attempts to `http://localhost:3000/api/v1/world-scale`, `api-base` resolution logs, GLSL parser stderr, and unmocked R3F DOM tag warnings.
- Duplicate API test team fixtures exist in `api/heartbeat/test_team_helpers_test.go`, `api/store/test_team_helpers_test.go`, and `api/teams/test_team_helpers_test.go`.
- Workspace-sandbox has a clearer model in `api/internal/testutil` with documented subpackages for `mocks`, `fixtures`, `db`, `httpx`, and `assertx`, plus a meta-test preventing production imports.

## Scope

In scope:

- Consolidate prompt-manager API test fakes, fixtures, HTTP helpers, and assertions.
- Consolidate prompt-manager UI test harnesses for providers, viewport, storage, API/service mocks, R3F, and TipTap.
- Add high-value behavior and contract tests for low-coverage API/CLI areas.
- Add meta-tests that enforce test utility boundaries.
- Improve requirement-to-test traceability for important business and playbook flows.
- Reduce noisy logs and accidental network access in unit tests.

Out of scope:

- Changing production behavior unrelated to testability.
- Rewriting the scenario architecture.
- Replacing Vitest, Go test, or the Vrooli scenario lifecycle.
- Installing new packages.
- Expanding the BAS playbook system beyond adding focused prompt-manager cases.

## Current Technical Context

Primary test entrypoints:

- `scenarios/prompt-manager/Makefile` delegates `make test` to `vrooli scenario test prompt-manager`.
- `scenarios/prompt-manager/.vrooli/testing.json` enables Go and Node unit tests plus business, integration, performance, lint, and structure gates.
- `scenarios/prompt-manager/ui/vitest.config.ts` uses jsdom, `src/test/setup.ts`, and forks pool.
- `scenarios/prompt-manager/ui/src/test/` already contains specialized helpers for R3F, shaders, TipTap, deep utilities, and stores.

API/CLI areas with existing tests but weak shared infrastructure:

- API fakes are mostly package-local, for example `api/teams/handlers_test.go`.
- Team fixtures are duplicated across `heartbeat`, `store`, and `teams`.
- API handler tests frequently construct `httptest.NewRequest` / `httptest.NewRecorder` directly.
- CLI tests focus on some command wiring and parity, but many command domains have near-zero statement coverage.

UI areas with existing tests but inconsistent harnesses:

- Components account for the largest UI test group, but many untested production components remain.
- Repeated local mocks exist for `localStorage`, routing, toast, heartbeat service, Monaco, TipTap, React Flow, and R3F.
- `src/test/index.ts` is a good start, but it only covers specialized R3F/deep utility exports and does not yet own routine React render harnesses.

## Target End State

Prompt-manager should have a documented, enforced test architecture:

- API test utilities live under a canonical test-only package, likely `api/internal/testutil/...`.
- Test fakes are one-per-seam, have constructors with sane defaults, expose failure knobs, and are reused by API tests.
- Domain fixtures use options instead of repeated object literals.
- HTTP handler tests use shared request/response helpers and domain assertions.
- CLI tests share a command harness that can exercise output, dry-run, API client behavior, and error handling consistently.
- UI tests import routine test setup from `@/test`, including `renderWithProviders`, query client setup, router setup, viewport/storage helpers, and canonical service mocks.
- Unit tests do not attempt real network calls unless explicitly marked integration.
- Requirement-critical flows have at least one unit/component/handler test plus one BAS or integration path where appropriate.
- Meta-tests catch production imports of test-only helpers.

## Implementation Strategy

### Phase 1: Baseline and Guardrails

1. Run the current gate before changing anything:

   ```bash
   cd scenarios/prompt-manager && make test
   ```

2. Record current package coverage from the unit log.
3. Add an internal testing note update in `docs/internal/UNIT_TEST_ARCHITECTURE.md` describing the intended shared utility layout.
4. Add a no-production-import meta-test after introducing any `internal/testutil` package.

Acceptance criteria:

- Current test suite is green before refactoring.
- There is a clear written target architecture before helper migration starts.

Progress:

- 2026-05-01: Started the baseline gate with `cd scenarios/prompt-manager && make test`.
- 2026-05-01: Updated `docs/internal/UNIT_TEST_ARCHITECTURE.md` with the shared API test utility target and production-import boundary.
- 2026-05-01: Added `api/internal/testutil/no_prod_import_test.go` to catch non-test imports of `prompt-manager/internal/testutil/...`.

### Phase 2: API Test Utility Unification

Create a prompt-manager equivalent of workspace-sandbox's API test utility shape:

```text
api/internal/testutil/
  assertx/
  fixtures/
  httpx/
  mocks/
  testutil.go
  no_prod_import_test.go
```

Initial extraction targets:

- Move duplicate team builders from:
  - `api/heartbeat/test_team_helpers_test.go`
  - `api/store/test_team_helpers_test.go`
  - `api/teams/test_team_helpers_test.go`
- Canonicalize fakes currently embedded in:
  - `api/teams/handlers_test.go`
  - `api/agents/handlers_test.go`
  - `api/skills/handlers_test.go`
  - `api/topics/handlers_test.go`
  - `api/aisearch/aisearch_test.go`
- Add `httpx` helpers for JSON request bodies, route variables, response decoding, and status assertions.

Design rules:

- Keep fakes hand-written and small.
- One fake per real seam.
- Constructors return sane defaults.
- Use explicit error knobs for failure-path tests.
- Production code must not import `prompt-manager/internal/testutil`.

Acceptance criteria:

- Duplicate team fixture implementations are removed or reduced to thin package-local wrappers.
- New handler tests can be written without rebuilding ad hoc stores.
- Existing API tests stay green.

Progress:

- 2026-05-01: Created `api/internal/testutil/{fixtures,httpx,mocks,assertx}` and documented the package contract.
- 2026-05-01: Added canonical team fixtures with functional options in `api/internal/testutil/fixtures`.
- 2026-05-01: Reduced heartbeat and teams duplicate team helpers to thin wrappers over shared fixtures, preserving their existing enabled/default-role drift explicitly through options.
- 2026-05-01: Added initial `httpx` helpers for request construction, mux vars, JSON request bodies, response decoding, and status assertions.
- 2026-05-02: Adopted `api/internal/testutil/httpx` in three handler-test areas: `agents`, `teams` export, and heartbeat retry/investigation handlers. The migration replaced package-local `httptest`/mux/status boilerplate while preserving existing assertions.
- 2026-05-02: Verified the touched API packages with `cd scenarios/prompt-manager/api && go test ./agents ./teams ./heartbeat`.

### Phase 3: CLI Harness and Contract Coverage

Build a shared CLI test harness around `cli/internal/appctx`, output capture, API base injection, dry-run behavior, and fake HTTP servers.

Priority command domains based on low coverage and business importance:

- `cli/skills`
- `cli/search`
- `cli/graph`
- `cli/topics`
- `cli/agents`
- `cli/members`
- `cli/testing`
- `cli/experiments`
- `cli/tags`

Recommended test classes:

- Command parses expected flags and rejects invalid combinations.
- Dry-run validates payloads without mutation.
- API errors produce useful user-facing output and nonzero status.
- List/show commands format empty and populated results.
- Mutating commands send expected request methods, paths, and bodies.
- CLI command coverage aligns with `docs/reference/cli-commands.md` and `cli/PARITY_AUDIT.md`.

Acceptance criteria:

- Each priority CLI domain has at least one success-path, one validation-error, and one API-error test.
- CLI tests use a common harness instead of one-off `httptest.Server` setup.
- Coverage for near-zero CLI command packages meaningfully increases.

Progress:

- 2026-05-01: Added `cli/internal/testutil` with a reusable `appctx.Context` fake, request recording, typed response/error injection, stdout/stderr capture, stdin-aware capture, and a production-import guard.
- 2026-05-01: Added behavior-focused `skills` CLI tests for filtered list success, validation-before-API, API-error surfacing, and `read` request payload construction.
- 2026-05-01: Added behavior-focused `search` CLI tests for AI search success, combined-output validation-before-API, and AI-error fallback to text search.
- 2026-05-01: Migrated the `discover` CLI tests from a package-local fake context to the shared CLI harness.
- 2026-05-02: Added behavior-focused `graph` CLI tests for summary output, missing node ID validation-before-API, popular-node API-error surfacing, and health type-filter request sequencing.
- 2026-05-02: Added behavior-focused `topics` CLI tests for list output, create validation-before-API, topic search request payload construction, accumulated skill output, and delete API-error surfacing.
- 2026-05-02: Verified the full prompt-manager CLI suite with `cd scenarios/prompt-manager/cli && go test ./...`.
- 2026-05-02: Added shared-harness CLI tests for the remaining low-coverage priority domains: `agents`, `members`, `tags`, `testing`, and `experiments`. These cover list/run/create success paths, validation-before-API paths, request payload construction, AI-search fallback sequencing, and API-error surfacing.
- 2026-05-02: Re-verified the full prompt-manager CLI suite with `cd scenarios/prompt-manager/cli && go test ./...`.

### Phase 4: UI Shared Test Harness

Expand `ui/src/test/index.ts` from specialized R3F exports into the canonical UI testing import.

Add focused utilities:

- `renderWithProviders(ui, options)` with router, query client, theme/toast providers as needed.
- `createTestQueryClient()` with retries disabled.
- `setViewport(width, height?)` and `restoreViewport()`.
- `mockLocalStorage()` / `resetStorageMocks()`.
- Service mock factories for heartbeat, skill, team, graph, and world-scale calls.
- API network guard that fails tests on unexpected `fetch` unless explicitly allowed.
- Common wrappers for Monaco, TipTap, React Flow, and R3F stubs.

Refactor the noisiest tests first:

- Tests that currently attempt real fetches, especially world-scale store/component paths.
- R3F tests that emit unknown DOM tag warnings because they are not using the R3F harness.
- Component tests with repeated provider boilerplate or repeated service mocks.

Acceptance criteria:

- UI unit logs no longer contain accidental `ECONNREFUSED` fetch attempts.
- Common provider and service mock setup lives in `ui/src/test`.
- New component tests can be authored with one import from `@/test`.

Progress:

- 2026-05-01: Added `ui/src/test` routine helpers for React Query clients, provider-aware component rendering, provider-aware hook rendering, localStorage mocks, viewport resizing, and explicit fetch guarding.
- 2026-05-01: Exported the routine UI helpers through `@/test` alongside the existing R3F/deep utility exports.
- 2026-05-01: Migrated `useSkillsData` and `useActionsData` tests to the shared `renderHookWithProviders` harness.
- 2026-05-01: Added `network.test.ts` to lock the unexpected-fetch contract and JSON response helper.
- 2026-05-01: Mocked the world-scale store seam in `WorldSettingsContent.test.tsx`, removing the accidental `http://localhost:3000/api/v1/world-scale` fetch from that suite.
- 2026-05-01: Verified `pnpm type-check`, focused UI tests, and full `pnpm test` for the prompt-manager UI.
- 2026-05-01: Added an explicit R3F DOM warning filter in `@/test` and applied it to `DynamicSky` and `SlimeAgent` suites.
- 2026-05-01: Removed accidental UI localhost calls from persisted world store imports and `SettingsDialog` AI status tests; `worldScaleStore` and `worldSeatsStore` now skip import-time fetches in Vitest while preserving explicit fetch behavior.
- 2026-05-01: Quieted expected-error suites by asserting logged errors in `worldSeatsStore` and `TaskKanbanBoard` tests instead of emitting stack traces.
- 2026-05-01: Removed `SearchHighlight` debug logging, filtered known GLSL parser diagnostics for built-in shader variables, enabled React Router future flags in local router wrappers, and wrapped GraphView focus interactions in async `act`.
- 2026-05-01: Re-verified `pnpm type-check` and full `pnpm test` for the prompt-manager UI. The remaining stdout noise is the intentionally exploratory `TipTapDiscovery.test.ts` output.

### Phase 5: Decision Boundary and Responsibility Tests

Add behavior-focused tests around the scenario's highest-risk decision boundaries:

- Team runtime and coordination defaults.
- Decision modes, pending decisions, defer/approve/retry behavior.
- Heartbeat prompt construction and member context inclusion/exclusion.
- Handoff extraction and run registry durability.
- Skill read/render/variant/experiment behavior.
- Graph health scoring and orphan/cycle detection.
- AI search budget, filtering, vector-store fallback, and text-search fallback.
- File-based store persistence and malformed data handling.

Testing style:

- Prefer table-driven tests for decision matrices.
- Avoid snapshot-only assertions for generated prompts; assert named contract fragments and excluded fragments.
- Inject clocks, clients, readers, stores, and vector/search seams instead of depending on globals.

Acceptance criteria:

- Tests name the decision being protected.
- Failure messages identify the broken contract, not just a generic diff.
- High-risk decision behavior is covered at the service/domain layer before UI or integration tests.

Progress:

- 2026-05-02: Added `/skills/read` experiment-selection decision tests covering three contracts: a running experiment can select a variant and override returned skill content, a control-arm selection keeps original skill content, and non-running experiments are rejected before returning content. The tests use deterministic 0/1 weights so they protect read-time variant behavior without random flake risk.

### Phase 6: Requirement Traceability and BAS Alignment

Use the requirement modules in `scenarios/prompt-manager/requirements/` as the coverage map.

Actions:

- Add `[REQ:...]` comments to key API/CLI/UI tests for requirement-critical flows.
- Ensure every requirement module has at least one automated coverage point or a documented reason why it is covered only by structural/business validation.
- Add BAS cases only where unit/component coverage cannot validate actual user behavior.

Likely BAS additions:

- CLI-backed skill read/list smoke path if BAS or scenario test infrastructure supports it.
- Team decision approval/defer UI flow.
- Graph health/settings interaction flow.
- Search mode fallback or filter flow beyond current skill search filter.

Acceptance criteria:

- Requirement-to-test mapping is visible from test names/comments or documentation.
- BAS remains focused on user-visible flows, not duplicating unit tests.

Progress:

- 2026-05-02: Added a requirement traceability table to `docs/internal/UNIT_TEST_ARCHITECTURE.md` mapping key requirement IDs to existing API/CLI/UI/BAS coverage points. The table also maps the new `/skills/read` experiment-aware tests under the broader skill-read API contract because the current requirements catalog does not define a dedicated experiment/variant requirement.

### Phase 7: Coverage Gates and Drift Prevention

After utility migration and focused coverage additions, introduce lightweight drift checks:

- API testutil import boundary meta-test.
- UI test helper import guidance in docs and optional lint/documentation check if an existing mechanism supports it.
- Coverage trend checks from scenario logs, without hard global thresholds until noisy/low-value packages are addressed.
- A short checklist in `docs/internal/UNIT_TEST_ARCHITECTURE.md` for new tests.

Acceptance criteria:

- Test architecture regressions are caught automatically where practical.
- Coverage gates do not incentivize shallow tests.
- Documentation tells future agents where to put fakes, fixtures, and harnesses.

Progress:

- 2026-05-02: Added fixture-package tests for `api/internal/testutil/fixtures` so the shared fixture package has direct contract coverage and no longer appears as an untested code package in standards scans.
- 2026-05-02: Removed the UI test-mode hardcoded `localhost:3000` API base in favor of relative `/api/v1`; standards now reports no high-severity findings.
- 2026-05-02: Stabilized `TestTriggerHeartbeat_DirectExecutionFallback` by waiting for the executor completion callback before allowing `t.TempDir()` cleanup.
- 2026-05-02: Full `make test` reached 10/11 passing phases after the fixes. Unit, playbooks, lint, standards, integration, business, and performance passed; smoke failed only because `ui/src/constants/selectors.manifest.json` had been regenerated after the running bundle started. Restarting prompt-manager via lifecycle and running `vrooli scenario ui-smoke prompt-manager` passed.

## Contract Decisions

- Shared test helpers are test-only infrastructure; production imports are prohibited.
- Fakes should model scenario seams, not internal implementation details.
- Handler tests should validate HTTP contracts and response bodies through shared assertions.
- CLI tests should validate command behavior as users experience it: args, output, exit status, and HTTP contract.
- UI tests should prefer accessible queries and stable selectors from `constants/selectors.ts` where selectors are part of the scenario contract.
- BAS playbooks cover real workflow confidence; unit/component tests cover decision detail.

## Testing Plan

Use these gates during implementation:

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager/cli && go test ./...
cd scenarios/prompt-manager/ui && pnpm test
cd scenarios/prompt-manager && make test
```

For risky refactors:

```bash
cd scenarios/prompt-manager/api && go test ./heartbeat ./teams ./store ./skills ./graph
cd scenarios/prompt-manager/cli && go test ./skills ./search ./graph ./topics ./teams
cd scenarios/prompt-manager/ui && pnpm test -- --run src/components src/hooks src/services src/stores
```

## Rollout Checklist

- [x] Baseline `make test` result captured.
- [x] API testutil package added and documented.
- [x] Production-import meta-test added for API testutil.
- [ ] Duplicate team fixtures consolidated.
- [x] Common API handler helpers adopted in at least three packages.
- [x] CLI command harness added.
- [x] Low-coverage CLI packages receive contract tests.
- [x] UI `@/test` exports routine render/provider/storage/network helpers.
- [x] Accidental UI network calls fail or are explicitly allowed.
- [x] Noisy UI warnings removed or intentionally documented.
- [x] Requirement-critical tests tagged or mapped.
- [ ] Final `make test` passes.

## Risks and Mitigations

- Risk: Consolidating fakes changes test semantics.
  - Mitigation: Start by wrapping existing behavior, migrate one package at a time, and run targeted package tests after each migration.

- Risk: Shared helpers become a dumping ground.
  - Mitigation: Keep subpackages responsibility-based: `fixtures`, `mocks`, `httpx`, `assertx`; avoid generic `helpers.go` files with unrelated concerns.

- Risk: Coverage work becomes shallow.
  - Mitigation: Prioritize low-coverage packages only where behavior is meaningful and tie tests to contracts, commands, or requirements.

- Risk: UI network guard breaks tests that intentionally exercise API behavior.
  - Mitigation: Provide an explicit `allowFetch` or `mockFetch` helper so tests declare intent.

- Risk: BAS runtime costs grow.
  - Mitigation: Add BAS cases sparingly and keep detailed decision matrices in unit/component tests.

## Non-Goals and Prohibited Patterns

- Do not install mocking or coverage packages without explicit approval.
- Do not bypass scenario lifecycle commands.
- Do not create broad snapshot tests as a substitute for behavior assertions.
- Do not move production code solely to satisfy coverage.
- Do not import test utilities from production files.
- Do not duplicate workspace-sandbox helpers blindly; adapt the structure to prompt-manager's seams.

## Definition of Done

The improvement stream is complete when:

- `cd scenarios/prompt-manager && make test` passes.
- Prompt-manager has a documented canonical test utility architecture.
- API package-local fake/fixture duplication is materially reduced.
- CLI low-coverage command domains have meaningful contract tests.
- UI tests have shared provider/storage/network/service harnesses and no accidental network calls.
- Requirement-critical flows are traceable to tests or BAS playbooks.
- Future agents can add tests by following the documented helper structure instead of inventing new local scaffolding.
