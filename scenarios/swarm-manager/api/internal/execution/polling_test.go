package execution

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
)

func newTestPollingService(t *testing.T, opts ...func(*ServiceConfig)) *Service {
	t.Helper()
	root := t.TempDir()
	cfg := ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: &stubAgentService{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return NewService(cfg)
}

func TestNewService_DifferWired(t *testing.T) {
	root := t.TempDir()

	agent := &fullAgentService{}
	svc := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
	})
	if svc.differ == nil {
		t.Fatal("expected differ to be non-nil when AgentService implements RunDiffer")
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

// TestPolling_FailsClosedOnUncorrelatedActiveRecord pins the post-migration
// invariant: every active record is runner-owned (OpExecutionID set). The
// legacy agent-manager poll driver is gone, so an active record WITHOUT an
// operation correlation is an impossible pre-cutover leftover — the sweep
// fails it closed (record failed, backlog item parked in in_review for the
// operator) instead of silently stranding it.
func TestPolling_FailsClosedOnUncorrelatedActiveRecord(t *testing.T) {
	svc := newTestPollingService(t)
	specPath := seedBacklogSpec(t, svc, "execute", "uncorrelated", "in_progress")

	rec := Record{
		ExecutionID: "exec-uncorrelated",
		BacklogKind: "execute",
		BacklogName: "uncorrelated",
		RunID:       "run-uncorrelated",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusFailed {
		t.Fatalf("uncorrelated active record status = %s, want StatusFailed (fail closed)", loaded[0].Status)
	}
	if !strings.Contains(loaded[0].FailureReason, "no operation correlation") {
		t.Fatalf("failure reason = %q, want no-operation-correlation explanation", loaded[0].FailureReason)
	}
	if got := loadSpecStatus(t, specPath); got != backlogStatusInReview {
		t.Fatalf("backlog status = %q, want %q (operator decides terminal)", got, backlogStatusInReview)
	}
}
