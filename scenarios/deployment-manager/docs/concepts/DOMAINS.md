# Domains — Deployment Manager

This document is the canonical ownership map for deployment-manager. Every
requirement validation reference must land in one of these product domains;
transport, storage, and shared test infrastructure are not product domains.

## Domain Inventory

| Domain | Responsibility | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| analysis | Inspect dependency graphs and calculate target fitness. | reporting | No product data. | API, CLI, UI | OT-P0-001..OT-P0-006, OT-P0-035..OT-P0-037 | `api/dependencies/`, `api/fitness/`, `cli/overview/`, `cli/domains/overview/`, `ui/src/features/dependencies/` |
| swaps | Recommend and apply dependency substitutions. | mutation | Profile swap configuration. | API, CLI | OT-P0-007..OT-P0-011 | `api/swaps/`, `cli/swaps/`, `cli/domains/swaps/` |
| profiles | Create and manage target deployment profiles and their configuration. | crud | `profiles`, profile versions. | API, CLI, UI | OT-P0-012..OT-P0-017, OT-P1-016..OT-P1-019 | `api/profiles/`, `api/internal/profiles/`, `cli/profiles/`, `cli/domains/profiles/`, `ui/src/features/profiles/` |
| secrets | Classify, template, and validate deployment secrets without storing values. | provider | Secret references and templates only. | API, CLI | OT-P0-018..OT-P0-022 | `api/secrets/`, `cli/domains/secrets/`, `cli/signing/` |
| validation | Validate profile readiness and deployment prerequisites. | validation | Validation references only. | API, CLI | OT-P0-023..OT-P0-027 | `api/profiles/`, `cli/validations/`, `cli/domains/validations/` |
| deployments | Govern releases, approvals, target verdicts, and explicit deployment refusals. | orchestration | Approvals, releases, evidence references. | API, Connect, CLI, UI | OT-P0-028..OT-P0-034, OT-P1-009..OT-P1-014, OT-P1-024..OT-P1-036, OT-P2-001..OT-P2-029 | `api/deployments/`, `api/internal/deployments/`, `api/internal/evidence/`, `api/internal/releases/`, `api/releases/`, `api/handlers/evidence/`, `cli/deployments/`, `cli/domains/deployments/`, `cli/releases/`, `cli/domains/releases/`, `ui/src/features/releases/`, `ui/src/features/evidence/` |
| monitoring | Ingest and present post-deployment telemetry and operational status. | reporting | Telemetry references and summaries. | API, UI | OT-P1-001..OT-P1-008 | `api/telemetry/`, `ui/src/features/telemetry/` |
| orchestration | Coordinate target-specific release workflows and agent handoffs. | orchestration | Release workflow state. | API, CLI | OT-P1-020..OT-P1-023 | `api/migrationtasks/`, `api/deployments/`, `cli/approvals/`, `cli/domains/approvals/` |

## Shared Concepts

Profiles, commits, target verdicts, evidence references, approvals, and release
records are shared vocabulary. Domain packages may consume their contracts but
must not silently assume ownership of another domain's storage.

## Non-Domains

- `api/server/` and `api/internal/database/` are composition and persistence
  infrastructure.
- `cli/internal/` and `ui/src/components/` are shared test/presentation
  infrastructure.
- The common evidence proto is a cross-ramp contract, not a deployment-manager
  product domain.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- [`../../requirements/README.md`](../../requirements/README.md)
