package queue

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// ensureSettings updates settings for the test and restores afterward.
func ensureSettings(t *testing.T, configure func(settings.Settings) settings.Settings) func() {
	t.Helper()
	prev := settings.GetSettings()
	next := configure(prev)
	settings.UpdateSettings(next)
	return func() { settings.UpdateSettings(prev) }
}

func newTestProcessorWithRecycler(t *testing.T) (*Processor, *recycler.Recycler, *tasks.Storage, func()) {
	t.Helper()
	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")
	for _, status := range []string{"pending", "in-progress", "completed", "failed"} {
		if err := os.MkdirAll(filepath.Join(queueDir, status), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", status, err)
		}
	}
	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatalf("mkdir prompts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptsDir, "sections.yaml"), []byte("sections: []"), 0o644); err != nil {
		t.Fatalf("write sections: %v", err)
	}

	storage := tasks.NewStorage(queueDir)
	assembler, err := prompts.NewAssembler(promptsDir, tempDir)
	if err != nil {
		t.Fatalf("assembler: %v", err)
	}

	broadcast := make(chan any, 4)
	rec := recycler.New(storage, nil)
	rec.Start()
	processor := NewProcessor(ProcessorDeps{
		Storage:   storage,
		Assembler: assembler,
		Broadcast: broadcast,
		Recycler:  rec,
	})

	cleanup := func() {
		rec.Stop()
		processor.Stop()
	}
	return processor, rec, storage, cleanup
}

func TestFinalizeTaskStatusEnqueuesRecyclerOnlyWhenAutoRequeue(t *testing.T) {
	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Recycler.EnabledFor = "both"
		return s
	})
	defer restore()

	processor, rec, storage, cleanup := newTestProcessorWithRecycler(t)
	defer cleanup()

	task := tasks.TaskItem{
		ID:                   "finalize-task",
		Type:                 "resource",
		Operation:            "generator",
		Status:               "in-progress",
		ProcessorAutoRequeue: true,
		CreatedAt:            "2025-01-01T00:00:00Z",
		UpdatedAt:            "2025-01-01T00:00:00Z",
	}
	if err := storage.SaveQueueItem(task, "in-progress"); err != nil {
		t.Fatalf("save task: %v", err)
	}

	before := rec.Stats().Enqueued

	if err := processor.finalizeTaskStatus(&task, "completed"); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	if rec.Stats().Enqueued-before != 1 {
		t.Fatalf("expected recycler to enqueue task when auto-requeue true")
	}

	// Now ensure auto-requeue false does not enqueue.
	before = rec.Stats().Enqueued
	task2 := task
	task2.ID = "no-auto"
	task2.ProcessorAutoRequeue = false
	if err := storage.SaveQueueItem(task2, "in-progress"); err != nil {
		t.Fatalf("save task2: %v", err)
	}
	if err := processor.finalizeTaskStatus(&task2, "completed"); err != nil {
		t.Fatalf("finalize no-auto: %v", err)
	}
	if rec.Stats().Enqueued-before != 0 {
		t.Fatalf("expected recycler not to enqueue when auto-requeue is false")
	}
}

// TestFinalizeTaskStatusToPendingCleansUpInProgress verifies that when a task
// in in-progress/ is finalized to "pending" (e.g., steering requeue or rate
// limit), the stale in-progress/ copy is removed and the task exists only in
// pending/.  This is a regression test for the bug where finalizeTaskStatus
// used the in-memory task.Status (already set to "pending" by the caller)
// instead of querying the filesystem for the actual location, causing a
// duplicate file that lingered until the next execution cycle.
func TestFinalizeTaskStatusToPendingCleansUpInProgress(t *testing.T) {
	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.CooldownSeconds = 0
		return s
	})
	defer restore()

	processor, _, storage, cleanup := newTestProcessorWithRecycler(t)
	defer cleanup()

	// Task is on disk in in-progress/, but the caller (handleRateLimitedExecution
	// or handleSteeringContinuation) has already set task.Status = "pending".
	task := tasks.TaskItem{
		ID:                   "requeue-task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "pending", // in-memory status set by caller
		ProcessorAutoRequeue: true,
		CreatedAt:            "2025-01-01T00:00:00Z",
		UpdatedAt:            "2025-01-01T00:00:00Z",
	}
	// Save to in-progress/ on disk (the real location before finalization).
	if err := storage.SaveQueueItem(task, "in-progress"); err != nil {
		t.Fatalf("save task to in-progress: %v", err)
	}

	// Finalize to pending — this is the path taken by rate-limit and steering requeue.
	if err := processor.finalizeTaskStatus(&task, "pending"); err != nil {
		t.Fatalf("finalize to pending: %v", err)
	}

	// Verify: task must exist in pending/
	pendingPath := filepath.Join(storage.QueueDir, "pending", "requeue-task.yaml")
	if _, err := os.Stat(pendingPath); err != nil {
		t.Fatalf("expected task in pending/ but got: %v", err)
	}

	// Verify: task must NOT exist in in-progress/ (the stale copy)
	inProgressPath := filepath.Join(storage.QueueDir, "in-progress", "requeue-task.yaml")
	if _, err := os.Stat(inProgressPath); err == nil {
		t.Fatalf("stale task file found in in-progress/ — duplicate not cleaned up")
	}
}
