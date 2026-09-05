# Flows — TypeScript Code Graph

This document is the canonical workflow and state-transition map for the scenario. Use it when behavior depends on ordered states, retries, cancellation, stale completion, or background coordination.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain query operations (`/health`, `extract` for a single quiet path) do not need a workflow model. Flows are documented here when ordering matters, when state crosses surfaces, or when a race condition could surface. The Node sidecar adds a non-trivial process-lifecycle flow that all other flows depend on.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Sidecar lifecycle | sidecar | API process boot, or sidecar crash | Sidecar process running and reachable; status visible on `/health`. | Long-running supervisor; spawn-with-backoff; restart on crash. | Level 1 (inventory) — Level 2 model planned alongside chaos tests. |
| Extract | graph | Consumer calls `Extract(project_path)`. | Returns a partial-or-complete Graph + Warnings[], or a typed catastrophic error. | Per-path serialization mutex (Go side + sidecar side); idempotent for the same source state. | Level 1 (inventory) — Level 2 (workflow model) planned. |
| Rewrite plan | rewrite | Consumer calls `RewritePlan(project_path, operations)`. | Returns `plan_id` + normalized operation log. No disk change. | Plan store is in-process; plan IDs derived deterministically. | Level 1 (inventory). Plan-ID stability tested. |
| Rewrite apply | rewrite | Consumer calls `RewriteApply(project_path, plan_id)`. | Executes file moves + import rewrites in order via `ts-morph`'s Project APIs, mutates the filesystem, returns operation log with per-op status. | Per-path serialization mutex; **non-atomic on partial failure**. | Level 1 (inventory). Partial-failure semantics intentionally not rolled back. |

## Flow Details

### Sidecar lifecycle

- **Owner domain**: `sidecar`.
- **Trigger**: API process boot (`main.go` initialization), or sidecar child process exit detected by the supervisor.
- **Inputs**: none — supervisor reads `sidecar/dist/index.js` path from configuration.
- **Steps**:
  1. Supervisor goroutine starts. Initial backoff is zero.
  2. Spawn `node sidecar/dist/index.js` as a child process with stdio piped to the Go side.
  3. Send a handshake: `{type: "handshake", api_version: <semver>}`. Sidecar responds with its own version. Reject incompatible versions.
  4. Mark sidecar `ready`. Begin servicing requests.
  5. Supervisor monitors the child process. On exit:
     - Mark sidecar `unhealthy`. Pending requests get `SidecarUnavailable`.
     - Apply exponential backoff: 100ms, 200ms, 400ms, ... capped at 5 seconds.
     - Respawn. Up to 5 attempts in 60 seconds before giving up and marking `permanently_unhealthy`.
  6. Periodic heartbeat: every 10 seconds, send `{type: "heartbeat"}`. If no response within 5 seconds, kill the child and respawn.
- **Outputs**: sidecar process running; status visible via `/health` and the explorer's diagnostics page.
- **Failure modes**:
  - `permanently_unhealthy` — exhausted restart budget. Manual `make restart` required.
  - `SidecarUnavailable` — returned to in-flight `graph` / `rewrite` requests during the restart gap.
  - `SidecarVersionMismatch` — handshake rejected; never enters service.
- **Retry/cancel behavior**: callers retry their `Extract` / `Rewrite` call after `SidecarUnavailable` — there's nothing the caller can do beyond wait for the supervisor to bring the sidecar back.
- **Tests**: `api/internal/sidecar/supervisor_test.go` (spawn + handshake), `api/internal/sidecar/restart_test.go` (kill child mid-call, verify restart), `api/internal/sidecar/heartbeat_test.go` (stalled-channel detection).

### Extract

- **Owner domain**: `graph`.
- **Trigger**: Connect-RPC `Extract(ExtractRequest{project_path})` call.
- **Inputs**: absolute filesystem path that contains exactly one `tsconfig.json`, or an explicit `tsconfig.json`; selected multi-project workspace roots are unsupported.
- **Steps**:
  1. Acquire per-path mutex (Go side) keyed by `filepath.Abs(project_path)`.
  2. Validate input: reject if no `tsconfig.json` discoverable, multiple `tsconfig.json` files, or the selected project directory is a multi-project workspace root.
  3. Verify sidecar is `ready`. If not, return `SidecarUnavailable`.
  4. Send `{type: "extract", request_id, project_path, options: {...}}` over IPC.
  5. Sidecar acquires its own per-path mutex (because `ts-morph` Project state is not safe to share).
  6. Sidecar constructs a `ts-morph` Project against the `tsconfig.json`, walks source files, emits nodes/edges/leading-comments/warnings.
  7. Sidecar responds with `{type: "extract_result", request_id, graph, warnings}` or `{type: "error", request_id, kind, message}`.
  8. Go side aggregates and normalizes into the shared `Graph` envelope, computes `graph_hash`.
  9. Release Go-side mutex; return.
- **Outputs**: `Graph` (possibly partial) + `Warnings[]` + `graph_hash`, or a typed catastrophic error.
- **Failure modes**:
  - `ExtractError{kind: no_tsconfig_found}` — no `tsconfig.json` discoverable.
  - `ExtractError{kind: multiple_tsconfig_files}` — ambiguous project root.
  - `ExtractError{kind: workspace_unsupported}` — selected project directory is a multi-project workspace root.
  - `ExtractError{kind: path_unreadable}` — filesystem error.
  - `SidecarUnavailable` — sidecar is not `ready`.
  - `SidecarTimeout{request_id, duration}` — IPC stalled past the request timeout.
  - Per-file parse / type-check failures → `Warnings[]`, not errors.
- **Retry/cancel behavior**: caller may retry. Cancellation through `context.Context` cancels the pending IPC request (sidecar receives a `{type: "cancel", request_id}` message).
- **Tests**: `api/internal/graph/extract_test.go` (unit, normalization), `api/internal/graph/concurrent_test.go` (per-path mutex), `api/internal/graph/integration_test.go` (fixture-driven), `api/internal/graph/performance_test.go` (SLA), `bas/fixtures/ts-jsdoc-tags/` integration test (leading-comment contract).

### Rewrite plan

- **Owner domain**: `rewrite`.
- **Trigger**: Connect-RPC `RewritePlan(RewriteRequest{project_path, operations})`.
- **Inputs**: `project_path` plus a list of `FileMove{from, to}` and/or `ImportRewrite{old_path, new_path}` operations.
- **Steps**:
  1. Validate project_path is a valid single-project TS source tree or explicit `tsconfig.json` (same validation as Extract).
  2. Normalize operations: deduplicate, sort by `(kind, from, old_path)`, reject self-moves and cycles.
  3. Compute `plan_id` as a deterministic content hash over the normalized operations. The plan store scopes by `project_path`.
  4. Validate operations: every `FileMove.from` must exist; every `ImportRewrite.old_path` must be imported by at least one source file. Verify by asking the sidecar (`{type: "validate_plan", ...}`).
  5. Validate path traversal: every `FileMove.to` and `ImportRewrite.new_path` must resolve inside the target project's root.
  6. Store the plan in an in-process plan registry keyed by `plan_id` (5-minute TTL).
  7. Return `RewriteResponse{plan_id, operations: <normalized>, dry_run: true}`.
- **Outputs**: `plan_id` + normalized operation list. **No filesystem change.**
- **Failure modes**:
  - `RewriteError{kind: invalid_operations, details}` — duplicates, cycles, or self-moves.
  - `RewriteError{kind: file_not_found, path}` — a FileMove source doesn't exist.
  - `RewriteError{kind: import_path_unused, path}` — an ImportRewrite source path is not imported anywhere.
  - `RewriteError{kind: path_traversal, path}` — destination resolves outside the project root.
  - `SidecarUnavailable` — sidecar is not `ready`.
- **Retry/cancel behavior**: idempotent. Same inputs → same `plan_id`.
- **Tests**: `api/internal/rewrite/plan_test.go` (unit, normalization + plan-ID determinism), `api/internal/rewrite/plan_traversal_test.go` (path-traversal validation).

### Rewrite apply

- **Owner domain**: `rewrite`.
- **Trigger**: Connect-RPC `RewriteApply(RewriteRequest{project_path, plan_id})`.
- **Inputs**: `project_path` and a `plan_id` returned by a previous `RewritePlan` call.
- **Steps**:
  1. Acquire per-path mutex (same lock as Extract).
  2. Look up plan by `plan_id`. Reject if missing/expired.
  3. Recompute plan from current ops; reject if content hash diverges (source moved between plan and apply).
  4. Send `{type: "apply", request_id, plan_id, operations}` to the sidecar.
  5. Sidecar acquires its own per-path mutex.
  6. Sidecar opens a `ts-morph` Project against the `tsconfig.json`.
  7. For each operation in order:
     - `FileMove{from, to}`: `project.getSourceFile(from).move(to)` then `project.save()`. `ts-morph` automatically updates import paths affected by the move and preserves formatting.
     - `ImportRewrite{old, new}`: walk source files, update `ImportDeclaration` paths via `ts-morph`, save.
  8. Each successful op streams back as `{type: "apply_progress", request_id, op_index, status: "applied"}`.
  9. On any op failure, sidecar stops. Sends `{type: "apply_partial", request_id, failed_op, completed_ops}`. **Disk is left in mid-state.**
  10. Go side aggregates per-op statuses into the response operation log.
  11. (P1) Persist the operation log entry to the SQLite Operation Log.
  12. Release mutex; return.
- **Outputs**: `RewriteResponse{plan_id, operations: <with per-op status>, dry_run: false}`.
- **Failure modes**:
  - `RewriteError{kind: plan_expired_or_invalid, plan_id}`
  - `RewriteError{kind: plan_content_mismatch, plan_id}`
  - `RewriteError{kind: apply_partial, failed_op, completed_ops}` — non-atomic mid-apply failure.
  - `SidecarUnavailable` — sidecar crashed before apply completed; disk may be torn.
- **Retry/cancel behavior**: not safely retryable after a partial failure. After git restore, caller must re-plan.
- **Tests**: `api/internal/rewrite/apply_test.go` (unit, per-op semantics), `api/internal/rewrite/apply_integration_test.go` (fixture-driven end-to-end), `api/internal/rewrite/apply_partial_failure_test.go` (panic injection between ops), `api/internal/rewrite/apply_formatting_test.go` (verify Prettier output unchanged after rewrite).

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| sidecar / lifecycle | `not_started`, `spawning`, `handshaking`, `ready`, `unhealthy`, `permanently_unhealthy` | `unhealthy → ready` without respawn, `permanently_unhealthy → ready` without manual restart | Supervisor goroutine state machine |
| rewrite / Plan lifecycle | `proposed`, `applied`, `failed`, `expired` | `failed → applied`, `expired → applied`, `applied → applied` (idempotent apply not supported) | In-process plan registry |
| rewrite / Per-op apply | `pending`, `applied`, `failed`, `not_attempted` | A `failed` op never advances; subsequent ops auto-mark `not_attempted` | apply executor's main loop (Go side) plus sidecar's apply loop |

Level 5 formal models are deferred. The sidecar lifecycle is the most complex flow and a Level 2 model is planned alongside the chaos tests (kill-and-restart). Promote to Level 4–5 if the sidecar's failure modes become harder to reason about as new IPC message types are added.

## Maturity Ladder

| Level | Status |
|---|---|
| 0 — Unmodeled risk | n/a (flows are documented). |
| 1 — Inventory | active (this document). |
| 2 — Workflow model | planned for sidecar lifecycle alongside chaos tests. |
| 3 — Matrix + traces | planned. |
| 4 — Declarative contract (`*.flow.json`) | deferred. |
| 5 — Checked formal model | deferred. |

## Production Shape

Standard react-vite template flow shape applies if/when Levels 4–5 are adopted (see template-inherited content below). For v1, flows live as plain Go code with test coverage; no `flow.json` artifacts yet.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Operation Log replay | Replay would let an operator re-apply a previously successful Rewrite plan after a git restore. | Considered for P2 once REQ-P1-002 lands. |
| Multi-Project sidecar sessions | Currently the sidecar serializes ts-morph Projects internally per call. A cached Project pool could speed up repeated extractions on the same path — but caching needs to be invalidated correctly on source changes. | Revisit if the no-cache decision becomes a measured bottleneck. |
| Sidecar version drift | API and sidecar are deployed together; runtime version skew is unlikely in v1 but possible in future hot-swap scenarios. | Handshake already rejects version mismatch; promote to a tracked flow if hot-swap becomes a real capability. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — matrix and trace testing
