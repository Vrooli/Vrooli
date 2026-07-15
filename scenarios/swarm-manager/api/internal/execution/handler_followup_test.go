package execution

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/promptmanager"
)

// followUpTestService creates a Service with seeded records, returning the
// service and the stub operation starter follow-up/fixup now route through.
func followUpTestService(t *testing.T, root string, records []Record, agent AgentSpawner) (*Service, *stubOperationStarter) {
	t.Helper()
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")
	store := NewStore(storePath)
	if err := store.Save(records); err != nil {
		t.Fatalf("seed records: %v", err)
	}
	svc := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    storePath,
		PlanRenderer: testPlanRenderer(),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	// Follow-up and fix-up start through the generic operation runner
	// (execution-followup / execution-fixup on the backlog-item target), so the
	// stub starter — not the agent spawner — records their launches.
	starter := &stubOperationStarter{}
	svc.SetOperationStarter(starter)
	return svc, starter
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
	mustWriteDeliverableFile(t, root, "idea", "followup-idea")

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

	svc, starter := followUpTestService(t, root, []Record{parentRecord}, agent)

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
	if starter.calls != 1 {
		t.Fatalf("expected 1 operation start, got %d", starter.calls)
	}
	if starter.req.Operation != operationExecutionFollowup {
		t.Fatalf("expected execution-followup operation, got %q", starter.req.Operation)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("expected 0 direct agent spawns (rerouted through the runner), got %d", agent.spawnCalls)
	}
	// The reroute tracks the live run + operation-execution correlation; TaskID is
	// no longer a follow-up concept (the runner owns the run association).
	if record.RunID == "" || record.OpExecutionID == "" {
		t.Fatalf("expected run + op execution IDs set, got run=%s op=%s", record.RunID, record.OpExecutionID)
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
	mustWriteDeliverableFile(t, root, "fix", "fixup-item")

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
		Finalization: &Finalization{
			Eligible:                true,
			Status:                  FinalizationStatusCompleted,
			Phase:                   FinalizationPhaseCompleted,
			AggregateClassification: FinalizationAggregateNeedsWork,
			AggregateSummary:        "Tests failing",
			Scenarios: []ScenarioFinalization{{
				ScenarioName: "fixup-item",
				Restart:      RestartResult{Status: FinalizationStatusCompleted},
				Health:       HealthCheckResult{Status: FinalizationStatusCompleted, SchemaValid: true},
				Review: ScenarioReviewStep{
					Status: FinalizationStatusCompleted,
					JobID:  "review-1",
					Result: &ReviewResult{
						JobID:          "review-1",
						Classification: "needs_work",
						Summary:        "Tests failing",
						Dimensions: []ReviewDimension{
							{Name: "tests", Status: "red", Details: "2 tests failing"},
						},
					},
				},
			}},
		},
		CreatedAt: "2026-03-24T00:00:00Z",
		UpdatedAt: "2026-03-24T01:00:00Z",
	}

	svc, starter := followUpTestService(t, root, []Record{parentRecord}, agent)

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
	// A fix-up-flavored follow-up routes to the execution-fixup operation.
	if starter.calls != 1 || starter.req.Operation != operationExecutionFixup {
		t.Fatalf("expected 1 execution-fixup operation start, got calls=%d op=%q", starter.calls, starter.req.Operation)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("expected 0 direct agent spawns, got %d", agent.spawnCalls)
	}
}

// TestFollowUp_ContinueCollapsesToFreshOperation pins the declarative-operations
// behavior: run_mode=continue no longer resumes the parent agent session (session
// continuation is not part of the operation model). It starts a fresh
// execution-followup operation whose caller context carries the note.
func TestFollowUp_ContinueCollapsesToFreshOperation(t *testing.T) {
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

	svc, starter := followUpTestService(t, root, []Record{parentRecord}, agent)

	record, err := svc.FollowUp(context.Background(), FollowUpRequest{
		ExecutionID:  "parent-continue-1",
		FollowUpType: "followup",
		Context:      "Please fix the remaining lint errors",
		RunMode:      "continue",
	})
	if err != nil {
		t.Fatalf("FollowUp error: %v", err)
	}
	if record.Status != StatusStarting {
		t.Fatalf("expected starting status for a fresh follow-up operation, got %s", record.Status)
	}
	// A fresh operation: the record does not inherit the parent run id.
	if record.RunID == "run-continue-1" {
		t.Fatal("expected a fresh run id, not the inherited parent run id")
	}
	if record.OpExecutionID == "" {
		t.Fatal("expected an operation-execution correlation id on the follow-up record")
	}
	if starter.calls != 1 || starter.req.Operation != operationExecutionFollowup {
		t.Fatalf("expected 1 execution-followup operation start, got calls=%d op=%q", starter.calls, starter.req.Operation)
	}
	// The follow-up note rides as a caller input for the mode's prompt.
	if got := starter.req.CallerInputs["FOLLOWUP_NOTE"]; got != "Please fix the remaining lint errors" {
		t.Fatalf("expected the note carried as FOLLOWUP_NOTE caller input, got %q", got)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("expected 0 direct agent spawns, got %d", agent.spawnCalls)
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

	svc, _ := followUpTestService(t, root, []Record{parentRecord}, agent)

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

	svc, _ := followUpTestService(t, root, []Record{}, agent)

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

// The former TestFollowUp_SessionExpired was removed with the run_mode=continue
// agent-session continuation path: the declarative-operations model has no session
// resume, so there is no session-expired error to surface (see
// TestFollowUp_ContinueCollapsesToFreshOperation).
