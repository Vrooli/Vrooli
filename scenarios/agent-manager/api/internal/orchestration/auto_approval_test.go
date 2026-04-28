// Tests for the applyAtRunEnd run-executor seam introduced by the
// agent-sandbox-audit-foundation initiative (Phase 3b cutover).
//
// These tests cover the six behaviors specified in
// execute/agent-manager-run-executor-apply-at-run-end-cutover/spec.json:
//
//  1. Success → applies (in-acceptance changes apply at run end)
//  2. Failure → applies (when ApplyOnFailure=true, the contract default)
//  3. Partial acceptance → split (in-acceptance applies; out-of-acceptance
//     remain as state=pending-review on the provenance record)
//  4. ManualReview=true → deferred (sandbox persists; run lands as
//     NeedsReview/Pending)
//  5. No-op empty → eager provenance entry exists; apply is a no-op write
//  6. ConversationID inheritance is forwarded to the workspace-sandbox call
//
// The defensive nil-config gate from the legacy tryAutoApproval path is
// preserved because resolveSandboxConfig() guarantees a non-nil config for
// sandboxed runs and a regression there should still be visible in events.
//
// See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.

package orchestration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
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
}

func (s *stubSandbox) Create(context.Context, sandbox.CreateRequest) (*sandbox.Sandbox, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSandbox) Get(context.Context, uuid.UUID) (*sandbox.Sandbox, error) {
	return nil, nil
}
func (s *stubSandbox) Delete(context.Context, uuid.UUID) error { return nil }
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
func (s *stubSandbox) Stop(context.Context, uuid.UUID) error  { return nil }
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
// Tests
// -----------------------------------------------------------------------------

func newTestExecutorWithRun(t *testing.T, cfg *domain.SandboxConfig, run *domain.Run, stub *stubSandbox) (*RunExecutor, *memEventStore) {
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
	exec := &RunExecutor{
		run:       run,
		sandbox:   stub,
		sandboxID: &sbxID,
		events:    ev,
	}
	return exec, ev
}

func newTestExecutor(t *testing.T, cfg *domain.SandboxConfig, stub *stubSandbox) (*RunExecutor, *memEventStore) {
	t.Helper()
	return newTestExecutorWithRun(t, cfg, nil, stub)
}

// TestApplyAtRunEnd_NilConfigEmitsWarning is the regression gate for the
// pre-2026-04-24 silent-fallthrough bug carried over from tryAutoApproval.
// resolveSandboxConfig() guarantees a non-nil config for sandboxed runs;
// landing here means an orchestrator path bypassed it.
func TestApplyAtRunEnd_NilConfigEmitsWarning(t *testing.T) {
	exec, ev := newTestExecutor(t, nil, &stubSandbox{})
	got := exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess)
	if got {
		t.Errorf("applyAtRunEnd with nil config should return false, got true")
	}
	if _, ok := ev.findMessage("apply-at-run-end skipped: run has no sandbox config"); !ok {
		t.Error("expected warn event describing nil sandbox config, got none")
	}
}

// (1) Success → apply.
func TestRunExecutor_SuccessApplies(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 2}}
	cfg := domain.DefaultSandboxConfig()
	exec, _ := newTestExecutor(t, cfg, stub)

	if !exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess) {
		t.Fatal("expected applyAtRunEnd to succeed")
	}
	if stub.applyHits != 1 {
		t.Errorf("expected exactly 1 ApplyAtRunEnd call, got %d", stub.applyHits)
	}
	if exec.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete, got %q", exec.run.Status)
	}
	if exec.run.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("expected ApprovalState=Approved, got %q", exec.run.ApprovalState)
	}
	if stub.applyReq.RunOutcome != "success" {
		t.Errorf("expected runOutcome=success on wire, got %q", stub.applyReq.RunOutcome)
	}
}

// (2) Failure → apply. ApplyOnFailure=true is the contract default; even a
// failed run must apply its in-acceptance changes so the audit trail is
// preserved.
func TestRunExecutor_FailureApplies(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1}}
	cfg := domain.DefaultSandboxConfig()
	exec, _ := newTestExecutor(t, cfg, stub)

	if !exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeFailure) {
		t.Fatal("expected apply on failure when ApplyOnFailure=true")
	}
	if stub.applyHits != 1 {
		t.Errorf("expected 1 ApplyAtRunEnd call on failure, got %d", stub.applyHits)
	}
	if stub.applyReq.RunOutcome != "failure" {
		t.Errorf("expected runOutcome=failure on wire, got %q", stub.applyReq.RunOutcome)
	}
}

// TestRunExecutor_FailureSkipsWhenApplyOnFailureFalse pins the operator
// opt-out behavior: ApplyOnFailure=false suppresses the apply on non-success.
func TestRunExecutor_FailureSkipsWhenApplyOnFailureFalse(t *testing.T) {
	stub := &stubSandbox{}
	cfg := domain.DefaultSandboxConfig()
	off := false
	cfg.ApplyOnFailure = &off
	exec, ev := newTestExecutor(t, cfg, stub)

	if exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeFailure) {
		t.Error("expected apply skipped when ApplyOnFailure=false on failure outcome")
	}
	if stub.applyHits != 0 {
		t.Errorf("expected 0 ApplyAtRunEnd calls when opted out, got %d", stub.applyHits)
	}
	if _, ok := ev.findMessage("applyOnFailure=false"); !ok {
		t.Error("expected info event explaining the skip")
	}
}

// (3) Partial acceptance → split. In-acceptance applies; remaining files
// stay on the sandbox as state=pending-review for operator follow-up.
func TestRunExecutor_PartialAcceptanceSplit(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{
		Success:   true,
		Applied:   2,
		Remaining: 1,
		IsPartial: true,
	}}
	cfg := domain.DefaultSandboxConfig()
	exec, ev := newTestExecutor(t, cfg, stub)

	if !exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess) {
		t.Fatal("expected partial apply to count as success")
	}
	if exec.run.Status != domain.RunStatusComplete {
		t.Errorf("partial apply must still mark run Complete, got %q", exec.run.Status)
	}
	if exec.run.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("partial apply must still mark run Approved (in-acceptance applied), got %q", exec.run.ApprovalState)
	}
	if _, ok := ev.findMessage("partial apply"); !ok {
		t.Error("expected info event explaining partial apply / pending-review")
	}
}

// (4) ManualReview=true → deferred. Sandbox persists past run end; the run
// terminates as NeedsReview/Pending so the AI Changes review queue surfaces
// it. No ApplyAtRunEnd call is issued.
func TestRunExecutor_ManualReviewDeferred(t *testing.T) {
	stub := &stubSandbox{}
	cfg := domain.DefaultSandboxConfig()
	cfg.ManualReview = true
	exec, ev := newTestExecutor(t, cfg, stub)

	if exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess) {
		t.Error("manualReview=true must defer apply (return false)")
	}
	if stub.applyHits != 0 {
		t.Errorf("expected 0 ApplyAtRunEnd calls under manualReview, got %d", stub.applyHits)
	}
	if exec.run.Status != domain.RunStatusNeedsReview {
		t.Errorf("expected Status=NeedsReview, got %q", exec.run.Status)
	}
	if exec.run.ApprovalState != domain.ApprovalStatePending {
		t.Errorf("expected ApprovalState=Pending, got %q", exec.run.ApprovalState)
	}
	if _, ok := ev.findMessage("manualReview=true"); !ok {
		t.Error("expected info event explaining the deferral")
	}
}

// (5) No-op (empty) → apply still issued. The contract requires eager
// provenance: every run lands a record so the audit trail is complete even
// when there were no changes. workspace-sandbox responds Applied=0 and the
// adapter records that as a no-op.
func TestRunExecutor_NoOpEmptyProvenance(t *testing.T) {
	stub := &stubSandbox{applyResult: &sandbox.ApplyAtRunEndResult{Success: true, Applied: 0}}
	cfg := domain.DefaultSandboxConfig()
	exec, ev := newTestExecutor(t, cfg, stub)

	if !exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess) {
		t.Fatal("expected apply to succeed for no-op run")
	}
	if stub.applyHits != 1 {
		t.Errorf("expected 1 ApplyAtRunEnd call for eager provenance, got %d", stub.applyHits)
	}
	if exec.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete on no-op, got %q", exec.run.Status)
	}
	if _, ok := ev.findMessage("empty provenance"); !ok {
		t.Error("expected info event acknowledging no-changes apply")
	}
}

// (6) ConversationID inheritance — when the run carries a populated
// ConversationID, it must reach the workspace-sandbox apply call so
// provenance is recorded against the correct agent thread.
func TestRunExecutor_ConversationIDForwarded(t *testing.T) {
	stub := &stubSandbox{}
	cfg := domain.DefaultSandboxConfig()
	run := &domain.Run{ConversationID: "conv-thread-123"}
	exec, _ := newTestExecutorWithRun(t, cfg, run, stub)

	exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess)
	if stub.applyReq.ConversationID != "conv-thread-123" {
		t.Errorf("expected ConversationID forwarded to ApplyAtRunEnd, got %q", stub.applyReq.ConversationID)
	}
}

// TestRunExecutor_ApplyFailurePreservesSandbox guards the failure-mode
// behavior: when workspace-sandbox returns an error, the run does NOT
// flip to Approved/Complete via this path — the sandbox is preserved for
// inspection. This is critical because a silent success on transport
// failure would lose audit fidelity.
func TestRunExecutor_ApplyFailurePreservesSandbox(t *testing.T) {
	stub := &stubSandbox{applyErr: errors.New("workspace-sandbox unreachable")}
	cfg := domain.DefaultSandboxConfig()
	exec, ev := newTestExecutor(t, cfg, stub)

	if exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess) {
		t.Error("apply on transport error must return false (do not silently mark Approved)")
	}
	if exec.run.ApprovalState == domain.ApprovalStateApproved {
		t.Errorf("ApprovalState must not be Approved when apply errored, got %q", exec.run.ApprovalState)
	}
	if _, ok := ev.findMessage("apply-at-run-end failed"); !ok {
		t.Error("expected warn event describing the apply failure")
	}
}

// TestRunExecutor_AutoApplyFalseSkipsApply pins the operator opt-out path.
func TestRunExecutor_AutoApplyFalseSkipsApply(t *testing.T) {
	stub := &stubSandbox{}
	cfg := domain.DefaultSandboxConfig()
	off := false
	cfg.AutoApply = &off
	exec, ev := newTestExecutor(t, cfg, stub)

	if exec.applyAtRunEnd(context.Background(), domain.ContractRunOutcomeSuccess) {
		t.Error("expected apply skipped when AutoApply=false")
	}
	if stub.applyHits != 0 {
		t.Errorf("expected 0 calls when AutoApply=false, got %d", stub.applyHits)
	}
	if _, ok := ev.findMessage("autoApply=false"); !ok {
		t.Error("expected info event explaining the skip")
	}
}

// Ensure runner package import stays referenced (used by other helpers in
// the test binary).
var _ = runner.ExecuteResult{}
