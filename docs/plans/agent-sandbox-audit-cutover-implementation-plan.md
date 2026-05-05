# Agent Sandbox Audit Foundation — Cutover Implementation Plan

> Plan target: **execute the load-bearing remainder of `agent-sandbox-audit-foundation`**
> (Phases 3a, 3b, 4) plus the on-deck verification gate (Phase E2E),
> sequenced so partial progress never leaves agent-manager in a broken state
> for swarm-manager (which depends on agent-manager to drive its own runs).

## 1. Purpose

Make `workspace-sandbox` the reliable, default audit path for every
agent-manager coding run by:

1. Wiring the `Provider.ApplyAtRunEnd` seam on agent-manager and pointing it at
   the existing workspace-sandbox `POST /api/v1/sandboxes/{id}/apply-at-run-end`
   endpoint (Phase 3a).
2. Cutting `RunExecutor` over from the legacy `tryAutoApproval` /
   `RequiresApproval` / `Acceptance.AutoApprove*` fields to a single shared
   `applyAtRunEnd` call site after every terminal handler, including the
   eager provenance write at sandbox setup and the `RunOutcome.ToContract`
   (7→4) mapping (Phase 3b).
3. Adding `ManualReviewTTL`-driven GC for abandoned `manualReview=true`
   sandboxes in workspace-sandbox `LifecycleReconciler` (Phase 4).
4. Producing the validation matrix from the locked contract
   (`research/agent-sandbox-auditability-contract`) as automated tests so
   the default rollout item (`execute/agent-manager-default-sandboxing-rollout`)
   has a real pass/fail gate (Phase E2E).

Out of this plan but tracked downstream:
- `execute/sandbox-provenance-schema-version-shared-package` (Phase 5) — must
  coordinate with `gct-pending-ai-provenance-hardening` on a different branch.
- `execute/agent-manager-default-sandboxing-rollout` — flips defaults; happens
  only after Phase E2E passes.
- `execute/agent-manager-spawn-surface-conversation-id-population` — cross-cuts
  web-console / cron / CLI; depends on Phase 3b.
- `protected-agent-sandboxing` initiative — explicitly later per locked
  sequencing (tracking-first; protected-mode containment is an enhancement on
  top of the same auditability contract). See § 12.

## 2. Required Reading

Future agent must run, in order:

```bash
# Plan-authoring substrate
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement

# Locked contract — load-bearing for every decision in this plan
swarm-manager backlog file-get --kind research --name agent-sandbox-auditability-contract --path conclusion.md
cat scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md

# Initiative + remaining items
swarm-manager initiatives context --name agent-sandbox-audit-foundation
swarm-manager initiatives file-get --name agent-sandbox-audit-foundation --path context/2026-04-09-sandbox-audit-foundation.md
swarm-manager backlog get --kind execute --name agent-manager-apply-at-run-end-provider-seam
swarm-manager backlog get --kind execute --name agent-manager-run-executor-apply-at-run-end-cutover
swarm-manager backlog get --kind execute --name workspace-sandbox-manual-review-ttl-gc
swarm-manager backlog get --kind execute --name sandbox-runtime-e2e-verification

# Upstream initiative (post-cutover)
swarm-manager initiatives context --name protected-agent-sandboxing
swarm-manager initiatives file-get --name protected-agent-sandboxing --path context/2026-04-09-protected-mode-brief.md
```

## 3. Problem Statement

The auditability contract is locked (April 26, 2026) and Phases 1–2 have
landed. Today:

- **Wire-level pieces exist** in workspace-sandbox: `ApplyAtRunEndRequest`,
  `Service.ApplyAtRunEnd`, the `POST .../apply-at-run-end` route, and per-file
  `state` ∈ {applied, pending-review, denied} provenance.
- **Domain-level pieces exist** in agent-manager: `SandboxConfig.{Mode,
  ManualReview, AutoApply, ApplyOnFailure, NetworkMode, NoLock}`,
  `DefaultSandboxConfig()`, `ContractRunOutcome`, `RunOutcome.ToContract()`.
- **But the run executor still routes through the deprecated path:**
  `handleSuccessfulCompletion` → `tryAutoApproval` → `Acceptance.AutoApprove`
  / `Acceptance.AutoReject` / `Acceptance.DisableAutoApproveIfEmpty`, and is
  gated on `ResolvedConfig.RequiresApproval`. The workspace-sandbox
  `apply-at-run-end` endpoint is not invoked anywhere from agent-manager.
- **Failure path does not apply** — `handleFailure` never calls
  auto-approval, so accepted changes from a failed-but-useful run are lost.
  This is the single biggest gap relative to the locked contract.
- **No eager provenance write** — sandboxes that produce no diff currently
  leave no trace; the contract requires an empty provenance record at
  sandbox setup so every run is auditable.
- **`manualReview=true` sandboxes leak indefinitely** — `ManualReviewTTL`
  is configured but `LifecycleReconciler` does not enforce it.

These gaps mean the default-rollout item cannot honestly flip `sandboxed=true`
as the platform default: the auditability promise still has holes in the
failure and no-op cases, and abandoned manual-review sandboxes accumulate.

The user's coupling concern — "swarm-manager itself uses agent-manager, so
breaking agent-manager mid-cutover breaks swarm-manager" — is a real risk and
shapes § 7 sequencing: each phase must leave both the legacy and
contract-faithful paths working until the final cutover, and the cutover
must be a single atomic commit (or PR) that flips all call sites.

## 4. Scope

**In scope:**
- agent-manager `Provider` seam: add `ApplyAtRunEnd`.
- agent-manager `WorkspaceSandboxProvider`: implement
  `ApplyAtRunEnd` against `POST /api/v1/sandboxes/{id}/apply-at-run-end`.
- agent-manager `RunExecutor`: replace the auto-approval branch in
  `handleSuccessfulCompletion`, add an apply-at-run-end branch in
  `handleFailure`, and write the eager-provenance call at sandbox setup.
- agent-manager domain types: drop `Acceptance.AutoApprove`,
  `Acceptance.AutoReject`, `Acceptance.DisableAutoApproveIfEmpty`,
  `RunConfig.RequiresApproval` (and any callers).
- workspace-sandbox `LifecycleReconciler`: enforce
  `LifecycleConfig.ManualReviewTTL`.
- Test coverage: the six run-executor cases listed in the spec, plus a
  clock-injected TTL test, plus the nine validation-matrix scenarios from
  the contract.

**Out of scope:**
- Cross-branch shared schema-version package (Phase 5; coordinate with
  `gct-pending-ai-provenance-hardening` separately).
- Default-rollout flips on UI / queue surfaces (separate item; gated on
  Phase E2E).
- `Run.ConversationID` / `Run.ParentRunID` population at non-default
  spawn surfaces (separate item; depends on this plan landing).
- All `protected-agent-sandboxing` items (process-launch through
  workspace-sandbox `/exec`, git/network guardrails, policy enforcement
  surface). Plan acknowledges them in § 12 but does not implement.

## 5. Current Technical Context

**agent-manager (this repo):**
- `path:scenarios/agent-manager/api/internal/orchestration/run_executor.go`
  - `handleSuccessfulCompletion` (~line 1403) — gated on
    `ResolvedConfig.RequiresApproval`; calls `tryAutoApproval`.
  - `handleFailure` (~line 1454) — does not apply.
  - `tryAutoApproval` (~line 1686) — branches on legacy
    `Acceptance.AutoApprove` / `AutoReject` /
    `DisableAutoApproveIfEmpty`.
  - `applySandboxLifecycle` is already invoked at terminal events.
- `path:scenarios/agent-manager/api/internal/adapters/sandbox/interface.go`
  - `Provider` interface has `Approve`, `Reject`, `PartialApprove`,
    `GetDiff`, etc. — does **not** have `ApplyAtRunEnd`.
- `path:scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox.go`
  - HTTP adapter; `doRequest` helper exists; pattern for new endpoint
    follows existing `Approve` / `Reject` calls.
- `path:scenarios/agent-manager/api/internal/domain/types.go`
  - `SandboxAcceptanceConfig` (line 204) holds the deprecated trio
    (autoApprove / autoReject / disableAutoApproveIfEmpty).
  - `SandboxConfig` (line 276) holds the new contract levers and is the
    target shape after cutover.
  - `DefaultSandboxConfig()` (line 332) returns the locked defaults.
  - `RunConfig.RequiresApproval` (line 54) is the legacy gate to remove.
- `path:scenarios/agent-manager/api/internal/domain/decisions.go`
  - `RunOutcome` (line 232) — 7 values.
  - `ContractRunOutcome` (line 286) — 4 values.
  - `RunOutcome.ToContract()` exists per `auditability_contract_test.go`.
- `path:scenarios/agent-manager/api/internal/orchestration/sandbox_config_test.go`
  - Fixtures exercise the deprecated fields; will need refactoring.

**workspace-sandbox:**
- `path:scenarios/workspace-sandbox/api/internal/sandbox/service.go`
  - `ApplyAtRunEnd` (line 1763) — implemented; validates request and
    routes through the same apply pipeline as operator approval, with
    `Source=SourceAgentManagerAutoApply`.
- `path:scenarios/workspace-sandbox/api/internal/handlers/handlers.go`
  - Route `POST /api/v1/sandboxes/{id}/apply-at-run-end` registered
    (line 207).
- `path:scenarios/workspace-sandbox/api/internal/types/auditability_contract_test.go`
  - `ApplyAtRunEndRequest` JSON contract is locked and tested.
- `path:scenarios/workspace-sandbox/api/internal/lifecycle/` (TBD path —
  verify with `rg -l LifecycleReconciler scenarios/workspace-sandbox`)
  - Hosts `LifecycleReconciler.ReconcileLifecycle`; needs a TTL branch.
- `path:scenarios/workspace-sandbox/api/internal/config/config.go`
  - Already exposes `LifecycleConfig.ManualReviewTTL` and
    `WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL`.

**Cross-cutting:**
- The locked contract is mirrored at
  `path:scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` and the
  conclusion file inside the research backlog item. Treat both as
  source-of-truth; if they disagree, the conclusion.md wins.

## 6. Target End State

After this plan lands:

1. `Provider.ApplyAtRunEnd(ctx, *ApplyAtRunEndRequest) (*ApprovalResult, error)`
   exists on the agent-manager sandbox provider seam, implemented in
   `WorkspaceSandboxProvider`, and covered by adapter tests against an
   `httptest.Server`.
2. Every terminal handler in `RunExecutor.Execute()` (`handleSuccessfulCompletion`,
   `handleFailure`, cancel/timeout paths) routes through a single shared
   `applyAtRunEnd(ctx)` helper instead of the legacy `tryAutoApproval`
   and `RequiresApproval` switches.
3. The eager provenance write happens at sandbox setup so every run —
   including no-op runs — leaves a provenance record correlating
   sandbox → run → conversation → cost, with `runOutcome` stamped on
   completion via `RunOutcome.ToContract()`.
4. `SandboxAcceptanceConfig.AutoApprove`, `.AutoReject`,
   `.DisableAutoApproveIfEmpty`, and `RunConfig.RequiresApproval` are
   removed from domain types, proto, generated code, and all call sites.
   `sandbox_config_test.go` fixtures are refactored. Persisted runs that
   still carry the old fields are normalized at DB read time (forward
   compat shim — see § 11).
5. `LifecycleReconciler.ReconcileLifecycle` auto-denies any
   `manualReview=true` sandbox whose `ManualReviewTTL` has expired since
   run end, recording the system source on the state transition.
6. The nine-behavior validation matrix from the locked contract is
   implemented as integration tests, surfaced as a single
   `make audit-validation` (or equivalent) target, and the pass/fail output
   is the exact gate consumed by `agent-manager-default-sandboxing-rollout`.

## 7. Implementation Strategy

Sequenced so that swarm-manager — which uses agent-manager to drive its
own runs — never sees a half-cut state. Each phase ends with a green
build + tests; only Phase 3b removes the legacy fields.

### Phase 3a — Add the seam (item: `agent-manager-apply-at-run-end-provider-seam`)

Effort: M. No behavior change.

1. Extend `Provider` in
   `path:scenarios/agent-manager/api/internal/adapters/sandbox/interface.go`
   with `ApplyAtRunEnd(ctx context.Context, req ApplyAtRunEndRequest) (*ApplyResult, error)`.
   - Define `ApplyAtRunEndRequest` and `ApplyResult` mirroring the
     workspace-sandbox `types.ApplyAtRunEndRequest` and
     `types.ApprovalResult` shapes (do not import workspace-sandbox; keep
     the seam adapter-owned).
2. Implement on `WorkspaceSandboxProvider` in `workspace_sandbox.go`
   following the existing `Approve` pattern: marshal body, call
   `doRequest("POST", "/api/v1/sandboxes/"+id+"/apply-at-run-end", body)`,
   decode the typed response, surface conflict / not-found errors via
   the existing structured error parsing.
3. Add adapter tests in `workspace_sandbox_test.go` covering: happy path,
   conflict (HTTP 409), not-found (HTTP 404), bad-request validation
   error, and source-must-be-agent-manager-auto-apply rejection.
4. Build + test agent-manager. **No call sites change yet.**

Exit criteria: `go test ./...` green in agent-manager; new method
unused (intentional).

### Phase 3b — Cutover (item: `agent-manager-run-executor-apply-at-run-end-cutover`)

Effort: L. **Atomic** — must land in one PR/commit because it removes
deprecated fields.

Order within the phase:

1. **Eager provenance write** — in `RunExecutor` (sandbox setup, near
   where `e.sandbox` and `e.sandboxID` are first populated), call a new
   `provider.WriteEagerProvenance(ctx, runID, sandboxID, conversationID, parentRunID)`
   helper (or extend `Create`/`Get` if the workspace-sandbox API already
   accepts it on creation — verify with the locked contract). The
   record's `runOutcome` is left empty; populated on terminal handler.
2. **Single shared apply-at-run-end helper** — add
   `(e *RunExecutor) applyAtRunEnd(ctx context.Context, outcome domain.ContractRunOutcome) bool`.
   Signature and behavior:
   - Resolves `effectiveSandboxConfig()`; bail with a warn event if nil
     (matches the existing `tryAutoApproval` defensive log).
   - Skips when `cfg.ManualReview == true` — sets state pending and
     returns `false` (the sandbox persists; operator approval comes
     later via any of the three surfaces).
   - Skips when `cfg.GetAutoApply() == false`.
   - Honors `cfg.GetApplyOnFailure()` for non-success outcomes.
   - Builds an `ApplyAtRunEndRequest{RunID, RunOutcome: outcome,
     ConversationID, CostUsd, Source: SourceAgentManagerAutoApply}` and
     invokes `e.sandbox.ApplyAtRunEnd(ctx, req)`.
   - On success, sets `run.ApprovalState`, `run.ApprovedBy =
     "applyAtRunEnd"`, `run.ApprovedAt`, `run.Status = Complete`.
   - On a `pending-review` partial result (out-of-acceptance files),
     records that the sandbox persists with `state=pending-review` and
     emits an info event; does not transition the run to NeedsReview —
     the run itself is complete.
3. **Replace `tryAutoApproval` call sites** — both
   `handleSuccessfulCompletion` and `handleFailure` (and any cancel /
   timeout terminal handler) call `e.applyAtRunEnd(ctx, e.classifyOutcome().ToContract())`
   immediately after they have set their terminal status fields, before
   `e.applySandboxLifecycle(...)`.
4. **Remove deprecated fields** — delete:
   - `SandboxAcceptanceConfig.AutoApprove`
   - `SandboxAcceptanceConfig.AutoReject`
   - `SandboxAcceptanceConfig.DisableAutoApproveIfEmpty`
   - `RunConfig.RequiresApproval`
   - `RunSummary.RequiresApproval` (mirror field at line 701)
   - The `tryAutoApproval`, `autoApprove`, `autoReject`,
     `autoApproveIfEmpty` methods on `RunExecutor`.
5. **Proto + generated code** — regenerate proto for any field deletions
   that touch the wire. Verify no other scenario depends on these
   fields by `rg "RequiresApproval|AutoApprove|DisableAutoApproveIfEmpty" --type go`.
6. **Refactor `sandbox_config_test.go`** to assert the new contract
   (`SandboxConfig` levers, not the deprecated trio).
7. **Add the six new RunExecutor tests** specified in the spec:
   - `TestRunExecutor_SuccessApplies` — success outcome, AutoApply=true,
     in-acceptance files apply.
   - `TestRunExecutor_FailureApplies` — exit-error outcome,
     ApplyOnFailure=true, in-acceptance files still apply.
   - `TestRunExecutor_PartialAcceptanceSplit` — some files in
     acceptanceAllow, some out; in-acceptance apply, out-of-acceptance
     persist as `state=pending-review`.
   - `TestRunExecutor_ManualReviewDeferred` — ManualReview=true defers
     apply; sandbox persists; run terminal status is Complete with
     ApprovalState=Pending.
   - `TestRunExecutor_NoOpEmptyProvenance` — no diff; eager provenance
     entry exists with `runOutcome=success` and zero file records.
   - `TestRunExecutor_ConversationIDInheritance` — when `Run.ConversationID`
     is empty but `Run.ParentRunID` is set, conversation id is inherited
     from the parent for the apply call.
8. **DB read-time normalization** — in the run repository (likely
   `path:scenarios/agent-manager/api/internal/database/repository_run.go`),
   on row decode, if a persisted JSON config still carries
   `acceptance.autoApprove==true` or `requiresApproval` flags, map them
   onto the new contract levers (`AutoApply`, `ManualReview`) so legacy
   rows continue to behave correctly. Cover with a read-time fixture
   test.

Exit criteria: `go build ./...` and `go test ./...` green across
agent-manager. `rg "RequiresApproval|AutoApprove|DisableAutoApproveIfEmpty"
scenarios/agent-manager` returns zero hits. Manual smoke: spawn a run
through the agent-manager UI with both `manualReview=true` and
`manualReview=false`, verify provenance lands as expected.

### Phase 4 — TTL GC (item: `workspace-sandbox-manual-review-ttl-gc`)

Effort: S. **Parallelizable** with Phase 3b — independent file set, no
shared types changed.

1. Locate `LifecycleReconciler.ReconcileLifecycle` in workspace-sandbox.
2. Add a branch: for each sandbox with `ManualReview=true` and
   `state ∈ {pending-review}` whose age since run end exceeds
   `LifecycleConfig.ManualReviewTTL` (per the injected clock), invoke
   the existing deny pipeline with `Source=SourceSystemTTLExpiry`
   (add the source enum value if it does not exist) and proceed to
   teardown via the existing `PolicyConfig.TeardownHooks`.
3. Test with a clock-injection fixture (the codebase already uses
   `clock.Clock` style injection in `gc/`; mirror that pattern).
4. Document the new env var behavior in
   `path:scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` (it is
   already mentioned; tighten to "enforced by reconciler since YYYY-MM-DD").

Exit criteria: workspace-sandbox tests green; new TTL test exercises
expiry → auto-deny → teardown end-to-end with a fake clock.

### Phase E2E — Validation matrix (item: `sandbox-runtime-e2e-verification`)

Effort: M-L. Runs after Phase 3b + Phase 4 land. This is the gate the
default-rollout item consumes.

For each of the nine behaviors in the locked contract Finding 5, write
an automated test exercising the agent-manager UI spawn surface and the
swarm-manager queue spawn surface. Cron + direct-CLI surfaces are out
of scope per the locked validation matrix.

The deliverable structure (mirrors the spec verbatim):

1. `tests/e2e/sandbox/` (or scenario-level integration tests) — nine
   test cases, one per locked behavior.
2. Focused regression tests around `path:packages/cli-core/cliutil/sandbox.go`
   and the Go scenario-restart code in `path:internal/scenario`,
   `path:internal/scenarioexec`, `path:internal/cli/vroolicli`,
   `path:internal/repocontractcheck`.
3. A pass/fail readiness checklist file at
   `path:docs/plans/agent-sandbox-validation-matrix-readiness.md` updated by
   the test runner output. The default-rollout item consumes this file
   verbatim.
4. Documentation of any behavior that cannot yet be covered automatically,
   plus the shortest path to cover it later.

Exit criteria: matrix green on agent-manager UI + swarm-manager queue
spawn surfaces; readiness checklist file generated; gaps explicitly
documented.

## 8. Contract Decisions

| Decision | Resolution | Source |
|----------|------------|--------|
| Apply-at-run-end source enum value | `SourceAgentManagerAutoApply` (already locked in workspace-sandbox `apply_at_run_end_test.go`) | workspace-sandbox tests |
| Out-of-acceptance file disposition | Persist as `state=pending-review` in same provenance record; surface in GCT review queue | Locked contract D-OOA |
| `manualReview=true` lifecycle | Sandbox persists past run end; auto-deny on `ManualReviewTTL` (Phase 4); approval from any of {GCT, agent-manager, workspace-sandbox} records originating surface | Locked contract D-MR |
| `runOutcome` mapping | `RunOutcome.ToContract()` 7→4 (already implemented and tested in `auditability_contract_test.go`) | agent-manager domain |
| Eager provenance for no-op runs | Provenance record written at sandbox creation; `runOutcome=success` stamped on completion; zero file rows | Locked contract D-EP |
| Legacy field removal vs deprecation window | Hard removal in Phase 3b — no deprecation window. DB read-time normalization handles persisted rows | User feedback (greenfield default) |
| Validation gate scope | agent-manager UI + swarm-manager queue spawn surfaces only. Cron and direct-CLI are separate items | Locked contract Finding 6 |

## 9. Testing Plan

Per `feedback_testing_over_manual.md`: every behavior is verified by an
automated test, not a manual checklist.

| Layer | Test additions |
|-------|---------------|
| agent-manager adapter | 5 `WorkspaceSandboxProvider.ApplyAtRunEnd` tests (Phase 3a) |
| agent-manager domain | Refactored `sandbox_config_test.go`; `repository_run.go` read-time normalization fixture |
| agent-manager run-executor | 6 cases listed in Phase 3b step 7 |
| workspace-sandbox lifecycle | Clock-injected `ManualReviewTTL` expiry → auto-deny → teardown |
| Cross-scenario E2E | 9 validation-matrix cases × 2 spawn surfaces (Phase E2E) |
| cli-core / scenario-restart | Regression tests around `cliutil/sandbox.go` and `path:internal/scenario*` (Phase E2E deliverable 2) |

Long-running command timeouts (per CLAUDE.md guidance): set `--timeout
600s` for full agent-manager `go test ./...` runs.

## 10. Rollout / Validation Checklist

**Per phase:**
- [ ] `go build ./...` green in the changed scenario.
- [ ] `gofumpt -w` applied to changed Go files.
- [ ] `golangci-lint run` green.
- [ ] Scenario restarted via `vrooli scenario restart agent-manager` (or
      `workspace-sandbox`) — health endpoint reachable.

**End of plan:**
- [ ] `rg "RequiresApproval|AutoApprove|DisableAutoApproveIfEmpty"
      scenarios/agent-manager` returns zero hits.
- [ ] Validation matrix readiness checklist written to
      `path:docs/plans/agent-sandbox-validation-matrix-readiness.md` with
      9/9 behaviors green on both spawn surfaces.
- [ ] swarm-manager queue spawn surface still drives runs end-to-end
      (since this plan touches the foundation it depends on).
- [ ] Each completed backlog item marked `status=completed` via
      `swarm-manager backlog update --kind execute --name <X>
      --data '{"status":"completed"}'` and archived.

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Phase 3b breaks swarm-manager mid-cutover (it depends on agent-manager) | M | High | Land Phase 3a first (no behavior change). Phase 3b lands as one atomic commit; restart agent-manager + swarm-manager together; smoke a swarm-manager queue spawn before pushing. |
| Persisted runs in DB still carry deprecated fields | High | M | DB read-time normalization shim in `repository_run.go` (Phase 3b step 8). Fixture-tested. |
| Proto / generated code drift hides callers | M | M | Regenerate, then `rg` over the whole tree (not just `path:scenarios/agent-manager`) for the deprecated names before merge. |
| `ApplyAtRunEnd` and `Approve` race (operator approves while terminal handler also auto-applies) | L | M | workspace-sandbox `ApplyAtRunEnd` is idempotent on already-applied sandboxes (verify in adapter test); on conflict, log warn and continue. |
| `manualReview=true` runs leak before Phase 4 lands | L | M | Phase 4 is parallelizable; land alongside or before Phase 3b. |
| Phase 5 schema-version package coordination delays persistence | M | M | Out of scope; new fields are wire-level and survive the round trip even if persistence lags. Document in `path:docs/plans/agent-sandbox-validation-matrix-readiness.md` as a known gap. |
| Protected-mode initiative changes process-launch path mid-flight | L | High | Protected mode is sequenced strictly after the default rollout (see § 12). Plan does not implement it. |

## 12. Coupling With `protected-agent-sandboxing`

The user flagged this initiative as a dependency. The locked contract and
the protected-mode brief both put protected mode **after** the audit
foundation — protected-mode items
(`execute/protected-sandbox-agent-launch`, `protected-sandbox-git-and-network-guardrails`,
`fix/protected-sandbox-policy-enforcement-surface`) explicitly depend on
`execute/sandbox-runtime-e2e-verification` and
`execute/agent-manager-sandbox-auto-apply-defaults`.

Implications for this plan:

1. **No protected-mode work in this plan.** The seam (`SandboxModeProtected`)
   is already reserved at the type level and rejected at validate time;
   protected-mode launch through workspace-sandbox `/exec` is intentionally
   future work.
2. **Cutover must not paint protected mode into a corner.** The
   `Provider.ApplyAtRunEnd` seam (Phase 3a) is process-launch agnostic —
   it operates on the sandbox by ID, not on the launch mechanism. So
   future protected-mode launch can adopt the same apply seam unchanged.
3. **Network mode contract.** `NetworkAccessLocalhost` is the locked
   default and is preserved by `DefaultSandboxConfig()`. Protected-mode
   guardrails (`none` / `localhost` / `full`) layer on top via
   workspace-sandbox runtime; agent-manager continues to pass the
   `NetworkMode` value through unchanged.
4. **Git allowlist.** Direct git restrictions belong in
   `protected-sandbox-git-and-network-guardrails`. The audit-foundation
   cutover does not touch git plumbing; GCT remains the trusted
   higher-trust mutation owner.
5. **Sequence after this plan completes:**
   - `execute/agent-manager-default-sandboxing-rollout` (flips defaults
     on UI + queue spawn surfaces).
   - `execute/sandbox-provenance-schema-version-shared-package`
     (cross-branch coord).
   - `execute/agent-manager-spawn-surface-conversation-id-population`
     (cron + CLI surfaces).
   - Then `protected-agent-sandboxing` items in the order their
     dependencies define.

## 13. Non-goals / Prohibited Patterns

- **No deprecation window for the legacy auto-approval fields.** They
  were already documented as deprecated in `types.go`; per
  `feedback_planning_guidelines.md` and `feedback_no_git_mutations.md`
  this is greenfield-default. Hard remove in Phase 3b.
- **No prompt-level changes to runners.** The agent must not need to
  know it is sandboxed (locked contract; restated in protected-mode
  brief).
- **No bolt-on of protected-mode behaviors into this plan.** Sequenced
  later for a reason.
- **No raw HTTP in the run executor.** All workspace-sandbox calls go
  through the `Provider` adapter (per `feedback_skills_use_cli_never_api.md`
  applied at the seam level).
- **No silent fallback when a spawn surface lacks contract support.**
  Per the rollout-item required outcome: explicit log/error rather than
  silent fallback.
- **No mass-update scripts.** Per CLAUDE.md, deprecated-field removal
  is done file-by-file with review.

## 14. Definition of Done

- [ ] Phase 3a: `Provider.ApplyAtRunEnd` exists, `WorkspaceSandboxProvider`
      implements it, 5 adapter tests pass.
- [ ] Phase 3b: `tryAutoApproval` deleted; `applyAtRunEnd` shared helper
      called from every terminal handler (success, failure, cancel,
      timeout); eager provenance write at sandbox setup;
      `RunOutcome.ToContract` wired; 6 run-executor tests pass;
      deprecated fields removed from domain + proto; DB read-time
      normalization in place.
- [ ] Phase 4: `LifecycleReconciler` enforces `ManualReviewTTL`;
      clock-injected expiry test passes; auto-deny records
      `Source=SourceSystemTTLExpiry`.
- [ ] Phase E2E: nine validation-matrix behaviors automated on
      agent-manager UI + swarm-manager queue spawn surfaces; readiness
      checklist file written; gaps documented.
- [ ] swarm-manager can spawn and complete a queue run end-to-end
      against the new path.
- [ ] All four backlog items marked `status=completed` and archived
      via `swarm-manager backlog update`.
- [ ] Plan acknowledged at the top of the next initiative review with
      pointers to the readiness checklist as the gate for
      `agent-manager-default-sandboxing-rollout`.
