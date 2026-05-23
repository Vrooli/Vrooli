# Integrations — Go Code Graph

This document is the canonical dependency contract for resources, other scenarios, and third-party services used by the scenario.

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
| `golang.org/x/tools/go/packages` | Go library | yes | `graph` (extraction) | Imported in `api/internal/graph/`; fixed load mode `NeedFiles \| NeedImports \| NeedTypes \| NeedSyntax \| NeedTypesInfo \| NeedName \| NeedDeps`. | Catastrophic load failure surfaces as a typed `ExtractError{kind: project_load_failed}`; partial parse errors surface as `Warnings[]` on the returned graph. |
| Target Go module | external source tree | yes | `graph`, `rewrite` | Absolute path passed in `ExtractRequest.scenario_path` / `RewriteRequest.scenario_path`; must contain exactly one `go.mod`. | Reject with `ExtractError{kind: no_go_mod_found}`, `ExtractError{kind: multiple_go_mod_files}`, or `ExtractError{kind: workspace_unsupported}` per OT-P0-004. |
| SQLite (embedded) | embedded storage | optional (P1 only) | `rewrite` (Operation Log persistence) | `SQLITE_PATH` lifecycle env var; modernc.org/sqlite driver. | If unreachable, Operation Log writes degrade silently with a warning; extraction and rewrite continue. v1 ships without persistence. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets. | Scenario must be started through lifecycle commands. |
| Shared proto: `common/v1/code_graph.proto` | proto schema | yes (build-time) | API, CLI, UI (codegen) | Imported by `packages/proto/schemas/go-code-graph/v1/`. | Codegen failure breaks build; no runtime impact. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Ollama | not-applicable | No embeddings, no LLM calls in v1. Pure deterministic parser. | Revisit only if a future P2 capability (e.g. embedding-based symbol-similarity ranking) is promoted. |
| Qdrant | not-applicable | Same as Ollama. | Same. |
| PostgreSQL | not-applicable | No multi-tenant state; SQLite suffices for the optional Operation Log. | Revisit if cross-tenant/host audit becomes a requirement. |

## Scenario Dependencies

go-code-graph is a **leaf** scenario at runtime — it does not call any other Vrooli scenario. It does, however, have one consumer commitment and a co-owned proto.

### Consumers (informational only — not runtime dependencies)

- **architecture-cartographer** consumes go-code-graph's `Extract` and `Rewrite` via Connect-RPC for its `graph` and `apply` domains. Cartographer's `api/internal/graph/gocodegraph/` adapter holds the client. See cartographer's [`INTEGRATIONS.md`](../../../architecture-cartographer/docs/concepts/INTEGRATIONS.md).

### Co-owned artifacts

- `packages/proto/schemas/common/v1/code_graph.proto` is co-owned with `typescript-code-graph` (planned). Changes to the envelope require coordination across both scenarios. Treat this proto as a stable cross-scenario contract.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | not-applicable | Fully local scenario. No third-party APIs, webhooks, auth providers, or data feeds. | Add only if a future capability genuinely requires one. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| Target Go module missing `go.mod` | `packages.Load` returns no packages | Return `ExtractError{kind: no_go_mod_found, path: <input>}` with a CLI hint to point at the module root. | Unit test against an empty directory fixture. |
| Target Go module has `go.work` | Workspace mode detected by checking for `go.work` in input or ancestors | Return `ExtractError{kind: workspace_unsupported}` with a hint to point at a specific module instead. | Unit test against a `go.work` fixture. |
| Target module fails to type-check | `packages.Load` returns packages with `Errors` populated | Return a **partial graph** with `Warnings[]` per failing file. Do not fail the call. | Integration test against the `bas/fixtures/go-broken-imports/` fixture (planned). |
| Concurrent `Extract` calls for same path | Mutex contention | Second call blocks on the per-path mutex; releases when first completes. No error. | Concurrency test in `api/internal/graph/concurrent_test.go`. |
| Rewrite apply crashes mid-operation | Process exit or panic between op N and op N+1 | Disk left in mid-state. Operator recovers via `git restore .`. Scenario does not roll back. | Integration test simulating panic via injected error after op 2 of 5. |
| Rewrite apply with stale `plan_id` | Plan ID not found in in-process plan store, or content hash mismatch | Return `RewriteError{kind: plan_expired_or_invalid, plan_id}`. Re-plan required. | Unit test. |
| SQLite unreachable (P1) | `PingContext` error | Operation Log writes degrade silently; warning logged. `/health` still reports overall scenario healthy (Operation Log is optional). | Health handler tests. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — handling integration outages
