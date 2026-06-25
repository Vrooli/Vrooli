package execution

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
)

func TestMapRunStatus_AllKnown(t *testing.T) {
	cases := []struct {
		input    string
		expected Status
	}{
		{"pending", StatusStarting},
		{"starting", StatusStarting},
		{"running", StatusRunning},
		{"needs_review", StatusNeedsReview},
		{"complete", StatusCompleted},
		{"failed", StatusFailed},
		{"cancelled", StatusCanceled},
		{"RUNNING", StatusRunning},
		{" complete ", StatusCompleted},
	}
	for _, tc := range cases {
		tracker := &runTracker{}
		got, _ := mapRunStatus(tc.input, "", tracker, 5)
		if got != tc.expected {
			t.Errorf("mapRunStatus(%q) = %s, want %s", tc.input, got, tc.expected)
		}
		if tracker.ConsecutiveUnknown != 0 {
			t.Errorf("mapRunStatus(%q) left ConsecutiveUnknown=%d, want 0", tc.input, tracker.ConsecutiveUnknown)
		}
	}
}

func TestMapRunStatus_FailedIncludesErrorMsg(t *testing.T) {
	tracker := &runTracker{}
	status, reason := mapRunStatus("failed", "out of memory", tracker, 5)
	if status != StatusFailed {
		t.Fatalf("expected StatusFailed, got %s", status)
	}
	if reason != "out of memory" {
		t.Fatalf("expected reason 'out of memory', got %q", reason)
	}
}

func TestMapRunStatus_FailedDefaultReason(t *testing.T) {
	tracker := &runTracker{}
	_, reason := mapRunStatus("failed", "", tracker, 5)
	if reason != "agent-manager run failed" {
		t.Fatalf("expected default reason, got %q", reason)
	}
}

func TestMapRunStatus_Unknown_GracePeriod(t *testing.T) {
	tracker := &runTracker{}
	threshold := 5

	for i := 1; i < threshold; i++ {
		status, _ := mapRunStatus("bogus_status", "", tracker, threshold)
		if status != StatusRunning {
			t.Fatalf("poll %d: expected StatusRunning during grace period, got %s", i, status)
		}
		if tracker.ConsecutiveUnknown != i {
			t.Fatalf("poll %d: expected ConsecutiveUnknown=%d, got %d", i, i, tracker.ConsecutiveUnknown)
		}
	}

	status, reason := mapRunStatus("bogus_status", "", tracker, threshold)
	if status != StatusFailed {
		t.Fatalf("expected StatusFailed after %d unknowns, got %s", threshold, status)
	}
	if reason == "" {
		t.Fatal("expected non-empty failure reason for unknown status")
	}
}

func TestMapRunStatus_Unknown_ResetOnKnown(t *testing.T) {
	tracker := &runTracker{ConsecutiveUnknown: 3}

	mapRunStatus("running", "", tracker, 5)
	if tracker.ConsecutiveUnknown != 0 {
		t.Fatalf("expected ConsecutiveUnknown reset to 0, got %d", tracker.ConsecutiveUnknown)
	}
}

type mockInspector struct {
	states map[string]agentmanager.RunState
	err    error
}

func (m *mockInspector) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	if m.err != nil {
		return agentmanager.RunState{}, m.err
	}
	if s, ok := m.states[runID]; ok {
		return s, nil
	}
	return agentmanager.RunState{}, m.err
}

func newTestPollingService(t *testing.T, inspector RunInspector, opts ...func(*ServiceConfig)) *Service {
	t.Helper()
	root := t.TempDir()
	cfg := ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: &stubAgentService{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	svc := NewService(cfg)
	svc.inspector = inspector
	return svc
}

func TestNewService_DifferWired(t *testing.T) {
	root := t.TempDir()

	agent := &fullAgentService{}
	svc := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
	})
	if svc.differ == nil {
		t.Fatal("expected differ to be non-nil when AgentService implements RunDiffer")
	}
	if svc.inspector == nil {
		t.Fatal("expected inspector to be non-nil when AgentService implements RunInspector")
	}
}

type fullAgentService struct {
	stubAgentService
}

func (f *fullAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{}, nil
}

func (f *fullAgentService) GetRunDiff(_ context.Context, _ string) (agentmanager.RunDiff, error) {
	return agentmanager.RunDiff{}, nil
}

func (f *fullAgentService) StopRun(_ context.Context, _ string) error {
	return nil
}

func (f *fullAgentService) ContinueRun(_ context.Context, _ string, _ string) error {
	return nil
}

func TestPolling_ConsecutiveErrors(t *testing.T) {
	errInspector := &mockInspector{err: agentmanager.ErrNotAvailable}
	maxErrors := 5
	svc := newTestPollingService(t, errInspector, func(cfg *ServiceConfig) {
		cfg.MaxConsecutiveErrors = maxErrors
	})

	records := []Record{{
		ExecutionID: "exec-1",
		BacklogKind: "idea",
		BacklogName: "test",
		RunID:       "run-1",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}}
	if err := svc.store.Save(records); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	for i := 0; i < maxErrors-1; i++ {
		svc.mu.Lock()
		_, _, _ = svc.refreshRunningLocked(ctx)
		svc.mu.Unlock()
	}

	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusRunning {
		t.Fatalf("expected StatusRunning before threshold, got %s", loaded[0].Status)
	}

	svc.mu.Lock()
	_, _, _ = svc.refreshRunningLocked(ctx)
	svc.mu.Unlock()

	loaded, _ = svc.store.Load()
	if loaded[0].Status != StatusFailed {
		t.Fatalf("expected StatusFailed after %d errors, got %s", maxErrors, loaded[0].Status)
	}
	if loaded[0].FailureReason == "" {
		t.Fatal("expected non-empty failure reason")
	}
}

func TestPolling_ErrorReset(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-1": {RunID: "run-1", Status: "running"},
		},
	}
	svc := newTestPollingService(t, inspector, func(cfg *ServiceConfig) {
		cfg.MaxConsecutiveErrors = 5
	})

	records := []Record{{
		ExecutionID: "exec-1",
		BacklogKind: "idea",
		BacklogName: "test",
		RunID:       "run-1",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}}
	if err := svc.store.Save(records); err != nil {
		t.Fatal(err)
	}

	tracker := svc.ensureRunTracker("run-1")
	tracker.ConsecutiveErrors = 4

	ctx := context.Background()
	svc.mu.Lock()
	_, _, _ = svc.refreshRunningLocked(ctx)
	svc.mu.Unlock()

	if tracker.ConsecutiveErrors != 0 {
		t.Fatalf("expected errors reset to 0, got %d", tracker.ConsecutiveErrors)
	}

	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusRunning {
		t.Fatalf("expected StatusRunning after reset, got %s", loaded[0].Status)
	}
}

func TestPolling_MaxAge(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-1": {RunID: "run-1", Status: "running"},
		},
	}
	svc := newTestPollingService(t, inspector, func(cfg *ServiceConfig) {
		cfg.MaxRunAge = 1 * time.Millisecond
	})

	records := []Record{{
		ExecutionID: "exec-1",
		BacklogKind: "idea",
		BacklogName: "test",
		RunID:       "run-1",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}}
	if err := svc.store.Save(records); err != nil {
		t.Fatal(err)
	}

	tracker := svc.ensureRunTracker("run-1")
	tracker.FirstSeen = time.Now().Add(-1 * time.Hour)

	ctx := context.Background()
	svc.mu.Lock()
	_, _, _ = svc.refreshRunningLocked(ctx)
	svc.mu.Unlock()

	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusFailed {
		t.Fatalf("expected StatusFailed after max-age, got %s", loaded[0].Status)
	}
	if loaded[0].FailureReason == "" {
		t.Fatal("expected non-empty failure reason for max-age timeout")
	}
}
