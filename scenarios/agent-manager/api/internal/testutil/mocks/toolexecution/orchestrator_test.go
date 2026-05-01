package toolexecution

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
)

func TestFakeOrchestratorDefaults(t *testing.T) {
	fake := NewFakeOrchestrator()

	task, err := fake.CreateTask(context.Background(), &domain.Task{Title: "test"})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if task.ID == uuid.Nil {
		t.Fatal("expected CreateTask to assign an ID")
	}

	run, err := fake.CreateRun(context.Background(), orchestration.CreateRunRequest{TaskID: task.ID})
	if err != nil {
		t.Fatalf("CreateRun returned error: %v", err)
	}
	if run.TaskID != task.ID || run.Status != domain.RunStatusPending {
		t.Fatalf("unexpected default run: %#v", run)
	}

	runs, err := fake.ListRuns(context.Background(), orchestration.RunListOptions{})
	if err != nil {
		t.Fatalf("ListRuns returned error: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected no default runs, got %d", len(runs))
	}
}

func TestFakeOrchestratorFuncOverrides(t *testing.T) {
	wantRun := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning}
	fake := NewFakeOrchestrator()
	fake.GetRunFunc = func(context.Context, uuid.UUID) (*domain.Run, error) {
		return wantRun, nil
	}

	got, err := fake.GetRun(context.Background(), wantRun.ID)
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if got != wantRun {
		t.Fatal("expected GetRun to return override result")
	}
}
