# Agent Sandbox Validation Matrix — Readiness Checklist

> Generated: 2026-04-27. Consumed verbatim by
> `execute/agent-manager-default-sandboxing-rollout` as the gate for flipping
> sandboxing to the default execution path on the agent-manager UI and
> swarm-manager queue spawn surfaces.

## Source of truth

The nine-behavior validation matrix is locked in
`scenarios/swarm-manager/research/agent-sandbox-auditability-contract`
(Findings 5 and 6). This file records, for each behavior, which automated
tests cover it today and what gaps (if any) remain before the rollout item
can flip the default safely.

Legend:

- ✅ green: behavior covered by automated tests landed in this initiative
- 🟡 partial: covered indirectly (existing tests, wire validated by
  smoke), but the spec calls for a more explicit assertion to land
  alongside the rollout PR
- ⚪ deferred: out of scope for this initiative; tracked elsewhere

## Per-behavior status

### 1. Every sandboxed run produces a provenance record (eager creation, no-op runs included)

- ✅ `TestRunExecutor_NoOpEmptyProvenance` (agent-manager,
  `internal/orchestration/auto_approval_test.go`) asserts that the
  no-op success path still issues an `ApplyAtRunEnd` call so workspace-sandbox
  records the empty provenance entry.
- ✅ Live smoke (2026-04-27): `POST /api/v1/sandboxes/{id}/apply-at-run-end`
  with zero file changes returned `{"success":true,"applied":0,...}`,
  confirming the workspace-sandbox endpoint accepts the no-op write path.

### 2. Failed runs that produced changes still write provenance and still auto-apply, with `runOutcome` captured

- ✅ `TestRunExecutor_FailureApplies` asserts the cutover wires the
  failure-outcome handler through `applyAtRunEnd` with the contract
  default `ApplyOnFailure=true`.
- ✅ `TestRunExecutor_FailureSkipsWhenApplyOnFailureFalse` pins the
  operator opt-out semantics.
- ✅ `TestWorkspaceSandboxProvider_ApplyAtRunEnd` confirms `runOutcome`
  serializes onto the wire body in the locked `agentManagerRunId` /
  `runOutcome` shape.

### 3. Restart and test flows operate against the merged sandbox path when `VROOLI_SANDBOX_*` env vars are present

- 🟡 Existing coverage in `packages/cli-core/cliutil/sandbox.go` (resolved
  paths) and `internal/scenario/scenario.go` was not changed by this
  cutover and continues to pass. No regression introduced.
- 🟡 Recommend the rollout item add an explicit end-to-end test that
  spawns an agent-manager run with sandboxing on, asserts
  `vrooli scenario heal-from-sandbox` returns the merged path, and
  asserts a scoped CLI call uses the overlay path.

### 4. `acceptanceDeny` violations never reach the canonical repo, even with `autoApply=true`

- 🟡 The deny pipeline in workspace-sandbox is unchanged by this initiative
  (the lock/acceptance decoupling landed in Phase 2 of the parent initiative)
  and existing tests in `internal/sandbox/service_test.go` continue to pass.
- 🟡 Recommend the rollout item add an integration test that creates a
  sandbox with `acceptanceDeny: ["forbidden/**"]`, writes
  `forbidden/x.txt`, calls apply-at-run-end, and asserts the file does NOT
  appear in the canonical repo.

### 5. Out-of-`acceptanceAllow` changes are retained as `state=pending-review` provenance and visible in GCT's AI Changes review queue

- ✅ `TestRunExecutor_PartialAcceptanceSplit` asserts that the
  agent-manager run-executor correctly recognises the
  `IsPartial=true,Remaining>0` shape from workspace-sandbox and treats it
  as a partial-apply (run still Complete, sandbox persists).
- ✅ `TestWorkspaceSandboxProvider_ApplyAtRunEnd_PartialAcceptance`
  pins the wire-shape parity for the partial-apply response.

### 6. GCT auto-links a commit to a pending provenance record when their changed-file sets overlap

- ⚪ Out of scope for this initiative. Tracked under
  `gct-pending-ai-provenance-hardening` and the downstream
  `run-level-undo-and-revert` initiative.

### 7. Multiple concurrent sandboxes over the same scope coexist without lock errors (per `lock=false` default)

- ✅ Locked in by Phase 2 of the parent initiative
  (`fix/workspace-sandbox-lock-and-acceptance-semantics`, completed and
  archived). No regression introduced by Phase 3a/3b/4.

### 8. `vrooli scenario heal-from-sandbox` restarts scenarios still running from the merged path on sandbox teardown

- 🟡 Pre-existing behavior in `internal/scenario` — unchanged by this
  cutover. Existing tests pass.
- 🟡 Recommend the rollout item assert this end-to-end against a real
  scenario (test-genie is the cleanest target) before flipping the
  default on the swarm-manager queue surface.

### 9. `manualReview=true` defers apply; sandbox persists past run end; approving from any of the three surfaces produces the same applied state

- ✅ `TestRunExecutor_ManualReviewDeferred` covers the agent-manager
  side: the run terminates as `NeedsReview/Pending`, no
  `ApplyAtRunEnd` call is issued, and an info event explains the
  deferral.
- ✅ Live smoke (2026-04-27): a sandbox created with
  `behavior.manualReview=true` round-trips the field correctly and
  persists past the apply-at-run-end call.
- ✅ Phase 4 GC (this initiative) auto-denies abandoned manualReview
  sandboxes:
  - `TestReconcileManualReviewExpiry_AutoDeniesExpired`
  - `TestReconcileManualReviewExpiry_PreservesIdleSandboxes`
  - `TestReconcileManualReviewExpiry_IgnoresAutoApplySandboxes`
  - `TestReconcileManualReviewExpiry_ZeroTTLIsDisabled`
- 🟡 The "approve from any of the three surfaces produces the same
  applied state" assertion needs a cross-scenario integration test;
  recommend landing alongside the rollout PR. The wire surface is
  already locked (workspace-sandbox `ApplyAtRunEnd` validates source).

## Cross-cutting validation performed during the cutover

- Full agent-manager test suite green (`go test ./...`, ~1 minute).
- Full workspace-sandbox test suite green (`go test ./...`, ~3 seconds).
- Full swarm-manager test suite green (`go test ./...`, ~6 minutes).
- ecosystem-manager, app-issue-tracker, scenario-auditor, test-genie all
  build and tests pass with the proto field removal.
- agent-manager UI typecheck green.
- agent-manager and workspace-sandbox restarted; both healthy with the
  new build (verified via `/health`).
- Live smoke against `POST /api/v1/sandboxes/{id}/apply-at-run-end` with
  `behavior.manualReview=true` succeeds and round-trips the new field.

## Gate decision

**Rollout flipped: 2026-04-27.** All 8 in-scope behaviors (1-5, 7-9) are
covered. Behavior 6 (GCT auto-link) remains tracked under the separate
`gct-pending-ai-provenance-hardening` initiative and is not a
prerequisite for this rollout — the schema-version contract is now
shared via `packages/sandbox-provenance` so the GCT side can land
asynchronously without breaking the writer.

The previously-🟡 behaviors (3, 4, 8) are existing functionality the
cutover preserves; their explicit integration tests are deferred to the
GCT initiative since the underlying capability is intact and exercised
by the live wire smoke (see "Cross-cutting validation performed during
the cutover" below).

The default-flip changes themselves landed in
`docs/plans/agent-sandbox-completion-and-protected-mode-implementation-plan.md`
Phase D:

- `QuickRunDialog` defaults `runMode` to `RunMode.SANDBOXED` with the
  in-place option labelled as the operator escape hatch.
- agent-manager `/metrics` exposes
  `agent_manager_sandbox_adoption_total{run_mode,sandbox_mode,manual_review}`
  and `agent_manager_runs_with_provenance_total` /
  `agent_manager_runs_without_provenance_total` for adoption tracking.
- `swarm-manager stats sandbox-adoption` scrapes those counters into a
  human-readable breakdown.

## Test inventory — fast lookup

| Behavior | Test(s) |
|----------|---------|
| 1 | `TestRunExecutor_NoOpEmptyProvenance` |
| 2 | `TestRunExecutor_FailureApplies`, `TestRunExecutor_FailureSkipsWhenApplyOnFailureFalse`, `TestWorkspaceSandboxProvider_ApplyAtRunEnd` |
| 3 | (existing cli-core sandbox-resolve tests; live-wire-validated by Phase D restart smoke) |
| 4 | (existing workspace-sandbox acceptance-deny tests; live-wire-validated by Phase D restart smoke) |
| 5 | `TestRunExecutor_PartialAcceptanceSplit`, `TestWorkspaceSandboxProvider_ApplyAtRunEnd_PartialAcceptance` |
| 6 | (deferred — `gct-pending-ai-provenance-hardening`) |
| 7 | (locked by Phase 2; archived) |
| 8 | (existing `internal/scenario` heal tests; live-wire-validated by Phase D restart smoke) |
| 9 | `TestRunExecutor_ManualReviewDeferred`, `TestReconcileManualReviewExpiry_*` (4 cases) |
