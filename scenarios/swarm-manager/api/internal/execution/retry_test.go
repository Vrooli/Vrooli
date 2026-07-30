package execution

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRetry_NewAttemptFromFailed(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "retry-idea", map[string]any{
		"name":        "retry-idea",
		"title":       "Retry Idea",
		"description": "desc",
		"status":      "failed",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "retry-idea")

	agent := &stubAgentService{}
	parentRecord := Record{
		ExecutionID:    "parent-1",
		BacklogKind:    "idea",
		BacklogName:    "retry-idea",
		PreviousStatus: "backlog",
		Status:         StatusFailed,
		Mode:           ModeYOLO,
		RunID:          "run-1",
		TaskID:         "task-1",
		FailureReason:  "agent hesitated",
		StartedAt:      "2026-04-25T00:00:00Z",
		FinishedAt:     "2026-04-25T00:30:00Z",
		CreatedAt:      "2026-04-25T00:00:00Z",
		UpdatedAt:      "2026-04-25T00:30:00Z",
	}

	svc, _ := followUpTestService(t, root, []Record{parentRecord}, agent)

	record, err := svc.Retry(context.Background(), RetryRequest{
		ExecutionID: "parent-1",
		Note:        "fixed agent-manager hesitation bug",
	})
	if err != nil {
		t.Fatalf("Retry error: %v", err)
	}
	if record.Status != StatusStarting {
		t.Fatalf("expected starting status, got %s", record.Status)
	}
	if record.ParentExecutionID != "parent-1" {
		t.Fatalf("expected parent_execution_id parent-1, got %s", record.ParentExecutionID)
	}
	if record.Operation != "retry" {
		t.Fatalf("expected operation retry, got %s", record.Operation)
	}
	if record.StartedBy != "swarm-manager:retry" {
		t.Fatalf("expected started_by swarm-manager:retry, got %s", record.StartedBy)
	}
	if record.FixupAttempt != 0 {
		t.Fatalf("expected fixup_attempt 0, got %d", record.FixupAttempt)
	}
	if record.ExecutionID == parentRecord.ExecutionID {
		t.Fatalf("retry must allocate a fresh execution id; got same as parent")
	}
	if record.Mode != ModeYOLO {
		t.Fatalf("expected mode carried from parent (yolo), got %s", record.Mode)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("plan-backed retry must not direct-spawn, got %d", agent.spawnCalls)
	}
	if record.RunID == "" {
		t.Fatal("expected declared workflow correlation on retry record")
	}
	correlation := workflowCorrelationFor(t, svc, record)
	if correlation.WorkflowKey != "swarm-manager/phased-plan-drain" || record.OpExecutionID != "" {
		t.Fatalf("retry must use phased-plan workflow, not operation runtime: %#v", record)
	}

	// Parent record must be untouched on disk.
	store := svc.store
	records, err := store.Load()
	if err != nil {
		t.Fatalf("load records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records after retry, got %d", len(records))
	}
	var parentAfter *Record
	for i := range records {
		if records[i].ExecutionID == "parent-1" {
			parentAfter = &records[i]
			break
		}
	}
	if parentAfter == nil {
		t.Fatal("parent record disappeared after retry")
	}
	if parentAfter.Status != StatusFailed {
		t.Errorf("parent status mutated: was failed, now %s", parentAfter.Status)
	}
	if parentAfter.FailureReason != "agent hesitated" {
		t.Errorf("parent failure_reason mutated: %q", parentAfter.FailureReason)
	}
	if parentAfter.FinishedAt != "2026-04-25T00:30:00Z" {
		t.Errorf("parent finished_at mutated: %q", parentAfter.FinishedAt)
	}
}

func TestRetry_FromCompletedAllowed(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "retry-completed", map[string]any{
		"name":        "retry-completed",
		"title":       "Retry Completed",
		"description": "desc",
		"status":      "completed",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "retry-completed")

	agent := &stubAgentService{}
	parent := Record{
		ExecutionID: "parent-c",
		BacklogKind: "idea",
		BacklogName: "retry-completed",
		Status:      StatusCompleted,
		Mode:        ModeYOLO,
		RunID:       "run-c",
		CreatedAt:   "2026-04-25T00:00:00Z",
		UpdatedAt:   "2026-04-25T00:30:00Z",
	}
	svc, _ := followUpTestService(t, root, []Record{parent}, agent)

	rec, err := svc.Retry(context.Background(), RetryRequest{ExecutionID: "parent-c"})
	if err != nil {
		t.Fatalf("Retry from completed should be allowed, got %v", err)
	}
	if rec.ParentExecutionID != "parent-c" {
		t.Errorf("expected parent linkage, got %q", rec.ParentExecutionID)
	}
}

func TestRetry_RejectsNonTerminal(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "retry-running", map[string]any{
		"name":        "retry-running",
		"title":       "Retry Running",
		"description": "desc",
		"status":      "in_progress",
		"priority":    3,
		"tags":        []string{},
	})

	agent := &stubAgentService{}
	parent := Record{
		ExecutionID: "parent-r",
		BacklogKind: "idea",
		BacklogName: "retry-running",
		Status:      StatusRunning,
		Mode:        ModeYOLO,
		RunID:       "run-r",
		CreatedAt:   "2026-04-25T00:00:00Z",
		UpdatedAt:   "2026-04-25T00:30:00Z",
	}
	svc, _ := followUpTestService(t, root, []Record{parent}, agent)

	_, err := svc.Retry(context.Background(), RetryRequest{ExecutionID: "parent-r"})
	if err == nil {
		t.Fatal("expected error retrying running execution")
	}
	if !strings.Contains(err.Error(), "cannot retry") {
		t.Errorf("error should mention retry restriction, got %q", err.Error())
	}
}

func TestRetry_NotFound(t *testing.T) {
	root := t.TempDir()
	agent := &stubAgentService{}
	svc, _ := followUpTestService(t, root, []Record{}, agent)

	_, err := svc.Retry(context.Background(), RetryRequest{ExecutionID: "missing"})
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !errors.Is(err, errNotFound) {
		t.Fatalf("expected errNotFound, got %v", err)
	}
}

func TestRetry_IdempotentWhenInFlight(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "retry-dedup", map[string]any{
		"name":        "retry-dedup",
		"title":       "Retry Dedup",
		"description": "desc",
		"status":      "failed",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "retry-dedup")

	agent := &stubAgentService{}
	parent := Record{
		ExecutionID: "parent-d",
		BacklogKind: "idea",
		BacklogName: "retry-dedup",
		Status:      StatusFailed,
		Mode:        ModeYOLO,
		CreatedAt:   "2026-04-25T00:00:00Z",
		UpdatedAt:   "2026-04-25T00:30:00Z",
	}
	svc, _ := followUpTestService(t, root, []Record{parent}, agent)

	first, err := svc.Retry(context.Background(), RetryRequest{ExecutionID: "parent-d"})
	if err != nil {
		t.Fatalf("first retry: %v", err)
	}
	if agent.spawnCalls != 0 {
		t.Fatalf("plan-backed retry must not direct-spawn, got %d", agent.spawnCalls)
	}

	// Second call while the first is still in-flight should return the same record.
	second, err := svc.Retry(context.Background(), RetryRequest{ExecutionID: "parent-d"})
	if err != nil {
		t.Fatalf("second retry: %v", err)
	}
	if second.ExecutionID != first.ExecutionID {
		t.Errorf("expected idempotent dedup; got different exec ids %s vs %s", first.ExecutionID, second.ExecutionID)
	}
	if agent.spawnCalls != 0 {
		t.Errorf("dedup retry must not direct-spawn, got %d", agent.spawnCalls)
	}

	records, err := svc.store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records (parent + 1 retry), got %d", len(records))
	}
}

func TestRetry_DoesNotPersistConsumerOwnedPrompt(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "retry-prompt", map[string]any{
		"name":        "retry-prompt",
		"title":       "Retry Prompt",
		"description": "desc",
		"status":      "failed",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "idea", "retry-prompt")

	agent := &stubAgentService{}
	parent := Record{
		ExecutionID: "parent-p",
		BacklogKind: "idea",
		BacklogName: "retry-prompt",
		Status:      StatusFailed,
		Mode:        ModeYOLO,
		CreatedAt:   "2026-04-25T00:00:00Z",
		UpdatedAt:   "2026-04-25T00:30:00Z",
		Finalization: &Finalization{
			Eligible:         true,
			AggregateSummary: "Tests failing — should NOT appear in retry prompt",
		},
	}
	svc, _ := followUpTestService(t, root, []Record{parent}, agent)

	rec, err := svc.Retry(context.Background(), RetryRequest{ExecutionID: "parent-p", Note: "trying again"})
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if rec.PromptTrace != nil {
		t.Fatalf("retry must not persist a consumer-built prompt, got %#v", rec.PromptTrace)
	}
	if correlation := workflowCorrelationFor(t, svc, rec); correlation.WorkflowKey != "swarm-manager/phased-plan-drain" {
		t.Fatalf("expected declared phased-plan workflow, got %q", correlation.WorkflowKey)
	}
}
