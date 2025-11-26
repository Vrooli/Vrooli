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
	processor := NewProcessor(storage, assembler, broadcast, rec)

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
