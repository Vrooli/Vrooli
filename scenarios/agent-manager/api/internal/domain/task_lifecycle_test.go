package domain

import "testing"

func TestTaskStatusTransitionOwnerValidatesLifecycleEdges(t *testing.T) {
	for _, edge := range [][2]TaskStatus{
		{TaskStatusQueued, TaskStatusRunning},
		{TaskStatusRunning, TaskStatusNeedsReview},
		{TaskStatusNeedsReview, TaskStatusRunning},
		{TaskStatusCompleted, TaskStatusRunning},
		{TaskStatusFailed, TaskStatusRunning},
		{TaskStatusQueued, TaskStatusCancelled},
	} {
		if err := ValidateTaskStatusTransition("task", edge[0], edge[1], "test"); err != nil {
			t.Errorf("legal Task edge %q -> %q rejected: %v", edge[0], edge[1], err)
		}
	}
	for _, edge := range [][2]TaskStatus{
		{TaskStatusApproved, TaskStatusRunning},
		{TaskStatusRejected, TaskStatusQueued},
		{TaskStatusCancelled, TaskStatusRunning},
		{TaskStatusQueued, TaskStatus("future")},
		{TaskStatus("future"), TaskStatusRunning},
	} {
		if err := ValidateTaskStatusTransition("task", edge[0], edge[1], "test"); err == nil {
			t.Errorf("invalid Task edge %q -> %q accepted", edge[0], edge[1])
		}
	}
}

func TestTaskTransitionStatusMutatesOnlyAfterValidation(t *testing.T) {
	task := &Task{Status: TaskStatusApproved}
	if err := task.TransitionStatus(TaskStatusRunning); err == nil {
		t.Fatal("terminal Task was reopened")
	}
	if task.Status != TaskStatusApproved {
		t.Fatalf("refused transition mutated Task to %q", task.Status)
	}
	task.Status = TaskStatusQueued
	if err := task.TransitionStatus(TaskStatusRunning); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskStatusRunning {
		t.Fatalf("transition status=%q, want running", task.Status)
	}
}
