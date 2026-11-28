# Integrations — Experience Manager

## Purpose Of This Document

This guide records Experience Manager's declared resources, scenario
dependencies, other scenario relationships, and integration contracts.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

No external Vrooli resources are declared in `.vrooli/service.json`.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None in v1. | deliberate | v1 is zero-ML and local: SQLite + in-repo spec files only. | qdrant/ollama/reranker enter with the P2 search leaf (OT-P2-003), mirroring business-health's search stack. |

## Scenario Dependencies

| Scenario | Enabled | Required | Startup policy | Purpose | Degraded behavior |
|---|---:|---:|---|---|---|
| browser-automation-studio | yes | no | `ignore` | Capture engine for structure reconciliation: drives scenario UIs and captures per-page/state screenshots plus accessibility trees that machine-tier claims are checked against. | Reconciliation checks report skipped (resource unavailable), never failed; spec contract validation, authoring studio, and fleet views work fully without it. |
| image-tools | yes | no | `ignore` | Optional AI-image rendering of spec pages in the workshop (openrouter-image BYOK provider). Deterministic wireframe rendering does not use it. | AI-image render option is unavailable; wireframe rendering and variant compare are unaffected. |

## Other Scenario Relationships

| Scenario | Relationship | Reason | Contract |
|---|---|---|---|
| workflow-health | file-level seam (no runtime call in v1) | Studio scaffolds `bas/cases` stubs from spec entries; workflow-health catalogs, safety-gates, and executes them. Spec↔case reference integrity checked both directions. P2 journey coherence builds on its execution. | Shared `bas/` file conventions + spec entry IDs referenced from cases. |
| business-health | sibling axis (no runtime call) | Validates this scenario's own PRD/requirements (business track); experience-manager is its experience-track mirror. | Canonical PRD template + requirements registry. |
| test-genie | consumer | Discovers and gates the `experience` phase declaratively — no test-genie code changes. | `.vrooli/test-genie.json` (phase, `scenario-validation/v1` contract, presence-keyed applicability, maturity ladders). |
| ui-health | boundary, not dependency | Charter split: ui-health owns built-correctly (including the axe harness and live runtime evidence); experience-manager owns declared intended-experience (accessible name, keyboard reachability, reading order, state affordances, reconciliation, saliency, journeys). No shared findings. | See `docs/internal/DECISIONS.md` boundary decision and [DOC: docs/reference/accessibility-validation.md]. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
