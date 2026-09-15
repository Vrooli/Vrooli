# Decisions — Brand Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-27 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-27 | **One coherent scenario** owns both branding authoring AND branding validation. | The scenario that generates/manages branding is the natural owner of validating it. The alternative — splitting into an authoring app + a separate validation app — was explicitly rejected by the user. | All domains (brands, generation, apply, discovery, validation) live in `brand-manager`. Validation is not a separate scenario. | Revisit only if the validation surface grows so large it warrants its own lifecycle. |
| 2026-06-27 | **Branding validation is a test-genie delegated phase**, not a scenario-auditor `external_rules` provider. | scenario-auditor is being phased out (hard to author/maintain rules, poor discoverability). The modern validation contract is `ScenarioValidationService` + the test-genie phase registry. | brand-manager implements `ScenarioValidationService`; test-genie registers a `branding` delegated phase on a new `FINDING_SOURCE_BRANDING` channel. No `external_rules_brand_manager.go`. This rebuild is one step in retiring scenario-auditor. | Revisit if the test-genie delegated-phase contract is itself replaced. |
| 2026-06-27 | **Regenerate the scenario shell from `react-vite`; port the transport-agnostic brains** rather than refactor in place. | The prior REST/gorilla-mux build (a full generation behind: no proto/Connect, programmatic CLI, flat handlers, plain-fetch UI) did not build (missing cli-core replace) and carried an orphaned validation surface. Regen replaces transport + orphan code for free. | Reusable Go (aigen/contrast/repository/apply/discovery/DESIGN-export algorithms) is ported into `api/internal/<domain>/`; all gorilla handlers, the REST UI client, and `audit_provider.go`/`standards.go`/`lighthouse.go` are discarded. | n/a (one-time rebuild). |
| 2026-06-27 | **SQLite is declared as embedded, not as a managed resource.** | The resource catalog (`.vrooli/schemas/resource-definitions.json`) has no `sqlite` resource — SQLite is the embedded `modernc.org/sqlite` library reached via the `SQLITE_PATH` env. Declaring a non-existent resource would fail schema validation. | `service.json` declares the AI provider resources (`ollama`, `openrouter`, both optional/graceful) and the `SQLITE_PATH`/`OLLAMA_URL`/`OPENROUTER_API_KEY` env; it does NOT declare a `sqlite` resource. | Revisit if a managed `sqlite` resource type is ever added to the catalog. |
| 2026-06-27 | **Defer agent-assisted apply + Lighthouse integration.** | The prior tree had agent-manager (P1 agent-assisted apply) and Lighthouse (P2) integrations. Neither is part of the core validation-as-phase goal. | No agent-manager dependency declared yet (P2 in PRD); no Lighthouse target. Keeps the rebuild scoped. | Revisit when agent-assisted apply is actually built (then declare agent-manager). |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
