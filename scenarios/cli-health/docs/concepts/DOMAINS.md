# Domains — CLI Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`notes` is a worked example from the template, not product scope.
Replace it after the first real domain is green.

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
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/cli-health/v1/health/` |
| validation | Validate every adopting scenario's `cli/manifest.json` against the schema and proto descriptors; emit findings for schema errors, unresolved bindings, orphan proto methods, and stale omission entries. | Query / report | Per-run validation reports (transient; not persisted in v1). | API, CLI, UI | OT-P0-001 | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validate/`, `ui/src/features/validation/`, `packages/proto/schemas/cli-health/v1/validation/` |
| search | Cross-scenario semantic and text search for CLI commands; indexes manifest entries plus `--help` fallback parses. | Search / index | Qdrant collection `cli-health-commands` (role-resolved dense vectors, cosine); transient indexed-record cache. | API, CLI, UI | OT-P0-002, OT-P0-003 | `api/internal/search/`, `api/internal/aisearch/`, `api/handlers/search/`, `cli/domains/search/`, `ui/src/features/search/`, `packages/proto/schemas/cli-health/v1/search/` |
| reindex | On-demand and scheduled rebuild of the search index; reconciler plus 5-minute sync loop. | Job / orchestration | Reindex job state (in-memory). | API, CLI, UI | OT-P0-004 | `api/internal/reindex/`, `api/handlers/reindex/`, `cli/domains/reindex/`, `ui/src/features/reindex/`, `packages/proto/schemas/cli-health/v1/reindex/` |
| notes | Worked CRUD reference; removed in Gate 7. | CRUD / entity (template only) | Notes and attachment metadata. | API, CLI, UI | None (template starter). | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/cli-health/v1/notes/` |

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

### validation

- Purpose: validate every adopting scenario's `cli/manifest.json`; emit findings for schema errors, unresolved bindings, orphan proto methods, and stale omission entries.
- Primary archetype: query / report.
- Owns: validation report construction, finding model (`severity`, `code`, `location`, `message`, `suggestion`).
- Does not own: persistent storage (reports are transient in v1); auto-fix logic (read-only by design per `feedback_no_git_mutations`).
- API: `api/internal/validation/`, `api/handlers/validation/`.
- CLI: `cli/domains/validate/`.
- UI: `ui/src/features/validation/`.
- Storage: none in v1.
- Requirements: OT-P0-001.
- Tests: per-finding unit tests, integration tests against all 6 adopting scenarios, deliberately-broken fixture regressions.

### search

- Purpose: semantic and text-mode cross-scenario CLI command discovery; primary substrate that replaces "grep the repo for an existing command".
- Primary archetype: search / index.
- Owns: indexed command records, embedding text composition, Qdrant collection schema.
- Does not own: natural-language-to-command UI (prompt-manager owns that); reindex orchestration (lives in `reindex` domain).
- API: `api/internal/search/`, `api/internal/aisearch/`, `api/handlers/search/`.
- CLI: `cli/domains/search/`.
- UI: `ui/src/features/search/`.
- Storage: Qdrant collection `cli-health-commands` (dimensions resolved from `embedding.default`, cosine, payload-hash drift detection).
- Requirements: OT-P0-002, OT-P0-003.
- Tests: discovery-source unit tests (manifest scan, `--help` fallback parser), embedding-stub stability test, integration test with real Qdrant, text-fallback test, recall@5 ≥ 0.8 on a hand-labeled query corpus at `testdata/search_queries.json`.

### reindex

- Purpose: on-demand rebuild of the search index plus scheduled reconciliation; mirrors prompt-manager's reconciler shape.
- Primary archetype: job / orchestration.
- Owns: reindex job state, sync-loop cadence, plan/apply phase coordination.
- Does not own: vector storage (Qdrant), embedding generation (Ollama), or the indexed-record schema (lives in `search`).
- API: `api/internal/reindex/`, `api/handlers/reindex/`.
- CLI: `cli/domains/reindex/`.
- UI: `ui/src/features/reindex/`.
- Storage: in-memory job state only.
- Requirements: OT-P0-004.
- Tests: reconciler test asserting one upsert and zero deletes after a single-manifest mutation; sync-loop cadence test.

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
