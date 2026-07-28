package execution

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/transitionrun"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// followUpTestService creates a Service with seeded records and a declared
// workflow seam. Follow-up and correction never launch an operation runner.
func followUpTestService(t *testing.T, root string, records []Record, agent AgentManagerAvailability) (*Service, *stubConclusionWorkflow) {
	t.Helper()
	registry, err := transitions.LoadDir(filepath.Join("..", "..", "..", ".vrooli", "swarm-transitions"))
	if err != nil {
		t.Fatalf("load transition registry: %v", err)
	}
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")
	store := NewStore(storePath)
	if err := store.Save(records); err != nil {
		t.Fatalf("seed records: %v", err)
	}
	svc := NewService(ServiceConfig{
		DataRoot:           root,
		StorePath:          storePath,
		PlanRenderer:       testPlanRenderer(),
		AgentService:       agent,
		PromptClient:       &promptmanager.MockClient{Result: "test prompt"},
		TransitionRegistry: registry,
	})
	workflow := &stubConclusionWorkflow{}
	svc.SetWorkWorkflow(workflow)
	svc.SetPhasedPlanWorkflow(&stubPhasedPlanWorkflow{})
	return svc, workflow
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

	svc, workflow := followUpTestService(t, root, []Record{parentRecord}, agent)

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
	if workflow.startCalls != 1 || workflow.invocation.WorkflowKey != "swarm-manager/work-follow-up" {
		t.Fatalf("expected work-follow-up workflow start, got calls=%d key=%q", workflow.startCalls, workflow.invocation.WorkflowKey)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("expected 0 direct agent spawns (rerouted through the runner), got %d", agent.spawnCalls)
	}
	if record.RunID == "" || record.AgentWorkflowExecutionID == "" || record.OpExecutionID != "" {
		t.Fatalf("expected workflow correlation without an operation execution, got %#v", record)
	}
}

func TestFollowUp_SourceProposalIsExactlyOnce(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "dedup-idea", map[string]any{"name": "dedup-idea", "title": "Dedup Idea", "description": "desc", "status": "completed", "priority": 3, "tags": []string{}})
	mustWriteDeliverableFile(t, root, "idea", "dedup-idea")
	parent := Record{ExecutionID: "parent-dedup", BacklogKind: "idea", BacklogName: "dedup-idea", Status: StatusCompleted, Mode: ModeYOLO}
	svc, workflow := followUpTestService(t, root, []Record{parent}, &stubAgentService{})
	request := FollowUpRequest{ExecutionID: parent.ExecutionID, FollowUpType: "followup", Context: "verified review finding", SourceProposalID: "proposal-review-1", SourceReviewRef: "review/idea/dedup-idea/round/1"}
	first, err := svc.FollowUp(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.FollowUp(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutionID != second.ExecutionID || first.FollowUpSourceProposalID != request.SourceProposalID || first.FollowUpSourceReviewRef != request.SourceReviewRef || workflow.startCalls != 1 {
		t.Fatalf("first=%+v second=%+v starts=%d", first, second, workflow.startCalls)
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

	svc, workflow := followUpTestService(t, root, []Record{parentRecord}, agent)

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
	if workflow.startCalls != 1 || workflow.invocation.WorkflowKey != "swarm-manager/work-correct" {
		t.Fatalf("expected work-correct workflow start, got calls=%d key=%q", workflow.startCalls, workflow.invocation.WorkflowKey)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("expected 0 direct agent spawns, got %d", agent.spawnCalls)
	}
}

// TestFollowUp_ContinueCollapsesToFreshWorkflow pins that a continuation request
// becomes a fresh, declared workflow with the note in its immutable snapshot.
func TestFollowUp_ContinueCollapsesToFreshWorkflow(t *testing.T) {
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

	svc, workflow := followUpTestService(t, root, []Record{parentRecord}, agent)

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
		t.Fatalf("expected starting status for a fresh follow-up workflow, got %s", record.Status)
	}
	// A fresh operation: the record does not inherit the parent run id.
	if record.RunID == "run-continue-1" {
		t.Fatal("expected a fresh run id, not the inherited parent run id")
	}
	if record.AgentWorkflowExecutionID == "" || record.OpExecutionID != "" {
		t.Fatal("expected a workflow correlation and no operation-execution correlation")
	}
	if workflow.startCalls != 1 || workflow.invocation.WorkflowKey != "swarm-manager/work-follow-up" {
		t.Fatalf("expected work-follow-up workflow start, got calls=%d key=%q", workflow.startCalls, workflow.invocation.WorkflowKey)
	}
	snapshot := workflow.invocation.Input.AsInterface().(map[string]any)["snapshot"].(map[string]any)
	if got := snapshot["operatorNote"]; got != "Please fix the remaining lint errors" {
		t.Fatalf("expected note in immutable snapshot, got %q", got)
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

func TestApplyWorkWorkflow_ExactlyOnce(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "work-apply", map[string]any{"name": "work-apply", "title": "Work Apply", "description": "desc", "status": "completed", "priority": 3, "tags": []string{}})
	mustWriteDeliverableFile(t, root, "idea", "work-apply")
	parent := Record{ExecutionID: "parent-work", BacklogKind: "idea", BacklogName: "work-apply", Status: StatusCompleted, Mode: ModeYOLO}
	svc, workflow := followUpTestService(t, root, []Record{parent}, &stubAgentService{})
	started, err := svc.FollowUp(context.Background(), FollowUpRequest{ExecutionID: parent.ExecutionID, FollowUpType: "followup"})
	if err != nil {
		t.Fatal(err)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "proposed", "summary": "needs review"}})
	if err != nil {
		t.Fatal(err)
	}
	workflow.completion = agentmanager.InvocationCompletion{ExecutionID: started.AgentWorkflowExecutionID, DefinitionDigest: started.AgentWorkflowDefinition, Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: workflow.invocation.Input, Output: output}
	first, err := svc.ApplyWorkWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("apply work workflow: %T %[1]v", err)
	}
	if first.Idempotent || first.Record.Status != StatusNeedsReview || transitionApplyStateFor(t, svc, first.Record.AgentWorkflowExecutionID) != transitionrun.ApplyStateComplete {
		t.Fatalf("unexpected first apply: %#v", first)
	}
	second, err := svc.ApplyWorkWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Idempotent || workflow.collectCalls != 1 {
		t.Fatalf("expected idempotent replay with one collect, got %#v collects=%d", second, workflow.collectCalls)
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
