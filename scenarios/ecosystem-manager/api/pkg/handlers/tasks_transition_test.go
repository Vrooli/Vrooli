package handlers

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// Ensure that moving a task back to pending does not flip auto-requeue on if the user disabled it.
func TestApplyStatusTransitionRespectsAutoRequeueWhenPending(t *testing.T) {
	handler := &TaskHandlers{}
	task := &tasks.TaskItem{
		ID:                   "manual-toggle",
		Status:               "pending", // Handlers set Status before applying transition
		ProcessorAutoRequeue: false,
	}

	now, err := handler.applyStatusTransitionLogic(task, task.ID, "completed", "pending")
	if err != nil {
		t.Fatalf("applyStatusTransitionLogic: %v", err)
	}
	if now == "" {
		t.Fatalf("expected timestamp to be set")
	}

	if task.Status != "pending" {
		t.Fatalf("expected status pending, got %s", task.Status)
	}
	if task.ProcessorAutoRequeue {
		t.Fatalf("expected ProcessorAutoRequeue to remain false when returning to pending")
	}
	if task.CooldownUntil != "" {
		t.Fatalf("expected cooldown cleared, got %s", task.CooldownUntil)
	}
}
