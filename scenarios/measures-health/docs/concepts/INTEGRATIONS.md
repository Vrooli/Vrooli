# Integrations — Measures Health

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
| SQLite | embedded storage | yes | API, notes reference | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

Declared in `.vrooli/service.json`. Both are optional (`required: false`,
`startup_policy: try_start`) and only the `index` domain needs them — coverage
`validation` works fully without either.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| qdrant | optional | Vector store for the central measures index (collection `measures-health-measures`). Used by the `index` domain. | Required once the `index` provider ships. |
| ollama | optional | `embedding.default` embeddings for the measures index + a small chat role for constrained param extraction (the engine `Completer`). | Required once the `index` provider ships. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| search-hub | optional (try_start) | The `index` domain self-registers its single `measures-health.measures` provider from `.vrooli/search.json` so search-hub routes analytical questions to the central index. | `search-hub providers register` (idempotent upsert via `searchregister-go`); the provider serves the shared `search-hub.v1.control.SearchControlService`. Degraded: registration retries then gives up; validation + query RPCs serve normally. |
| test-genie | consumer | test-genie's `measures` phase shells `measures-health validate scenario <name> --json` and maps findings into `FINDING_SOURCE_MEASURES`. | Public CLI surface (`validate scenario --json`); no runtime coupling — test-genie calls measures-health, not the reverse. |
| target scenarios (any) | read-only subject | `validation` reads each target's `cli/manifest.json` + `packages/proto/schemas/<s>/v1/domain/*.proto`; the `index` execution-proxy POSTs to a target's measures serve endpoint resolved via api-core discovery. | Read-only filesystem + (probe/execute) the target's own measures endpoint. Never client-computes URLs. |

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
