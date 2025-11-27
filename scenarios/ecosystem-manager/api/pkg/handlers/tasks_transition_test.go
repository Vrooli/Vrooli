package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// Manual move from a terminal state to pending should re-enable auto-requeue and respect cooldown.
func TestManualTransitionFromTerminalToPendingEnablesAutoRequeue(t *testing.T) {
	queueDir := t.TempDir()
	storage := tasks.NewStorage(queueDir)
	if err := os.MkdirAll(filepath.Join(queueDir, tasks.StatusCompleted), 0o755); err != nil {
		t.Fatalf("mkdir completed: %v", err)
	}
	task := tasks.TaskItem{
		ID:                   "terminal-to-pending",
		Status:               tasks.StatusCompleted,
		ProcessorAutoRequeue: false, // manual terminal moves disable it
	}
	if err := storage.SaveQueueItem(task, tasks.StatusCompleted); err != nil {
		t.Fatalf("save task: %v", err)
	}

	lc := tasks.Lifecycle{Store: storage}
	outcome, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusPending,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentManual,
		},
	})
	if err != nil {
		t.Fatalf("apply transition: %v", err)
	}

	if outcome.Task.Status != tasks.StatusPending {
		t.Fatalf("expected pending, got %s", outcome.Task.Status)
	}
	if !outcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue to be re-enabled when leaving terminal state manually")
	}
	if !outcome.Effects.StartIfSlotAvailable {
		t.Fatalf("expected StartIfSlotAvailable to be signaled for pending transition")
	}
}

// Manual completion should lock auto-requeue off, apply cooldown, and restarting to active should request a force start.
func TestManualCompletionLocksAutoRequeueAndCooldown(t *testing.T) {
	queueDir := t.TempDir()
	storage := tasks.NewStorage(queueDir)
	if err := os.MkdirAll(filepath.Join(queueDir, tasks.StatusInProgress), 0o755); err != nil {
		t.Fatalf("mkdir in-progress: %v", err)
	}
	task := tasks.TaskItem{
		ID:                   "manual-complete",
		Status:               tasks.StatusInProgress,
		ProcessorAutoRequeue: true,
	}
	if err := storage.SaveQueueItem(task, tasks.StatusInProgress); err != nil {
		t.Fatalf("save task: %v", err)
	}

	lc := tasks.Lifecycle{Store: storage}

	completeOutcome, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusCompleted,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentManual,
		},
	})
	if err != nil {
		t.Fatalf("apply completion: %v", err)
	}
	if completeOutcome.Task.Status != tasks.StatusCompleted {
		t.Fatalf("expected completed, got %s", completeOutcome.Task.Status)
	}
	if completeOutcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue disabled on manual completion")
	}
	if completeOutcome.Task.CooldownUntil == "" {
		t.Fatalf("expected cooldown to be applied on manual completion")
	}
	if !completeOutcome.Effects.TerminateProcess {
		t.Fatalf("expected termination flag when leaving in-progress to completed")
	}

	restartOutcome, err := lc.ApplyTransition(tasks.TransitionRequest{
		TaskID:   task.ID,
		ToStatus: tasks.StatusInProgress,
		TransitionContext: tasks.TransitionContext{
			Intent: tasks.IntentManual,
		},
	})
	if err != nil {
		t.Fatalf("apply restart: %v", err)
	}
	if restartOutcome.Task.Status != tasks.StatusInProgress {
		t.Fatalf("expected in-progress, got %s", restartOutcome.Task.Status)
	}
	if !restartOutcome.Task.ProcessorAutoRequeue {
		t.Fatalf("expected auto-requeue re-enabled when leaving terminal state")
	}
	if !restartOutcome.Effects.ForceStart {
		t.Fatalf("expected force-start flag when manually moving to active")
	}
}
