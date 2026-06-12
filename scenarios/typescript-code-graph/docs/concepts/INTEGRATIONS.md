# Integrations — TypeScript Code Graph

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
| `ts-morph` (npm) | Node library | yes | Node sidecar (delegated to by `graph`, `rewrite`) | Pinned major version in `sidecar/package.json`. | If the sidecar can't load `ts-morph`, spawn fails; `sidecar` reports `unhealthy`; API returns `SidecarUnavailable` to all `graph` and `rewrite` calls. |
| Node runtime (≥20.x) | Local runtime | yes | Node sidecar | Provided by the react-vite template's lifecycle. | If Node is missing or below required version, spawn fails; same SidecarUnavailable surface. |
| Target TS project | external source tree | yes | `graph`, `rewrite` | Absolute path passed in `ExtractRequest.project_path` / `RewriteRequest.project_path`; may be a project root or explicit `tsconfig.json` and must resolve to exactly one TS project. | Reject with `ExtractError{kind: no_tsconfig_found}`, `ExtractError{kind: multiple_tsconfig_files}`, or `ExtractError{kind: workspace_unsupported}` per OT-P0-005. |
| SQLite (embedded) | embedded storage | optional (P1 only) | `rewrite` (Operation Log persistence) | `SQLITE_PATH` lifecycle env var; modernc.org/sqlite driver. | If unreachable, Operation Log writes degrade silently; extraction and rewrite continue. v1 ships without persistence. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets. | Scenario must be started through lifecycle commands. The sidecar is spawned as a child of the API process; lifecycle has no direct knowledge of it. |
| Shared proto: `common/v1/code_graph.proto` | proto schema | yes (build-time) | API, CLI, UI (codegen) | Imported by `packages/proto/schemas/typescript-code-graph/v1/`. Co-owned with `go-code-graph`. | Codegen failure breaks build; no runtime impact. |
| IPC channel (stdio or Unix socket) | local IPC | yes | `sidecar` | JSON-over-stdio (or local Unix socket per platform). Defined in `api/internal/sidecar/protocol.go` + `sidecar/src/protocol.ts`. | Channel stall → request-level timeout returns typed timeout error. Channel close → supervisor restarts the sidecar with backoff. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Ollama | not-applicable | No embeddings, no LLM calls in v1. Pure deterministic parser. | Revisit only if a future P2 capability (e.g. embedding-based symbol-similarity ranking) is promoted. |
| Qdrant | not-applicable | Same as Ollama. | Same. |
| PostgreSQL | not-applicable | No multi-tenant state; SQLite suffices for the optional Operation Log. | Revisit if cross-tenant/host audit becomes a requirement. |

## Scenario Dependencies

typescript-code-graph is a **leaf** scenario at runtime — it does not call any other Vrooli scenario. It has two consumer commitments and a co-owned proto.

### Consumers (informational only — not runtime dependencies)

- **architecture-cartographer** consumes typescript-code-graph's `Extract` and `Rewrite` via Connect-RPC for its `graph` and `apply` domains. Cartographer's `api/internal/graph/tscodegraph/` adapter holds the client. See cartographer's [`INTEGRATIONS.md`](../../../architecture-cartographer/docs/concepts/INTEGRATIONS.md).
- **react-component-library** (planned consumer) will migrate off its current regex-based JSDoc tag scraping onto this scenario's `Extract` + leading-comment metadata. Tracked in this scenario's PRD as OT-P1-006 (cross-scenario coordination commitment). The rcl-side work executes inside the react-component-library scenario.

### Co-owned artifacts

- `packages/proto/schemas/common/v1/code_graph.proto` is co-owned with `go-code-graph`. Changes to the envelope require coordination across both scenarios. Treat this proto as a stable cross-scenario contract.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | not-applicable | Fully local scenario. No third-party APIs, webhooks, auth providers, or data feeds. | Add only if a future capability genuinely requires one. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| Target TS project missing `tsconfig.json` | sidecar's Project initialization fails | Return `ExtractError{kind: no_tsconfig_found, path: <input>}` with a CLI hint to point at the project root. | Unit test against an empty directory fixture. |
| Target TS project has multiple `tsconfig.json` at same level | sidecar detects ambiguity | Return `ExtractError{kind: multiple_tsconfig_files}` with a hint to point at a specific tsconfig. | Unit test. |
| Target TS project is part of pnpm workspace | sidecar detects `pnpm-workspace.yaml` ancestor | Return `ExtractError{kind: workspace_unsupported}` with a hint to point at a specific project. | Unit test. |
| Target project fails to parse | sidecar emits per-file parse errors | Return a **partial graph** with `Warnings[]` per failing file. Do not fail the call. | Integration test against the `bas/fixtures/ts-broken-imports/` fixture (planned). |
| Concurrent `Extract` calls for same path | Mutex contention at Go side; additional serialization inside sidecar | Second call blocks on the per-path mutex; releases when first completes. No error. | Concurrency test in `api/internal/graph/concurrent_test.go`. |
| Rewrite apply crashes mid-operation | Process exit or panic between op N and op N+1, or sidecar crash mid-apply | Disk left in mid-state. Operator recovers via `git restore .`. Scenario does not roll back. Sidecar is restarted by the supervisor. | Integration test simulating panic via injected error after op 2 of 5; sidecar-kill test. |
| Sidecar crashes during normal operation | Supervisor goroutine detects child exit | Restart with exponential backoff (max 5 attempts in 60 seconds). `health` reports `degraded` during the gap. In-flight requests return `SidecarUnavailable`. | Chaos test: kill sidecar mid-call; verify restart and recovery. |
| Sidecar IPC stalls | Per-request timeout (default 30s) | Return `SidecarTimeout{request_id, duration}`. Supervisor probes the sidecar via heartbeat; if heartbeat also stalls, kill and restart. | Chaos test: inject latency at the sidecar's stdio writer. |
| Sidecar repeatedly fails to start | Supervisor exhausts restart budget | `health` reports `unhealthy`. API returns `SidecarUnavailable` until manual intervention (`make restart`). | Integration test: corrupt the sidecar binary; verify failure surfaces. |
| Rewrite apply with stale `plan_id` | Plan ID not found in in-process plan store, or content hash mismatch | Return `RewriteError{kind: plan_expired_or_invalid, plan_id}`. Re-plan required. | Unit test. |
| SQLite unreachable (P1) | `PingContext` error | Operation Log writes degrade silently; warning logged. `/health` still reports overall scenario healthy (Operation Log is optional). | Health handler tests. |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — handling integration outages
