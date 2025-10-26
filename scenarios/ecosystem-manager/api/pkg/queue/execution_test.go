package queue

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// setupExecutionTestProcessor creates a test processor with minimal dependencies
func setupExecutionTestProcessor(t *testing.T) (*Processor, *tasks.Storage, string) {
	tmpDir := t.TempDir()
	queueDir := filepath.Join(tmpDir, "queue")
	for _, dir := range []string{"pending", "in-progress", "completed", "failed"} {
		if err := os.MkdirAll(filepath.Join(queueDir, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create minimal prompts structure
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(filepath.Join(promptsDir, "sections"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create minimal sections.yaml
	sectionsYAML := `sections:
  - name: test
    content: "Test section"`
	if err := os.WriteFile(filepath.Join(promptsDir, "sections.yaml"), []byte(sectionsYAML), 0644); err != nil {
		t.Fatal(err)
	}

	storage := tasks.NewStorage(queueDir)
	assembler, err := prompts.NewAssembler(promptsDir)
	if err != nil {
		t.Fatal(err)
	}

	broadcast := make(chan interface{}, 100)
	processor := NewProcessor(30*time.Second, storage, assembler, broadcast)

	return processor, storage, tmpDir
}

// TestExecutionRegistryLifecycle verifies the complete lifecycle of execution registration
func TestExecutionRegistryLifecycle(t *testing.T) {
	processor, _, _ := setupExecutionTestProcessor(t)

	taskID := "test-task-123"
	agentTag := "test-agent"

	// Test 1: Reserve execution
	processor.reserveExecution(taskID, agentTag, time.Now())

	if !processor.IsTaskRunning(taskID) {
		t.Error("Expected task to be running after reservation")
	}

	execState, ok := processor.getExecution(taskID)
	if !ok {
		t.Error("Expected execution to be registered")
	}
	if execState.agentTag != agentTag {
		t.Errorf("Expected agent tag %q, got %q", agentTag, execState.agentTag)
	}

	// Test 2: Register execution with command
	cmd := exec.Command("sleep", "0.1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	processor.registerExecution(taskID, agentTag, cmd, time.Now())

	execState, ok = processor.getExecution(taskID)
	if !ok || execState.cmd == nil {
		t.Error("Expected execution to have command after registration")
	}

	// Wait for command to finish
	cmd.Wait()

	// Test 3: Verify task still shows as running (until unregistered)
	if !processor.IsTaskRunning(taskID) {
		t.Error("Task should still be running until explicitly unregistered")
	}

	// Test 4: Unregister execution
	processor.unregisterExecution(taskID)

	if processor.IsTaskRunning(taskID) {
		t.Error("Task should not be running after unregistration")
	}

	_, ok = processor.getExecution(taskID)
	if ok {
		t.Error("Execution should not exist after unregistration")
	}
}

// TestTaskCompletionRaceCondition tests that reconciliation doesn't interfere with completing tasks
func TestTaskCompletionRaceCondition(t *testing.T) {
	processor, storage, _ := setupExecutionTestProcessor(t)

	// Create a task in in-progress
	task := tasks.TaskItem{
		ID:     "test-race-task",
		Title:  "Test Race Condition",
		Type:   "scenario",
		Status: "in-progress",
	}
	if err := storage.SaveQueueItem(task, "in-progress"); err != nil {
		t.Fatal(err)
	}

	// Register execution (simulating task is running)
	processor.reserveExecution(task.ID, "test-agent", time.Now())

	// Verify task is recognized as running
	if !processor.IsTaskRunning(task.ID) {
		t.Error("Task should be running")
	}

	// Simulate the critical window: task completes but hasn't finalized status yet
	// This is the window where the bug would occur

	var wg sync.WaitGroup
	raceCaught := false
	var mu sync.Mutex

	// Goroutine 1: Reconciliation (runs periodically)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Small delay to let completion start

		external := processor.getExternalActiveTaskIDs()
		internal := processor.getInternalRunningTaskIDs()
		moved := processor.reconcileInProgressTasks(external, internal)

		mu.Lock()
		if len(moved) > 0 && moved[0] == task.ID {
			raceCaught = true
		}
		mu.Unlock()
	}()

	// Goroutine 2: Task completion (unregisters then finalizes)
	wg.Add(1)
	go func() {
		defer wg.Done()

		// OLD BUGGY CODE would do:
		// processor.unregisterExecution(task.ID)  // <-- unregister FIRST
		// time.Sleep(20 * time.Millisecond)       // <-- window for race
		// processor.finalizeTaskStatus(&task, "completed")

		// NEW FIXED CODE does:
		processor.finalizeTaskStatus(&task, "completed") // finalize FIRST
		time.Sleep(20 * time.Millisecond)                // ensure overlap
		processor.unregisterExecution(task.ID)           // unregister AFTER
	}()

	wg.Wait()

	// Verify: reconciliation should NOT have moved the task
	mu.Lock()
	caught := raceCaught
	mu.Unlock()

	if caught {
		t.Error("RACE CONDITION DETECTED: Reconciliation moved task during completion")
	}

	// Verify final state: task should be in completed
	finalStatus, err := storage.CurrentStatus(task.ID)
	if err != nil {
		t.Fatalf("Failed to get final status: %v", err)
	}
	if finalStatus != "completed" {
		t.Errorf("Expected task in 'completed', got %q", finalStatus)
	}
}

// TestInternalRunningTasksAccuracy verifies getInternalRunningTaskIDs returns accurate state
func TestInternalRunningTasksAccuracy(t *testing.T) {
	processor, _, _ := setupExecutionTestProcessor(t)

	// Initially empty
	running := processor.getInternalRunningTaskIDs()
	if len(running) != 0 {
		t.Errorf("Expected 0 running tasks, got %d", len(running))
	}

	// Add multiple tasks
	tasks := []string{"task-1", "task-2", "task-3"}
	for _, tid := range tasks {
		processor.reserveExecution(tid, "agent-"+tid, time.Now())
	}

	running = processor.getInternalRunningTaskIDs()
	if len(running) != len(tasks) {
		t.Errorf("Expected %d running tasks, got %d", len(tasks), len(running))
	}

	for _, tid := range tasks {
		if _, exists := running[tid]; !exists {
			t.Errorf("Expected task %s to be running", tid)
		}
	}

	// Remove one task
	processor.unregisterExecution("task-2")

	running = processor.getInternalRunningTaskIDs()
	if len(running) != 2 {
		t.Errorf("Expected 2 running tasks after removal, got %d", len(running))
	}
	if _, exists := running["task-2"]; exists {
		t.Error("task-2 should not be running after unregistration")
	}
}

// TestExecutionLifecycle verifies execution registry lifecycle
func TestExecutionLifecycle(t *testing.T) {
	processor, _, _ := setupExecutionTestProcessor(t)

	taskID := "lifecycle-test-task"

	// Start a real process
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Register execution
	processor.registerExecution(taskID, "test-agent", cmd, time.Now())

	// Simulate coordinated wait (what execution.go does now)
	waitChan := make(chan error, 1)
	go func() {
		waitChan <- cmd.Wait()
		close(waitChan)
	}()

	// Verify processor reports task as running
	if !processor.IsTaskRunning(taskID) {
		t.Error("Processor should report task as running")
	}

	// Wait for process to complete
	<-waitChan

	// Even after process completes, execution registry keeps it until explicitly unregistered
	if !processor.IsTaskRunning(taskID) {
		t.Error("Task should still be in registry until unregistered")
	}

	// Now unregister from executions
	processor.unregisterExecution(taskID)

	if processor.IsTaskRunning(taskID) {
		t.Error("Task should not be running after unregistration")
	}
}

// TestConcurrentExecutionRegistration tests thread-safety of execution registry
func TestConcurrentExecutionRegistration(t *testing.T) {
	processor, _, _ := setupExecutionTestProcessor(t)

	const numTasks = 50
	var wg sync.WaitGroup

	// Concurrently register tasks
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("concurrent-task-%d", id)
			processor.reserveExecution(taskID, "agent", time.Now())
		}(i)
	}

	wg.Wait()

	// Verify no data corruption
	running := processor.getInternalRunningTaskIDs()
	if len(running) != numTasks {
		t.Errorf("Expected %d tasks, got %d (possible race condition)", numTasks, len(running))
	}

	// Concurrently unregister
	for taskID := range running {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			processor.unregisterExecution(id)
		}(taskID)
	}

	wg.Wait()

	// Verify all removed
	running = processor.getInternalRunningTaskIDs()
	if len(running) != 0 {
		t.Errorf("Expected 0 tasks after cleanup, got %d", len(running))
	}
}

// TestReserveExecutionIdempotency tests that reserving an already-reserved execution is safe
func TestReserveExecutionIdempotency(t *testing.T) {
	processor, _, _ := setupExecutionTestProcessor(t)

	taskID := "idempotent-task"
	startTime := time.Now()

	// Reserve once
	processor.reserveExecution(taskID, "agent-1", startTime)

	execState1, _ := processor.getExecution(taskID)

	// Reserve again with different agent (simulating retry)
	processor.reserveExecution(taskID, "agent-2", time.Time{})

	execState2, _ := processor.getExecution(taskID)

	// Agent should be updated, but start time preserved
	if execState2.agentTag != "agent-2" {
		t.Errorf("Expected agent tag to update to 'agent-2', got %q", execState2.agentTag)
	}

	if execState2.started != execState1.started {
		t.Error("Start time should be preserved on re-reservation")
	}

	if execState2.started != startTime {
		t.Error("Original start time should be preserved")
	}
}
