// Tests for the finalize seam: ApplyAtRunEnd, ApplySandboxLifecycle, and
// the per-status lifecycle dispatch.
//
// These tests pin the contract that prevented the 2026-04-28 mount-leak
// incident from recurring:
//
//   - Sandbox teardown MUST run even when the caller's ctx is already
//     cancelled. Before the fix, applySandboxLifecycle was called with
//     execCtx (which carries the run's 60-min execution deadline); a slow
//     applyAtRunEnd could exhaust the deadline, after which Delete failed
//     silently and the fuse-overlayfs mount was leaked. After the fix,
//     ApplySandboxLifecycle uses a detached context.
//
//   - Every run reaches RunPhaseCompleted. Finalize advances the phase
//     ladder unconditionally; failure during teardown emits a warn event
//     but does not block phase advancement.
//
// Plus the auditability-contract behaviors for ApplyAtRunEnd:
//   1. Success → applies (in-acceptance changes apply at run end)
//   2. Failure → applies (when ApplyOnFailure=true, the contract default)
//   3. Partial acceptance → split (in-acceptance applies; out-of-acceptance
//      remain as state=pending-review on the provenance record)
//   4. ManualReview=true → deferred (sandbox persists; run lands as
//      NeedsReview/Pending)
//   5. No-op empty → eager provenance entry exists; apply is a no-op write
//   6. ConversationID inheritance is forwarded to the workspace-sandbox call

package phases

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration/testutil/assertx"
	"agent-manager/internal/orchestration/testutil/fixtures"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Test helpers
// -----------------------------------------------------------------------------

// finalizeFixture bundles the run, sandbox, and event store wired into a
// shape every test in this file consumes.
type finalizeFixture struct {
	run       *domain.Run
	sandbox   *mocks.FakeSandboxProvider
	sandboxID uuid.UUID
	events    *mocks.FakeEventStore
	deps      Deps
}

func newFinalizeFixture(t *testing.T, cfg *domain.SandboxConfig, stub *mocks.FakeSandboxProvider) *finalizeFixture {
	t.Helper()
	return newFinalizeFixtureWithRun(t, cfg, nil, stub)
}

func newFinalizeFixtureWithRun(t *testing.T, cfg *domain.SandboxConfig, run *domain.Run, stub *mocks.FakeSandboxProvider) *finalizeFixture {
	t.Helper()
	if run == nil {
		run = fixtures.NewRun(t, uuid.Nil, uuid.Nil)
	}
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	run.RunMode = domain.RunModeSandboxed
	run.SandboxConfig = cfg
	sbxID := uuid.New()
	ev := mocks.NewFakeEventStore()
	return &finalizeFixture{
		run:       run,
		sandbox:   stub,
		sandboxID: sbxID,
		events:    ev,
		deps: Deps{
			Events: ev,
			Levers: config.DefaultLevers(),
		},
	}
}

// sandboxedRunCfg returns a SandboxConfig that mirrors what
// resolveSandboxConfig produces for an auto-apply protected run:
// DeleteOn=[terminal] so finalize will issue Delete on terminal events.
func sandboxedRunCfg() *domain.SandboxConfig {
	return fixtures.NewSandboxConfig(nil,
		fixtures.WithSandboxDeleteOn(domain.SandboxLifecycleTerminal),
	)
}

// -----------------------------------------------------------------------------
// ApplySandboxLifecycle / Finalize regression gates
// -----------------------------------------------------------------------------

// TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx is the
// regression gate for the 2026-04-28 incident.
func TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ApplySandboxLifecycle(ctx, ApplySandboxLifecycleInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Event:     domain.SandboxLifecycleRunCompleted,
		Reason:    "regression",
	})

	if stub.DeleteCallCount() != 1 {
		t.Fatalf("expected 1 Delete call despite cancelled caller ctx, got %d (regression: 2026-04-28 mount leak)", stub.DeleteCallCount())
	}
	deleteCtxErrs := stub.DeleteContextErrs()
	if len(deleteCtxErrs) != 1 {
		t.Fatalf("expected 1 Delete context capture, got %d", len(deleteCtxErrs))
	}
	if deleteCtxErrs[0] != nil {
		t.Errorf("Delete was called with a context that already had ctx.Err()=%v — teardown ctx is not detached", deleteCtxErrs[0])
	}
}

func TestFinalize_AdvancesPhaseToCompleted(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	assertx.RunPhase(t, fx.run, domain.RunPhaseCompleted)
}

func TestFinalize_DeletesSandboxOnSuccess(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.DeleteCallCount() != 1 {
		t.Errorf("expected 1 Delete on successful run, got %d", stub.DeleteCallCount())
	}
	if stub.StopCallCount() != 0 {
		t.Errorf("expected 0 Stop calls when DeleteOn matches, got %d", stub.StopCallCount())
	}
}

func TestFinalize_DeletesSandboxOnFailure(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusFailed

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.DeleteCallCount() != 1 {
		t.Errorf("expected 1 Delete on failed run (DeleteOn=[terminal] matches RunFailed), got %d", stub.DeleteCallCount())
	}
}

func TestFinalize_DeleteFailureDoesNotBlockPhaseAdvance(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.DeleteErr = errors.New("workspace-sandbox unreachable")
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	assertx.RunPhase(t, fx.run, domain.RunPhaseCompleted)
	ops := fx.events.TypedEvents(fx.run.ID, domain.EventTypeSandboxOperation)
	if len(ops) != 1 {
		t.Fatalf("expected exactly one sandbox.operation event, got %d", len(ops))
	}
	payload, err := eventlog.Decode(domain.EventTypeSandboxOperation, ops[0].SchemaVersion, ops[0].Data.(*domain.TypedEventData).Body)
	if err != nil {
		t.Fatalf("decode sandbox.operation payload: %v", err)
	}
	op, ok := payload.(*eventlog.SandboxOperationPayload)
	if !ok {
		t.Fatalf("decoded payload is %T, want *SandboxOperationPayload", payload)
	}
	if op.Operation != eventlog.SandboxOpDelete {
		t.Errorf("Operation = %q, want %q", op.Operation, eventlog.SandboxOpDelete)
	}
	if op.Success {
		t.Error("expected Success=false on Delete failure")
	}
	if op.Message == "" {
		t.Error("expected Message to carry the underlying error")
	}
}

func TestFinalize_NoOpForInPlaceRun(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.RunMode = domain.RunModeInPlace
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.DeleteCallCount() != 0 || stub.StopCallCount() != 0 {
		t.Errorf("in-place run should not touch sandbox: deleteHits=%d stopHits=%d", stub.DeleteCallCount(), stub.StopCallCount())
	}
	assertx.RunPhase(t, fx.run, domain.RunPhaseCompleted)
}

func TestFinalize_StopsSandboxWhenLifecycleSaysStop(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.StopOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	cfg.Lifecycle.DeleteOn = nil

	stub := mocks.NewFakeSandboxProvider()
	fx := newFinalizeFixture(t, cfg, stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.StopCallCount() != 1 {
		t.Errorf("expected 1 Stop call when StopOn=[terminal] and DeleteOn empty, got %d", stub.StopCallCount())
	}
	if stub.DeleteCallCount() != 0 {
		t.Errorf("expected 0 Delete calls under StopOn lifecycle, got %d", stub.DeleteCallCount())
	}
}

func TestLifecycleEventForStatus_MapsAllTerminalStatuses(t *testing.T) {
	cases := []struct {
		status domain.RunStatus
		want   domain.SandboxLifecycleEvent
	}{
		{domain.RunStatusComplete, domain.SandboxLifecycleRunCompleted},
		{domain.RunStatusFailed, domain.SandboxLifecycleRunFailed},
		{domain.RunStatusCancelled, domain.SandboxLifecycleRunCancelled},
		{domain.RunStatusNeedsReview, domain.SandboxLifecycleRunCompleted},
		{domain.RunStatusRunning, domain.SandboxLifecycleRunFailed},
	}
	for _, tc := range cases {
		got := LifecycleEventForStatus(tc.status)
		if got != tc.want {
			t.Errorf("LifecycleEventForStatus(%s) = %s, want %s", tc.status, got, tc.want)
		}
	}
}

// -----------------------------------------------------------------------------
// ApplyAtRunEnd auditability-contract tests
// -----------------------------------------------------------------------------

// TestApplyAtRunEnd_NilConfigEmitsWarning is the regression gate for the
// pre-2026-04-24 silent-fallthrough bug carried over from tryAutoApproval.
func TestApplyAtRunEnd_NilConfigEmitsWarning(t *testing.T) {
	fx := newFinalizeFixture(t, nil, mocks.NewFakeSandboxProvider())
	got := ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	})
	if got {
		t.Errorf("ApplyAtRunEnd with nil config should return false, got true")
	}
	if _, ok := fx.events.FindLogMessage("apply-at-run-end skipped: run has no sandbox config"); !ok {
		t.Error("expected warn event describing nil sandbox config, got none")
	}
}

// (1) Success → apply.
func TestApplyAtRunEnd_SuccessApplies(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{Success: true, Applied: 2, TotalSizeBytes: 3072, DiffPath: "/api/v1/sandboxes/diff", CommitHash: "full-commit"}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Fatal("expected ApplyAtRunEnd to succeed")
	}
	if stub.ApplyAtRunEndCallCount() != 1 {
		t.Errorf("expected exactly 1 ApplyAtRunEnd call, got %d", stub.ApplyAtRunEndCallCount())
	}
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete, got %q", fx.run.Status)
	}
	if fx.run.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("expected ApprovalState=Approved, got %q", fx.run.ApprovalState)
	}
	if fx.run.ChangedFiles != 2 || fx.run.TotalSizeBytes != 3072 || fx.run.DiffPath != "/api/v1/sandboxes/diff" || fx.run.CommitHash != "full-commit" {
		t.Fatalf("persisted attribution = files:%d bytes:%d diff:%q commit:%q", fx.run.ChangedFiles, fx.run.TotalSizeBytes, fx.run.DiffPath, fx.run.CommitHash)
	}
	applyReqs := stub.ApplyAtRunEndRequests()
	if len(applyReqs) != 1 {
		t.Fatalf("expected 1 ApplyAtRunEnd request, got %d", len(applyReqs))
	}
	if applyReqs[0].RunOutcome != "success" {
		t.Errorf("expected runOutcome=success on wire, got %q", applyReqs[0].RunOutcome)
	}
}

// (2) Failure → apply (ApplyOnFailure=true is the contract default).
func TestApplyAtRunEnd_FailureApplies(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeFailure,
	}) {
		t.Fatal("expected apply on failure when ApplyOnFailure=true")
	}
	if stub.ApplyAtRunEndCallCount() != 1 {
		t.Errorf("expected 1 ApplyAtRunEnd call on failure, got %d", stub.ApplyAtRunEndCallCount())
	}
	applyReqs := stub.ApplyAtRunEndRequests()
	if len(applyReqs) != 1 {
		t.Fatalf("expected 1 ApplyAtRunEnd request, got %d", len(applyReqs))
	}
	if applyReqs[0].RunOutcome != "failure" {
		t.Errorf("expected runOutcome=failure on wire, got %q", applyReqs[0].RunOutcome)
	}
}

func TestApplyAtRunEnd_FailureSkipsWhenApplyOnFailureFalse(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	cfg := domain.DefaultSandboxConfig()
	off := false
	cfg.ApplyOnFailure = &off
	fx := newFinalizeFixture(t, cfg, stub)

	if ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeFailure,
	}) {
		t.Error("expected apply skipped when ApplyOnFailure=false on failure outcome")
	}
	if stub.ApplyAtRunEndCallCount() != 0 {
		t.Errorf("expected 0 ApplyAtRunEnd calls when opted out, got %d", stub.ApplyAtRunEndCallCount())
	}
	if _, ok := fx.events.FindLogMessage("applyOnFailure=false"); !ok {
		t.Error("expected info event explaining the skip")
	}
}

// (3) Partial acceptance → split.
func TestApplyAtRunEnd_PartialAcceptanceSplit(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{
		Success:        true,
		Applied:        2,
		Remaining:      1,
		IsPartial:      true,
		TotalSizeBytes: 512,
		DiffPath:       "/api/v1/sandboxes/partial/diff",
		CommitHash:     "partial-commit",
	}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Fatal("expected partial apply to count as success")
	}
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("partial apply must still mark run Complete, got %q", fx.run.Status)
	}
	if fx.run.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("partial apply must still mark run Approved (in-acceptance applied), got %q", fx.run.ApprovalState)
	}
	if _, ok := fx.events.FindLogMessage("partial apply"); !ok {
		t.Error("expected info event explaining partial apply / pending-review")
	}
	if fx.run.ChangedFiles != 2 || fx.run.TotalSizeBytes != 512 || fx.run.CommitHash != "partial-commit" {
		t.Fatalf("partial attribution = files:%d bytes:%d commit:%q", fx.run.ChangedFiles, fx.run.TotalSizeBytes, fx.run.CommitHash)
	}
}

// (4) ManualReview=true → deferred.
func TestApplyAtRunEnd_ManualReviewDeferred(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	cfg := domain.DefaultSandboxConfig()
	cfg.ManualReview = true
	fx := newFinalizeFixture(t, cfg, stub)

	if ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Error("manualReview=true must defer apply (return false)")
	}
	if stub.ApplyAtRunEndCallCount() != 0 {
		t.Errorf("expected 0 ApplyAtRunEnd calls under manualReview, got %d", stub.ApplyAtRunEndCallCount())
	}
	if fx.run.Status != domain.RunStatusNeedsReview {
		t.Errorf("expected Status=NeedsReview, got %q", fx.run.Status)
	}
	if fx.run.ApprovalState != domain.ApprovalStatePending {
		t.Errorf("expected ApprovalState=Pending, got %q", fx.run.ApprovalState)
	}
	if _, ok := fx.events.FindLogMessage("manualReview=true"); !ok {
		t.Error("expected info event explaining the deferral")
	}
}

// (5) No-op (empty) → apply still issued.
func TestApplyAtRunEnd_NoOpEmptyProvenance(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{Success: true, Applied: 0}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Fatal("expected apply to succeed for no-op run")
	}
	if stub.ApplyAtRunEndCallCount() != 1 {
		t.Errorf("expected 1 ApplyAtRunEnd call for eager provenance, got %d", stub.ApplyAtRunEndCallCount())
	}
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete on no-op, got %q", fx.run.Status)
	}
	if _, ok := fx.events.FindLogMessage("empty provenance"); !ok {
		t.Error("expected info event acknowledging no-changes apply")
	}
	if fx.run.ChangedFiles != 0 || fx.run.TotalSizeBytes != 0 || fx.run.CommitHash != "" {
		t.Fatalf("empty apply must not invent attribution, got files:%d bytes:%d commit:%q", fx.run.ChangedFiles, fx.run.TotalSizeBytes, fx.run.CommitHash)
	}
}

// Regression of the 2026-04-28 silent-COMPLETE-despite-error bug.
func TestApplyAtRunEnd_FailureWithEmptyProvenanceStaysFailed(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{Success: true, Applied: 0}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)
	fx.run.Status = domain.RunStatusFailed

	got := ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeFailure,
	})
	if got {
		t.Error("ApplyAtRunEnd must return false for empty-provenance failure (no change to apply)")
	}
	if fx.run.Status != domain.RunStatusFailed {
		t.Errorf("Status must remain Failed after empty-provenance failure apply; got %q (regression of 2026-04-28 silent-COMPLETE bug)", fx.run.Status)
	}
	if fx.run.ApprovalState == domain.ApprovalStateApproved {
		t.Errorf("ApprovalState must not be Approved when nothing was applied on a failed run; got %q", fx.run.ApprovalState)
	}
	if _, ok := fx.events.FindLogMessage("empty provenance"); !ok {
		t.Error("expected info event acknowledging the no-changes apply was attempted")
	}
}

func TestApplyAtRunEnd_FailureWithPartialProvenancePreservesRunnerFailure(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{Success: true, Applied: 2}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)
	fx.run.Status = domain.RunStatusFailed

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeFailure,
	}) {
		t.Fatal("expected apply to succeed when failed run produced changes")
	}
	if fx.run.Status != domain.RunStatusFailed {
		t.Errorf("failed run must preserve runner failure status after finalization; got %q", fx.run.Status)
	}
	if fx.run.FinalizationStatus != domain.RunFinalizationStatusSucceeded {
		t.Errorf("finalization status = %q, want succeeded", fx.run.FinalizationStatus)
	}
}

// (6) ConversationID inheritance.
func TestApplyAtRunEnd_ConversationIDForwarded(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	cfg := domain.DefaultSandboxConfig()
	run := &domain.Run{ConversationID: "conv-thread-123"}
	fx := newFinalizeFixtureWithRun(t, cfg, run, stub)

	ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	})
	applyReqs := stub.ApplyAtRunEndRequests()
	if len(applyReqs) != 1 {
		t.Fatalf("expected 1 ApplyAtRunEnd request, got %d", len(applyReqs))
	}
	if applyReqs[0].ConversationID != "conv-thread-123" {
		t.Errorf("expected ConversationID forwarded to ApplyAtRunEnd, got %q", applyReqs[0].ConversationID)
	}
}

func TestApplyAtRunEnd_CheckpointOnUsesTurnCheckpoint(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTurnCompleted}
	run := &domain.Run{ConversationID: "conv-thread-123"}
	fx := newFinalizeFixtureWithRun(t, cfg, run, stub)

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Fatal("expected turn checkpoint to succeed")
	}
	if stub.TurnCheckpointCallCount() != 1 {
		t.Fatalf("expected 1 TurnCheckpoint call, got %d", stub.TurnCheckpointCallCount())
	}
	if stub.ApplyAtRunEndCallCount() != 0 {
		t.Fatalf("expected 0 final ApplyAtRunEnd calls, got %d", stub.ApplyAtRunEndCallCount())
	}
	reqs := stub.TurnCheckpointRequests()
	if reqs[0].ConversationID != "conv-thread-123" {
		t.Errorf("expected ConversationID forwarded to TurnCheckpoint, got %q", reqs[0].ConversationID)
	}
	if reqs[0].RunOutcome != "success" {
		t.Errorf("expected runOutcome=success on checkpoint request, got %q", reqs[0].RunOutcome)
	}
}

func TestApplyAtRunEnd_FailurePreservesSandbox(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	stub.ApplyAtRunEndErr = errors.New("workspace-sandbox unreachable")
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)
	fx.run.Status = domain.RunStatusComplete

	if ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Error("apply on transport error must return false (do not silently mark Approved)")
	}
	if fx.run.ApprovalState == domain.ApprovalStateApproved {
		t.Errorf("ApprovalState must not be Approved when apply errored, got %q", fx.run.ApprovalState)
	}
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("runner status = %q, want complete despite finalization failure", fx.run.Status)
	}
	if fx.run.FinalizationStatus != domain.RunFinalizationStatusFailed {
		t.Errorf("finalization status = %q, want failed", fx.run.FinalizationStatus)
	}
	if fx.run.FinalizationError == "" {
		t.Fatal("expected finalization error to be recorded")
	}
	if _, ok := fx.events.FindLogMessage("apply-at-run-end failed"); !ok {
		t.Error("expected warn event describing the apply failure")
	}
}

func TestApplyAtRunEnd_RetriesCheckpointAfterEnsuringWorkspaceSandbox(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	var calls int32
	stub.TurnCheckpointFunc = func(ctx context.Context, req sandbox.TurnCheckpointRequest) (*sandbox.TurnCheckpointResult, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			return nil, &domain.SandboxError{
				SandboxID:   &req.SandboxID,
				Operation:   "turn_checkpoint",
				Cause:       errors.New("dial tcp 127.0.0.1:15120: connect: connection refused"),
				IsTransient: true,
				CanRetry:    true,
			}
		}
		return &sandbox.TurnCheckpointResult{
			SandboxID: req.SandboxID,
			Status:    sandbox.SandboxStatusCheckpointed,
			Success:   true,
			Applied:   1,
			AppliedAt: time.Now(),
		}, nil
	}
	cfg := fixtures.NewSandboxConfig(nil)
	cfg.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTurnCompleted}
	fx := newFinalizeFixture(t, cfg, stub)
	levers := config.DefaultLevers()
	levers.Sandbox.OperationMaxAttempts = 2
	levers.Sandbox.OperationInitialBackoff = time.Millisecond
	levers.Sandbox.OperationMaxBackoff = time.Millisecond
	ensurer := &fakeWorkspaceSandboxEnsurer{}
	fx.deps.Levers = levers
	fx.deps.WorkspaceSandbox = ensurer

	if !ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Fatal("expected ApplyAtRunEnd checkpoint retry to succeed")
	}
	if stub.TurnCheckpointCallCount() != 2 {
		t.Fatalf("expected two checkpoint attempts, got %d", stub.TurnCheckpointCallCount())
	}
	if ensurer.CallCount() != 1 {
		t.Fatalf("expected one workspace-sandbox ensure call, got %d", ensurer.CallCount())
	}
}

func TestApplyAtRunEnd_AutoApplyFalseSkipsApply(t *testing.T) {
	stub := mocks.NewFakeSandboxProvider()
	cfg := domain.DefaultSandboxConfig()
	off := false
	cfg.AutoApply = &off
	fx := newFinalizeFixture(t, cfg, stub)

	if ApplyAtRunEnd(context.Background(), ApplyAtRunEndInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
		Outcome:   domain.ContractRunOutcomeSuccess,
	}) {
		t.Error("expected apply skipped when AutoApply=false")
	}
	if stub.ApplyAtRunEndCallCount() != 0 {
		t.Errorf("expected 0 calls when AutoApply=false, got %d", stub.ApplyAtRunEndCallCount())
	}
	if _, ok := fx.events.FindLogMessage("autoApply=false"); !ok {
		t.Error("expected info event explaining the skip")
	}
}
