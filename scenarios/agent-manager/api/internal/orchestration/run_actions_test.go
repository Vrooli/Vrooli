package orchestration_test

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
)

// TestAttachRunActionsList_ActionsPopulated verifies that ListRuns populates
// action flags for every run in the result set.
func TestAttachRunActionsList_ActionsPopulated(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		Name:       "test-actions",
		ProfileKey: "test-actions-" + uuid.New().String()[:8], RoleRef: "code.default",
	})

	task := mustCreateTask(t, svc, ctx, &domain.Task{
		Title:       "actions-test",
		Description: "test task for action flags",
		ScopePath:   "src/",
	})

	// Create two runs
	for i := 0; i < 2; i++ {
		_, _ = svc.CreateRun(ctx, orchestration.CreateRunRequest{
			TaskID:         task.ID,
			AgentProfileID: &profile.ID,
			Prompt:         "Test prompt",
		})
	}

	runs, err := svc.ListRuns(ctx, orchestration.RunListOptions{})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}

	for i, run := range runs {
		if run.Actions == nil {
			t.Errorf("run[%d] (%s): Actions is nil, expected populated", i, run.ID)
		}
	}
}

// TestAttachRunActionsList_EmptySlice ensures no panic and no unnecessary work
// when the run list is empty.
func TestAttachRunActionsList_EmptySlice(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	runs, err := svc.ListRuns(ctx, orchestration.RunListOptions{
		TagPrefix: "nonexistent-tag-" + uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected 0 runs, got %d", len(runs))
	}
}
