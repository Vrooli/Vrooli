# Integrations — Performance Health

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
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

These are the planned dependencies the real scenario composes (declared in
`.vrooli/service.json` as code lands per the implementation plan). They are not
yet wired in this documentation-first initialization.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| code-facts | planned | Source of actual surfaces + UI framework for code-facts-gated tier detection (with a filesystem fallback that records a degraded reason). | `CodeFactsService.DescribeCodeFacts` over Connect. |
| browser-automation-studio | planned | Raw perf-capture mechanism (CDP trace + web-vitals). performance-health owns tier meaning; BAS stays agnostic. | BAS `capture --capture perf` over Connect. |
| test-genie | planned | Consumes performance-health as its `Performance` phase provider (axes ① build-time + ③ Lighthouse) and runs the provider-contract scan. | shared `scenario-validation/v1 ScenarioValidationService`. |
| structure-health | planned | Relinquishes its `perf` domain — the resource-aware startup benchmark (axis ②) is re-homed here. | greenfield move; no runtime call. |
| cli-health | planned | Registers performance-health's CLI verbs into the search-hub command index for discoverability. | CLI manifest → command index. |

## Shared Packages

| Package | Used For | Contract |
|---|---|---|
| `packages/maturity-go` | Maturity-ladder validation + finding-impact → global signals. | `.vrooli/maturity.json` (provider=`performance-health`). |
| `packages/maturity-go/autofix` | Shared format-preserving readiness autofix substrate. | `FixClass`/`Candidate`/`Fixer`/`Registry`, dry-run default. |
| `packages/api-core/metrics` | `common.v1.ExecutionMetrics` resource envelope for startup measurements. | host-load-aware metrics collector. |
| `packages/vrooli-cli-go` | Typed CLI client for any `vrooli <cmd> --json` calls (e.g. scenario restart/status). | typed cliv1 contracts. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| Lighthouse CLI | planned | The performance-health-owned Lighthouse runner uses its own Chrome (NOT via BAS); silent-skips when absent. | `.vrooli/lighthouse.json` thresholds. |
| Chrome / Browserless | planned (indirect) | Reached via BAS perf-capture and the Lighthouse runner; the audit pipeline skips cleanly when unavailable. | indirect, through BAS / Lighthouse. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
