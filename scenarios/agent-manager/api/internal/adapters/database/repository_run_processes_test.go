package database

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

func TestManagedRunProcessProjectionPreservesCandidatePopulation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "process candidates", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	want := make(map[uuid.UUID]int)
	for _, status := range []domain.RunStatus{domain.RunStatusPending, domain.RunStatusStarting, domain.RunStatusRunning, domain.RunStatusNeedsReview, domain.RunStatusParked, domain.RunStatusComplete, domain.RunStatusFailed, domain.RunStatusCancelled, domain.RunStatusUnknown} {
		for _, mode := range []domain.ExecutionMode{"", domain.ExecutionModeCodecPipe, domain.ExecutionModeInteractive, domain.ExecutionModeImported, domain.ExecutionModeAttached, "legacy-mode"} {
			for _, pid := range []int{0, 1, 4321} {
				run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: status, ExecutionMode: mode, RunnerPID: pid, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone}
				if err := repos.Runs.Create(ctx, run); err != nil {
					t.Fatal(err)
				}
				if pid > 1 && mode != domain.ExecutionModeAttached && !status.IsTerminal() {
					want[run.ID] = pid
				}
			}
		}
	}
	// NULL and empty legacy modes normalize to codec_pipe, including in this
	// projection. Neither may silently escape the managed-ownership check.
	for _, value := range []any{nil, ""} {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, RunnerPID: 7654, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, "UPDATE runs SET execution_mode = ? WHERE id = ?", value, run.ID); err != nil {
			t.Fatal(err)
		}
		want[run.ID] = run.RunnerPID
	}
	reader := repos.Runs.(repository.ManagedRunProcessReader)
	got, err := reader.ListManagedRunProcesses(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d; want %d", len(got), len(want))
	}
	for _, candidate := range got {
		if want[candidate.ID] != candidate.RunnerPID {
			t.Fatalf("unexpected candidate: %+v", candidate)
		}
		delete(want, candidate.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing candidates: %v", want)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reader.ListManagedRunProcesses(canceled); err == nil {
		t.Fatal("canceled inventory read reported success")
	}
}
