# Domains — Development Toolchain Validator

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`notes` is a worked example from the template, retained as a reference pattern. All P0 DTV domains (OT-P0-001 through OT-P0-008) are implemented and listed in the Domain Inventory below; the core validate-skill-against-golden loop was proven live end-to-end 2026-05-29 (see [`../internal/PROGRESS.md`](../internal/PROGRESS.md)).

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
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/development-toolchain-validator/v1/health/` |
| notes | Worked CRUD reference with attachment upload exception. | CRUD / entity | Notes and attachment metadata. | API, CLI, UI | Template starter only. | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/development-toolchain-validator/v1/notes/` |
| golden | Register, list, update, delete, and regenerate template-pristine golden scenarios. | CRUD / entity | `goldens` table (slug, template, version pin, path, timestamps). | API, CLI, UI | OT-P0-001 (Golden Registry & Regeneration). | `api/internal/golden/`, `api/handlers/golden/`, `cli/domains/golden/`, `ui/src/features/golden/`, `packages/proto/schemas/development-toolchain-validator/v1/golden/` |
| skill_catalog | Mirror prompt-manager's steer-skill catalog (id, version, content hash); expose drift. | Integration / sync | `skill_catalog` table. | API, CLI, UI | OT-P0-002 (Skill Catalog Sync). | `api/internal/skill_catalog/`, `api/integrations/prompt_manager/`, `cli/domains/skill_catalog/` |
| manifest | Per-(skill, golden) expected-diff manifest with allowed paths, convergence, and template+skill version pins. | CRUD / entity | `manifests` table + clear-stale overrides. | API, CLI, UI | OT-P0-003 (Expected-Diff Manifest). | `api/internal/manifest/`, `cli/domains/manifest/`, `ui/src/surfaces/manifests/` |
| validation_run | Execute a (skill\|tool, golden) run under sandboxed agent-manager; capture diff + summary; evaluate against the manifest; produce a verdict. | Orchestration / workflow | In-flight run state (terminal records persisted by validation_record). | API, CLI, UI | OT-P0-004, OT-P0-005. | `api/internal/validation_run/`, `api/integrations/agent_manager/`, `api/integrations/dev_tools/`, `ui/src/surfaces/runs/` |
| validation_record | Append-only history of terminal verdicts with metrics, version pins, and diff hash. | Storage / ledger | `validation_records` table. | API, CLI, UI | OT-P0-006 (Validation Record Storage). | `api/internal/validation_record/`, `cli/domains/record/` |
| staleness | Derive manifest staleness from template/skill version drift; clear-stale override. | Reporting / derivation | Derived (reads manifests + catalog + overrides). | API, CLI, UI | OT-P0-007 (Staleness Tracking). | `api/internal/staleness/`, `cli/domains/staleness/` |
| report | Read-only roll-ups: golden summary, tuple history, coverage grid. | Reporting / query | Derived (composes the domains above). | API, CLI, UI | OT-P0-008 (Validation Report API). | `api/internal/report/`, `api/handlers/report/`, `cli/domains/report/` |

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

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Planned Domains

All P0 domains derived from the PRD have landed and are listed in the Domain Inventory above (verified live 2026-05-29 — see [`../internal/PROGRESS.md`](../internal/PROGRESS.md)). Tooling-baseline validation (OT-P0-005) is implemented inside the `validation_run` domain (the tool runner is a sibling seam to the agent-manager runner), not as a separate domain.

| Domain | PRD Ref | Purpose |
|---|---|---|
| None pending. | — | All P0 domains are implemented; future P1/P2 capabilities (template-version-watcher, skill-maturity-score, convergence-tracking, trend-detection, coverage-map, bulk-revalidation, etc.) live in `requirements/11-21` and will be added here when their slices land. |

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
