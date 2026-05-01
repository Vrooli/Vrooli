# System Invariants

This document captures critical invariants that must be maintained across all changes to the Swarm Manager. These are the foundational guarantees that enable safe operation, retries, and predictable behavior.

## Replay/Idempotency Invariants

Understanding how operations behave under retries is critical for building robust systems. This section documents the idempotency characteristics of each state-mutating operation.

### Key Definitions

- **Idempotent**: Running the operation multiple times produces the same result as running it once
- **Replay-Safe**: The operation can be safely retried without causing data corruption or duplicates
- **Conflict-Signaling**: The operation returns a clear signal (e.g., 409) when a duplicate is attempted

### API Operations

| Operation | Idempotent | Replay-Safe | Signal | Notes |
|-----------|------------|-------------|--------|-------|
| `GET /api/v1/backlog` | ✅ Yes | ✅ Yes | N/A | Read-only |
| `GET /api/v1/backlog/{kind}/{name}` | ✅ Yes | ✅ Yes | N/A | Read-only |
| `POST /api/v1/backlog` | ❌ No | ✅ Yes | 409 Conflict | Duplicate name returns 409 |
| `PUT /api/v1/backlog/{kind}/{name}` | ⚠️ Partial | ✅ Yes | 200 OK | Same data, but `Updated` timestamp changes |
| `DELETE /api/v1/backlog/{kind}/{name}` | ✅ Yes | ✅ Yes | 204 No Content | Returns 204 whether resource exists or not |

### Operation Details

#### Create (POST /api/v1/backlog)

**Idempotency Key**: `name` field (sanitized to folder-safe format)

**Behavior under replay**:
1. First call: Creates backlog item, returns 201 Created
2. Subsequent calls: Returns 409 Conflict (resource exists)

**Code location**: `api/internal/backlog/handler.go:123-187`

**Test coverage**: `TestCreate_ReplaySafe` in `handler_test.go`

**Why this is safe**: The 409 response tells the client that the resource already exists. The client can then fetch the existing resource to get the created data. This pattern is superior to silently succeeding because it helps detect programming errors where duplicate creates might indicate a bug.

#### Update (PUT /api/v1/backlog/{kind}/{name})

**Idempotency**: Partial - core data is idempotent, but `Updated` timestamp always changes

**Behavior under replay**:
1. First call: Updates backlog item, sets new `Updated` timestamp, returns 200 OK
2. Subsequent calls: Same data values, but `Updated` timestamp reflects latest call

**Code location**: `api/internal/backlog/handler.go:189-229`

**Test coverage**: `TestUpdate_Idempotent` in `handler_test.go`

**Why timestamp change is acceptable**: The `Updated` field is meant to track "when was this last modified" which is technically true - a retry is still a modification event. If exact idempotency is required, use an `If-Match` header with ETags (not currently implemented).

**Future enhancement**: Consider adding ETag support for conditional updates.

#### Delete (DELETE /api/v1/backlog/{kind}/{name})

**Idempotency**: ✅ Full - calling delete twice produces the same result

**Behavior under replay**:
1. First call: Removes backlog item if exists, returns 204 No Content
2. Subsequent calls: Resource already gone, returns 204 No Content

**Code location**: `api/internal/backlog/handler.go:231-254`

**Test coverage**: `TestDelete_Idempotent` in `handler_test.go`

**Why 204 instead of 404**: Returning 404 for "already deleted" breaks idempotency. If a client's first DELETE request succeeded but the response was lost (network error), the client would retry and get 404, which could be misinterpreted as "resource never existed" rather than "delete was successful". By always returning 204, we signal "the resource is definitely not there" which is the desired end state.

### File System Operations

All file system operations are inherently idempotent at the OS level:

| Operation | Idempotent | Notes |
|-----------|------------|-------|
| `os.MkdirAll` | ✅ Yes | No-op if directory exists |
| `os.WriteFile` | ✅ Yes | Overwrites existing file atomically |
| `os.RemoveAll` | ✅ Yes | No error if path doesn't exist |
| `os.Stat` | ✅ Yes | Read-only check |

**Note**: While `os.WriteFile` is idempotent, it's not atomic across crashes. If the write is interrupted, the file could be corrupted. For production use, consider write-to-temp-then-rename pattern.

### UI Considerations

Currently, the UI only uses read operations (queries). When mutations are added, they should follow these patterns:

1. **Use React Query's `useMutation`** with proper `onSuccess`/`onError` handlers
2. **Disable submit button during mutation** to prevent double-clicks
3. **Show pending state** so users know the operation is in progress
4. **Invalidate queries on success** to refresh stale data
5. **Handle 409 Conflict** for create operations as "already exists" (may redirect to existing resource)

### Safe Retry Patterns

When implementing retry logic:

1. **GET requests**: Always safe to retry
2. **DELETE requests**: Always safe to retry (idempotent)
3. **PUT requests**: Safe to retry with same payload (core data is idempotent)
4. **POST requests**: Check for 409 Conflict - indicates resource was created on previous attempt

### Risky Patterns to Avoid

1. **Auto-generated IDs on POST**: If the server generates an ID, retries create duplicates. Use client-provided idempotency keys.
2. **Non-atomic multi-step operations**: If step 2 fails, step 1 may need to be undone. Use transactions or compensating actions.
3. **Silent success on duplicate**: Always signal when an operation was a no-op due to existing state.
4. **Time-based deduplication**: Using "only process if timestamp > last_processed" can miss retries. Use explicit idempotency keys.

### Testing Idempotency

All idempotency tests follow this pattern:

```go
// 1. Perform operation (may set up initial state)
// 2. Assert first call behavior
// 3. Perform same operation again
// 4. Assert second call produces same (or acceptable) result
// 5. Verify no duplicate state was created
```

Tests are tagged with `[REQ:REQ-P0-002]` for requirement traceability.

### Terminal State Writers (closed loop)

Backlog item terminal statuses (`completed`, `failed`, `needs_followup`) form a closed loop with exactly two writers:

| Direction | Writer | Trigger | Audit record |
|-----------|--------|---------|--------------|
| Forward (active → terminal) | `backlog.Handler.ReviewDecide` | `POST /api/v1/backlog/{kind}/{name}/review-decide` | `review/decisions/{ts}-{accept\|fail\|followup}.json` |
| Backward (terminal → in_progress) | `backlog.Handler.reopenForRetry` | called only from `backlog.Handler.Retry` (`POST /api/v1/backlog/{kind}/{name}/retry`) | `review/decisions/{ts}-reopen.json` |

**Invariant:** no other code path may transition an item into or out of a terminal status. The `update_patch.go` PATCH handler explicitly rejects status changes when the existing status is in a review-gated state, and rejects user-driven flips into terminal states without going through `review-decide`. The `executionQueuer` API surface deliberately exposes `RetryLatestForBacklog` (which calls into `execution.Service.Retry`) but the *item-level* status flip stays inside `backlog.Handler` so the audit-record write and event emission cannot be skipped.

**Why this matters:** stats math depends on `EmitBacklogStatusChanged` firing on every transition; the audit folder is the only durable history of *why* the item moved. Skipping either breaks observability. New writers must justify themselves and update this table.

### Retry-as-New-Attempt

`execution.Service.Retry` and `execution.Service.RetryLatestForBacklog` create a *new* `Record` parented to the prior one (`ParentExecutionID = parent.ExecutionID`). The parent record's `Status`, `FailureReason`, `Finalization`, `StartedAt`, `FinishedAt`, and timestamps are NEVER mutated. Stats engines depend on this: `execOutcome` is keyed by `execution_id` and the rollup counts each attempt distinctly.

**Idempotency:** if a retry of a given parent is already in flight (any non-terminal status with `ParentExecutionID == parent.ExecutionID && Operation == "retry"`), `Retry` returns the existing in-flight record instead of creating a duplicate. This dedups double-clicks and racing HTTP retries without persistent idempotency keys.

**Eligible parent statuses:** `completed`, `failed`, `canceled`, `needs_fixup`. Calling Retry on `pending`, `starting`, `running`, `validating`, or `needs_review` returns 400 — the parent must reach a stable state before retry.

### Initiative Operating Mode Invariants

Non-default initiative operating modes are strict orchestration flows. They are not a UI convenience layer over item-level execution.

| Invariant | Enforced by | Why it matters |
|-----------|-------------|----------------|
| Generic initiative metadata updates cannot change mode | `api/internal/initiatives` update validation and operating-mode switch routes | Mode changes have cancellation, lock, event, and workspace side effects that must stay in one lifecycle boundary |
| Round actions never silently default to `item-level` | `api/internal/operatingmode` handler/service mode resolution | Prevents operators and clients from applying non-default round controls to the wrong execution model |
| Completed rounds must satisfy the registered phase output contract | `api/internal/operatingmode` registry, parser, artifact applier, and refresher | Stats, phase transitions, and artifact state cannot treat malformed or incomplete agent output as success |
| A failed phase start leaves no active reserved/running round unless a real AgentManager run owns the lock | `api/internal/operatingmode/phase_runner.go` | Prevents stale rounds from blocking future initiative progress |
| Backlog sync mutations are run-id validated and source-attributed | `api/internal/operatingmode/backlog_reconciler.go` | Keeps direct backlog edits out of agent output and preserves audit metadata on backlog/status/proposal events |
| UI round cards render parsed view models instead of raw payload decisions | `ui/src/components/initiative/operating-mode/round-view-model.ts` and `round-card.tsx` | Prevents payload-schema drift across React components and makes sync-action rules testable without rendering |

### Operating Mode Authoring Invariants

Operating-mode methodology behavior must stay registry-owned so new modes remain easy to add and safe to validate.

| Invariant | Enforced by | Why it matters |
|-----------|-------------|----------------|
| Concrete mode behavior lives in focused mode definition files | `api/internal/operatingmode/mode_*.go` and `ValidateRegistry` | Makes the authoring surface obvious for future agents and prevents hidden cross-package mode facts |
| Initiative mode definitions use registry policies for transitions, artifacts, metrics, prompts, profiles, backlog sync, locks, and capabilities | `api/internal/operatingmode/definition_builder.go`, `registry.go`, `state.go`, `artifact_applier.go`, `workspace.go` | Keeps shared framework code generic instead of accumulating mode-specific branches |
| Transition routing is declared through `PhaseGraph.Transitions` and `TransitionRules` | `api/internal/operatingmode/state.go` and registry validation | Prevents handlers, UI, CLI, or stats from becoming alternate state machines |
| Derived artifact writes are declared through phase `ResultBindings` | `api/internal/operatingmode/artifact_applier.go` and registry validation | Ensures new mode artifacts can be added without hardcoded mode/path branches |
| Prompt catalog entries for operating-mode phases are generated from registry metadata | `api/internal/operatingmode/prompt_catalog_entries.go` and `ValidatePromptCatalog` | Prevents catalog ID, skill ID, mode, phase, and output path drift |
| Replan and acceptance metrics are opt-in registry semantics | `api/internal/operatingmode.MetricsPolicy` and `api/internal/stats/engine.go` | Lets new modes define meaningful statistics without phase-name assumptions |
| UI and CLI consume backend-declared capabilities | `api/internal/operatingmode/workspace.go`, UI service normalization, and CLI output structs | Keeps presentation code out of business-rule inference |
| New phase purposes do not require shared activity or lock constants | Registry purpose token validation and initiative-owned activity validation | Lets a mode author add phases without editing unrelated shared packages |
| A synthetic non-production mode exercises authoring seams | `api/internal/operatingmode/synthetic_mode_test.go` | Catches accidental regressions toward production-mode hardcoding |

### Audit Trail

| Date | Author | Change |
|------|--------|--------|
| 2026-01-28 | Claude (Phase 19) | Initial idempotency invariants documentation; made DELETE idempotent (204 instead of 404); added replay safety tests |
| 2026-04-25 | Retry-as-new-attempt rewrite | Documented terminal-state writer pair (review-decide forward, reopenForRetry backward); replaced in-place `execution.Retry` with new-attempt semantics; added `RetryLatestForBacklog` and `POST /api/v1/backlog/{kind}/{name}/retry` route |
| 2026-04-30 | Operating-mode hardening | Documented non-default operating-mode lifecycle, output-contract, round-action, backlog-sync, and UI view-model invariants |
| 2026-05-01 | Operating-mode authoring architecture | Documented registry-owned authoring invariants for definitions, transitions, result bindings, prompt catalog, metrics, capabilities, purposes, and synthetic-mode coverage |
