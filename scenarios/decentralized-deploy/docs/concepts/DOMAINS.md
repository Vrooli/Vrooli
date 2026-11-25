# Domains — Decentralized Deploy

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `template-manager detemplate <scenario>` removes every
fenced example once your real domains are green.


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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/decentralized-deploy/v1/shared/health.proto` |
| deployment | Coordinate storage and naming adapter operations and retain deployment state. | Let Vrooli scenarios deploy, resolve, renew and remove portable service artifacts with provider-specific behavior behind replaceable contracts. | Deployment records and adapter/path references; persistent storage is not wired yet. | service | external-integration | Deployment, StoredContent, NamedContent, EffectGate | `api/internal/deployment/`, `data/provider-path-catalog.json`, future CLI/API handlers |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/decentralized-deploy/v1/notes/` |

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
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### deployment

- Purpose: own a complete storage/name operation and its lifecycle receipts.
- Primary archetype: service / orchestration.
- Secondary traits: external integration, idempotent teardown, gated effects.
- Owns: deployment state, storage and naming adapter contracts, effect decisions, fault hooks, provider/path fixture catalogue.
- Does not own: compute provider selection or provisioning; the caller supplies a Compute Manager reference.
- API: planned receipt-bearing deploy, resolve, renew and teardown operations; no API handler is wired yet.
- CLI: planned `deploy`, `resolve`, `renew`, `teardown` commands; not wired yet.
- UI: generated template UI is placeholder until a deployment lifecycle view is implemented.
- Storage: in-memory operation map in the initial library; durable repository not yet implemented.
- Tests: lifecycle contract, effect gate refusal, fault injection, compensation and real offline Kubo test are in `api/internal/deployment/*_test.go`.
- Related docs: [`ARCHITECTURE.md`](ARCHITECTURE.md), [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

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
| None yet. | No additional domain is in scope beyond storage/name deployment, effect gating and provider/path catalogue. | Revisit when a concrete requirement introduces an independently owned capability. |

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
