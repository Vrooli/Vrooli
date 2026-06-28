# Domains — Brand Manager

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/brand-manager/v1/shared/health.proto` |
| brands | Own brand entities + immutable version history (CRUD, optimistic concurrency, idempotency). | The core authored artifact every other domain references. | `brands`, `brand_versions` tables. | crud | service | Brand, BrandVersion | `api/internal/brands/`, `cli/domains/brands/`, `ui/src/features/brands/`, `packages/proto/schemas/brand-manager/v1/brands/` |
| assignments | Link a brand+version to a target scenario and track what/when/which-version was applied (incl. partial element application). | Records which scenario wears which brand. | `assignments` table. | crud | service | Assignment | `api/internal/assignments/`, `packages/proto/schemas/brand-manager/v1/assignments/` |
| assets | Store and serve brand binary assets (logo, favicon, icon) on the filesystem with metadata in SQLite. | Durable home for generated/imported brand imagery. | `assets` table + `~/.vrooli/brand-manager/assets/{brand_id}/`. | storage | service | Asset | `api/internal/assets/`, `packages/proto/schemas/brand-manager/v1/assets/` |
| contrast | Compute WCAG AA contrast ratios for color pairings. | Keep authored palettes accessible; shared by generation + validation. | No data (pure algorithm). | computation | — | ContrastRatio | `api/internal/contrast/`, `packages/proto/schemas/brand-manager/v1/contrast/` |
| generation | AI-generate brand text (palette, typography, copy) + images (logo, favicon) via the AIProviderChain (Ollama→OpenRouter). | Make producing a compliant brand cheap. | No data (calls providers; results persisted via brands/assets). | integration | service | AIProviderChain | `api/internal/generation/`, `packages/proto/schemas/brand-manager/v1/generation/` |
| apply | Apply brand elements to a target scenario: CSS custom properties with `/* brand-manager:<element> */` markers, `manifest.json` `_brand` keys, favicon path, atomic asset copy; export root `DESIGN.md`. | Turns an assigned brand into real files in a scenario. | No data (writes target scenario files). | service | workflow | ApplyEngine, DesignExport | `api/internal/apply/`, `packages/proto/schemas/brand-manager/v1/apply/` |
| discovery | Scan an existing scenario's state (service.json, theme/token files, static assets, manifests, DESIGN.md) and propose a draft brand with confidence scores; pluggable per-framework scanners. | Bootstrap brands from what scenarios already have. | No data (proposes drafts). | analysis | service | Scanner, DiscoveryResult | `api/internal/discovery/`, `packages/proto/schemas/brand-manager/v1/discovery/` |
| validation | **Headline.** Serve `ScenarioValidationService` (ValidateScenario/PreviewFix/ApplyFix) so test-genie runs a `branding` delegated phase: severity-gated branding findings + maturity ladder + deterministic auto-fix. Findings flow on `FINDING_SOURCE_BRANDING`. | Every scenario's branding is continuously validated + auto-fixed inside the standard test loop. | No data (scans targets). | validation | service | BrandingRule, MaturityLadder | `api/internal/validation/`, `api/validation_connect.go`, `packages/proto/schemas/brand-manager/v1/validation/` |

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
