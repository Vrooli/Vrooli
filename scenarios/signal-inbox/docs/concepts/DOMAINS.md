# Domains — Signal Inbox

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Signal Inbox is organized around the six product domains in the inventory
below. `health` is an operational support surface, not a product boundary.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/signal-inbox/v1/shared/health.proto` |
| signals | Own the append-only journal of captured material. | Guarantee that anything captured is stored exactly once and never lost. | `signal`, `signal_media`, initial capture annotations. | crud | service | Signal, SourceIdentity, ContentHash | `api/internal/signals/`, `api/handlers/signals/`, `cli/domains/signals/`, `ui/src/features/capture/` |
| sources | Register source adapters, enforce risk tiers, and run imports. | Get material in without risking the operator's platform accounts. | `adapter_state`, `import_run`. | service | workflow | Adapter, RiskTier, SafetyEnvelope, ImportRun | `api/internal/sources/`, `api/handlers/sources/`, `cli/domains/sources/`, `ui/src/features/sources/` |
| enrichment | Resolve a source to text, delegating to the scenario that owns each medium. | Make a signal readable and embeddable regardless of what it arrived as. | `signal_enrichment` append-only extraction attempts. | service | integration | Extractor, ExtractionResult | `api/internal/enrichment/` |
| categories | Own the operator's category set, taxonomies, and classification. | Organize the corpus the way the operator thinks about it, not the way the system does. | `category`, `taxonomy`, `classification`. | crud | service | Category, Taxonomy, Proposal, Confirmation | `api/internal/categories/`, `api/handlers/categories/`, `cli/domains/categories/`, `ui/src/features/categories/` |
| triage | Own disposition, annotations, outcome links, and the review queue. | Make a signal's handled-state and its history legible, so nothing is reconsidered twice. | `disposition`, `annotation`. | workflow | crud | Disposition, Annotation, OutcomeLink | `api/internal/triage/`, `api/handlers/triage/`, `cli/domains/triage/`, `ui/src/features/triage/` |
| retrieval | Own structured query, semantic search, the ambient view, and federation. | Answer both exact and natural-language questions over the whole corpus. | `embedding` index; owns the search descriptor. | reporting | query, integration | Query, AmbientView, ProviderDescriptor | `api/internal/retrieval/`, `api/handlers/retrieval/`, `cli/domains/retrieval/`, `ui/src/features/search/`, `.vrooli/search.json` |

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
| Consumer routing | Per-consumer ownership, event publication, and saved views are P1/P2 work. | A consuming scenario needs routed signals or durable saved views. |

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
