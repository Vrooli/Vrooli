package execution

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/promptmanager"
)

// stubContinuer implements the runContinuer interface for testing follow-up continuation.
type stubContinuer struct {
	calls   int
	lastRun string
	lastMsg string
	err     error
}

func (s *stubContinuer) ContinueRun(_ context.Context, runID string, message string) error {
	s.calls++
	s.lastRun = runID
	s.lastMsg = message
	return s.err
}

// followUpTestService creates a Service with seeded records, returning the service and store path.
func followUpTestService(t *testing.T, root string, records []Record, agent agentSpawner) *Service {
	t.Helper()
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")
	store := NewStore(storePath)
	if err := store.Save(records); err != nil {
		t.Fatalf("seed records: %v", err)
	}
	svc := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    storePath,
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	return svc
}

func TestFollowUp_NewRunFromCompleted(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "followup-idea", map[string]any{
		"name":        "followup-idea",
		"title":       "Follow-up Idea",
		"description": "desc",
		"status":      "completed",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "followup-idea")

	agent := &stubAgentService{}
	parentRecord := Record{
		ExecutionID:    "parent-exec-1",
		BacklogKind:    "idea",
		BacklogName:    "followup-idea",
		PreviousStatus: "backlog",
		Status:         StatusCompleted,
		Mode:           ModeYOLO,
		RunID:          "run-parent-1",
		TaskID:         "task-parent-1",
		CreatedAt:      "2026-03-24T00:00:00Z",
		UpdatedAt:      "2026-03-24T01:00:00Z",
	}

	svc := followUpTestService(t, root, []Record{parentRecord}, agent)

	record, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "parent-exec-1",
		FollowUpType: "followup",
		RunMode:      "new",
	})
	if err != nil {
		t.Fatalf("FollowUp error: %v", err)
	}
	if record.Status != StatusStarting {
		t.Fatalf("expected starting status, got %s", record.Status)
	}
	if record.ParentExecutionID != "parent-exec-1" {
		t.Fatalf("expected parent_execution_id parent-exec-1, got %s", record.ParentExecutionID)
	}
	if record.BacklogKind != "idea" {
		t.Fatalf("expected backlog_kind idea, got %s", record.BacklogKind)
	}
	if record.BacklogName != "followup-idea" {
		t.Fatalf("expected backlog_name followup-idea, got %s", record.BacklogName)
	}
	if record.Operation != "followup" {
		t.Fatalf("expected operation followup, got %s", record.Operation)
	}
	if record.StartedBy != "swarm-manager:follow-up" {
		t.Fatalf("expected started_by swarm-manager:follow-up, got %s", record.StartedBy)
	}
	if record.FixupAttempt != 0 {
		t.Fatalf("expected fixup_attempt 0 for followup type, got %d", record.FixupAttempt)
	}
	if agent.spawnCalls != 1 {
		t.Fatalf("expected 1 spawn call, got %d", agent.spawnCalls)
	}
	if record.TaskID == "" || record.RunID == "" {
		t.Fatalf("expected task/run IDs set, got task=%s run=%s", record.TaskID, record.RunID)
	}
}

func TestFollowUp_FixupFromNeedsFixup(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "fix", "fixup-item", map[string]any{
		"name":        "fixup-item",
		"title":       "Fixup Item",
		"description": "desc",
		"status":      "needs_fixup",
		"priority":    2,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "fix", "fixup-item")

	agent := &stubAgentService{}
	parentRecord := Record{
		ExecutionID:    "parent-fixup-1",
		BacklogKind:    "fix",
		BacklogName:    "fixup-item",
		PreviousStatus: "backlog",
		Status:         StatusNeedsFixup,
		Mode:           ModeYOLO,
		RunID:          "run-fixup-1",
		TaskID:         "task-fixup-1",
		FixupAttempt:   1,
		ReviewResult: &ReviewResult{
			JobID:          "review-1",
			Classification: "needs_work",
			Summary:        "Tests failing",
			Dimensions: []ReviewDimension{
				{Name: "tests", Status: "red", Details: "2 tests failing"},
			},
		},
		CreatedAt: "2026-03-24T00:00:00Z",
		UpdatedAt: "2026-03-24T01:00:00Z",
	}

	svc := followUpTestService(t, root, []Record{parentRecord}, agent)

	record, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "parent-fixup-1",
		FollowUpType: "fixup",
		RunMode:      "new",
	})
	if err != nil {
		t.Fatalf("FollowUp error: %v", err)
	}
	if record.FixupAttempt != 2 {
		t.Fatalf("expected fixup_attempt 2 (parent was 1), got %d", record.FixupAttempt)
	}
	if record.ParentExecutionID != "parent-fixup-1" {
		t.Fatalf("expected parent_execution_id parent-fixup-1, got %s", record.ParentExecutionID)
	}
	if record.Operation != "fixup" {
		t.Fatalf("expected operation fixup, got %s", record.Operation)
	}
	if record.Status != StatusStarting {
		t.Fatalf("expected starting status, got %s", record.Status)
	}
	if agent.spawnCalls != 1 {
		t.Fatalf("expected 1 spawn call, got %d", agent.spawnCalls)
	}
}

func TestFollowUp_ContinueRun(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "continue-idea", map[string]any{
		"name":        "continue-idea",
		"title":       "Continue Idea",
		"description": "desc",
		"status":      "completed",
		"priority":    3,
		"tags":        []string{},
	})

	agent := &stubAgentService{}
	continuer := &stubContinuer{}
	parentRecord := Record{
		ExecutionID:    "parent-continue-1",
		BacklogKind:    "idea",
		BacklogName:    "continue-idea",
		PreviousStatus: "backlog",
		Status:         StatusCompleted,
		Mode:           ModeYOLO,
		RunID:          "run-continue-1",
		TaskID:         "task-continue-1",
		CreatedAt:      "2026-03-24T00:00:00Z",
		UpdatedAt:      "2026-03-24T01:00:00Z",
	}

	svc := followUpTestService(t, root, []Record{parentRecord}, agent)
	svc.continuer = continuer

	record, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "parent-continue-1",
		FollowUpType: "followup",
		Context:      "Please fix the remaining lint errors",
		RunMode:      "continue",
	})
	if err != nil {
		t.Fatalf("FollowUp error: %v", err)
	}
	if record.Status != StatusRunning {
		t.Fatalf("expected running status for continue, got %s", record.Status)
	}
	if record.RunID != "run-continue-1" {
		t.Fatalf("expected inherited run ID run-continue-1, got %s", record.RunID)
	}
	if record.TaskID != "task-continue-1" {
		t.Fatalf("expected inherited task ID task-continue-1, got %s", record.TaskID)
	}
	if continuer.calls != 1 {
		t.Fatalf("expected 1 ContinueRun call, got %d", continuer.calls)
	}
	if continuer.lastRun != "run-continue-1" {
		t.Fatalf("expected ContinueRun called with run-continue-1, got %s", continuer.lastRun)
	}
	// Agent spawn should NOT be called for continue mode.
	if agent.spawnCalls != 0 {
		t.Fatalf("expected 0 spawn calls for continue, got %d", agent.spawnCalls)
	}
}

func TestFollowUp_RejectsRunningState(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "running-idea", map[string]any{
		"name":        "running-idea",
		"title":       "Running Idea",
		"description": "desc",
		"status":      "in_progress",
		"priority":    3,
		"tags":        []string{},
	})

	agent := &stubAgentService{}
	parentRecord := Record{
		ExecutionID:    "parent-running-1",
		BacklogKind:    "idea",
		BacklogName:    "running-idea",
		PreviousStatus: "backlog",
		Status:         StatusRunning,
		Mode:           ModeYOLO,
		RunID:          "run-running-1",
		CreatedAt:      "2026-03-24T00:00:00Z",
		UpdatedAt:      "2026-03-24T01:00:00Z",
	}

	svc := followUpTestService(t, root, []Record{parentRecord}, agent)

	_, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "parent-running-1",
		FollowUpType: "followup",
		RunMode:      "new",
	})
	if err == nil {
		t.Fatal("expected error for running state")
	}
	if !errors.Is(err, nil) {
		// Verify the error message mentions the state restriction.
		expected := `cannot follow up execution in "running" state`
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err.Error())
		}
	}
}

func TestFollowUp_NotFound(t *testing.T) {
	root := t.TempDir()
	agent := &stubAgentService{}

	svc := followUpTestService(t, root, []Record{}, agent)

	_, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "nonexistent-id",
		FollowUpType: "followup",
		RunMode:      "new",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent execution")
	}
	if !errors.Is(err, errNotFound) {
		t.Fatalf("expected errNotFound, got %v", err)
	}
}

func TestFollowUp_SessionExpired(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "expired-idea", map[string]any{
		"name":        "expired-idea",
		"title":       "Expired Idea",
		"description": "desc",
		"status":      "completed",
		"priority":    3,
		"tags":        []string{},
	})

	agent := &stubAgentService{}
	continuer := &stubContinuer{err: errors.New("session_expired: session has timed out")}
	parentRecord := Record{
		ExecutionID:    "parent-expired-1",
		BacklogKind:    "idea",
		BacklogName:    "expired-idea",
		PreviousStatus: "backlog",
		Status:         StatusCompleted,
		Mode:           ModeYOLO,
		RunID:          "run-expired-1",
		TaskID:         "task-expired-1",
		CreatedAt:      "2026-03-24T00:00:00Z",
		UpdatedAt:      "2026-03-24T01:00:00Z",
	}

	svc := followUpTestService(t, root, []Record{parentRecord}, agent)
	svc.continuer = continuer

	_, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "parent-expired-1",
		FollowUpType: "followup",
		RunMode:      "continue",
	})
	if err == nil {
		t.Fatal("expected error for session expired")
	}
	if !errors.Is(err, errSessionExpired) {
		t.Fatalf("expected errSessionExpired, got %v", err)
	}
}
