# Domains — Extraction Range

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real infrastructure domain the scaffold ships. The
`range` domain owns challenges, target pages and scoring. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `template-manager detemplate <scenario>` removes every
fenced example once your real domains are green.

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

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/extraction-range/v1/shared/health.proto` |
| range | Publish ten families of owned extraction challenges; accept tool submissions and return deterministic scores. | Let agents compare extraction quality on known-answer pages before client work. | Level JSON files, held expected records, best score per tool/level in the current process. | service | scoring, experiment | Level, Result, Tool | `api/internal/rangeengine/`, `api/handlers/range/`, `cli/domains/range/`, `packages/proto/schemas/extraction-range/v1/range/`, `api/internal/rangeengine/levels/` |
| _(your domain)_ | What product capability it owns. | Why the capability exists for users or operators. | Its tables, if any. | service | — | DomainVocabulary | `api/internal/<domain>/` |

## Domain Details

### range

- Purpose: provide an owned website extraction benchmark with answer-key grading.
- Primary archetype: service / practice ground; its `/practice/{id}` pages are the target sites.
- Secondary traits: deterministic scoring and a composable data corpus.
- Owns: the challenge data schema and JSON files, task briefs, public practice-page behavior, matching/score calculation, and persistent best scores.
- Does not own: the client extraction toolkit, browser automation runtime, real-site crawling, or real-site bot handling.
- API: Connect `RangeService` plus documented REST routes `GET /levels`, `GET /levels/{id}`, `POST /score`, `GET /scoreboard`; crawler target pages are `/practice/{id}`.
- CLI: `range list`, `range get`, `range submit`, `range scoreboard`.
- UI: operator workbench provides family filtering, challenge details, record submission, score results and the saved scoreboard; experience validation is still being completed.
- Storage: level definitions and held answers are embedded JSON. Best scores use the range domain's SQLite table and survive API process restarts.
- Requirements: `EXTRACTI-P0-001` through `EXTRACTI-P0-003` in `requirements/`.
- Tests: `api/internal/rangeengine/engine_test.go` and `api/handlers/range/site_test.go`.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md), [`../internal/SEAMS.md`](../internal/SEAMS.md).

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
