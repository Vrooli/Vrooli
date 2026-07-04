# Domains — Experience Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `vrooli scenario detemplate <scenario>` removes every
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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/experience-manager/v1/shared/health.proto` |
| _(your domain)_ | What product capability it owns. | Why the capability exists for users or operators. | Its tables, if any. | service | — | DomainVocabulary | `api/internal/<domain>/` |

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
| `spec` | Experience spec contract: parser for `experience/` (index, pages, journeys), JSON-schema + cross-file reference validation, tier/open-world semantics, frozen finding codes. The keystone domain — everything else consumes it. | OT-P0-001 implementation start. |
| `provider` | `ScenarioValidationService` mount for the `experience` phase: findings, metrics, maturity ladders, fix preview/apply; declarative discovery via `.vrooli/test-genie.json`. | OT-P0-002. |
| `reconcile` | BAS-driven capture (screenshot + a11y tree per page/state) and machine-tier claim checks with per-claim evidence; skipped-never-failed degradation. | OT-P0-003, after BAS ships a11y-tree capture. |
| `studio` | Form-based spec authoring sessions with live validation and the zero-findings round-trip property; CLI parity. | OT-P0-004. |
| `render` | Deterministic wireframe rendering with claim annotations, N-variant compare, promote-to-spec; optional AI-image mode composing image-tools. | OT-P1-001. |
| `scaffold` | Derive `bas/cases` stubs from spec entries for workflow-health governance; spec↔case reference-integrity findings both directions. | OT-P1-002. |
| `attest` | Manual-tier claim attestation ledger with expiry; expired attestations surface as findings. | OT-P1-004. |
| `fleet` | Compute-on-read sweep of spec coverage/depth across all scenarios, worst-first. | OT-P1-005. |
| `perception` | Pixel-side parsing + saliency importance vs. declared communication priorities; advisory before gating, off the CI hot path. Engine home (here vs. image-tools) deliberately undecided. | OT-P2-001. |

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
