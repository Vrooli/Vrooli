# Integrations — Experience Manager

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None in v1. | deliberate | v1 is zero-ML and local: SQLite + in-repo spec files only. | qdrant/ollama/reranker enter with the P2 search leaf (OT-P2-003), mirroring business-health's search stack. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| browser-automation-studio | declared (optional, `try_start`) | Single capture engine for reconciliation: per-page/state screenshot + accessibility tree. | `.vrooli/service.json` dependency; BAS Connect API. Degraded: reconciliation records unavailable evidence and never marks a claim satisfied. |
| image-tools | declared (optional, `on_demand`) | Optional AI-image rendering of spec pages in the workshop via its `openrouter-image` BYOK provider. Wireframe rendering never uses it. | `.vrooli/service.json` dependency; image-tools API. Degraded: AI render option unavailable; wireframe/compare unaffected. |
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
