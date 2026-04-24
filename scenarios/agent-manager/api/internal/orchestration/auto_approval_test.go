package orchestration

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

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
	diffResult  *sandbox.DiffResult
	diffErr     error
	approveErr  error
	approveHits int
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
func (s *stubSandbox) GetDiff(_ context.Context, _ uuid.UUID) (*sandbox.DiffResult, error) {
	if s.diffErr != nil {
		return nil, s.diffErr
	}
	if s.diffResult != nil {
		return s.diffResult, nil
	}
	return &sandbox.DiffResult{}, nil
}

func (s *stubSandbox) Approve(_ context.Context, _ sandbox.ApproveRequest) (*sandbox.ApproveResult, error) {
	s.approveHits++
	if s.approveErr != nil {
		return nil, s.approveErr
	}
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

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// newTestExecutor wires the minimum fields needed for tryAutoApproval tests.
func newTestExecutor(t *testing.T, cfg *domain.SandboxConfig, stub *stubSandbox) (*RunExecutor, *memEventStore) {
	t.Helper()
	runID := uuid.New()
	sbxID := uuid.New()
	run := &domain.Run{
		ID:            runID,
		RunMode:       domain.RunModeSandboxed,
		SandboxConfig: cfg,
	}
	ev := &memEventStore{}
	exec := &RunExecutor{
		run:       run,
		sandbox:   stub,
		sandboxID: &sbxID,
		events:    ev,
	}
	return exec, ev
}

// TestTryAutoApproval_NilConfigEmitsWarning is the regression gate for the
// original bug: before 2026-04-24 a nil SandboxConfig caused a silent false
// return, leaving runs stuck in NEEDS_REVIEW with no explanation in the event
// stream. The fix emits a warn-level event so the next occurrence is visible.
func TestTryAutoApproval_NilConfigEmitsWarning(t *testing.T) {
	exec, ev := newTestExecutor(t, nil, &stubSandbox{})

	got := exec.tryAutoApproval(context.Background())
	if got {
		t.Errorf("tryAutoApproval with nil config should return false, got true")
	}
	if _, ok := ev.findMessage("auto-approval skipped: run has no sandbox config"); !ok {
		t.Error("expected warn event describing nil sandbox config, got none")
	}
}

// TestAutoApproveIfEmpty_EmptySandboxSucceeds verifies the happy path: an
// empty sandbox is auto-approved and the run lands in Complete/Approved.
func TestAutoApproveIfEmpty_EmptySandboxSucceeds(t *testing.T) {
	stub := &stubSandbox{
		diffResult: &sandbox.DiffResult{
			Stats: sandbox.DiffStats{FilesChanged: 0},
		},
	}
	cfg := &domain.SandboxConfig{Acceptance: domain.SandboxAcceptanceConfig{}}
	exec, ev := newTestExecutor(t, cfg, stub)

	ok := exec.autoApproveIfEmpty(context.Background())
	if !ok {
		t.Fatal("autoApproveIfEmpty should succeed for an empty sandbox")
	}
	if stub.approveHits != 1 {
		t.Errorf("expected exactly 1 Approve call, got %d", stub.approveHits)
	}
	if exec.run.Status != domain.RunStatusComplete {
		t.Errorf("expected Status=Complete, got %q", exec.run.Status)
	}
	if exec.run.ApprovalState != domain.ApprovalStateApproved {
		t.Errorf("expected ApprovalState=Approved, got %q", exec.run.ApprovalState)
	}
	if exec.run.ApprovedBy != "auto-approve-empty" {
		t.Errorf("expected ApprovedBy=auto-approve-empty, got %q", exec.run.ApprovedBy)
	}
	if _, ok := ev.findMessage("auto-approved empty sandbox"); !ok {
		t.Error("expected info event confirming auto-approval")
	}
}

// TestAutoApproveIfEmpty_NonEmptyEmitsReviewReason is the second event-emission
// gate: when a sandbox has real changes, the run legitimately needs review,
// but the reason should appear in the event stream rather than silently
// disappearing.
func TestAutoApproveIfEmpty_NonEmptyEmitsReviewReason(t *testing.T) {
	stub := &stubSandbox{
		diffResult: &sandbox.DiffResult{
			Stats: sandbox.DiffStats{FilesChanged: 3},
		},
	}
	cfg := &domain.SandboxConfig{Acceptance: domain.SandboxAcceptanceConfig{}}
	exec, ev := newTestExecutor(t, cfg, stub)

	ok := exec.autoApproveIfEmpty(context.Background())
	if ok {
		t.Error("autoApproveIfEmpty should return false when there are changes")
	}
	if stub.approveHits != 0 {
		t.Errorf("expected 0 Approve calls for non-empty sandbox, got %d", stub.approveHits)
	}
	got, ok := ev.findMessage("3 files changed")
	if !ok {
		t.Error("expected info event mentioning file count, got none")
	}
	if ok && got.level != "info" {
		t.Errorf("expected info level event, got %q", got.level)
	}
}

// TestAutoApproveIfEmpty_GetDiffErrorEmitsWarn verifies that a failure to
// fetch the diff emits a warn event (not silent).
func TestAutoApproveIfEmpty_GetDiffErrorEmitsWarn(t *testing.T) {
	stub := &stubSandbox{diffErr: errors.New("diff service down")}
	cfg := &domain.SandboxConfig{Acceptance: domain.SandboxAcceptanceConfig{}}
	exec, ev := newTestExecutor(t, cfg, stub)

	ok := exec.autoApproveIfEmpty(context.Background())
	if ok {
		t.Error("autoApproveIfEmpty should return false when GetDiff fails")
	}
	if _, ok := ev.findMessage("failed to get diff"); !ok {
		t.Error("expected warn event when GetDiff errors")
	}
}
