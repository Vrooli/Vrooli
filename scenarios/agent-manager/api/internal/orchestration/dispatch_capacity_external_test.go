package orchestration_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
)

// TestOrchestrator_SlotEnforcement_IgnoresAttachedSessions verifies that
// attached interactive sessions never consume dispatch capacity: operator
// terminals must not starve delivery workers.
func TestOrchestrator_SlotEnforcement_IgnoresAttachedSessions(t *testing.T) {
	svc, runRepo := newTestOrchestratorWithLimit(t, 2)
	ctx := context.Background()

	profile := &domain.AgentProfile{ID: uuid.New(), Name: "attached-slot-profile", CreatedAt: time.Now(), UpdatedAt: time.Now(), RoleRef: "code.default"}
	mustCreateProfile(t, svc, ctx, profile)
	task := &domain.Task{ID: uuid.New(), Title: "Attached Slot Task", ScopePath: "src/", Status: domain.TaskStatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	mustCreateTask(t, svc, ctx, task)

	// Three live attached sessions, more than the limit, plus one dispatched run.
	for i := 0; i < 3; i++ {
		attached := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, RunMode: domain.RunModeInPlace, ExecutionMode: domain.ExecutionModeAttached, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		if err := runRepo.Create(ctx, attached); err != nil {
			t.Fatalf("create attached run: %v", err)
		}
	}
	dispatched := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, RunMode: domain.RunModeInPlace, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := runRepo.Create(ctx, dispatched); err != nil {
		t.Fatalf("create dispatched run: %v", err)
	}

	if _, err := svc.CreateRun(ctx, authenticatedCreateRunFixture(orchestration.CreateRunRequest{TaskID: task.ID, AgentProfileID: &profile.ID, Prompt: "fits beside attached sessions"})); err != nil {
		if _, isCapacity := err.(*domain.CapacityExceededError); isCapacity {
			t.Fatalf("attached sessions consumed dispatch capacity: %v", err)
		}
	}

	// A second dispatched run fills the limit; the next one is refused.
	second := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, RunMode: domain.RunModeInPlace, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := runRepo.Create(ctx, second); err != nil {
		t.Fatalf("create second dispatched run: %v", err)
	}
	_, err := svc.CreateRun(ctx, authenticatedCreateRunFixture(orchestration.CreateRunRequest{TaskID: task.ID, AgentProfileID: &profile.ID, Prompt: "over the dispatch limit"}))
	capErr, ok := err.(*domain.CapacityExceededError)
	if !ok {
		t.Fatalf("expected *domain.CapacityExceededError once dispatched runs fill the limit, got %T: %v", err, err)
	}
	if capErr.Current < 2 || capErr.Maximum != 2 {
		t.Fatalf("capacity counted attached sessions or wrong limit: %+v", capErr)
	}
}
