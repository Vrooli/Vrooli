package queue

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func setupTestProcessor(t *testing.T) (*Processor, string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")

	for _, status := range []string{"pending", "in-progress", "completed", "failed"} {
		if err := os.MkdirAll(filepath.Join(queueDir, status), 0o755); err != nil {
			t.Fatalf("Failed to create queue dir %s: %v", status, err)
		}
	}

	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}

	// Create minimal sections.yaml
	sectionsYAML := `sections: []`
	if err := os.WriteFile(filepath.Join(promptsDir, "sections.yaml"), []byte(sectionsYAML), 0o644); err != nil {
		t.Fatalf("Failed to create sections.yaml: %v", err)
	}

	storage := tasks.NewStorage(queueDir)
	assembler, err := prompts.NewAssembler(promptsDir)
	if err != nil {
		t.Fatalf("Failed to create assembler: %v", err)
	}

	broadcast := make(chan any, 10)
	processor := NewProcessor(30*time.Second, storage, assembler, broadcast)

	cleanup := func() {
		processor.Stop()
	}

	return processor, tempDir, cleanup
}

func TestProcessor_GetQueueStatus(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	status := processor.GetQueueStatus()

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	// Check required fields
	requiredFields := []string{"processor_active", "pending_count", "executing_count", "completed_count", "failed_count", "available_slots"}
	for _, field := range requiredFields {
		if _, ok := status[field]; !ok {
			t.Errorf("Expected %s field", field)
		}
	}
}

func TestProcessor_StartStop(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	// Initially should be stopped
	status := processor.GetQueueStatus()
	if status["processor_active"].(bool) {
		t.Error("Processor should initially be inactive")
	}

	// Start processor - but processor_active requires BOTH Start() AND settings.Active=true
	// This is a dual-safety feature to prevent unintended task execution
	processor.Start()
	time.Sleep(100 * time.Millisecond)

	// After Start(), processor is running but not active (settings.Active defaults to false)
	status = processor.GetQueueStatus()
	if status["processor_active"].(bool) {
		t.Error("Processor should require settings.Active=true to be fully active (safety feature)")
	}

	// Verify the processor is internally running (just not processing due to settings)
	processor.mu.Lock()
	if !processor.isRunning {
		processor.mu.Unlock()
		t.Error("Processor isRunning should be true after Start()")
	} else {
		processor.mu.Unlock()
	}

	// Stop processor
	processor.Stop()
	time.Sleep(100 * time.Millisecond)

	status = processor.GetQueueStatus()
	if status["processor_active"].(bool) {
		t.Error("Processor should be inactive after Stop()")
	}

	// Verify the processor is internally stopped
	processor.mu.Lock()
	if processor.isRunning {
		processor.mu.Unlock()
		t.Error("Processor isRunning should be false after Stop()")
	} else {
		processor.mu.Unlock()
	}
}

func TestProcessor_MaxConcurrent(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	status := processor.GetQueueStatus()
	maxConcurrent, ok := status["max_concurrent"].(int)
	if !ok {
		t.Fatal("Expected max_concurrent to be int")
	}

	if maxConcurrent <= 0 {
		t.Errorf("Expected positive max_concurrent, got %d", maxConcurrent)
	}
}

func TestProcessor_AvailableSlots(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	status := processor.GetQueueStatus()
	availableSlots, ok := status["available_slots"].(int)
	if !ok {
		t.Fatal("Expected available_slots to be int")
	}

	maxConcurrent := status["max_concurrent"].(int)

	// When no tasks are executing, available slots should equal max concurrent
	if availableSlots != maxConcurrent {
		t.Errorf("Expected available_slots (%d) to equal max_concurrent (%d)", availableSlots, maxConcurrent)
	}
}

func TestProcessor_WithPendingTask(t *testing.T) {
	processor, tempDir, cleanup := setupTestProcessor(t)
	defer cleanup()

	queueDir := filepath.Join(tempDir, "queue")
	storage := tasks.NewStorage(queueDir)

	task := tasks.TaskItem{
		ID:        "test-task-pending",
		Type:      "resource",
		Operation: "generator",
		Target:    "test-resource",
		Status:    "pending",
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	if err := storage.SaveQueueItem(task, "pending"); err != nil {
		t.Fatalf("Failed to save test task: %v", err)
	}

	status := processor.GetQueueStatus()
	pendingCount := status["pending_count"].(int)

	if pendingCount == 0 {
		t.Error("Expected at least one pending task")
	}
}

func TestProcessor_MultipleStates(t *testing.T) {
	processor, tempDir, cleanup := setupTestProcessor(t)
	defer cleanup()

	queueDir := filepath.Join(tempDir, "queue")
	storage := tasks.NewStorage(queueDir)

	states := []string{"pending", "completed", "failed"}
	for i, state := range states {
		task := tasks.TaskItem{
			ID:        "test-task-" + state + "-" + string(rune('0'+i)),
			Type:      "resource",
			Operation: "generator",
			Target:    "test-resource-" + state,
			Status:    state,
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}

		if err := storage.SaveQueueItem(task, state); err != nil {
			t.Fatalf("Failed to save %s task: %v", state, err)
		}
	}

	status := processor.GetQueueStatus()

	if status["pending_count"].(int) == 0 {
		t.Error("Expected pending tasks")
	}

	if status["completed_count"].(int) == 0 {
		t.Error("Expected completed tasks")
	}

	if status["failed_count"].(int) == 0 {
		t.Error("Expected failed tasks")
	}
}

func TestProcessor_ConcurrentAccess(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_ = processor.GetQueueStatus()
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestProcessor_GetResumeDiagnostics(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	diagnostics := processor.GetResumeDiagnostics()

	// ResumeDiagnostics is a struct, not a pointer
	// Verify it was returned (slices are initialized as empty, not nil)
	// Just check that the function executed without panic
	_ = diagnostics
}

func TestProcessor_RateLimitPause(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	t.Run("IsNotPausedInitially", func(t *testing.T) {
		paused, _ := processor.IsRateLimitPaused()
		if paused {
			t.Error("Processor should not be rate-limited initially")
		}
	})

	t.Run("BecomesRateLimitPaused", func(t *testing.T) {
		// Simulate a rate limit pause
		processor.handleRateLimitPause(300) // 5 minutes

		paused, pauseUntil := processor.IsRateLimitPaused()
		if !paused {
			t.Error("Processor should be rate-limited after handleRateLimitPause")
		}

		if pauseUntil.IsZero() {
			t.Error("Expected non-zero pause time")
		}

		if time.Until(pauseUntil) < 4*time.Minute {
			t.Error("Expected pause duration of at least 4 minutes")
		}
	})

	t.Run("ManualReset", func(t *testing.T) {
		processor.handleRateLimitPause(300)
		processor.ResetRateLimitPause()

		paused, _ := processor.IsRateLimitPaused()
		if paused {
			t.Error("Processor should not be rate-limited after reset")
		}
	})

	t.Run("CapsMaxPauseDuration", func(t *testing.T) {
		// Request a very long pause (5 hours = 18000 seconds)
		processor.handleRateLimitPause(18000)

		paused, pauseUntil := processor.IsRateLimitPaused()
		if !paused {
			t.Error("Processor should be rate-limited")
		}

		// Should be capped at 4 hours (14400 seconds)
		duration := time.Until(pauseUntil)
		if duration > 4*time.Hour+time.Minute {
			t.Errorf("Expected pause capped at 4 hours, got %v", duration)
		}
	})
}

func TestProcessor_ReconcileInProgressTasks(t *testing.T) {
	processor, tempDir, cleanup := setupTestProcessor(t)
	defer cleanup()

	queueDir := filepath.Join(tempDir, "queue")
	storage := tasks.NewStorage(queueDir)

	t.Run("MovesOrphanTasksToPending", func(t *testing.T) {
		// Create an orphan task (in-progress but not actually running)
		orphanTask := tasks.TaskItem{
			ID:        "orphan-task-1",
			Type:      "resource",
			Operation: "generator",
			Status:    "in-progress",
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		if err := storage.SaveQueueItem(orphanTask, "in-progress"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		// Call reconcile with empty active sets (no processes running)
		external := make(map[string]struct{})
		internal := make(map[string]struct{})
		moved := processor.reconcileInProgressTasks(external, internal)

		if len(moved) == 0 {
			t.Error("Expected orphan task to be moved")
		}

		// Verify task was moved to pending
		task, status, err := storage.GetTaskByID("orphan-task-1")
		if err != nil {
			t.Fatalf("GetTaskByID: %v", err)
		}

		if status != "pending" {
			t.Errorf("Expected status=pending, got %s", status)
		}

		if task.Status != "pending" {
			t.Errorf("Expected task.Status=pending, got %s", task.Status)
		}
	})

	t.Run("KeepsActiveTasksInProgress", func(t *testing.T) {
		// Create a legitimately running task
		activeTask := tasks.TaskItem{
			ID:        "active-task-1",
			Type:      "scenario",
			Operation: "improver",
			Status:    "in-progress",
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		if err := storage.SaveQueueItem(activeTask, "in-progress"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		// Mark as externally active
		external := map[string]struct{}{
			"active-task-1": {},
		}
		internal := make(map[string]struct{})

		moved := processor.reconcileInProgressTasks(external, internal)

		// Should not move active tasks
		if len(moved) != 0 {
			t.Errorf("Expected no tasks moved, got %d", len(moved))
		}

		// Verify task is still in-progress
		_, status, err := storage.GetTaskByID("active-task-1")
		if err != nil {
			t.Fatalf("GetTaskByID: %v", err)
		}

		if status != "in-progress" {
			t.Errorf("Expected status=in-progress, got %s", status)
		}
	})
}

func TestProcessor_ResetForResume(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	t.Run("ClearsRateLimitPause", func(t *testing.T) {
		processor.handleRateLimitPause(600) // 10 minutes

		summary := processor.ResetForResume()

		if !summary.RateLimitCleared {
			t.Error("Expected rate limit to be cleared")
		}

		if summary.ActionsTaken == 0 {
			t.Error("Expected ActionsTaken > 0")
		}

		paused, _ := processor.IsRateLimitPaused()
		if paused {
			t.Error("Processor should not be rate-limited after reset")
		}
	})

	t.Run("ProvidesDetailedSummary", func(t *testing.T) {
		processor.handleRateLimitPause(300)

		summary := processor.ResetForResume()

		// Summary should have useful information
		if summary.ActionsTaken == 0 {
			t.Error("Expected some actions taken")
		}

		// Should have notes or cleared flags
		hasInfo := summary.RateLimitCleared ||
			len(summary.AgentsStopped) > 0 ||
			len(summary.ProcessesTerminated) > 0 ||
			len(summary.TasksMovedToPending) > 0 ||
			len(summary.Notes) > 0

		if !hasInfo {
			t.Error("Expected summary to contain actionable information")
		}
	})
}

func TestProcessor_Pause(t *testing.T) {
	processor, _, cleanup := setupTestProcessor(t)
	defer cleanup()

	t.Run("PausesProcessor", func(t *testing.T) {
		processor.Pause()

		processor.mu.Lock()
		paused := processor.isPaused
		processor.mu.Unlock()

		if !paused {
			t.Error("Expected processor to be paused")
		}
	})

	t.Run("ResumeWithResetUnpauses", func(t *testing.T) {
		processor.Pause()

		summary := processor.ResumeWithReset()

		processor.mu.Lock()
		paused := processor.isPaused
		processor.mu.Unlock()

		if paused {
			t.Error("Expected processor to be unpaused after ResumeWithReset")
		}

		// Summary should be returned
		if summary.ActionsTaken < 0 {
			t.Error("Expected valid summary")
		}
	})
}
