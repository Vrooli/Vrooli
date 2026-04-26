# Plan: Wire agent-manager run executor + workspace-sandbox apply path to the locked auditability contract

## Purpose

Make sandboxing the seamless, audit-by-default execution path for agent-manager coding runs. Encode the locked defaults from the auditability contract (`mode=tracking`, `manualReview=false`, `autoApply=true`, `applyOnFailure=true`, `lock=false`, `networkMode=localhost`, eager sandbox creation, agent-side awareness=none) into the run executor and the workspace-sandbox apply call path, and produce per-run + per-file provenance that GCT can render.

## Greenfield Declaration

**This is greenfield work.** The auditability contract supersedes the existing `SandboxConfig.Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and run-level `ResolvedConfig.RequiresApproval` indirections. Do not preserve compatibility shims, dual-name fields, `// removed` comments, or `_unused` aliases. The new per-run levers (`manualReview`, `autoApply`, `applyOnFailure`, `mode`, `networkMode`) replace the old field surface; old fields are removed in this item, not deprecated. Tool-level `RequiresApproval` in the workspace-sandbox `toolregistry` is preserved as a separate concern (canonical-repo-modifying tools that bypass the sandbox, e.g. direct git commit).

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
- **Upstream initiative member (critical dependency):** `execute/gct-pending-ai-provenance-hardening` — owns the `runOutcome` (run group) and `state` (per-file) provenance schema additions. This item's provenance writes depend on that schema landing first or in parallel.

## Problem Statement

The agent-manager run executor today:

1. Creates a sandbox **lazily** only when `RunMode == RunModeSandboxed` and routes the run-end apply through `tryAutoApproval`, which mixes three concerns: empty-sandbox auto-approval (`autoApproveIfEmpty`), explicit auto-approve (`autoApprove`), and auto-reject (`autoReject`). The defaults live on `SandboxConfig.Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and the run-level `ResolvedConfig.RequiresApproval` flag.
2. Does not produce provenance correlating sandbox → run → conversation → cost → outcome on no-op runs (eager creation is missing).
3. Has no per-run lever for `manualReview`, no concept of "apply identical regardless of run outcome with `runOutcome` recorded", and no path that persists a sandbox beyond run end.
4. Bypasses the locked contract's terminology entirely — `manualReview`, `autoApply`, `applyOnFailure`, `mode`, `networkMode` do not exist in the codebase yet.

The workspace-sandbox API exposes `Approve` / `Reject` / `PartialApprove` but no run-context-aware Apply call that takes `agent_manager_run_id`, `conversation_id`, `cost`, and `runOutcome` and writes per-file `state`. That writer surface needs to land here on the agent-manager side, with the workspace-sandbox surface accepting the new metadata fields.

## Scope (In / Out)

**In scope:**

- Encode locked defaults at the agent-manager `Run` domain level: replace the existing `SandboxConfig.Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `ResolvedConfig.RequiresApproval` field surface with the contract-named levers (`mode`, `manualReview`, `autoApply`, `applyOnFailure`, `lock`, `networkMode`, `acceptanceAllow`, `acceptanceDeny`).
- Eager sandbox creation: every sandboxed run creates the sandbox at `setupWorkspace` time (already the case) AND writes an empty provenance record at run start so no-op runs leave a durable audit trail.
- Run-end apply for `manualReview=false`: replace the `handleSuccessfulCompletion → tryAutoApproval` indirection with a single `applyAtRunEnd` step that runs identically for success / failure / cancelled / timeout outcomes (apply behavior identical regardless of `runOutcome`; outcome is metadata, not a gate).
- Sandbox-persists-beyond-run-end when `manualReview=true`: the run executor records run end without applying; the sandbox stays alive until an operator approves or denies via any of the three approval surfaces (GCT, agent-manager UI, workspace-sandbox UI).
- Provenance writes: `agent_manager_run_id`, `conversation_id`, `cost`, `runOutcome` on `ProvenanceRunGroup`; per-file `state` ∈ {`applied`, `pending-review`, `denied`} on `ProvenanceFile`. Workspace-sandbox owns state transitions; agent-manager only supplies the run-context metadata.
- Garbage-collection rule for abandoned `manualReview=true` sandboxes (open detail explicitly delegated to this item — see Decision D1).
- Tests: success+apply, failure+apply, partial-acceptance split (in-acceptance applied + out-of-acceptance pending-review on same run), `manualReview=true` deferred apply with sandbox persistence, no-op run empty-provenance.
- Documentation: a one-paragraph note pointing future contributors to `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` from `scenarios/agent-manager/api/internal/orchestration/run_executor.go` package doc.

**Out of scope:**

- Provenance schema migrations (`runOutcome`, `state`) — these land in `execute/gct-pending-ai-provenance-hardening` (upstream initiative). Sequencing handled in Decision D3.
- AI Changes UI / GCT surface changes — same upstream item.
- End-to-end spawn-surface validation matrix — `execute/sandbox-runtime-e2e-verification` (downstream sibling).
- Flipping the default in agent-manager UI / swarm-manager queue / cron / CLI spawn surfaces — `execute/agent-manager-default-sandboxing-rollout` (downstream sibling).
- Protected-mode containment / runtime guardrails — `protected-agent-sandboxing` (upstream initiative). Requesting `mode=protected` in this item must error with a clear "reserved" message.
- Tool-level `RequiresApproval` flags in workspace-sandbox `toolregistry` — preserved as-is (separate concern: canonical-repo-modifying tools that bypass the sandbox).
- Removing `RunModeInPlace` — preserved as an escape hatch; this item does not flip the default RunMode (that belongs to `agent-manager-default-sandboxing-rollout`).

## Scope (Acceptance Globs)

- `acceptance_allow`: `scenarios/agent-manager/**`, `scenarios/workspace-sandbox/**` (already set).
- `acceptance_deny`: none required (no secrets / generated code / vendor dirs in the change set).

Planned changes by glob:

- `scenarios/agent-manager/api/internal/domain/types.go` — replace `SandboxAcceptanceConfig.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `RunConfig.RequiresApproval` with the new lever surface; add `RunOutcome` mapping for `runOutcome` field.
- `scenarios/agent-manager/api/internal/domain/validation.go` — validate the new fields; reject `mode=protected` with a "reserved" error.
- `scenarios/agent-manager/api/internal/orchestration/run_executor.go` — eager provenance write at sandbox setup, replace `tryAutoApproval` with `applyAtRunEnd`, gate on `manualReview`, emit `runOutcome` to apply call.
- `scenarios/agent-manager/api/internal/orchestration/run_executor_test.go` — five new test cases enumerated above.
- `scenarios/agent-manager/api/internal/adapters/sandbox/interface.go` — extend `ApproveRequest` with run-context metadata (or add a new `ApplyAtRunEnd` method — see Decision D4).
- `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox.go` — wire new metadata to the workspace-sandbox HTTP client.
- `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox_test.go` — coverage for the new request shape.
- `scenarios/workspace-sandbox/api/internal/types/types.go` — extend the apply request shape to accept `runId`, `conversationId`, `cost`, `runOutcome`.
- `scenarios/workspace-sandbox/api/internal/sandbox/service.go` — accept the new metadata; persist on apply (per-file `state` and run-group `runOutcome`); enforce sandbox-persists-beyond-run-end semantics for `manualReview=true`.
- `scenarios/workspace-sandbox/api/internal/handlers/handlers.go` — request decode + response shape changes.
- `scenarios/workspace-sandbox/api/internal/config/config.go` — add `ManualReviewSandboxTTL` (or equivalent — see Decision D1) for GC.
- `scenarios/workspace-sandbox/api/internal/gc/*` — implement the GC rule (Decision D1).

## Current Technical Context

| File | Relevance |
|---|---|
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:619-711` | `setupWorkspace` / `createSandboxWorkspace` — eager creation already runs at this point for sandboxed runs, but no provenance write yet. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1247-1273` | `SandboxEnvVars` injects `VROOLI_SANDBOX_ID/MERGED/SCOPE`. Already aligned with the contract. No change needed. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1403-1452` | `handleSuccessfulCompletion` — current entry point for the apply path (via `tryAutoApproval`). Replaced by `applyAtRunEnd`. |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1454-1485` | `handleFailure` / `handleCancellation` — currently DO NOT call apply. New behavior: apply identical regardless of outcome (Decision D4). |
| `scenarios/agent-manager/api/internal/orchestration/run_executor.go:1686-1789` | `tryAutoApproval` / `autoApprove` / `autoReject` / `autoApproveIfEmpty` — entire indirection replaced by `applyAtRunEnd` keyed on `manualReview` + `acceptance`. |
| `scenarios/agent-manager/api/internal/domain/types.go:204-225` | `SandboxAcceptanceConfig` + `SandboxConfig` — field surface to be replaced. |
| `scenarios/agent-manager/api/internal/domain/types.go:467-472` | `RunMode` (`sandboxed` / `in_place`) — preserved; in-place remains an escape hatch. |
| `scenarios/agent-manager/api/internal/domain/decisions.go:227-277` | `RunOutcome` already exists with `success/exit_error/exception/cancelled/timeout/sandbox_fail/runner_fail`. Map to contract's `success/failure/cancelled/timeout` for the provenance write (Decision D5). |
| `scenarios/agent-manager/api/internal/adapters/sandbox/interface.go:164-170` | `ApproveRequest` shape — extended (or replaced — see Decision D4). |
| `scenarios/workspace-sandbox/api/internal/sandbox/service.go` | Service-layer apply path; receives the new metadata. |
| `scenarios/workspace-sandbox/api/internal/types/types.go` | Request/response types for the apply path. |
| `scenarios/workspace-sandbox/api/internal/config/config.go:168-227` | `PolicyConfig.RequireHumanApproval` / `DefaultNoLock` / `TeardownHooks`. New `ManualReviewSandboxTTL` (or equivalent) lands here. |

## Target End State

1. Every sandboxed agent-manager run creates a sandbox eagerly at `setupWorkspace` AND writes an empty provenance record correlating sandbox → run → conversation → cost. No-op runs leave an empty provenance entry.
2. `manualReview=false` (default) runs apply at run end identically for `success`, `failure`, `cancelled`, `timeout` outcomes, with `runOutcome` recorded on `ProvenanceRunGroup`. In-acceptance changes get `state=applied`; out-of-acceptance changes get `state=pending-review` and remain in the sandbox.
3. `manualReview=true` runs do NOT apply at run end. All changes persist as `state=pending-review`. The sandbox persists beyond run end until an operator approves or denies. The originating approval surface is recorded on the resulting state transition.
4. Per-run config surface accepts `mode`, `manualReview`, `autoApply`, `applyOnFailure`, `lock`, `networkMode`, `acceptanceAllow`, `acceptanceDeny`. None pass through to the agent prompt or runner. `mode=protected` errors as "reserved".
5. The old `SandboxAcceptanceConfig.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `RunConfig.RequiresApproval` fields are removed from the codebase (greenfield).
6. Tool-level `RequiresApproval` in the workspace-sandbox `toolregistry` continues to gate canonical-repo-modifying tools (e.g. direct git commit). Unchanged.
7. Abandoned `manualReview=true` sandboxes are garbage-collected per Decision D1.
8. Five new tests in `run_executor_test.go` pass: success+apply, failure+apply, partial-acceptance split, `manualReview=true` deferred apply, no-op empty-provenance.
9. A package-doc note in `run_executor.go` points future contributors to `AUDITABILITY_CONTRACT.md`.

## Implementation Strategy

### Phase 1 — Domain field surface (agent-manager)

1. In `scenarios/agent-manager/api/internal/domain/types.go`, replace `SandboxAcceptanceConfig.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` and `RunConfig.RequiresApproval` with a new `SandboxAuditConfig` (or equivalent — Decision D2) holding `Mode`, `ManualReview`, `AutoApply`, `ApplyOnFailure`, `NetworkMode`. Keep `Allow` / `Deny` on the existing `SandboxAcceptanceConfig`.
2. Update `SandboxConfig` to embed the new fields; default values via `DefaultSandboxConfig()` matching the contract's locked table.
3. In `scenarios/agent-manager/api/internal/domain/validation.go`, reject `Mode=protected` with a "reserved" error; reject unsupported `NetworkMode` values.
4. Update all call-sites that read the removed fields. None of these fields are set by agent prompts or the runner — confirm in code review.

### Phase 2 — Workspace-sandbox apply surface

5. In `scenarios/workspace-sandbox/api/internal/types/types.go`, extend the apply request to accept `agent_manager_run_id`, `conversation_id`, `cost`, `run_outcome` (∈ {`success`, `failure`, `cancelled`, `timeout`}). The provenance schema fields (`runOutcome` on `ProvenanceRunGroup`, `state` on `ProvenanceFile`) are owned by `execute/gct-pending-ai-provenance-hardening` — sequencing in Decision D3.
6. In `scenarios/workspace-sandbox/api/internal/sandbox/service.go`, accept the new metadata in `Apply` and persist it. The per-file `state` defaults to `applied` for in-acceptance changes; `pending-review` for out-of-acceptance. Preserve sandbox lifetime when any file is `pending-review` (sandbox persists until all files transition).
7. Add `ManualReviewSandboxTTL` (or equivalent — Decision D1) to `PolicyConfig`. Wire to existing GC.

### Phase 3 — Run-executor wiring

8. In `scenarios/agent-manager/api/internal/orchestration/run_executor.go`, write an empty provenance record on successful sandbox creation in `createSandboxWorkspace` (eager creation contract).
9. Replace `handleSuccessfulCompletion → tryAutoApproval` with `applyAtRunEnd`, called from a single shared point (Decision D4) that runs identically for success / failure / cancelled / timeout. Pass `runOutcome` (mapped from `domain.RunOutcome` per Decision D5) plus run-context metadata (run_id, conversation_id, cost) to the apply call.
10. When `manualReview=true`, skip the apply call. Mark the run terminal but instruct the sandbox to persist (the lifecycle config's `DeleteOn` / `StopOn` are bypassed in this case — sandbox stays until operator action).
11. Remove `tryAutoApproval`, `autoApprove`, `autoReject`, `autoApproveIfEmpty` (greenfield).

### Phase 4 — GC rule for abandoned manualReview=true sandboxes

12. Implement the rule chosen in Decision D1.

### Phase 5 — Tests

13. Five test cases in `run_executor_test.go` (table-driven where shape allows):
    - `TestRunExecutor_ManualReviewFalse_AppliesOnSuccess` — success outcome + auto-apply path, in-acceptance changes only → `state=applied`.
    - `TestRunExecutor_ManualReviewFalse_AppliesOnFailure` — failure outcome + auto-apply still runs (apply identical regardless of `runOutcome`).
    - `TestRunExecutor_ManualReviewFalse_PartialAcceptanceSplit` — single run with one in-acceptance file (`state=applied`) + one out-of-acceptance file (`state=pending-review`) on the same run group.
    - `TestRunExecutor_ManualReviewTrue_DefersApply` — apply not called at run end; sandbox persists; provenance entries all `state=pending-review`.
    - `TestRunExecutor_NoOpRun_WritesEmptyProvenance` — agent makes no edits; eager-creation provenance entry exists with file count 0.
14. Workspace-sandbox-side tests for the new request shape and per-file `state` writes.
15. GC tests for the manualReview-TTL path (Decision D1).

### Phase 6 — Documentation

16. Add a short package-doc paragraph at the top of `run_executor.go` pointing to `scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` as the canonical reference for sandbox/auditability behavior.

### Final: Cleanup & Verification

17. Run type checking (`go build ./...`) and fix ALL errors, even pre-existing ones in modified files.
18. Run linter (`golangci-lint run`) and fix ALL warnings in modified files.
19. Run unit tests (`go test ./...` in both `scenarios/agent-manager/api` and `scenarios/workspace-sandbox/api`) and fix any failures.
20. `vrooli scenario restart agent-manager` and `vrooli scenario restart workspace-sandbox`.
21. Verify health: `curl -s http://localhost:<agent-manager-port>/health` and the workspace-sandbox equivalent.

## Contract Decisions

| ID | Topic | Status |
|---|---|---|
| D1 | GC rule for abandoned `manualReview=true` sandboxes | **TBD — Round 1** |
| D2 | Migration approach for old `Acceptance.{AutoApprove,AutoReject,DisableAutoApproveIfEmpty}` + `RequiresApproval` field surface | **TBD — Round 1** |
| D3 | Provenance schema dependency sequencing with `execute/gct-pending-ai-provenance-hardening` | **TBD — Round 1** |
| D4 | Apply trigger placement in run executor lifecycle | **TBD — Round 1** |
| D5 | Mapping `domain.RunOutcome` (7 values) to contract's `runOutcome` (4 values) | **TBD — Round 1** |

## Testing Plan

| # | Behavior | File | Layer |
|---|---|---|---|
| 1 | `manualReview=false` + success outcome → applies in-acceptance changes; provenance file states `applied` | `run_executor_test.go` | Unit (mock sandbox provider) |
| 2 | `manualReview=false` + failure outcome → applies identically; `runOutcome=failure` recorded | `run_executor_test.go` | Unit |
| 3 | Partial-acceptance split: one in-acceptance file `applied`, one out-of-acceptance file `pending-review` on same run group | `run_executor_test.go` | Unit |
| 4 | `manualReview=true` → apply NOT called at run end; sandbox persists; all files `pending-review` | `run_executor_test.go` | Unit |
| 5 | No-op run (no edits) → empty provenance record exists with eager creation metadata | `run_executor_test.go` | Unit |
| 6 | Workspace-sandbox apply request accepts `agent_manager_run_id`, `conversation_id`, `cost`, `run_outcome` and persists per-file `state` | `service_test.go` | Service-layer integration |
| 7 | Abandoned `manualReview=true` sandbox is GC'd per the chosen rule | `gc_test.go` | Unit |
| 8 | Validation rejects `mode=protected` with "reserved" error | `validation_test.go` | Unit |
| 9 | Removed fields (`Acceptance.AutoApprove` etc.) no longer compile in any reference | Compile-time | Build check |

## Rollout / Validation Checklist

- [ ] Provenance schema fields (`runOutcome` on `ProvenanceRunGroup`, `state` on `ProvenanceFile`) available per Decision D3.
- [ ] All five run-executor tests pass.
- [ ] Workspace-sandbox apply tests pass.
- [ ] GC tests pass.
- [ ] `go build ./...` clean in both scenarios.
- [ ] `golangci-lint run` clean in modified files.
- [ ] Both scenarios restart healthy.
- [ ] Manual sanity: spawn an agent-manager run via the existing UI surface; verify a provenance entry appears in GCT's AI Changes tab with run_id / conversation_id / cost / runOutcome populated (gated on `gct-pending-ai-provenance-hardening` UI work — see Decision D3).
- [ ] `mode=protected` request errors with "reserved".

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `gct-pending-ai-provenance-hardening` schema lands later than planned, blocking provenance writes | Medium | High | Decision D3 pins the sequencing strategy. If schema-first is chosen, this item blocks until schema lands. If parallel, this item writes against a feature-flagged stub and the full-fidelity contract becomes observable when the flag flips. |
| Greenfield removal of `Acceptance.AutoApprove` etc. breaks downstream callers | Medium | Medium | Search all references in agent-manager + cli-core + test-genie + web-console + swarm-manager before removing; update or remove every call-site in the same change set. |
| Sandbox-persists-beyond-run-end leaks resources if GC rule is too lenient | Medium | Medium | Decision D1. The rule must have an upper bound; ops-visible metric (`pending_review_sandbox_age_seconds`) optional but recommended. |
| `runOutcome` mapping from `domain.RunOutcome` (7 values) to contract's 4 values is lossy | Low | Low | Decision D5 documents the mapping explicitly. The lossiness is by design (contract wants 4 buckets for GCT rendering). |
| Apply-on-failure reveals partial / broken work to the canonical repo | Medium | Medium | Acceptance allow/deny is the gate. Operators who fear apply-on-failure should set narrower `acceptanceAllow` or opt into `manualReview=true`. Documentation note in `AUDITABILITY_CONTRACT.md`. |
| Removing `tryAutoApproval` regresses the existing empty-sandbox auto-approval ergonomics | Low | Low | The eager-provenance + auto-apply default + acceptance gating preserve the same observable behavior for the empty case (no files → no apply needed → run terminal). Test #5 pins this. |

## Non-goals / Prohibited Patterns

- Do NOT preserve `Acceptance.AutoApprove` / `Acceptance.AutoReject` / `Acceptance.DisableAutoApproveIfEmpty` / `RunConfig.RequiresApproval` as deprecated aliases. Greenfield.
- Do NOT add a feature flag for "old apply behavior". The contract is the contract.
- Do NOT pass `manualReview` / `autoApply` / `mode` etc. into the agent prompt or runner. Agent-side awareness is none.
- Do NOT couple `runOutcome` to apply gating. Apply behavior is identical regardless of outcome; outcome is metadata only.
- Do NOT split a single run group across `applied` and `pending-review` runs. Per-file `state` keeps the run group whole.
- Do NOT touch tool-level `RequiresApproval` in workspace-sandbox `toolregistry`. Separate concern.
- Do NOT remove `RunModeInPlace`. That's `agent-manager-default-sandboxing-rollout`'s job.

## Definition of Done

1. All five run-executor tests pass.
2. Workspace-sandbox apply path persists `agent_manager_run_id`, `conversation_id`, `cost`, `runOutcome` (run group) and per-file `state`.
3. The old field surface (`Acceptance.AutoApprove` etc., `RunConfig.RequiresApproval`) is removed from the codebase; `git grep` confirms zero references in production code.
4. `mode=protected` errors with "reserved".
5. GC rule for abandoned `manualReview=true` sandboxes lands per Decision D1 with passing tests.
6. `vrooli scenario restart agent-manager` and `vrooli scenario restart workspace-sandbox` both produce healthy services.
7. `go build ./...` and `golangci-lint run` clean in modified files (including pre-existing issues).
8. Package-doc reference to `AUDITABILITY_CONTRACT.md` lands in `run_executor.go`.

## Cross-Initiative Implications (for orchestrator)

- This item is the writer side. The reader / schema side lives in `execute/gct-pending-ai-provenance-hardening` (initiative `git-control-tower-ai-provenance`, upstream). The exact sequencing is Decision D3.
- Downstream `execute/sandbox-runtime-e2e-verification` cannot start its multi-surface validation until this item lands. Flag if D3 selects schema-first sequencing — the chain becomes serial.
- Downstream `execute/agent-manager-default-sandboxing-rollout` flips the spawn-surface defaults; nothing in this item affects the default `RunMode` (still `in_place` or whatever spawn surfaces choose today).
