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
	"strings"
	"sync"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// In-memory event store for assertion
// -----------------------------------------------------------------------------

type capturedEvent struct {
	level   string
	message string
}

type memEventStore struct {
	mu     sync.Mutex
	events []capturedEvent
}

func (m *memEventStore) Append(_ context.Context, _ uuid.UUID, evts ...*domain.RunEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range evts {
		if e == nil {
			continue
		}
		log, ok := e.Data.(*domain.LogEventData)
		if !ok || log == nil {
			continue
		}
		m.events = append(m.events, capturedEvent{level: log.Level, message: log.Message})
	}
	return nil
}

func (m *memEventStore) Get(context.Context, uuid.UUID, event.GetOptions) ([]*domain.RunEvent, error) {
	return nil, nil
}

func (m *memEventStore) Stream(context.Context, uuid.UUID, event.StreamOptions) (<-chan *domain.RunEvent, error) {
	ch := make(chan *domain.RunEvent)
	close(ch)
	return ch, nil
}
func (m *memEventStore) Count(context.Context, uuid.UUID) (int64, error) { return 0, nil }
func (m *memEventStore) Delete(context.Context, uuid.UUID) error         { return nil }

func (m *memEventStore) findMessage(substr string) (capturedEvent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if strings.Contains(e.message, substr) {
			return e, true
		}
	}
	return capturedEvent{}, false
}

// -----------------------------------------------------------------------------
// Minimal sandbox.Provider stub
// -----------------------------------------------------------------------------

type stubSandbox struct {
	applyReq    *sandbox.ApplyAtRunEndRequest
	applyResult *sandbox.ApplyAtRunEndResult
	applyErr    error
	applyHits   int

	deleteHits   int
	deleteErr    error
	deleteCtxErr error
	stopHits     int
	stopErr      error
}

func (s *stubSandbox) Create(context.Context, sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSandbox) Get(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
	return nil, nil
}

func (s *stubSandbox) Delete(ctx context.Context, _ uuid.UUID) error {
	s.deleteHits++
	s.deleteCtxErr = ctx.Err()
	return s.deleteErr
}

func (s *stubSandbox) GetWorkspacePath(context.Context, uuid.UUID) (string, error) {
	return "", nil
}
func (s *stubSandbox) IsAvailable(context.Context) (bool, string) { return true, "" }
func (s *stubSandbox) GetDiff(context.Context, uuid.UUID) (*sandbox.DiffResult, error) {
	return &sandbox.DiffResult{}, nil
}

func (s *stubSandbox) Approve(context.Context, sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
	return &sandbox.ApproveResult{Success: true}, nil
}
func (s *stubSandbox) Reject(context.Context, uuid.UUID, string) error { return nil }
func (s *stubSandbox) PartialApprove(context.Context, sandbox.PartialApproveRequest) (*sandbox.ApproveResult, error) {
	return &sandbox.ApproveResult{Success: true}, nil
}

func (s *stubSandbox) Stop(_ context.Context, _ uuid.UUID) error {
	s.stopHits++
	return s.stopErr
}
func (s *stubSandbox) Start(context.Context, uuid.UUID) error { return nil }
func (s *stubSandbox) ValidatePath(context.Context, string, string) (*sandbox.PathValidationResult, error) {
	return &sandbox.PathValidationResult{Valid: true}, nil
}

func (s *stubSandbox) ExecProcess(context.Context, sandbox.ExecProcessRequest) (*sandbox.ExecProcessResult, error) {
	return &sandbox.ExecProcessResult{ExitCode: 0}, nil
}

func (s *stubSandbox) ApplyAtRunEnd(_ context.Context, req sandbox.ApplyAtRunEndRequest) (*sandbox.ApplyAtRunEndResult, error) {
	s.applyHits++
	reqCopy := req
	s.applyReq = &reqCopy
	if s.applyErr != nil {
		return nil, s.applyErr
	}
	if s.applyResult != nil {
		return s.applyResult, nil
	}
	return &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1}, nil
}

// -----------------------------------------------------------------------------
// Test helpers
// -----------------------------------------------------------------------------

// finalizeFixture bundles the run, sandbox, and event store wired into a
// shape every test in this file consumes.
type finalizeFixture struct {
	run       *domain.Run
	sandbox   *stubSandbox
	sandboxID uuid.UUID
	events    *memEventStore
	deps      Deps
}

func newFinalizeFixture(t *testing.T, cfg *domain.SandboxConfig, stub *stubSandbox) *finalizeFixture {
	t.Helper()
	return newFinalizeFixtureWithRun(t, cfg, nil, stub)
}

func newFinalizeFixtureWithRun(t *testing.T, cfg *domain.SandboxConfig, run *domain.Run, stub *stubSandbox) *finalizeFixture {
	t.Helper()
	if run == nil {
		run = &domain.Run{}
	}
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	run.RunMode = domain.RunModeSandboxed
	run.SandboxConfig = cfg
	sbxID := uuid.New()
	ev := &memEventStore{}
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
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	return cfg
}

// -----------------------------------------------------------------------------
// ApplySandboxLifecycle / Finalize regression gates
// -----------------------------------------------------------------------------

// TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx is the
// regression gate for the 2026-04-28 incident.
func TestApplySandboxLifecycle_DeletesEvenWithCancelledCallerCtx(t *testing.T) {
	stub := &stubSandbox{}
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

	if stub.deleteHits != 1 {
		t.Fatalf("expected 1 Delete call despite cancelled caller ctx, got %d (regression: 2026-04-28 mount leak)", stub.deleteHits)
	}
	if stub.deleteCtxErr != nil {
		t.Errorf("Delete was called with a context that already had ctx.Err()=%v — teardown ctx is not detached", stub.deleteCtxErr)
	}
}

func TestFinalize_AdvancesPhaseToCompleted(t *testing.T) {
	stub := &stubSandbox{}
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if fx.run.Phase != domain.RunPhaseCompleted {
		t.Errorf("expected run.Phase=%s after finalize, got %s", domain.RunPhaseCompleted, fx.run.Phase)
	}
}

func TestFinalize_DeletesSandboxOnSuccess(t *testing.T) {
	stub := &stubSandbox{}
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.deleteHits != 1 {
		t.Errorf("expected 1 Delete on successful run, got %d", stub.deleteHits)
	}
	if stub.stopHits != 0 {
		t.Errorf("expected 0 Stop calls when DeleteOn matches, got %d", stub.stopHits)
	}
}

func TestFinalize_DeletesSandboxOnFailure(t *testing.T) {
	stub := &stubSandbox{}
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusFailed

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.deleteHits != 1 {
		t.Errorf("expected 1 Delete on failed run (DeleteOn=[terminal] matches RunFailed), got %d", stub.deleteHits)
	}
}

func TestFinalize_DeleteFailureDoesNotBlockPhaseAdvance(t *testing.T) {
	stub := &stubSandbox{deleteErr: errors.New("workspace-sandbox unreachable")}
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if fx.run.Phase != domain.RunPhaseCompleted {
		t.Errorf("phase must advance to Completed even when Delete errors, got %s", fx.run.Phase)
	}
	if _, ok := fx.events.findMessage("failed to delete sandbox"); !ok {
		t.Error("expected warn event recording the Delete error")
	}
}

func TestFinalize_NoOpForInPlaceRun(t *testing.T) {
	stub := &stubSandbox{}
	fx := newFinalizeFixture(t, sandboxedRunCfg(), stub)
	fx.run.RunMode = domain.RunModeInPlace
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.deleteHits != 0 || stub.stopHits != 0 {
		t.Errorf("in-place run should not touch sandbox: deleteHits=%d stopHits=%d", stub.deleteHits, stub.stopHits)
	}
	if fx.run.Phase != domain.RunPhaseCompleted {
		t.Errorf("phase ladder must still advance for in-place runs, got %s", fx.run.Phase)
	}
}

func TestFinalize_StopsSandboxWhenLifecycleSaysStop(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.StopOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	cfg.Lifecycle.DeleteOn = nil

	stub := &stubSandbox{}
	fx := newFinalizeFixture(t, cfg, stub)
	fx.run.Status = domain.RunStatusComplete

	Finalize(FinalizeInput{
		Deps:      fx.deps,
		Run:       fx.run,
		SandboxID: &fx.sandboxID,
		Sandbox:   fx.sandbox,
	})

	if stub.stopHits != 1 {
		t.Errorf("expected 1 Stop call when StopOn=[terminal] and DeleteOn empty, got %d", stub.stopHits)
	}
	if stub.deleteHits != 0 {
		t.Errorf("expected 0 Delete calls under StopOn lifecycle, got %d", stub.deleteHits)
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
	fx := newFinalizeFixture(t, nil, &stubSandbox{})
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
	if _, ok := fx.events.findMessage("apply-at-run-end skipped: run has no sandbox config"); !ok {
		t.Error("expected warn event describing nil sandbox config, got none")
	}
}

// (1) Success → apply.
func TestApplyAtRunEnd_SuccessApplies(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 2}}
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
	if stub.applyHits != 1 {
		t.Errorf("expected exactly 1 ApplyAtRunEnd call, got %d", stub.applyHits)
	}
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete, got %q", fx.run.Status)
	}
	if fx.run.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("expected ApprovalState=Approved, got %q", fx.run.ApprovalState)
	}
	if stub.applyReq.RunOutcome != "success" {
		t.Errorf("expected runOutcome=success on wire, got %q", stub.applyReq.RunOutcome)
	}
}

// (2) Failure → apply (ApplyOnFailure=true is the contract default).
func TestApplyAtRunEnd_FailureApplies(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1}}
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
	if stub.applyHits != 1 {
		t.Errorf("expected 1 ApplyAtRunEnd call on failure, got %d", stub.applyHits)
	}
	if stub.applyReq.RunOutcome != "failure" {
		t.Errorf("expected runOutcome=failure on wire, got %q", stub.applyReq.RunOutcome)
	}
}

func TestApplyAtRunEnd_FailureSkipsWhenApplyOnFailureFalse(t *testing.T) {
	stub := &stubSandbox{}
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
	if stub.applyHits != 0 {
		t.Errorf("expected 0 ApplyAtRunEnd calls when opted out, got %d", stub.applyHits)
	}
	if _, ok := fx.events.findMessage("applyOnFailure=false"); !ok {
		t.Error("expected info event explaining the skip")
	}
}

// (3) Partial acceptance → split.
func TestApplyAtRunEnd_PartialAcceptanceSplit(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{
		Success:   true,
		Applied:   2,
		Remaining: 1,
		IsPartial: true,
	}}
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
	if _, ok := fx.events.findMessage("partial apply"); !ok {
		t.Error("expected info event explaining partial apply / pending-review")
	}
}

// (4) ManualReview=true → deferred.
func TestApplyAtRunEnd_ManualReviewDeferred(t *testing.T) {
	stub := &stubSandbox{}
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
	if stub.applyHits != 0 {
		t.Errorf("expected 0 ApplyAtRunEnd calls under manualReview, got %d", stub.applyHits)
	}
	if fx.run.Status != domain.RunStatusNeedsReview {
		t.Errorf("expected Status=NeedsReview, got %q", fx.run.Status)
	}
	if fx.run.ApprovalState != domain.ApprovalStatePending {
		t.Errorf("expected ApprovalState=Pending, got %q", fx.run.ApprovalState)
	}
	if _, ok := fx.events.findMessage("manualReview=true"); !ok {
		t.Error("expected info event explaining the deferral")
	}
}

// (5) No-op (empty) → apply still issued.
func TestApplyAtRunEnd_NoOpEmptyProvenance(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 0}}
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
	if stub.applyHits != 1 {
		t.Errorf("expected 1 ApplyAtRunEnd call for eager provenance, got %d", stub.applyHits)
	}
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete on no-op, got %q", fx.run.Status)
	}
	if _, ok := fx.events.findMessage("empty provenance"); !ok {
		t.Error("expected info event acknowledging no-changes apply")
	}
}

// Regression of the 2026-04-28 silent-COMPLETE-despite-error bug.
func TestApplyAtRunEnd_FailureWithEmptyProvenanceStaysFailed(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 0}}
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
	if _, ok := fx.events.findMessage("empty provenance"); !ok {
		t.Error("expected info event acknowledging the no-changes apply was attempted")
	}
}

func TestApplyAtRunEnd_FailureWithPartialProvenanceMarksComplete(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 2}}
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
	if fx.run.Status != domain.RunStatusComplete {
		t.Errorf("failed run with applied changes must flip to Complete (audit-of-change contract); got %q", fx.run.Status)
	}
}

// (6) ConversationID inheritance.
func TestApplyAtRunEnd_ConversationIDForwarded(t *testing.T) {
	stub := &stubSandbox{}
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
	if stub.applyReq.ConversationID != "conv-thread-123" {
		t.Errorf("expected ConversationID forwarded to ApplyAtRunEnd, got %q", stub.applyReq.ConversationID)
	}
}

func TestApplyAtRunEnd_FailurePreservesSandbox(t *testing.T) {
	stub := &stubSandbox{applyErr: errors.New("workspace-sandbox unreachable")}
	cfg := domain.DefaultSandboxConfig()
	fx := newFinalizeFixture(t, cfg, stub)

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
	if _, ok := fx.events.findMessage("apply-at-run-end failed"); !ok {
		t.Error("expected warn event describing the apply failure")
	}
}

func TestApplyAtRunEnd_AutoApplyFalseSkipsApply(t *testing.T) {
	stub := &stubSandbox{}
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
	if stub.applyHits != 0 {
		t.Errorf("expected 0 calls when AutoApply=false, got %d", stub.applyHits)
	}
	if _, ok := fx.events.findMessage("autoApply=false"); !ok {
		t.Error("expected info event explaining the skip")
	}
}
