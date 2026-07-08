# Domains — Cleanup Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`cleanup` and `health` are the current domains. Keep this inventory aligned
with proto, API, CLI, UI, and test ownership as the cleanup surface grows.

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
| cleanup | Build preview-first cleanup plans, enforce policy gates, apply approved provider actions, and record audit events. | Give operators and agents one safe orchestration point for reclaimable storage. | Policy snapshots, plans, apply attempts, audit events. | service | mutation | Provider, Plan, PolicyProfile, AuditEvent | `api/internal/cleanup/`, `api/internal/orchestrator/`, `api/internal/providers/`, `api/handlers/cleanup/`, `cli/domains/cleanup/`, `packages/proto/schemas/cleanup-manager/v1/cleanup/` |
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/cleanup-manager/v1/shared/health.proto` |

## Domain Details

### cleanup

- Purpose: orchestrate cleanup without letting providers mutate host state
  outside preview, policy, approval, idempotency, and audit contracts.
- Primary archetype: service / orchestration.
- Secondary traits: policy enforcement, safety tiers, idempotent mutation,
  provider registry, audit trail.
- Owns: provider metadata contracts, policy profiles, plans, apply attempts,
  audit events, cleanup CLI commands, and cleanup UI console panels.
- Does not own: private deletion rules inside owner scenarios or raw host
  mutation APIs.
- API: `api/internal/cleanup/`, `api/internal/orchestrator/`,
  `api/internal/providers/`, `api/handlers/cleanup/`.
- CLI: `cli/domains/cleanup/`.
- UI: dashboard cleanup console in `ui/src/pages/DashboardPage.tsx`.
- Storage: in-memory Phase 4 store; SQLite persistence is deferred.
- Requirements: CLN-P0-001 through CLN-P0-005.
- Tests: provider metadata, policy profiles, provider previews, orchestrator,
  Connect handler, CLI registration, and UI console tests.

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
- Requirements: operational readiness support for cleanup-manager.
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
| Persistent cleanup store | Current Phase 4 store is in memory. | Add when retention and replay durability are required beyond process lifetime. |

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
