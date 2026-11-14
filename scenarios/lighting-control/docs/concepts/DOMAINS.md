# Domains — Lighting Control

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/lighting-control/v1/shared/health.proto` |
| control | Backend availability, stable controller/zone enrollment, capabilities, static color/brightness/effects, manual Off, and desired-state reconciliation. | Operators need safe, repeatable control despite backend/device reconnects. | Enrollment and desired static state. | service | query | Controller, Zone, Capability, DesiredState | `api/internal/control/`, `api/handlers/control/`, `cli/domains/control/`, `packages/proto/schemas/lighting-control/v1/control/` |
| policy | Presets, telemetry inputs, effective policy, freshness, hysteresis, debounce, brightness cap, manual priority, expiry, and static fallback. | Operators need useful automatic behavior with bounded and explainable inputs. | Preset configuration and active manual/policy request. | service | reporting | Preset, PolicyRequest, Evidence | `api/internal/policy/`, `api/handlers/policy/`, `cli/domains/policy/`, `packages/proto/schemas/lighting-control/v1/policy/` |

## Domain Details

### control

- Owns enrollment, stable identity mapping, capabilities, desired color/brightness/effect, reconciliation triggers, and command result truth.
- Physical observation is an optional separate fact. A successful SDK acknowledgement is never converted into an observed-output claim.
- Reconciliation is event/lifecycle driven and bounded; periodic work must not rescan devices per frame.
- Durable identity uses OpenRGB serials or reported device locations; transient SDK handles only route a write after resolving the durable identity at discovery. Missing or duplicate identity fails closed. Zone identity uses name/count fingerprints, not zone indexes.

### policy

- Owns configurable Off/load/work/health/combined presets and the explanation of the effective choice.
- Uses system-monitor readings and active Agent Manager run records through their existing APIs. Missing or stale sources produce timestamped fallback evidence and static policy.
- High utilization alone is not machine-health failure. Manual Off outranks automatic policy; expiring requests return to the configured policy.
- Capability gaps resolve to a static safe fallback and are surfaced per zone.

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
