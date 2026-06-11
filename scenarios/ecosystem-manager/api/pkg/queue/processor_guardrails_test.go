package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

type mockImportanceProvider struct {
	scores map[string]float64
}

func (m mockImportanceProvider) Importance(_ context.Context, scenario string) (float64, bool, error) {
	score, ok := m.scores[scenario]
	return score, ok, nil
}

type mockMaturityGapProvider struct {
	scores map[string]float64
}

func (m mockMaturityGapProvider) MaturityGap(_ context.Context, task tasks.TaskItem) (float64, bool, error) {
	score, ok := m.scores[task.ID]
	return score, ok, nil
}

// newGuardrailProcessor creates a processor with real storage backed by a temp directory,
// suitable for testing ForceStartTask and StartTaskIfSlotAvailable.
func newGuardrailProcessor(t *testing.T) (*Processor, *tasks.Storage) {
	t.Helper()

	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")
	for _, status := range tasks.QueueStatuses {
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

	broadcast := make(chan any, 16)
	mockRegistry := NewMockExecutionRegistry()
	mockRecycler := &MockRecycler{}

	processor := NewProcessor(ProcessorDeps{
		Storage:     storage,
		Assembler:   assembler,
		Broadcast:   broadcast,
		Registry:    mockRegistry,
		Recycler:    mockRecycler,
		TaskLogsDir: filepath.Join(tempDir, "logs", "task-runs"),
	})

	return processor, storage
}

// savePendingTask creates and persists a task in the pending queue.
func savePendingTask(t *testing.T, storage *tasks.Storage, id string) tasks.TaskItem {
	t.Helper()
	task := tasks.TaskItem{
		ID:                   id,
		Title:                "Test task " + id,
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "pending",
		Priority:             "medium",
		ProcessorAutoRequeue: true,
		CreatedAt:            "2025-01-01T00:00:00Z",
		UpdatedAt:            "2025-01-01T00:00:00Z",
	}
	if err := storage.SaveQueueItem(task, "pending"); err != nil {
		t.Fatalf("save task %s: %v", id, err)
	}
	return task
}

func TestSelectPendingTask_DefaultUsesExistingPriorityOrdering(t *testing.T) {
	processor, _ := newGuardrailProcessor(t)
	processor.SetSchedulingSignalProviders(
		mockImportanceProvider{scores: map[string]float64{"low-importance": 0.1, "high-importance": 1}},
		mockMaturityGapProvider{scores: map[string]float64{"critical-task": 0.1, "medium-task": 1}},
	)

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.ImportanceAwareScheduling = false
		return s
	})
	defer restore()

	pending := []tasks.TaskItem{
		{
			ID:                   "medium-task",
			Target:               "high-importance",
			Priority:             "medium",
			ProcessorAutoRequeue: true,
		},
		{
			ID:                   "critical-task",
			Target:               "low-importance",
			Priority:             "critical",
			ProcessorAutoRequeue: true,
		},
	}

	selected := processor.selectPendingTask(pending)
	if selected.task == nil || selected.task.ID != "critical-task" {
		t.Fatalf("selected %v, want critical-task", selected.task)
	}
}

func TestSelectPendingTask_ImportanceAwareUsesImportanceTimesMaturityGap(t *testing.T) {
	processor, _ := newGuardrailProcessor(t)
	processor.SetSchedulingSignalProviders(
		mockImportanceProvider{scores: map[string]float64{"low-importance": 0.2, "high-importance": 0.9}},
		mockMaturityGapProvider{scores: map[string]float64{"critical-task": 0.2, "medium-task": 0.9}},
	)

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.ImportanceAwareScheduling = true
		return s
	})
	defer restore()

	pending := []tasks.TaskItem{
		{
			ID:                   "critical-task",
			Target:               "low-importance",
			Priority:             "critical",
			ProcessorAutoRequeue: true,
		},
		{
			ID:                   "medium-task",
			Target:               "high-importance",
			Priority:             "medium",
			ProcessorAutoRequeue: true,
		},
	}

	selected := processor.selectPendingTask(pending)
	if selected.task == nil || selected.task.ID != "medium-task" {
		t.Fatalf("selected %v, want medium-task", selected.task)
	}
	if selected.score != 0.81 {
		t.Fatalf("score = %v, want 0.81", selected.score)
	}
}

// --- ForceStartTask tests ---

func TestForceStartTask_RejectsWhenPaused(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-paused")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()
	processor.Pause()

	err := processor.ForceStartTask("task-paused", false)
	if err == nil {
		t.Fatal("expected error when processor is paused")
	}
	if err.Error() != "processor is paused" {
		t.Fatalf("expected 'processor is paused' error, got: %v", err)
	}
}

func TestForceStartTask_RejectsWhenInactive(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-inactive")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = false
		return s
	})
	defer restore()

	err := processor.ForceStartTask("task-inactive", false)
	if err == nil {
		t.Fatal("expected error when processor is inactive")
	}
	if err.Error() != "processor is inactive" {
		t.Fatalf("expected 'processor is inactive' error, got: %v", err)
	}
}

func TestForceStartTask_RejectsNilProcessor(t *testing.T) {
	var processor *Processor
	err := processor.ForceStartTask("task-1", false)
	if err == nil {
		t.Fatal("expected error for nil processor")
	}
}

func TestForceStartTask_RejectsEmptyTaskID(t *testing.T) {
	processor, _ := newGuardrailProcessor(t)

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	err := processor.ForceStartTask("", false)
	if err == nil {
		t.Fatal("expected error for empty task ID")
	}
}

func TestForceStartTask_SkipsDuplicateLaunch(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-dup")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	// Pre-register the task as running
	processor.registry.ReserveExecution("task-dup", "agent-dup", time.Now())

	err := processor.ForceStartTask("task-dup", false)
	if err != nil {
		t.Fatalf("expected nil error for already-running task, got: %v", err)
	}
}

func TestForceStartTask_RejectsAutoRequeueDisabled(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)

	task := tasks.TaskItem{
		ID:                   "task-locked",
		Title:                "Locked task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "pending",
		Priority:             "medium",
		ProcessorAutoRequeue: false,
		CreatedAt:            "2025-01-01T00:00:00Z",
		UpdatedAt:            "2025-01-01T00:00:00Z",
	}
	if err := storage.SaveQueueItem(task, "pending"); err != nil {
		t.Fatalf("save: %v", err)
	}

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	err := processor.ForceStartTask("task-locked", false)
	if err == nil {
		t.Fatal("expected error for auto-requeue disabled task with allowOverflow=false")
	}
}

// --- StartTaskIfSlotAvailable tests ---

func TestStartTaskIfSlotAvailable_SkipsWhenPaused(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-paused-slot")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()
	processor.Pause()

	err := processor.StartTaskIfSlotAvailable("task-paused-slot")
	if err != nil {
		t.Fatalf("expected nil error (silent skip) when paused, got: %v", err)
	}

	// Verify task was NOT started
	if processor.registry.IsTaskRunning("task-paused-slot") {
		t.Fatal("task should not have been started while paused")
	}
}

func TestStartTaskIfSlotAvailable_SkipsWhenInactive(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-inactive-slot")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = false
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()

	err := processor.StartTaskIfSlotAvailable("task-inactive-slot")
	if err != nil {
		t.Fatalf("expected nil error (silent skip) when inactive, got: %v", err)
	}

	if processor.registry.IsTaskRunning("task-inactive-slot") {
		t.Fatal("task should not have been started while inactive")
	}
}

func TestStartTaskIfSlotAvailable_SkipsWhenNotRunning(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-not-running")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	// Don't start the processor
	err := processor.StartTaskIfSlotAvailable("task-not-running")
	if err != nil {
		t.Fatalf("expected nil error when not running, got: %v", err)
	}
}

func TestStartTaskIfSlotAvailable_SkipsWhenNoSlots(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-no-slots")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		s.Slots = 1
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()

	// Fill the single slot
	processor.registry.ReserveExecution("occupying-task", "agent-1", time.Now())

	err := processor.StartTaskIfSlotAvailable("task-no-slots")
	if err != nil {
		t.Fatalf("expected nil error when no slots, got: %v", err)
	}

	if processor.registry.IsTaskRunning("task-no-slots") {
		t.Fatal("task should not have been started with no available slots")
	}
}

func TestStartTaskIfSlotAvailable_SkipsNonPendingTask(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)

	// Save task in completed status
	task := tasks.TaskItem{
		ID:                   "task-completed",
		Title:                "Completed task",
		Type:                 "scenario",
		Operation:            "improver",
		Status:               "completed",
		Priority:             "medium",
		ProcessorAutoRequeue: true,
		CreatedAt:            "2025-01-01T00:00:00Z",
		UpdatedAt:            "2025-01-01T00:00:00Z",
	}
	if err := storage.SaveQueueItem(task, "completed"); err != nil {
		t.Fatalf("save: %v", err)
	}

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		s.Slots = 2
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()

	err := processor.StartTaskIfSlotAvailable("task-completed")
	if err != nil {
		t.Fatalf("expected nil error for non-pending task, got: %v", err)
	}
}

func TestStartTaskIfSlotAvailable_RejectsNilProcessor(t *testing.T) {
	var processor *Processor
	err := processor.StartTaskIfSlotAvailable("task-1")
	if err == nil {
		t.Fatal("expected error for nil processor")
	}
}

func TestStartTaskIfSlotAvailable_RejectsEmptyTaskID(t *testing.T) {
	processor, _ := newGuardrailProcessor(t)
	err := processor.StartTaskIfSlotAvailable("")
	if err == nil {
		t.Fatal("expected error for empty task ID")
	}
}

// --- ProcessQueue guardrail tests ---

func TestProcessQueue_SkipsWhenPaused(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-pq-paused")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	processor.Pause()
	processor.ProcessQueue()

	// Verify no tasks were started
	if processor.registry.Count() > 0 {
		t.Fatal("ProcessQueue should not start tasks when paused")
	}
}

func TestProcessQueue_SkipsWhenInactive(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-pq-inactive")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = false
		return s
	})
	defer restore()

	processor.ProcessQueue()

	if processor.registry.Count() > 0 {
		t.Fatal("ProcessQueue should not start tasks when settings inactive")
	}
}

func TestProcessQueue_SkipsWhenRateLimited(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-pq-ratelimit")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		return s
	})
	defer restore()

	// Activate rate limit pause
	processor.rateLimiter.HandlePause(600)

	processor.ProcessQueue()

	if processor.registry.Count() > 0 {
		t.Fatal("ProcessQueue should not start tasks when rate limited")
	}
}

func TestProcessQueue_SkipsWhenExecutionLimitReached(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-pq-limit")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		s.ExecutionLimit = 1
		return s
	})
	defer restore()

	// Simulate execution limit already reached
	processor.incrementExecutionCount()

	processor.ProcessQueue()

	if processor.registry.Count() > 0 {
		t.Fatal("ProcessQueue should not start tasks when execution limit reached")
	}
}

// --- Guardrail consistency: all entry points respect the same checks ---

func TestAllEntryPointsRespectPausedState(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-consistency")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = true
		s.Slots = 5
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()
	processor.Pause()

	// All three entry points should refuse to start tasks
	entryPoints := []struct {
		name string
		fn   func() error
	}{
		{"ForceStartTask", func() error { return processor.ForceStartTask("task-consistency", true) }},
		{"StartTaskIfSlotAvailable", func() error { return processor.StartTaskIfSlotAvailable("task-consistency") }},
		{"ProcessQueue", func() error { processor.ProcessQueue(); return nil }},
	}

	for _, ep := range entryPoints {
		t.Run(ep.name, func(t *testing.T) {
			_ = ep.fn() // Some return errors, some don't
			if processor.registry.IsTaskRunning("task-consistency") {
				t.Fatalf("%s started a task despite processor being paused", ep.name)
			}
		})
	}
}

func TestAllEntryPointsRespectInactiveSettings(t *testing.T) {
	processor, storage := newGuardrailProcessor(t)
	savePendingTask(t, storage, "task-inactive-all")

	restore := ensureSettings(t, func(s settings.Settings) settings.Settings {
		s.Active = false
		s.Slots = 5
		return s
	})
	defer restore()

	processor.Start()
	defer processor.Stop()

	entryPoints := []struct {
		name string
		fn   func() error
	}{
		{"ForceStartTask", func() error { return processor.ForceStartTask("task-inactive-all", true) }},
		{"StartTaskIfSlotAvailable", func() error { return processor.StartTaskIfSlotAvailable("task-inactive-all") }},
		{"ProcessQueue", func() error { processor.ProcessQueue(); return nil }},
	}

	for _, ep := range entryPoints {
		t.Run(ep.name, func(t *testing.T) {
			_ = ep.fn()
			if processor.registry.IsTaskRunning("task-inactive-all") {
				t.Fatalf("%s started a task despite settings being inactive", ep.name)
			}
		})
	}
}
