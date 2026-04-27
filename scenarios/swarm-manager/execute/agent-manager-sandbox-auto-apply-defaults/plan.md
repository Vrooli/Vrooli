# Plan: Wire agent-manager run executor + workspace-sandbox apply path to the locked auditability contract

> **All decisions resolved.** Round 1: D1=A (7-day TTL from run end), D2=A (greenfield wholesale replacement), D3=B (parallel with schema-version contract), D4=A (single shared `applyAtRunEnd` call site), D5=A (7→4 RunOutcome mapping). Round 2: D6=A (new `POST /api/v1/sandboxes/{id}/apply-at-run-end` endpoint + `Provider.ApplyAtRunEnd` method), D7=A (new `Run.ConversationID` field; child runs inherit from parent via `ParentRunID`, else fresh UUID), D8=A (typed `Source` enum on requests + audit events), D9=A (`ManualReviewTTL` on `LifecycleConfig`, env var `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`, consumed by existing `LifecycleReconciler`). See `workshop/round-001.json` and `workshop/round-002.json`.

## Purpose

Make sandboxing the seamless, audit-by-default execution path for agent-manager coding runs. Encode the locked defaults from the auditability contract (`mode=tracking`, `manualReview=false`, `autoApply=true`, `applyOnFailure=true`, `lock=false`, `networkMode=localhost`, eager sandbox creation, agent-side awareness=none) into the run executor and the workspace-sandbox apply call path, and produce per-run + per-file provenance that GCT can render.

## Greenfield Declaration

**This is greenfield work.** The auditability contract supersedes the existing `SandboxConfig.Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and run-level `ResolvedConfig.RequiresApproval` indirections. Per Decision D2 (Round 1, option A), the new per-run levers (`manualReview`, `autoApply`, `applyOnFailure`, `mode`, `networkMode`) replace the old field surface wholesale; old fields are removed in this item, not deprecated, and no compatibility shims, dual-name fields, `// removed` comments, or `_unused` aliases are preserved. Tool-level `RequiresApproval` in the workspace-sandbox `toolregistry` is preserved as a separate concern (canonical-repo-modifying tools that bypass the sandbox, e.g. direct git commit).

## Required Reading

Before executing this plan, the implementing agent should read the following skills:

```bash
prompt-manager skill read seam-discovery-and-enforcement test boundary-of-responsibility-enforcement implementation-plan-authoring
```

- `seam-discovery-and-enforcement` — The agent-manager `sandbox.Provider` interface is the seam between agent-manager and workspace-sandbox. New per-run levers must flow through this seam without leaking into the agent prompt or the runner; tests must hit the seam, not the wire.
- `test` — Apply assertion-quality discipline to the new run-executor tests. The spec enumerates five test cases (success-apply, failure-apply, partial-acceptance split, manualReview-deferred, no-op empty-provenance) — each must assert observable contract behavior, not implementation detail.
- `boundary-of-responsibility-enforcement` — Workspace-sandbox owns state transitions (`applied` / `pending-review` / `denied`); agent-manager only writes the run-context metadata onto the apply call. Do not duplicate the state machine on the agent-manager side.
- `implementation-plan-authoring` — Plan format and quality gates (already followed by this scaffold).

## Source-of-Truth References

- **Contract (source of truth):** `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md` Findings 1–6.
- **Contract mirror (workspace-sandbox docs):** `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md`.
- **Initiative neighborhood:** `agent-sandbox-audit-foundation`. Sibling members:
  - `research/agent-sandbox-auditability-contract` (completed) — produced the contract.
  - `fix/workspace-sandbox-lock-and-acceptance-semantics` (completed) — decoupled `NoLock` from acceptance evaluation. This item now takes the orthogonality as given.
  - `execute/sandbox-runtime-e2e-verification` (downstream) — owns the spawn-surface validation matrix. This item ships the writer side; e2e adoption gates on its completion.
  - `execute/agent-manager-default-sandboxing-rollout` (downstream) — flips the default in spawn surfaces after this item + e2e verification land.
- **Upstream initiative member (critical dependency):** `execute/gct-pending-ai-provenance-hardening` — owns the `runOutcome` (run group) and `state` (per-file) provenance schema additions. This item's provenance writes depend on that schema landing first or in parallel; **per Decision D3 the two items run in parallel**, coordinating via a shared typed schema-version contract (see "Parallelization Contract" below).

## Problem Statement

The agent-manager run executor today:

1. Creates a sandbox **lazily** only when `RunMode == RunModeSandboxed` and routes the run-end apply through `tryAutoApproval`, which mixes three concerns: empty-sandbox auto-approval (`autoApproveIfEmpty`), explicit auto-approve (`autoApprove`), and auto-reject (`autoReject`). The defaults live on `SandboxConfig.Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and the run-level `ResolvedConfig.RequiresApproval` flag.
2. Does not produce provenance correlating sandbox → run → conversation → cost → outcome on no-op runs (eager creation is missing).
3. Has no per-run lever for `manualReview`, no concept of "apply identical regardless of run outcome with `runOutcome` recorded", and no path that persists a sandbox beyond run end.
4. Bypasses the locked contract's terminology entirely — `manualReview`, `autoApply`, `applyOnFailure`, `mode`, `networkMode` do not exist in the codebase yet.

The workspace-sandbox API exposes `Approve` / `Reject` / `Discard` (single unified `/api/v1/sandboxes/{id}/approve` endpoint with `Mode ∈ {all, files, hunks}` selector — see info i4) but no run-context-aware Apply call that takes `agent_manager_run_id`, `conversation_id`, `cost`, and `runOutcome` and writes per-file `state`. That writer surface needs to land here on the agent-manager side, with the workspace-sandbox surface accepting the new metadata fields. Per Decision D6, this lands as a dedicated `POST /api/v1/sandboxes/{id}/apply-at-run-end` route with a new `ApplyAtRunEndRequest` type (separate from the operator `ApprovalRequest`) and a matching `Provider.ApplyAtRunEnd` method on the agent-manager seam.

## Scope (In / Out)

**In scope:**

- Encode locked defaults at the agent-manager `Run` domain level: replace the existing `SandboxConfig.Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `ResolvedConfig.RequiresApproval` field surface with the contract-named levers (`mode`, `manualReview`, `autoApply`, `applyOnFailure`, `lock`, `networkMode`, `acceptanceAllow`, `acceptanceDeny`).
- Eager sandbox creation: every sandboxed run creates the sandbox at `setupWorkspace` time (already the case) AND writes an empty provenance record at run start so no-op runs leave a durable audit trail.
- Run-end apply for `manualReview=false`: replace the `handleSuccessfulCompletion → tryAutoApproval` indirection with a single `applyAtRunEnd` step (Decision D4) that runs identically for success / failure / cancelled / timeout outcomes (apply behavior identical regardless of `runOutcome`; outcome is metadata, not a gate).
- Sandbox-persists-beyond-run-end when `manualReview=true`: the run executor records run end without applying; the sandbox stays alive until an operator approves or denies via any of the three approval surfaces (GCT, agent-manager UI, workspace-sandbox UI). Per Decision D8, the originating surface is recorded on the resulting state transition via a typed `Source` enum field on `ApprovalRequest`, the new `ApplyAtRunEndRequest`, and audit events; values ∈ {`agent-manager-auto-apply`, `git-control-tower`, `agent-manager-ui`, `workspace-sandbox-ui`, `cli`}.
- Provenance writes: `agent_manager_run_id`, `conversation_id`, `cost`, `runOutcome` on `ProvenanceRunGroup`; per-file `state` ∈ {`applied`, `pending-review`, `denied`} on `ProvenanceFile`. Workspace-sandbox owns state transitions; agent-manager only supplies the run-context metadata. Per Decision D7, `conversation_id` is sourced from a new `Run.ConversationID` field populated by the spawner: child runs inherit the value from `ParentRunID`'s run when the chain is set, otherwise the spawner generates a fresh UUID.
- Garbage-collection rule for abandoned `manualReview=true` sandboxes: **per Decision D1, a 7-day TTL from run end with auto-deny-on-expiry**, configurable via `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`. Per Decision D9, the TTL lives on `LifecycleConfig` (alongside `DefaultTTL` / `IdleTimeout`) and is consumed by the existing `LifecycleReconciler.ReconcileLifecycle` 15-minute tick — no new ticker.
- Tests: success+apply, failure+apply, partial-acceptance split (in-acceptance applied + out-of-acceptance pending-review on same run), `manualReview=true` deferred apply with sandbox persistence, no-op run empty-provenance, GC at TTL expiry.
- Documentation: a one-paragraph note pointing future contributors to `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` from `scenarios/agent-manager/api/internal/orchestration/run_executor.go` package doc.

**Out of scope:**

- Provenance schema migrations (`runOutcome`, `state`) — these land in `execute/gct-pending-ai-provenance-hardening` (upstream initiative). Coordinated via the schema-version contract per Decision D3.
- AI Changes UI / GCT surface changes — same upstream item.
- End-to-end spawn-surface validation matrix — `execute/sandbox-runtime-e2e-verification` (downstream sibling).
- Flipping the default in agent-manager UI / swarm-manager queue / cron / CLI spawn surfaces — `execute/agent-manager-default-sandboxing-rollout` (downstream sibling).
- Protected-mode containment / runtime guardrails — `protected-agent-sandboxing` (upstream initiative). Requesting `mode=protected` in this item must error with a clear "reserved" message.
- Tool-level `RequiresApproval` flags in workspace-sandbox `toolregistry` — preserved as-is (separate concern: canonical-repo-modifying tools that bypass the sandbox).
- Removing `RunModeInPlace` — preserved as an escape hatch; this item does not flip the default RunMode (that belongs to `agent-manager-default-sandboxing-rollout`). Confirmed: `handleSuccessfulCompletion` already short-circuits for `RunMode==InPlace` (run_executor.go:1423-1426); `handleFailure`/`handleCancellation` skip sandbox operations entirely.

## Scope (Acceptance Globs)

- `acceptance_allow`: `scenarios/agent-manager/**`, `scenarios/workspace-sandbox/**` (already set).
- `acceptance_deny`: none required (no secrets / generated code / vendor dirs in the change set).

Planned changes by glob:

- `scenarios/agent-manager/api/internal/domain/types.go` — replace `SandboxAcceptanceConfig.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `RunConfig.RequiresApproval` with the new lever surface; add `Run.ConversationID` field (per D7) populated by spawners; add `RunOutcome` mapping for the contract's `runOutcome` field.
- `scenarios/agent-manager/api/internal/domain/validation.go` — validate the new fields; reject `mode=protected` with a "reserved" error.
- `scenarios/agent-manager/api/internal/orchestration/run_executor.go` — eager provenance write at sandbox setup, replace `tryAutoApproval` with `applyAtRunEnd`, gate on `manualReview`, emit `runOutcome` to apply call. When the spawner did not set `ConversationID` and a `ParentRunID` is present, inherit the parent run's `ConversationID`; else generate a fresh UUID at run-creation time (per D7).
- `scenarios/agent-manager/api/internal/orchestration/run_executor_test.go` — five new test cases enumerated above plus `ConversationID` inheritance.
- `scenarios/agent-manager/api/internal/adapters/sandbox/interface.go` — add a new `Provider.ApplyAtRunEnd(ctx, ApplyAtRunEndRequest) (ApplyAtRunEndResponse, error)` method (per D6). Existing `Approve` / `Reject` / `PartialApprove` / `Stop` / `Start` methods remain unchanged.
- `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox.go` — wire the new method to `POST /api/v1/sandboxes/{id}/apply-at-run-end` with the new request shape.
- `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox_test.go` — coverage for the new endpoint and request shape.
- `scenarios/workspace-sandbox/api/internal/types/types.go` — new `ApplyAtRunEndRequest` (per D6) carrying `agent_manager_run_id`, `conversation_id`, `cost`, `run_outcome`, `source` (typed enum per D8); add typed `Source` enum (`agent-manager-auto-apply` / `git-control-tower` / `agent-manager-ui` / `workspace-sandbox-ui` / `cli`); add `Source` field to existing `ApprovalRequest` and audit-event types (per D8).
- `scenarios/workspace-sandbox/api/internal/sandbox/service.go` — accept the new metadata; persist on apply (per-file `state` and run-group `runOutcome`); enforce sandbox-persists-beyond-run-end semantics for `manualReview=true`. Per-file state-machine logic shared between `Approve` and `ApplyAtRunEnd` lives in a single internal helper to prevent drift (see Risks).
- `scenarios/workspace-sandbox/api/internal/handlers/handlers.go` — register `POST /api/v1/sandboxes/{id}/apply-at-run-end`; decode `ApplyAtRunEndRequest`; response shape mirrors the existing `/approve` response with per-file state breakdown.
- `scenarios/workspace-sandbox/api/internal/config/config.go` — add `ManualReviewTTL time.Duration` to `LifecycleConfig` (per D9); env var `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`; default `7 * 24 * time.Hour` (per D1).
- `scenarios/workspace-sandbox/api/internal/gc/lifecycle.go` — add `pending-review` branch to `LifecycleReconciler.ReconcileLifecycle` consuming the new TTL from `LifecycleConfig`.

## Current Technical Context

| File | Relevance |
|---|---|
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:619-711` | `setupWorkspace` / `createSandboxWorkspace` — eager creation already runs at this point for sandboxed runs, but no provenance write yet. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1247-1273` | `SandboxEnvVars` injects `VROOLI_SANDBOX_ID/MERGED/SCOPE`. Already aligned with the contract. No change needed. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1403-1452` | `handleSuccessfulCompletion` — current entry point for the apply path (via `tryAutoApproval`). Replaced by `applyAtRunEnd` per D4. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1454-1485` | `handleFailure` / `handleCancellation` — currently DO NOT call apply. New behavior: apply identical regardless of outcome (D4 single shared call site). |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1686-1789` | `tryAutoApproval` / `autoApprove` / `autoReject` / `autoApproveIfEmpty` — entire indirection replaced by `applyAtRunEnd` keyed on `manualReview` + `acceptance` (per D2 wholesale removal). |
| `scenarios/agent-manager/api/internal/domain/types.go:204-225` | `SandboxAcceptanceConfig` + `SandboxConfig` — field surface to be replaced. |
| `scenarios/agent-manager/api/internal/domain/types.go:376` | `Run.SessionID` — runner-specific Claude Code resume token; **NOT** the agent-manager conversation ID (see info i3). |
| `scenarios/agent-manager/api/internal/domain/types.go:467-472` | `RunMode` (`sandboxed` / `in_place`) — preserved; in-place remains an escape hatch. |
| `scenarios/agent-manager/api/internal/domain/types.go:527, 950-963` | Cost: `RunSummary.CostEstimate` (float64 USD); detailed breakdown in `CostEventData` (Input/Output/CacheCreation/CacheRead/Total USD). Apply call passes the total. |
| `scenarios/agent-manager/api/internal/domain/decisions.go:227-277` | `RunOutcome` already exists with `success/exit_error/exception/cancelled/timeout/sandbox_fail/runner_fail`. Mapping per D5 below. |
| `scenarios/agent-manager/api/internal/adapters/sandbox/interface.go:26-63` | `Provider` interface methods: `Create`, `Get`, `Delete`, `GetWorkspacePath`, `GetDiff`, `Approve`, `Reject`, `PartialApprove`, `Stop`, `Start`, `IsAvailable`, `ValidatePath`. New method per D6. |
| `scenarios/agent-manager/api/internal/adapters/sandbox/interface.go:164-170` | `ApproveRequest` shape — `SandboxID`, `Actor`, `CommitMsg`, `Force`. Extended or replaced per D6. |
| `scenarios/workspace-sandbox/api/internal/handlers/handlers.go:206-208` | Routes: `/api/v1/sandboxes/{id}/approve`, `/reject`, `/discard`. New apply-at-run-end route per D6. |
| `scenarios/workspace-sandbox/api/internal/types/types.go:356-377` | `ApprovalRequest` shape (`Mode ∈ all/files/hunks`, `Actor`, `CommitMsg`, `OverrideAcceptance`, `Force`, `CreateCommit`). |
| `scenarios/workspace-sandbox/api/internal/sandbox/service.go` | Service-layer apply path; receives the new metadata. |
| `scenarios/workspace-sandbox/api/internal/config/config.go:120-160` | `LifecycleConfig` — `DefaultTTL`, `IdleTimeout`, `TerminalCleanupDelay`, `GCInterval` (default 15 min). Sibling for `ManualReviewTTL` per D9. |
| `scenarios/workspace-sandbox/api/internal/config/config.go:166-233` | `PolicyConfig` — `RequireHumanApproval`, `DefaultNoLock`, `TeardownHooks`, etc. Alternative location for `ManualReviewTTL` per D9. |
| `scenarios/workspace-sandbox/api/internal/gc/lifecycle.go:36-58, 76` | `LifecycleReconciler` ticker (`time.NewTicker(cfg.Lifecycle.GCInterval)`) + `ReconcileLifecycle` iterating sandbox statuses. New `pending-review` branch lands here. |

## Target End State

1. Every sandboxed agent-manager run creates a sandbox eagerly at `setupWorkspace` AND writes an empty provenance record correlating sandbox → run → conversation → cost. No-op runs leave an empty provenance entry.
2. `manualReview=false` (default) runs apply at run end identically for `success`, `failure`, `cancelled`, `timeout` outcomes (D4 single shared call site), with `runOutcome` recorded on `ProvenanceRunGroup`. In-acceptance changes get `state=applied`; out-of-acceptance changes get `state=pending-review` and remain in the sandbox.
3. `manualReview=true` runs do NOT apply at run end. All changes persist as `state=pending-review`. The sandbox persists beyond run end until an operator approves or denies. Per D8, the originating approval surface is recorded on the resulting state transition via a typed `Source` enum (∈ {`agent-manager-auto-apply`, `git-control-tower`, `agent-manager-ui`, `workspace-sandbox-ui`, `cli`}).
4. Per-run config surface accepts `mode`, `manualReview`, `autoApply`, `applyOnFailure`, `lock`, `networkMode`, `acceptanceAllow`, `acceptanceDeny`. None pass through to the agent prompt or runner. `mode=protected` errors as "reserved".
5. The old `SandboxAcceptanceConfig.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `RunConfig.RequiresApproval` fields are removed from the codebase (D2 greenfield wholesale).
6. Tool-level `RequiresApproval` in the workspace-sandbox `toolregistry` continues to gate canonical-repo-modifying tools (e.g. direct git commit). Unchanged.
7. Abandoned `manualReview=true` sandboxes auto-deny-and-tear-down at 7 days post-run-end (D1), driven by the existing `LifecycleReconciler` consuming `LifecycleConfig.ManualReviewTTL` (D9).
8. Every sandboxed run carries a `Run.ConversationID` (per D7): inherited from `ParentRunID`'s run when chained, else a fresh UUID generated at run-creation time. The value is included verbatim on the apply call.
9. Five new tests in `run_executor_test.go` pass: success+apply, failure+apply, partial-acceptance split, `manualReview=true` deferred apply, no-op empty-provenance.
10. A package-doc note in `run_executor.go` points future contributors to `AUDITABILITY_CONTRACT.md`.

## Implementation Strategy

### Phase 1 — Domain field surface (agent-manager)

1. In `scenarios/agent-manager/api/internal/domain/types.go`, replace `SandboxAcceptanceConfig.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `RunConfig.RequiresApproval` with new contract-named fields (`Mode`, `ManualReview`, `AutoApply`, `ApplyOnFailure`, `NetworkMode`). Keep `Allow` / `Deny` on the existing `SandboxAcceptanceConfig`.
2. Update `SandboxConfig` to embed the new fields; default values via `DefaultSandboxConfig()` matching the contract's locked table.
3. Add `Run.ConversationID string` (per D7). Populated by spawners; if left empty at run creation and `Run.ParentRunID` is set, the run-creation path resolves it from the parent run's `ConversationID`. If still empty after that, a fresh UUID is generated at run-creation time. Do NOT reuse `Run.SessionID` (info i3 — runner-specific Claude Code resume token, unrelated lifetime). Coordinate with downstream spawn surfaces (web-console, swarm-manager queue, cron, CLI) to populate the field where they already know they're continuing a thread; the inheritance/UUID fallback covers the gap until they do.
4. In `scenarios/agent-manager/api/internal/domain/validation.go`, reject `Mode=protected` with a "reserved" error; reject unsupported `NetworkMode` values.
5. Update all call-sites that read the removed fields. None of these fields are set by agent prompts or the runner — confirm in code review.

### Phase 2 — Workspace-sandbox apply surface (per Decisions D6, D8, D9)

6. Add `POST /api/v1/sandboxes/{id}/apply-at-run-end` (per D6) with new `ApplyAtRunEndRequest` carrying:
    - `agent_manager_run_id string`
    - `conversation_id string`
    - `cost float64` (USD)
    - `run_outcome string` (∈ {`success`, `failure`, `cancelled`, `timeout`} per the D5 mapping)
    - `source Source` (typed enum per D8)
    - File / hunk selection mirroring the existing acceptance partitioning the server already does for `Approve`.
   Response shape mirrors the existing `/approve` response with a per-file `state` breakdown (∈ {`applied`, `pending-review`}) so the agent-manager seam can surface counts without an extra call.
7. Add the typed `Source` enum (per D8) to `scenarios/workspace-sandbox/api/internal/types/types.go`: ∈ {`agent-manager-auto-apply`, `git-control-tower`, `agent-manager-ui`, `workspace-sandbox-ui`, `cli`}. Add the `Source` field to existing `ApprovalRequest` and audit-event types so audit queries don't fork; `Actor` continues to carry user/system identity classification (unchanged).
8. In `scenarios/workspace-sandbox/api/internal/sandbox/service.go`, accept the new metadata and persist it. Per-file `state` defaults to `applied` for in-acceptance changes; `pending-review` for out-of-acceptance. Preserve sandbox lifetime when any file is `pending-review` (sandbox persists until all files transition). Factor the per-file state-machine logic into a single internal helper shared by `Approve` and `ApplyAtRunEnd` to prevent drift.
9. Add `ManualReviewTTL time.Duration` to `LifecycleConfig` (per D9), env var `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`, default `7 * 24 * time.Hour` (per D1). Sibling to `DefaultTTL` / `IdleTimeout` / `TerminalCleanupDelay` / `GCInterval`.

### Phase 3 — Run-executor wiring

10. In `scenarios/agent-manager/api/internal/orchestration/run_executor.go`, write an empty provenance record on successful sandbox creation in `createSandboxWorkspace` (eager creation contract).
11. Replace `handleSuccessfulCompletion → tryAutoApproval` with `applyAtRunEnd`, called from a single shared point at the end of `Execute()` after all terminal handlers (Decision D4 option A). Route through the new `Provider.ApplyAtRunEnd` seam (per D6). Gate on `RunMode==Sandboxed && !manualReview`. Pass `runOutcome` (mapped per D5 below) plus run-context metadata (`agent_manager_run_id`, `conversation_id`, `cost`, `source=agent-manager-auto-apply`) to the apply call.
12. When `manualReview=true`, skip the apply call. Mark the run terminal but instruct the sandbox to persist (the lifecycle config's `DeleteOn` / `StopOn` are bypassed in this case — sandbox stays until operator action or TTL expiry per D1/D9).
13. Remove `tryAutoApproval`, `autoApprove`, `autoReject`, `autoApproveIfEmpty` (greenfield, per D2).

**RunOutcome mapping (per Decision D5 option A):**

| `domain.RunOutcome` | Contract `runOutcome` |
|---|---|
| `success` | `success` |
| `cancelled` | `cancelled` |
| `timeout` | `timeout` |
| `exit_error` | `failure` |
| `exception` | `failure` |
| `sandbox_fail` | `failure` |
| `runner_fail` | `failure` |

### Phase 4 — GC rule for abandoned manualReview=true sandboxes (per Decisions D1, D9)

14. Add a `pending-review` branch to `LifecycleReconciler.ReconcileLifecycle` (lifecycle.go:76). At each tick (default 15 min, `WORKSPACE_SANDBOX_GC_INTERVAL`), find sandboxes whose status is `pending-review` and where `now - run_end_at > LifecycleConfig.ManualReviewTTL`. For each:
    a. Mark all `pending-review` `ProvenanceFile` entries as `state=denied` (auto-deny on expiry).
    b. Tear down the sandbox via the existing teardown path (`TeardownHooks` honored).
    c. Emit an audit event with `source=workspace-sandbox-gc` (per D8 — note: `workspace-sandbox-gc` is an internal source value not enumerated in the operator-facing `Source` enum; it lives on audit events only) and `actor=manual-review-ttl-expiry`.
15. Optional ops metric: `pending_review_sandbox_age_seconds` histogram for observability.

### Phase 5 — Parallelization Contract (per Decision D3)

16. Land `execute/gct-pending-ai-provenance-hardening` and this item in parallel against a shared typed schema-version contract:
    - **Field surface**: `runOutcome string` on `ProvenanceRunGroup`, `state string` on `ProvenanceFile` (∈ {`applied`, `pending-review`, `denied`}). Schema-version constant lives in a shared Go package owned by the upstream item.
    - **Compatibility**: this item writes the new fields unconditionally; readers (GCT UI) tolerate empty values during the gap. Integration tests gate the operator-visible feature flip on both items merging.
    - **Failure mode**: if the schema item slips, this item still writes the wire fields; provenance simply reads as "v0" until the schema lands.

### Phase 6 — Tests

17. Five test cases in `run_executor_test.go` (table-driven where shape allows):
    - `TestRunExecutor_ManualReviewFalse_AppliesOnSuccess` — success outcome + auto-apply path, in-acceptance changes only → `state=applied`.
    - `TestRunExecutor_ManualReviewFalse_AppliesOnFailure` — failure outcome + auto-apply still runs (apply identical regardless of `runOutcome`).
    - `TestRunExecutor_ManualReviewFalse_PartialAcceptanceSplit` — single run with one in-acceptance file (`state=applied`) + one out-of-acceptance file (`state=pending-review`) on the same run group.
    - `TestRunExecutor_ManualReviewTrue_DefersApply` — apply not called at run end; sandbox persists; provenance entries all `state=pending-review`.
    - `TestRunExecutor_NoOpRun_WritesEmptyProvenance` — agent makes no edits; eager-creation provenance entry exists with file count 0.
    - `TestRunExecutor_ConversationID_InheritsFromParentRun` (per D7) — child run with `ParentRunID` set inherits parent's `ConversationID`; standalone run with no parent gets a fresh UUID; spawner-supplied value is preserved verbatim.
18. Workspace-sandbox-side tests for the new `/apply-at-run-end` endpoint and per-file `state` writes (`handlers_test.go`, `service_test.go`); include a typed-`Source`-enum round-trip test (per D8).
19. GC tests for the manualReview-TTL path (`gc/lifecycle_test.go`); use injectable clock to fast-forward past `LifecycleConfig.ManualReviewTTL`.

### Phase 7 — Documentation

20. Add a short package-doc paragraph at the top of `run_executor.go` pointing to `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` as the canonical reference for sandbox/auditability behavior.

### Final: Cleanup & Verification

21. Run type checking (`go build ./...`) and fix ALL errors, even pre-existing ones in modified files.
22. Run linter (`golangci-lint run`) and fix ALL warnings in modified files.
23. Run unit tests (`go test ./...` in both `scenarios/agent-manager/api` and `scenarios/workspace-sandbox/api`) and fix any failures.
24. `vrooli scenario restart agent-manager` and `vrooli scenario restart workspace-sandbox`.
25. Verify health: `curl -s http://localhost:<agent-manager-port>/health` and the workspace-sandbox equivalent.

## Contract Decisions

| ID | Topic | Status |
|---|---|---|
| D1 | GC rule for abandoned `manualReview=true` sandboxes | **Resolved (R1) — A: 7-day TTL from run end, auto-deny + teardown, configurable via `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`** |
| D2 | Migration approach for old field surface | **Resolved (R1) — A: greenfield wholesale replacement; old fields removed in this item** |
| D3 | Sequencing with `execute/gct-pending-ai-provenance-hardening` | **Resolved (R1) — B: parallel via shared typed schema-version contract** |
| D4 | Apply trigger placement in run executor lifecycle | **Resolved (R1) — A: single shared `applyAtRunEnd` call site at end of `Execute()`** |
| D5 | Mapping `domain.RunOutcome` (7 values) to contract's `runOutcome` (4 values) | **Resolved (R1) — A: see mapping table in Phase 3** |
| D6 | Apply API surface shape (extend `/approve` vs new `/apply-at-run-end` vs typed wrapper) | **Resolved (R2) — A: new `POST /api/v1/sandboxes/{id}/apply-at-run-end` endpoint with new `ApplyAtRunEndRequest` type and matching `Provider.ApplyAtRunEnd` method** |
| D7 | Source of `conversation_id` (new field on Run vs reuse RunID/SwarmID) | **Resolved (R2) — A: new `Run.ConversationID` field; spawner populates; child runs inherit from `ParentRunID`'s run, else a fresh UUID is generated at run-creation time** |
| D8 | Originating-approval-surface field shape (typed `Source` enum vs Actor prefix vs ActorType reuse) | **Resolved (R2) — A: typed `Source` enum on `ApprovalRequest`, `ApplyAtRunEndRequest`, and audit events ∈ {`agent-manager-auto-apply`, `git-control-tower`, `agent-manager-ui`, `workspace-sandbox-ui`, `cli`}** |
| D9 | `ManualReviewTTL` config placement (`LifecycleConfig` vs `PolicyConfig` vs per-run override) | **Resolved (R2) — A: `LifecycleConfig.ManualReviewTTL`; env var `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`; consumed by existing `LifecycleReconciler.ReconcileLifecycle`** |

## Testing Plan

| # | Behavior | File | Layer |
|---|---|---|---|
| 1 | `manualReview=false` + success outcome → applies in-acceptance changes; provenance file states `applied` | `run_executor_test.go` | Unit (mock sandbox provider) |
| 2 | `manualReview=false` + failure outcome → applies identically; `runOutcome=failure` recorded | `run_executor_test.go` | Unit |
| 3 | Partial-acceptance split: one in-acceptance file `applied`, one out-of-acceptance file `pending-review` on same run group | `run_executor_test.go` | Unit |
| 4 | `manualReview=true` → apply NOT called at run end; sandbox persists; all files `pending-review` | `run_executor_test.go` | Unit |
| 5 | No-op run (no edits) → empty provenance record exists with eager creation metadata | `run_executor_test.go` | Unit |
| 6 | `POST /api/v1/sandboxes/{id}/apply-at-run-end` accepts `agent_manager_run_id`, `conversation_id`, `cost`, `run_outcome`, typed `source` enum and persists per-file `state` | `handlers_test.go` + `service_test.go` | Handler + service-layer integration |
| 7 | Abandoned `manualReview=true` sandbox is GC'd at 7-day TTL: all files transition `pending-review`→`denied`, sandbox torn down, audit event with `source=workspace-sandbox-gc` emitted | `gc/lifecycle_test.go` | Unit (clock injection) |
| 8 | Validation rejects `mode=protected` with "reserved" error | `validation_test.go` | Unit |
| 9 | Validation rejects unsupported `networkMode` values | `validation_test.go` | Unit |
| 10 | Removed fields (`Acceptance.AutoApprove` etc., `RunConfig.RequiresApproval`) no longer compile in any reference | Compile-time | Build check |
| 11 | RunOutcome mapping: `exit_error`/`exception`/`sandbox_fail`/`runner_fail` → `failure` on the wire | `run_executor_test.go` | Unit (table-driven) |
| 12 | `Run.ConversationID` inheritance: child run with `ParentRunID` set inherits parent's value; standalone run gets a fresh UUID; spawner-supplied value is preserved verbatim | `run_executor_test.go` | Unit |
| 13 | Typed `Source` enum round-trips through `ApprovalRequest`, `ApplyAtRunEndRequest`, and audit events; unknown values rejected at decode | `types_test.go` + `handlers_test.go` | Unit |

## Rollout / Validation Checklist

- [ ] Provenance schema fields (`runOutcome` on `ProvenanceRunGroup`, `state` on `ProvenanceFile`) coordinated with `gct-pending-ai-provenance-hardening` per the schema-version contract (D3).
- [ ] All run-executor tests pass (six cases including `ConversationID` inheritance).
- [ ] Workspace-sandbox apply tests pass for the new `/apply-at-run-end` route + typed `Source` enum round-trip.
- [ ] GC tests pass (clock-injected 7-day TTL expiry).
- [ ] `go build ./...` clean in both scenarios.
- [ ] `golangci-lint run` clean in modified files.
- [ ] Both scenarios restart healthy.
- [ ] Manual sanity: spawn an agent-manager run via the existing UI surface; verify a provenance entry appears in GCT's AI Changes tab with run_id / conversation_id / cost / runOutcome populated (gated on `gct-pending-ai-provenance-hardening` UI work — coordinated via D3).
- [ ] `mode=protected` request errors with "reserved".
- [ ] `git grep` confirms zero references to the removed legacy field surface.
- [ ] `git grep` confirms `Provider.ApplyAtRunEnd` is the only call path for the auto-apply flow on agent-manager (no remaining `Approve(Mode=all)` shortcuts from the run executor).

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `gct-pending-ai-provenance-hardening` schema lands later than planned, blocking provenance writes | Medium | High | D3=B parallel sequencing via shared typed schema-version contract. Each item ships its side; the operator-visible feature gates on both merging. If the schema item slips, this item still writes the wire fields; provenance reads as "v0" until the schema lands. |
| Greenfield removal of `Acceptance.AutoApprove` etc. breaks downstream callers (cli-core, test-genie, web-console, swarm-manager) | Medium | Medium | Search all references in the listed scenarios before removing; update or remove every call-site in the same change set. Compile-time test #10 catches drift. |
| In-flight runs at the time of deploy carry the old field surface in persisted state | Medium | Medium | Greenfield removal includes a one-shot migration helper that maps any persisted `Acceptance.AutoApprove==true` records to `manualReview=false, autoApply=true` semantics on read; document the read-time normalization in the migration helper. |
| Sandbox-persists-beyond-run-end leaks resources if GC rule is too lenient | Medium | Medium | D1=A: 7-day TTL upper bound. D9 places the field in `LifecycleConfig` so existing reconciler picks it up. Optional ops metric `pending_review_sandbox_age_seconds`. |
| GC false-positive: 7-day TTL expires while operator was about to approve | Low | Low | Document the 7-day rule prominently in `AUDITABILITY_CONTRACT.md`. Operators with longer review cycles set `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL` higher. Auto-deny-on-expiry is reversible (operator can manually re-spawn an equivalent run). |
| `runOutcome` mapping from `domain.RunOutcome` (7 values) to contract's 4 values is lossy | Low | Low | D5=A: mapping table documented in Phase 3 + test #11. Lossiness is by design; failures are bucketed for GCT rendering. Original `domain.RunOutcome` remains on the agent-manager Run record for triage. |
| Apply-on-failure reveals partial / broken work to the canonical repo | Medium | Medium | Acceptance allow/deny is the gate. Operators who fear apply-on-failure should set narrower `acceptanceAllow` or opt into `manualReview=true`. Documentation note in `AUDITABILITY_CONTRACT.md`. |
| Removing `tryAutoApproval` regresses the existing empty-sandbox auto-approval ergonomics | Low | Low | Eager-provenance + auto-apply default + acceptance gating preserve the same observable behavior for the empty case (no files → no apply needed → run terminal). Test #5 pins this. |
| `Run.SessionID` mistakenly used as `conversation_id` | Low | Medium | Info i3 explicitly flags this. D7 picks the canonical source. Code review catches stragglers. |
| New `apply-at-run-end` route diverges from operator `/approve` over time | Low | Medium | Per D6 the two are intentionally separate routes/types; per Phase 2 step 8 a single internal service-layer helper backs both so per-file state-machine logic doesn't fork. Workspace-sandbox tests cover both routes against the helper. |

## Non-goals / Prohibited Patterns

- Do NOT preserve `Acceptance.AutoApprove` / `Acceptance.AutoReject` / `Acceptance.DisableAutoApproveIfEmpty` / `RunConfig.RequiresApproval` as deprecated aliases. D2=A greenfield.
- Do NOT add a feature flag for "old apply behavior". The contract is the contract.
- Do NOT pass `manualReview` / `autoApply` / `mode` etc. into the agent prompt or runner. Agent-side awareness is none.
- Do NOT couple `runOutcome` to apply gating. Apply behavior is identical regardless of outcome; outcome is metadata only.
- Do NOT split a single run group across `applied` and `pending-review` runs. Per-file `state` keeps the run group whole.
- Do NOT touch tool-level `RequiresApproval` in workspace-sandbox `toolregistry`. Separate concern.
- Do NOT remove `RunModeInPlace`. That's `agent-manager-default-sandboxing-rollout`'s job.
- Do NOT reuse `Run.SessionID` as `conversation_id`. They have unrelated lifetimes (info i3).
- Do NOT bypass the existing `LifecycleReconciler` ticker by adding a new GC ticker for `pending-review`. Add a branch to the existing reconciler (D9=A).

## Definition of Done

1. All run-executor tests pass (six cases including `ConversationID` inheritance per D7).
2. `POST /api/v1/sandboxes/{id}/apply-at-run-end` (per D6) accepts and persists `agent_manager_run_id`, `conversation_id`, `cost`, `runOutcome` (run group) and per-file `state`. Typed `Source` enum (per D8) carries the originating surface on `ApprovalRequest`, `ApplyAtRunEndRequest`, and audit events.
3. The old field surface (`Acceptance.AutoApprove` etc., `RunConfig.RequiresApproval`) is removed from the codebase; `git grep` confirms zero references in production code.
4. `mode=protected` errors with "reserved".
5. GC rule for abandoned `manualReview=true` sandboxes lands at 7-day TTL via `LifecycleConfig.ManualReviewTTL` (per D9); clock-injected tests verify auto-deny + teardown + audit event.
6. `vrooli scenario restart agent-manager` and `vrooli scenario restart workspace-sandbox` both produce healthy services.
7. `go build ./...` and `golangci-lint run` clean in modified files (including pre-existing issues).
8. Package-doc reference to `AUDITABILITY_CONTRACT.md` lands in `run_executor.go`.
9. Schema-version contract artifact (per D3) lands in a shared package with `gct-pending-ai-provenance-hardening`.
10. `Run.ConversationID` (per D7) is populated by the run-creation path: spawner-supplied wins, else inherit from `ParentRunID`'s run, else fresh UUID. Test #12 pins all three branches.

## Cross-Initiative Implications (for orchestrator)

- This item is the writer side. The reader / schema side lives in `execute/gct-pending-ai-provenance-hardening` (initiative `git-control-tower-ai-provenance`, upstream). Per D3 the two items run in parallel against a shared typed schema-version contract; the orchestrator should track both and gate operator-visible features on both merging.
- Downstream `execute/sandbox-runtime-e2e-verification` cannot start its multi-surface validation until this item lands.
- Downstream `execute/agent-manager-default-sandboxing-rollout` flips the spawn-surface defaults; nothing in this item affects the default `RunMode` (still `in_place` or whatever spawn surfaces choose today).
- Per D7 (conversation_id source), spawn surfaces (web-console, swarm-manager queue, cron, CLI) should be flagged by the orchestrator to populate `Run.ConversationID` where they already know they are continuing an agent thread (e.g. swarm-manager queue resuming a swarm, agent-manager UI "continue conversation"). The inheritance-from-`ParentRunID` + UUID-fallback logic in this item covers the gap until each spawner is updated, so this is a quality-of-life follow-up rather than a blocker.
