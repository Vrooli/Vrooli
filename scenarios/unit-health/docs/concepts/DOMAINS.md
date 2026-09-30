# Domains — Unit Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The template's starter domain was removed. Every domain below is
product scope.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | reporting | No product data. | API, UI | Scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/unit-health/v1/health/` |
| validation | Validate a scenario's test maturity: discover surfaces, plan and optionally run tests, analyze coverage/architecture/quality, assess maturity. | reporting | Validation run history, cached evidence. | API, CLI, UI | `requirements/` modules tied to `PRD.md`. | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validate/`, `ui/src/features/validation/`, `packages/proto/schemas/unit-health/v1/validation/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/` (`GET /health`, also `/api/v1/health`).
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### validation

- Purpose: answer "how mature is this scenario's unit testing?" for one
  scenario or filesystem path, with normalized findings and a shared
  maturity assessment.
- Primary archetype: analysis / report. One entrypoint,
  `ValidationService.ValidateScenario`, composes an internal pipeline.
- Pipeline (all inside `api/internal/validation/`, driven by
  `service.go`):
  1. Discovery — `api/internal/discovery/` asks Code Facts for surfaces
     and parse units, then normalizes them into test workspaces.
  2. Plan — `plan.go` derives per-workspace canonical test commands and
     degraded-state findings.
  3. Execution (optional, `include_execution`) — `api/internal/executor/`
     runs commands under a per-command timeout, no-output watchdog,
     process-group cleanup, and weighted admission caps.
  4. Analyzers — `coverage.go` (Go profiles / LCOV / Vitest summaries),
     `architecture.go` (co-location, shared test-utils, injectable
     seams), `quality.go` + `native_quality.go` (the 7-rule catalog in
     `api/internal/testquality/catalog.json`), `reliability.go` and
     `diagnostics.go` (flake and runtime-growth signals from run history),
     `requirement_registry.go` (requirement traceability),
     `policy.go` (unit policy projections and waivers).
  5. Assessment — the `maturity` block of `.vrooli/test-genie.json`
     defines capability ladders (`surface_discovery`,
     `execution_readiness`, `framework_config`, `test_architecture`,
     `coverage_quality`, `stability_traceability`); findings map onto
     them through `maturity-go/assessment`.
- Owns: `ValidateScenarioRequest`/`Response`, findings and finding
  codes, execution plans, coverage targets, test-quality and
  traceability reports, evidence cache keys, and run-history rows.
- Does not own: the shared `scenario_validation.v1` contract (owned by
  the platform proto; this domain implements it), Code Facts discovery
  semantics, or the maturity ladder vocabulary in `common.v1`.
- API: `api/handlers/validation/` mounts the scenario-local
  `ValidationService` and the shared `ScenarioValidationService`
  (`ValidateScenario`, `ValidateTarget`, `DescribeProvider`,
  `PreviewFix`, `ApplyFix`).
- CLI: `cli/domains/validate/` — `unit-health validate scenario <name>`
  with `--path`, `--workspace`, `--execution`, `--fast-test-only`,
  `--json`, bound declaratively in `cli/manifest.json`.
- UI: `ui/src/features/validation/ScenarioValidationWorkbench.tsx`,
  `ui/src/api/validation.ts`.
- Storage: `api/internal/runhistory/schema.sql` (run timing/status) and
  the filesystem evidence cache under the scenario cache directory.
- Requirements: see `requirements/` and `PRD.md`.
- Tests: analyzer, executor, discovery, handler, CLI manifest coverage,
  UI workbench, and end-to-end validation tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Finding | One normalized validation observation with a code, severity, workspace, and evidence. | validation domain, `validation.proto`. |
| Maturity ladder | Per-capability rung definitions in the `maturity` block of `.vrooli/test-genie.json`. | validation domain (spec), `maturity-go` (assessor). |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

The sub-capabilities below live inside the `validation` domain today as
packages or files rather than separate domains. Split one out only when
it needs its own proto service, storage, or surface.

| Candidate Domain | Current Home | Split Trigger |
|---|---|---|
| discovery (Code Facts intake) | `api/internal/discovery/` | A second consumer besides validation. |
| execution | `api/internal/executor/` | Execution needs its own RPC or scheduling state. |
| testquality | `api/internal/testquality/` (rule catalog) | Rules need to be authored or served independently. |
| diagnostics / runhistory | `api/internal/runhistory/`, `diagnostics.go` | History queries need their own read API. |
| maturity (assessor) | `maturity-go/assessment` reading the `maturity` block of `.vrooli/test-genie.json` | Never inside this scenario; the assessor is shared. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If any of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
