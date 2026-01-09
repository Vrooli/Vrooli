package queue

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestShouldContinueTaskRespectsManualDisable(t *testing.T) {
	integration := AutoSteerIntegration{}
	task := tasks.TaskItem{
		ID:                   "task-123",
		AutoSteerProfileID:   "profile-abc",
		ProcessorAutoRequeue: false,
	}

	shouldContinue, err := integration.ShouldContinueTask(&task, "scenario-name")
	if err != nil {
		t.Fatalf("expected no error when auto-enqueue disabled, got %v", err)
	}
	if shouldContinue {
		t.Fatalf("expected shouldContinue=false when ProcessorAutoRequeue is false")
	}
}
