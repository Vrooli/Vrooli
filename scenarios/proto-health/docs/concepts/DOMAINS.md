# Domains — Proto Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`notes` is a worked example from the template, not product scope.
Keep it as a reference until the first real domain is green, then
delete it in the same change that proves the replacement pattern.

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
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/proto-health/v1/health/` |
| impact | Compare proto contracts against baselines and report downstream file-level blast radius. | Advisory impact analysis | No persisted product data. | API, CLI | PROTO-SURFACES-008 | `api/internal/impact/`, `api/handlers/impact/`, `cli/domains/impact/`, `packages/proto/schemas/proto-health/v1/impact/` |
| validation | Validate one scenario's proto contract structure and return stable findings. | Validator / findings producer | No persisted product data in v1. | API, CLI, UI, test-genie | PROTO-VAL-001, PROTO-VAL-002 | Planned: `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validation/`, `ui/src/features/validation/`, `packages/proto/schemas/proto-health/v1/validation/` |
| protosurface | Read descriptor/repo facts and expose one scenario's proto inventory. | Fact surface / read model | No persisted product data in v1. | API, CLI, UI, downstream analyzers | PROTO-SURF-001, PROTO-SURF-002 | Planned: `api/internal/protosurface/`, `api/handlers/protosurface/`, `ui/src/features/protosurface/`, `packages/proto/schemas/proto-health/v1/protosurface/` |
| notes | Worked CRUD reference with attachment upload exception. | CRUD / entity | Notes and attachment metadata. | API, CLI, UI | Template starter only. | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/proto-health/v1/notes/` |

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

### notes

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### impact

- Purpose: compare current proto contracts against a baseline and make
  breaking-change blast radius visible before promotion.
- Primary archetype: advisory impact analysis.
- Secondary traits: baseline resolver, Buf compatibility parser,
  consumer reconciliation report.
- Owns: impact RPC/CLI, baseline scope resolution, Buf breaking command
  execution, compatibility classification, and v1 file-level consumer
  reconciliation.
- Does not own: git-control-tower baseline storage, symbol-level
  consumer proof, or code-facts/code-graph language analysis.
- API: `api/handlers/impact/`.
- CLI: `cli/domains/impact/`.
- Storage: none required; baselines are read from git and
  git-control-tower.
- Requirements: PROTO-SURFACES-008.
- Tests: impact service unit tests over parser, scope metadata, and
  consumer reconciliation.
- Related docs: [`../guides/contract-impact.md`](../guides/contract-impact.md).

### validation

- Purpose: convert documented proto conventions into executable,
  stable findings for one scenario at a time.
- Primary archetype: validator / findings producer.
- Secondary traits: maturity signal producer, test-genie phase source.
- Owns: finding codes, severity tiers, check orchestration, and
  suggestions for local proto issues.
- Does not own: cross-scenario graph analysis, dependency manifest
  drift, or authoritative dead-proto detection.
- API: planned `ProtoHealthService.ValidateScenario`.
- CLI: planned `proto-health validate scenario <name> --json`.
- UI: planned per-scenario findings view.
- Storage: none required in v1.
- Requirements: PROTO-VAL-001, PROTO-VAL-002.
- Tests: table-driven checker tests over fake descriptor/repo facts.

### protosurface

- Purpose: expose a reusable structured inventory of one scenario's
  proto surface.
- Primary archetype: fact surface / read model.
- Secondary traits: downstream analyzer interface, descriptor
  introspection substrate.
- Owns: descriptor loading model, scenario-owned file filtering,
  service/RPC/message/field/import/annotation inventory, adoption
  signals, and transport-world facts.
- Does not own: validation severity decisions or fleet graph
  computation.
- API: planned `ProtoHealthService.DescribeScenarioProtos`.
- CLI: planned `proto-health describe scenario <name> --json`.
- UI: planned surface detail panels.
- Storage: none required in v1.
- Requirements: PROTO-SURF-001, PROTO-SURF-002.
- Tests: deterministic fixture-to-surface assertions.

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
| search provider | Semantic proto reuse search is out of scope for v1. | `DescribeScenarioProtos` is stable and downstream consumers need indexed reuse suggestions. |
| dependency graph | Belongs to scenario-dependency-analyzer, not proto-health. | Downstream plan consumes proto surface facts. |

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
