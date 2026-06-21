# Domains — Performance Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships today. The domains in the
inventory below marked **planned** are the bounded contexts this scenario will
build in later implementation phases (see `PRD.md` operational targets and the
`requirements/` modules); they are documented here up front so the architecture
is intentional, but their code does not exist yet in this documentation-first
initialization. The scaffold also ships one clearly fenced worked example domain
(`notes`, never product scope) as a copyable reference; `vrooli scenario
detemplate <scenario>` removes every fenced example once a real domain is green.

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

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/performance-health/v1/shared/health.proto` |
| readiness | Code-facts-gated capture tier detection + perf-build infra detection + format-preserving autofix. | Validation / autofix | No persisted data. | API, CLI, UI | `02-tier-and-readiness` (PH-TIER-001..003) | `api/internal/readiness/`, `api/internal/autofix/`, `cli/domains/readiness/`, `ui/src/features/readiness/` |
| capture | Conduct the profile-mode capture pipeline over BAS perf-capture; tier-0-never-fails; graceful headless skip. | Orchestration | No persisted data. | API, CLI | `03-capture-orchestration` (PH-CAP-001..003) | `api/internal/capture/`, `cli/domains/audit/` |
| analysis | Deterministic trace analysis → located findings + before/after comparison. | Analysis / query | No persisted data. | API, CLI, UI | `04-analysis` (PH-ANALYSIS-001..003) | `api/internal/analysis/`, `cli/domains/audit/`, `ui/src/features/audit/` |
| lighthouse | performance-health-owned Lighthouse runner (own Chrome) with config thresholds + silent-skip. | Capture / query | No persisted data. | API, CLI | `05-lighthouse-and-benchmarks` (PH-LH-001..002) | `api/internal/lighthouse/`, `cli/domains/lighthouse/` |
| benchmark | Build-time (axis ①) benchmarks: time go build + UI build with thresholds. | Measurement | No persisted data. | API, CLI | `05-lighthouse-and-benchmarks` (PH-BENCH-001) | `api/internal/benchmark/`, `cli/domains/benchmark/` |
| budgets | Declarative per-scenario budgets with ratchet + baseline-diff gating. | Policy / gate | Budget config. | API, CLI, UI | `06-budgets-trends-fleet` (PH-BUDGET-001..002) | `api/internal/budgets/`, `cli/domains/budget/`, `ui/src/features/budgets/` |
| trend | Additive SQLite trend store; newest-first reads scoped per scenario. | Persistence / query | Per-run measurements + ExecutionMetrics. | API, CLI, UI | `06-budgets-trends-fleet` (PH-TREND-001) | `api/internal/trend/`, `cli/domains/trend/`, `ui/src/features/trend/` |
| fleet | Deterministic structured offender queries (no budget, slow build, regressed, tier distribution). | Aggregation / query | No persisted data. | API, CLI, UI | `06-budgets-trends-fleet` (PH-FLEET-001) | `api/internal/fleet/`, `cli/domains/fleet/`, `ui/src/features/fleet/` |
| startup | Resource-aware startup benchmark (axis ②, migrated from structure-health) with self-restart guard. | Measurement / persistence | Per-run startup measurements. | API, CLI | `07-startup` (PH-STARTUP-001..002) | `api/internal/startup/`, `cli/domains/startup/` |
| validation | Dual-mount: native services + shared `scenario-validation/v1` provider for Test Genie. | Provider | No persisted data. | API | `01-scenario-boundary` (PH-BOUND-001..003) | `api/handlers/validation/`, `api/internal/assessment/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| None yet. | Generated scaffold. | Add after PRD-specific requirements identify future capability boundaries. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
