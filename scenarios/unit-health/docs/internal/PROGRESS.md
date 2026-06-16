# Progress — Unit Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-16 | agi | done | Phase 0 + Phase 1 (product contract). Generated from react-vite. Authored PRD, 5 requirements modules, `.vrooli/maturity.json` (L0–L5 + 24 finding codes), `docs/reference/maturity.md`, planned-domain map in DOMAINS.md, service.json deps. API builds green. Example `notes` domain intentionally left in place (compiling scaffold) — see handoff below. |
| 2026-06-16 | agi | done | Phase 4 (bounded test execution engine). New `internal/executor` package: `Bounded` runner (per-command timeout, no-output watchdog via `tailWriter`, process-group kill through `Cmd.Cancel`+`WaitDelay`=3s backstop — fixes Go inherited-pipe hang, unix/other build-tagged `setProcessGroup`/`killGroup`, 8KiB tail excerpts, `GOWORK=off`+`CI=1` env), `RunAll` (order-preserving bounded concurrency), failure classification (test_failure/missing_dependency/misconfiguration/timeout_hang/no_output_stall/system). Service gains `Executor`/`MaxConcurrency`; `Validate` runs the plan only when `req.IncludeExecution`, populates `CommandResults`, maps non-pass results → TEST_EXECUTION_FAILURE/TEST_DEPENDENCY_MISSING/TEST_TIMEOUT_HANG/TEST_MISCONFIGURATION findings (added to `codeSeverity`). Plan now executes the coverage command (Go `-coverprofile=coverage.out`, vitest `test:coverage`) when available. CLI shows an Execution section. Tests: real sh-fixture executor tests (pass/nonzero/timeout/no-output-stall/missing-cmd/RunAll-order, 2s) + fake-executor service tests (skip-without-flag/failure→failed/timeout→hang/pass). Validation: api+cli `go test ./...` green; LIVE `unit-health validate scenario uhsmoke --path /tmp/uhsmoke2 --execution` ran `go test -covermode=atomic -coverprofile=coverage.out ./...` → passed exit=0 125ms. NOTE: coverage artifacts land in conventional in-workspace locations (Go `coverage.out`, vitest `coverage/`); Phase 5 reads them there. |
| 2026-06-16 | agi | done | Phase 3 (Code Facts intake + test plan builder). New `internal/discovery` package = Code Facts client (mirrors quality-health `surfaces.go`: `Discoverer`/`Locator`/`CodeFactsClient`/`DefaultLocator`, surfaces+parse_units describe, filesystem fallback w/ explicit DegradedReason, added Python indicator detection + `cli-core` go.mod replace for `api-core/discovery`). New `internal/validation/plan.go` = workspace resolver + dry-run plan: canonical-framework per language (Go `go test`, React/Vite `vitest`, Python `pytest` degraded), package.json manifest reader, degraded/noncanonical detectors → 7 finding codes (TEST_SURFACE_ABSENT, UNSUPPORTED_PARSE_UNIT, TEST_FRAMEWORK_MISSING, TEST_FRAMEWORK_NONCANONICAL, COVERAGE_CONFIG_MISSING, PACKAGE_MANAGER_MISMATCH, TEST_MISCONFIGURATION). Service now injects `Discoverer`/`Locator`/`Spec`, computes local maturity via `assessment.LocalMaturity`, derives status (passed/failed/degraded). CLI human output now shows Workspaces + Test plan; exit nonzero on error findings. Tests: fake-discoverer service tests (go ready / jest noncanonical→failed L1 / vitest ready / python degraded / no-surface degraded L0) + fake-CodeFactsReport discovery tests. Validation: api+cli `go test ./...` green; LIVE `unit-health validate scenario unit-health` + `code-facts` (human + --json) → passed, L5, 3 workspaces, real Code Facts intake, schema-valid JSON. |
| 2026-06-16 | agi | done | Phase 2 (Proto/API/CLI foundations). Authored `unit-health/v1/validation/validation.proto` (`ValidationService.ValidateScenario`, full domain message set: TestSurface/TestWorkspace/ExecutionPlan/CommandResult/CoverageTarget/ValidationFinding/Diagnostic/MaturitySummary/ValidationCounts + `common.v1.MaturityAssessment`); regenerated proto (Go/TS/Py + descriptor). Deleted `notes` proto. Swapped `notes`→`validation` across API (`handlers/validation` + `internal/validation` skeleton service + registry + main.go, dropped the measures block), CLI (`domains/validate`, manifest `validate scenario`, gen-endpoints seed), and UI (`api/validation.ts` + `features/validation/ScenarioValidationWorkbench`, deleted `features/notes`/`api/notes.ts`/`NotesPage`). Added `maturity-go` dep to api/go.mod; tidied cli/go.mod. Added missing `RESTExceptionPayloads` type + health-endpoint `proto_payloads` (template was older than quality-health's). **example-domain-removed gate PASSES.** Validation: api+cli `go test ./...` green; UI lint+test+build green; `proto-health validate scenario unit-health` passed (L5, 0 err); `cli-health validate scenario unit-health` passed (L5, 0 err); live `unit-health validate scenario unit-health` (human + --json) returns a schema-valid, honestly-degraded stub assessment. |

## Phase 5 Handoff (next agent — start here)

**State:** Discovery, dry-run plan, AND bounded execution are landed and green
(Phases 3–4). `unit-health validate scenario <name> [--execution]` discovers
surfaces, plans canonical commands, runs them bounded (timeout/no-output/
process-group), returns `CommandResults` + execution findings, and computes
local maturity. What's still missing is **analysis of artifacts**: coverage
parsing, test-architecture checks, test-quality checks, flake/runtime
diagnostics, and a richer maturity assessor.

**Engine layout (as built):**
- `api/internal/discovery/discovery.go` — Code Facts client.
- `api/internal/executor/{executor,watch,procattr_*}.go` — bounded runner.
- `api/internal/validation/plan.go` — `buildPlan` + finding codes + `codeSeverity`.
- `api/internal/validation/service.go` — `Service{Discoverer, Locator, Spec,
  Executor, MaxConcurrency, Now}`; `Validate` orchestrates discover → plan →
  (optional) execute → status → local maturity. `execute()` + `executionFinding()`
  live here.
- `api/handlers/validation/handler.go` flat-copies `Response`→proto + builds the
  shared `common.v1.MaturityAssessment` (coverage-dimension codes →
  `FINDING_SOURCE_COVERAGE`, else `FINDING_SOURCE_STANDARDS`).

**Phase 5 steps (per plan §7 Phase 5) — analyzers + maturity assessor:**
1. **Coverage analyzer** (`internal/analyzers/coverage` or similar): after
   execution, parse Go cover profiles (`coverage.out` written in each Go
   workspace dir) and Vitest/LCOV summaries (`<ui>/coverage/`). Populate
   `Response.Coverage` (`CoverageTarget` per file/surface) and emit `LOW_COVERAGE`
   / `COVERAGE_ABSENT` (dimension=coverage). Wire a per-scenario threshold
   (default e.g. 0 to start; read `.vrooli/testing.json` if present).
2. **Test architecture analyzer** (static, source-fact driven, no execution
   needed): co-location, `internal/testutil`/`src/test-utils` presence, no test
   helper imported from production (there's already a `no_prod_import_test.go`
   convention in the template), injectable seams. Emit TEST_NOT_COLOCATED,
   TEST_UTIL_MISSING, TEST_HELPER_FROM_PRODUCTION, MISSING_INJECTABLE_SEAM.
3. **Test quality analyzer:** skipped/only tests, no/weak assertions, render-only
   tests, excessive snapshots, untagged requirements, missing edge cases. Emit
   TEST_SKIPPED_OR_ONLY, TEST_NO_ASSERTION, TEST_RENDER_ONLY,
   TEST_EXCESSIVE_SNAPSHOTS, TEST_MISSING_EDGE_CASES, TEST_UNTAGGED_REQUIREMENT.
4. **Diagnostics/history analyzer:** populate `Response.Diagnostics` from command
   durations + hang evidence; emit TEST_FLAKE_SUSPECTED, TEST_RUNTIME_GROWTH.
5. **Maturity assessor:** already wired via `assessment.LocalMaturity` in
   `service.assessMaturity`; just ensure every NEW finding code has a
   `codeSeverity` entry AND a `.vrooli/maturity.json` mapping (all Phase-5 codes
   already exist in maturity.json — only add `codeSeverity` entries). Consider
   validating at startup that every spec code is reachable.
6. Fixture-heavy tests per analyzer.

All finding codes for Phase 5 already exist in `.vrooli/maturity.json` (LOW_COVERAGE,
COVERAGE_ABSENT, TEST_NOT_COLOCATED, TEST_UTIL_MISSING, TEST_HELPER_FROM_PRODUCTION,
MISSING_INJECTABLE_SEAM, TEST_SKIPPED_OR_ONLY, TEST_NO_ASSERTION, TEST_RENDER_ONLY,
TEST_EXCESSIVE_SNAPSHOTS, TEST_MISSING_EDGE_CASES, TEST_FLAKE_SUSPECTED,
TEST_RUNTIME_GROWTH, TEST_UNTAGGED_REQUIREMENT) — they just need producers + `codeSeverity`.

**Status vocabulary:** `passed` / `failed` (any error finding) / `degraded`.
CLI exits nonzero when `counts.errors > 0`. Keep stable for Test Genie (Phase 7).

**Restart gotcha:** the scenario compiles from source on `vrooli scenario
restart unit-health`; the CLI auto-rebuilds per-invocation but the **API server
binary does not** — restart the scenario after API changes before live testing.

**Known pre-existing scaffolding debt (NOT introduced by Phase 2; shared with
sibling quality-health):**
- Full `vrooli scenario test unit-health` STANDARDS phase is red (~32 high) —
  the fleet-wide freshly-generated-template standards campaign, not this work.
- `dependencies` phase: governance-review warnings for the scenario's deps
  (need recording in approved-dependency memory) — template baseline.
- `proto` phase WARNINGS only (errors=0 now): `errors`/`health` protos still
  carry `@template react-vite/example` (sibling does too); `errors` domain has
  no handler dir. Accepted scaffold state.
- `docs/manifest.json` nav still lists the old `notes` CRUD domain in a couple
  of entries (lines ~884/924/1416); the `bas/cases/routed-database/` case still
  references `@selector/notes.list`. These are doc/UI-test scaffolding — clean
  up when the UI is built out (Phase 6) and during doc passes; smoke/playbooks
  currently pass regardless.

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
