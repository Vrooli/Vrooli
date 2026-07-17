# System Invariants

This document captures critical invariants that must be maintained across all changes to the Swarm Manager. These are the foundational guarantees that enable safe operation, retries, and predictable behavior.

## Replay/Idempotency Invariants

### Agent Manager workshop-round pilot

- A workshop start is keyed by the backlog identity, immutable snapshot version, and canonical operator-note digest. Retrying the same request may return the same workflow execution; changing the item, accumulated rounds, or operator intent produces a different request identity.
- Applying an execution requires the current backlog snapshot version to equal the version captured by the workflow input. A stale result fails closed before any backlog mutation.
- Swarm Manager exclusively creates `workshop/workflow-provenance-<execution-id>.json` before materializing a round. That claim is the durable exactly-once boundary across HTTP retries and process restarts.
- A replay that finds complete provenance returns the recorded artifact without collecting or mutating again. If a crash left a claim without its named artifact, recovery may collect the same execution and finish only that artifact.
- `abstained` is a successful, typed result: it records provenance but intentionally creates no round artifact. Agent Manager never writes backlog state.

### Agent Manager phased-plan pilot

- Only the primary plan-backed execution start uses the phased-plan workflow.
  Retry, fixup, follow-up, and research conclusion remain on their existing
  paths; there is no dual-path fallback at the selected callsite.
- The start record pins workflow execution id, definition digest, rendered-plan
  frontier digest, backlog entity version, and bounded attempt provenance.
- Result application recomputes the current plan and entity snapshot and fails
  closed before mutation if any pinned correlation differs.
- `agent_workflow_apply_state=claimed` is a durable recovery checkpoint. A
  restart finishes the idempotent backlog transition from the persisted typed
  result without recollecting external state; `complete` makes later callbacks
  local no-ops.
- An approval decision is stored before the stable-key `slice_approved` signal
  is delivered. Redelivery cannot replace the original actor or decision time.
- The consumer cancels the workflow aggregate, never an arbitrary child Run.
  Agent Manager never calls Swarm or writes an execution or backlog record.

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
| Round actions resolve an explicit mode and never silently fall back to the member-item strategy | `api/internal/operatingmode` handler/service mode resolution | Prevents operators and clients from applying non-default round controls to the wrong execution model |
| Completed rounds must satisfy the registered phase output contract | `api/internal/operatingmode` registry, parser, artifact applier, and refresher | Stats, phase transitions, and artifact state cannot treat malformed or incomplete agent output as success |
| A failed phase start leaves no active reserved/running round unless a real AgentManager run owns the lock | `api/internal/operatingmode/phase_runner.go` | Prevents stale rounds from blocking future initiative progress |
| Backlog sync mutations are run-id validated and source-attributed | `api/internal/operatingmode/backlog_reconciler.go` | Keeps direct backlog edits out of agent output and preserves audit metadata on backlog/status/proposal events |
| UI round cards render parsed view models instead of raw payload decisions | `ui/src/components/initiative/operating-mode/round-view-model.ts` and `round-card.tsx` | Prevents payload-schema drift across React components and makes sync-action rules testable without rendering |
| Every new mode round belongs to one immutable execution manifest | `api/internal/operatingmode/execution.go`, `phase_runner.go`, and `rounds.go` | Registry edits cannot rewrite the definition graph used by an existing round |
| Every phase-mode execution pins one transitive compiled input contract and digest | `api/internal/operatingmode/input_contract.go` and `execution.go` | Logical specs, explicit sources, aliases, and delegated-mode bindings validate before manifest creation; runtime rendering prefers the pinned artifact |
| A logical input has exactly one source and sensitivity-safe retention | `CompileInputContract` | Missing/competing sources, provider type/target mismatches, derived cycles, and sensitive `retention=value` fail before filesystem mutation |
| Caller inputs are validated once before execution creation and replay by digest | `ValidateCallerInputSnapshot` and `Store.ContinueOrCreateExecutionWithInputs` | Missing, unknown, mistyped, out-of-bounds, oversized, sensitive, or unreplayable values leave no manifest or round; a retry may omit inputs or repeat the identical normalized snapshot, but cannot replace it |
| A `plan-execution` target resolves to a canonical plan id + execution id, never a workspace file path | `plan-execution` target adapter | The unmanaged `plan-ref` workspace-file target kind and its `resolveWorkspacePlanRef` adapter were removed in the declarative-operations rebuild (0 live instances); a plan target is a provider-neutral plan execution, and the domain `plan_ref` field on items/initiatives is an associated reference, not a target |
| Dynamic phase preflight is read-only until compare-and-reserve | `Store.PrepareRound`, `Store.CompareAndCreateRound`, and `startResolvedPhase` | Provider resolution, pinned rendering, trace generation, and spawn-request assembly use a proposed round plus execution/round digest; any intervening state change returns `ErrRoundPreflightStale` and cannot allocate a duplicate round |
| A run ID maps to at most one execution/round owner | `Store.IndexRunOwner` | Replays are idempotent; a conflicting second owner is an explicit ambiguity error rather than last-write-wins |
| At most one nonterminal execution may exist for a target and mode | `Store.ContinueOrCreateExecution` and `collectRunContext` | Resume never chooses an execution by recency or hidden precedence; multiple resumable manifests require repair |

Execution creation uses `execution_id` as its durable identity. Round numbers remain
mode/scope-global during the compatibility window, so existing round links remain
unambiguous while new bytes live under `executions/<execution-id>/rounds/`.
Repeating run-owner registration with the same `(execution_id, round)` is a no-op;
registering the same run against a different owner fails with
`ErrRunOwnerAmbiguous` and preserves the first mapping.

Execution caller-input snapshots are canonical JSON. Their SHA-256 digest is
verified after reload independently of JSON indentation, and public Connect/UI
projections expose the compiled request contract plus retention metadata without
copying retained caller values onto the wire.

Legacy adoption is replay-safe: the execution id is a deterministic digest of
scope, mode, pinned definition digest, and original round envelopes. The
transformed manifest and rounds validate in a staging directory before the flat
round directory moves to its byte-preserving backup; ambiguous histories remain
untouched and are excluded from continuation context.

Phase-start retries must restart from `PrepareRound` after
`ErrRoundPreflightStale`; callers must not reuse the prior rendered prompt or
silently advance to another round number. Exclusive round creation is the
commit point. Prompt/provider/trace failures before that point leave no round,
lock, run, or failure event, while failures after reservation are persisted as
terminal attempts so recovery can distinguish “never applied” from “reserved
once and failed.”

### Operating Mode Authoring Invariants

Operating-mode methodology behavior must stay data-owned and interpreted by the
generic operating-mode engine so new modes remain easy to add and safe to
validate.

| Invariant | Enforced by | Why it matters |
|-----------|-------------|----------------|
| Concrete mode behavior lives in `scenarios/swarm-manager/modes/<id>/mode.json` plus `example-runs/*.json` | `api/internal/operatingmode/loader.go`, `modevalidation.go`, and `ValidateRegistry` | Makes the authoring surface obvious for future agents and prevents hidden cross-package mode facts |
| Initiative mode data declares transitions, artifacts, metrics, prompt skill routing, profiles, backlog sync, locks, and capabilities | `api/internal/operatingmode/loader.go`, `registry.go`, `state.go`, `artifact_applier.go`, `workspace.go` | Keeps shared framework code generic instead of accumulating mode-specific branches |
| Transition routing is declared through data-backed phase graph transitions | `api/internal/operatingmode/state.go`, `guard.go`, and registry validation | Prevents handlers, UI, CLI, or stats from becoming alternate state machines |
| Derived artifact writes are declared through phase result bindings in mode data | `api/internal/operatingmode/artifact_applier.go` and registry validation | Ensures new mode artifacts can be added without hardcoded mode/path branches |
| Prompt catalog entries for operating-mode phases are generated from mode prompt metadata | `api/internal/operatingmode/prompt_catalog_entries.go` and `ValidatePromptCatalog` | Prevents catalog ID, skill ID, mode, phase, and output path drift |
| Replan and acceptance metrics are opt-in mode-data semantics | `api/internal/operatingmode.MetricsPolicy` and `api/internal/stats/engine.go` | Lets new modes define meaningful statistics without phase-name assumptions |
| UI and CLI consume backend-declared capabilities | `api/internal/operatingmode/workspace.go`, UI service normalization, and CLI output structs | Keeps presentation code out of business-rule inference |
| New phase purposes do not require shared activity or lock constants | Registry purpose token validation and initiative-owned activity validation | Lets a mode author add phases without editing unrelated shared packages |
| Synthetic mode data exercises authoring seams | `api/internal/operatingmode/synthetic_mode_test.go` and `api/internal/operatingmode/testdata/` | Catches accidental regressions toward production-mode hardcoding |

### Declarative Agent-Operations Invariants

The declarative agent-operations layer (see
[`../concepts/AGENT-OPERATIONS.md`](../concepts/AGENT-OPERATIONS.md)) adds four
load-bearing invariants that keep autonomous spawning legible and fail-closed.

| Invariant | Enforced by | Why it matters |
|-----------|-------------|----------------|
| Every autonomous agent spawn flows through the generic operation runner; no domain package spawns directly | Architecture tests in `api/internal/archtest` (spawn AST test, agent-manager import allowlist, opsrunner/opsbridge/opscatalog purity tests, removed-vocabulary test) plus the engine's `agentactivity` chokepoint | One spawn path means one place to pin provenance, correlate a workflow, and attribute a run; a bespoke call site would be unversioned and invisible to the workflow instance |
| Revision **labels** are advisory; **digests** are the enforcement identity | Execution provenance pins compiled-mode/contract/binding/policy digests; reproduce fails with `ErrDigestMismatch` and deleted revisions fail closed | A run reproduces against the exact content it pinned, not a mutable `1.0.0` label; content drift is caught even when the label is unchanged |
| An active execution record with a `RunID` but no `OpExecutionID` fails closed | `execution/polling.go` `inspectRunningRecordsLocked` | After migration an uncorrelated active record is impossible; it is marked failed and the item lands in `in_review` rather than being silently stranded by a poll driver that no longer exists |
| `item-level` is a member-item strategy sentinel, never a mode folder | Operating-mode loader/scaffold reject id `item-level`; `operatingmode.NormalizeMode` / `IsMemberItemStrategySentinel` map the sentinel (string `item-level` or blank) to the strategy | Prevents a phase-less pseudo-mode from re-appearing as an authorable, selectable methodology; the strategy is initiative workflow configuration, not a loop |

## Baseline Modes Engagement Invariants

Shadow-mode engagements (plan P-b/P-c) isolate a live scenario from an in-progress candidate. These invariants keep the isolation sound and the policy levers single-sourced.

| Invariant | Enforced by | Why it matters |
|-----------|-------------|----------------|
| The shadow restore point is opened (clean working tree captured) BEFORE the overlay merge is approved | `execution.processEngagementHold` (open → ApproveRun ordering); a `baseline start` failure aborts before approve | A merge without a restore point would let a live restart rebuild the candidate — the platform isolation floor (P-a) depends on the copy already existing |
| A scenario has at most one open engagement across all owners | `EngagementStore.HolderOf` + `checkExclusivityAtStart` (block-at-start) + the diff-level conflict check in `processEngagementHold` | One working tree can hold one candidate; two owners editing it concurrently would corrupt each other's baseline |
| An engagement set is owned by the backlog item, never the per-run Record | `ownerKeyFor` (single definition) + `EngagementStore`; fixups share the owner key | The set must outlive the main run + every fixup and survive until review-decide; per-run keying would close it too early or lose it |
| Promote/abandon happen only at review-decide (the atomic accept/reject), never at finalization | `CloseOwnerEngagements` via `AddItemTerminalHandler`; finalization no longer closes | A candidate must not be blessed into live before the human accept |
| Validation restart/health route to `@shadow` for shadow-engaged scenarios; live is untouched | `execution.shadowTargetFor` feeding `runScenarioRestartAndHealth` | Validates the candidate without disturbing the live instance still serving the baseline |

### Audit Trail

| Date | Author | Change |
|------|--------|--------|
| 2026-01-28 | Claude (Phase 19) | Initial idempotency invariants documentation; made DELETE idempotent (204 instead of 404); added replay safety tests |
| 2026-04-25 | Retry-as-new-attempt rewrite | Documented terminal-state writer pair (review-decide forward, reopenForRetry backward); replaced in-place `execution.Retry` with new-attempt semantics; added `RetryLatestForBacklog` and `POST /api/v1/backlog/{kind}/{name}/retry` route |
| 2026-04-30 | Operating-mode hardening | Documented non-default operating-mode lifecycle, output-contract, round-action, backlog-sync, and UI view-model invariants |
| 2026-05-01 | Operating-mode authoring architecture | Documented registry-owned authoring invariants for definitions, transitions, result bindings, prompt catalog, metrics, capabilities, purposes, and synthetic-mode coverage |
| 2026-06-07 | Shadow-mode engagement rework (plan P-b/P-c) | Replaced per-execution live-mode engagement with owner-keyed shadow engagements: pre-merge hold (ManualReview → diff-driven `baseline start --mode shadow` → ApproveRun), block-at-start exclusivity, `@shadow` validation routing, promote/abandon moved to review-decide. Added the Baseline Modes Engagement Invariants table |
