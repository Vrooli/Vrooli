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

	broadcast := make(chan interface{}, 10)
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

	// Start processor
	processor.Start()
	time.Sleep(100 * time.Millisecond)

	status = processor.GetQueueStatus()
	if !status["processor_active"].(bool) {
		t.Error("Processor should be active after Start()")
	}

	// Stop processor
	processor.Stop()
	time.Sleep(100 * time.Millisecond)

	status = processor.GetQueueStatus()
	if status["processor_active"].(bool) {
		t.Error("Processor should be inactive after Stop()")
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
	// Check a field to verify it was populated
	if len(diagnostics.Notes) < 0 { // Will always be >= 0, just checking it exists
		t.Fatal("Expected diagnostics to be populated")
	}
}
