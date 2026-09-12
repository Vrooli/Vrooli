# Flows — Go Code Graph

This document is the canonical workflow and state-transition map for the scenario. Use it when behavior depends on ordered states, retries, cancellation, stale completion, or background coordination.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain query operations (`/health`, `extract` for a single quiet path) do not need a workflow model. Flows are documented here when ordering matters, when state crosses surfaces, or when a race condition could surface.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Extract | graph | Consumer calls `Extract(module_path)`. | Returns a partial-or-complete Graph + Warnings[], or a typed catastrophic error. | Per-path serialization mutex; idempotent for the same source state. | Level 1 (inventory) — Level 2 (workflow model) planned alongside concurrent-extraction tests. |
| Rewrite plan | rewrite | Consumer calls `RewritePlan(module_path, operations)`. | Returns `plan_id` + normalized operation log. No disk change. | Plan store is in-process; plan IDs derived deterministically from normalized op list. | Level 1 (inventory). Plan-ID stability tested. |
| Rewrite apply | rewrite | Consumer calls `RewriteApply(module_path, plan_id)`. | Executes file moves + import rewrites in order, mutates the filesystem, returns operation log with per-op status. | Per-path serialization mutex; **non-atomic on partial failure** — operator recovers via git. | Level 1 (inventory). Partial-failure semantics intentionally not rolled back. |

## Flow Details

### Extract

- **Owner domain**: `graph`.
- **Trigger**: Connect-RPC `Extract(ExtractRequest{module_path})` call from cartographer, CLI, or UI explorer.
- **Inputs**: absolute filesystem path that contains exactly one `go.mod` and is not inside a `go.work` workspace.
- **Steps**:
  1. Acquire per-path mutex keyed by `filepath.Abs(module_path)`.
  2. Validate input: reject if no `go.mod` discoverable, multiple `go.mod` files, or `go.work` workspace detected.
  3. Configure `packages.Config` with load mode `NeedFiles | NeedImports | NeedTypes | NeedSyntax | NeedTypesInfo | NeedName | NeedDeps`, exclude `vendor/` by default (REQ-P1-003).
  4. Call `packages.Load(...)`. Capture both `Packages` and per-package `Errors`.
  5. For each loaded package, walk top-level declarations and emit nodes (`go_type`, `go_func`, `go_var`, `go_const`, `go_interface`, `go_method`).
  6. For each import statement, emit an `imports` edge.
  7. For intra-package references between top-level decls, emit `references` edges.
  8. Aggregate parse errors and unresolved-import errors into `Warnings[]`.
  9. Compute `graph_hash` (deterministic content hash over the normalized graph).
  10. Serialize to the shared `Graph` proto envelope (sorted by deterministic keys).
  11. Release mutex; return.
- **Outputs**: `Graph` (possibly partial) + `Warnings[]` + `graph_hash`, or a typed catastrophic error.
- **Failure modes**:
  - `ExtractError{kind: no_go_mod_found}` — no `go.mod` discoverable.
  - `ExtractError{kind: multiple_go_mod_files}` — ambiguous module root.
  - `ExtractError{kind: workspace_unsupported}` — `go.work` detected.
  - `ExtractError{kind: path_unreadable}` — filesystem error.
  - Per-file parse / type-check failures → `Warnings[]`, not errors.
- **Retry/cancel behavior**: caller may retry. The per-path mutex serializes concurrent identical calls (the second call blocks, does not error). Cancellation through `context.Context` propagates to `packages.Load` where supported.
- **Tests**: `api/internal/graph/extract_test.go` (unit, normalization), `api/internal/graph/concurrent_test.go` (per-path mutex), `api/internal/graph/integration_test.go` (fixture-driven), `api/internal/graph/performance_test.go` (SLA).

### Rewrite plan

- **Owner domain**: `rewrite`.
- **Trigger**: Connect-RPC `RewritePlan(RewriteRequest{module_path, operations})`.
- **Inputs**: `module_path` plus a list of `FileMove{from, to}` and/or `ImportRewrite{old_path, new_path}` operations.
- **Steps**:
  1. Validate module_path is a valid single-module Go project (same validation as Extract).
  2. Normalize operations: deduplicate, sort by `(kind, from, old_path)`, reject self-moves and cycles.
  3. Compute `plan_id` as a deterministic content hash over the normalized ops + module_path.
  4. Validate operations against the current filesystem: every `FileMove.from` must exist; every `ImportRewrite.old_path` must be present in at least one `.go` file under the module.
  5. Store the plan in an in-process plan registry keyed by `plan_id` (5-minute TTL).
  6. Return `RewriteResponse{plan_id, operations: <normalized>, dry_run: true}`.
- **Outputs**: `plan_id` + normalized operation list. **No filesystem change.**
- **Failure modes**:
  - `RewriteError{kind: invalid_operations, details}` — duplicates, cycles, or self-moves.
  - `RewriteError{kind: file_not_found, path}` — a FileMove source doesn't exist.
  - `RewriteError{kind: import_path_unused, path}` — an ImportRewrite source path is not imported anywhere.
- **Retry/cancel behavior**: idempotent. Same inputs → same `plan_id`.
- **Tests**: `api/internal/rewrite/plan_test.go` (unit, normalization + plan-ID determinism).

### Rewrite apply

- **Owner domain**: `rewrite`.
- **Trigger**: Connect-RPC `RewriteApply(RewriteRequest{module_path, plan_id})`.
- **Inputs**: `module_path` and a `plan_id` returned by a previous `RewritePlan` call.
- **Steps**:
  1. Acquire per-path mutex (same lock as Extract — only one mutation in flight per path).
  2. Look up plan by `plan_id`. Reject if missing/expired.
  3. Recompute plan from current ops; reject if content hash diverges from `plan_id` (source moved between plan and apply).
  4. For each operation in order:
     - `FileMove{from, to}`: `os.Rename`, then walk all `.go` files in the module and update any import paths affected by the move via Go AST rewriting.
     - `ImportRewrite{old, new}`: walk all `.go` files in the module, update import declarations via Go AST rewriting, write modified files back.
  5. For each successful op, append to the response operation log with status `applied`.
  6. On any op failure, stop. Record the failed op with status `failed` and remaining ops with status `not_attempted`. **Disk is left in mid-state.** Operator recovers via git.
  7. (P1) Persist the operation log entry to the SQLite Operation Log.
  8. Release mutex; return.
- **Outputs**: `RewriteResponse{plan_id, operations: <with per-op status>, dry_run: false}`.
- **Failure modes**:
  - `RewriteError{kind: plan_expired_or_invalid, plan_id}` — plan TTL expired or never existed.
  - `RewriteError{kind: plan_content_mismatch, plan_id}` — filesystem changed between plan and apply.
  - `RewriteError{kind: apply_partial, failed_op, completed_ops}` — non-atomic mid-apply failure. Disk is mid-state; the caller (and operator) must reconcile via git.
- **Retry/cancel behavior**: not safely retryable after a partial failure. After git restore, caller must re-plan (the plan may not match a clean tree).
- **Tests**: `api/internal/rewrite/apply_test.go` (unit, per-op semantics), `api/internal/rewrite/apply_integration_test.go` (fixture-driven, FileMove + ImportRewrite end-to-end), `api/internal/rewrite/apply_partial_failure_test.go` (panic injection between ops to verify mid-state semantics).

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| rewrite / Plan lifecycle | `proposed`, `applied`, `failed`, `expired` | `failed → applied`, `expired → applied`, `applied → applied` (idempotent apply not supported in v1) | In-process plan registry rejects stale/unknown plan IDs |
| rewrite / Per-op apply | `pending`, `applied`, `failed`, `not_attempted` | A `failed` op never advances; subsequent ops auto-mark `not_attempted` | apply executor's main loop |

Level 5 formal models (`flow.json` + Quint) are deferred. The flows here are small enough that the test matrix and partial-failure scenarios cover correctness. Promote to Level 5 if cross-call sequencing complexity grows (e.g. when the Operation Log adds replay).

## Maturity Ladder

| Level | Status |
|---|---|
| 0 — Unmodeled risk | n/a (flows are documented). |
| 1 — Inventory | active (this document). |
| 2 — Workflow model | planned for Rewrite apply once partial-failure tests exist. |
| 3 — Matrix + traces | planned alongside Level 2. |
| 4 — Declarative contract (`*.flow.json`) | deferred. |
| 5 — Checked formal model | deferred. |

## Production Shape

Standard react-vite template flow shape applies if/when Levels 4–5 are adopted (see template-inherited content below). For v1, flows live as plain Go code with test coverage; no `flow.json` artifacts yet.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| Operation Log replay | Replay would let an operator re-apply a previously successful Rewrite plan after a git restore. | Considered for P2 once REQ-P1-002 lands and operators report needing it. |
| Concurrent multi-path extraction | Already implemented (parallel across paths) but worth a Level 2 model if regressions appear. | Promote when concurrency-related bugs are reported. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md) — matrix and trace testing
