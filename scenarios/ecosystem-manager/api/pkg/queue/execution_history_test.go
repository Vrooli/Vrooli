package queue

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// setupTestExecutionProcessor creates a test processor with temporary directories for execution history tests
func setupTestExecutionProcessor(t *testing.T) (*Processor, func()) {
	t.Helper()
	resetTimings := SetTimingScaleForTests(0.01)
	t.Cleanup(resetTimings)

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "execution-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	queueDir := filepath.Join(tmpDir, "queue")
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		t.Fatalf("Failed to create queue dir: %v", err)
	}

	storage := tasks.NewStorage(queueDir)
	assembler := &prompts.Assembler{} // Mock assembler
	broadcast := make(chan any, 10)

	processor := NewProcessor(storage, assembler, broadcast, nil)

	cleanup := func() {
		processor.Stop()
		os.RemoveAll(tmpDir)
		close(broadcast)
	}

	return processor, cleanup
}

// TestSavePromptToHistory verifies that prompts are saved correctly
func TestSavePromptToHistory(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	taskID := "test-task-001"
	executionID := "2025-10-27_120000"
	promptContent := "This is a test prompt for execution history"

	// Save prompt
	relPath, err := processor.savePromptToHistory(taskID, executionID, promptContent)
	if err != nil {
		t.Fatalf("savePromptToHistory failed: %v", err)
	}

	// Verify relative path format
	expectedRelPath := filepath.Join(taskID, "executions", executionID, "prompt.txt")
	if relPath != expectedRelPath {
		t.Errorf("Expected relative path %q, got %q", expectedRelPath, relPath)
	}

	// Verify file exists and contains correct content
	fullPath := filepath.Join(processor.taskLogsDir, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read saved prompt: %v", err)
	}

	if string(content) != promptContent {
		t.Errorf("Expected prompt content %q, got %q", promptContent, string(content))
	}

	// Verify directory structure
	execDir := filepath.Join(processor.taskLogsDir, taskID, "executions", executionID)
	if _, err := os.Stat(execDir); os.IsNotExist(err) {
		t.Errorf("Execution directory %q was not created", execDir)
	}

	t.Logf("Successfully saved prompt to %s", fullPath)
}

// TestSaveOutputToHistory verifies that output logs are saved correctly
func TestSaveOutputToHistory(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	taskID := "test-task-002"
	executionID := "2025-10-27_120001"

	// Create a fake log file in the old location (simulating finalizeTaskLogs)
	oldLogPath := filepath.Join(processor.taskLogsDir, taskID+".log")
	if err := os.MkdirAll(filepath.Dir(oldLogPath), 0755); err != nil {
		t.Fatalf("Failed to create log dir: %v", err)
	}
	logContent := "Task execution output\nLine 2\nLine 3"
	if err := os.WriteFile(oldLogPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to create fake log: %v", err)
	}

	// Save output to history
	relPath, err := processor.saveOutputToHistory(taskID, executionID)
	if err != nil {
		t.Fatalf("saveOutputToHistory failed: %v", err)
	}

	// Verify relative path format
	expectedRelPath := filepath.Join(taskID, "executions", executionID, "output.log")
	if relPath != expectedRelPath {
		t.Errorf("Expected relative path %q, got %q", expectedRelPath, relPath)
	}

	// Verify output was moved to history location
	newLogPath := filepath.Join(processor.taskLogsDir, relPath)
	content, err := os.ReadFile(newLogPath)
	if err != nil {
		t.Fatalf("Failed to read moved output: %v", err)
	}

	if string(content) != logContent {
		t.Errorf("Expected output content %q, got %q", logContent, string(content))
	}

	// Verify old log was removed
	if _, err := os.Stat(oldLogPath); !os.IsNotExist(err) {
		t.Errorf("Old log file %q should have been removed", oldLogPath)
	}

	t.Logf("Successfully moved output log to %s", newLogPath)
}

// TestSaveExecutionMetadata verifies that metadata is saved correctly
func TestSaveExecutionMetadata(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	history := ExecutionHistory{
		TaskID:         "test-task-003",
		ExecutionID:    "2025-10-27_120002",
		AgentTag:       "test-agent-123",
		ProcessID:      12345,
		StartTime:      time.Now().Add(-5 * time.Minute),
		EndTime:        time.Now(),
		Duration:       "5m0s",
		Success:        true,
		ExitReason:     "completed",
		PromptSize:     25000,
		PromptPath:     "test-task-003/executions/2025-10-27_120002/prompt.txt",
		OutputPath:     "test-task-003/executions/2025-10-27_120002/output.log",
		TimeoutAllowed: "30m0s",
		RateLimited:    false,
		RetryAfter:     0,
	}

	// Save metadata
	err := processor.saveExecutionMetadata(history)
	if err != nil {
		t.Fatalf("saveExecutionMetadata failed: %v", err)
	}

	// Verify metadata file exists
	metadataPath := filepath.Join(processor.taskLogsDir, history.TaskID, "executions", history.ExecutionID, "metadata.json")
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	// Verify JSON is valid and contains expected data
	var loaded ExecutionHistory
	if err := json.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("Failed to parse metadata JSON: %v", err)
	}

	// Verify key fields
	if loaded.TaskID != history.TaskID {
		t.Errorf("Expected TaskID %q, got %q", history.TaskID, loaded.TaskID)
	}
	if loaded.ExecutionID != history.ExecutionID {
		t.Errorf("Expected ExecutionID %q, got %q", history.ExecutionID, loaded.ExecutionID)
	}
	if loaded.Success != history.Success {
		t.Errorf("Expected Success %v, got %v", history.Success, loaded.Success)
	}
	if loaded.ExitReason != history.ExitReason {
		t.Errorf("Expected ExitReason %q, got %q", history.ExitReason, loaded.ExitReason)
	}

	t.Logf("Successfully saved and verified metadata at %s", metadataPath)
}

// TestLoadExecutionHistory verifies loading history for a single task
func TestLoadExecutionHistory(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	taskID := "test-task-004"

	// Create multiple execution histories
	executions := []ExecutionHistory{
		{
			TaskID:      taskID,
			ExecutionID: "2025-10-27_100000",
			Success:     true,
			ExitReason:  "completed",
			StartTime:   time.Now().Add(-2 * time.Hour),
			EndTime:     time.Now().Add(-2*time.Hour + 5*time.Minute),
		},
		{
			TaskID:      taskID,
			ExecutionID: "2025-10-27_110000",
			Success:     false,
			ExitReason:  "failed",
			StartTime:   time.Now().Add(-1 * time.Hour),
			EndTime:     time.Now().Add(-1*time.Hour + 3*time.Minute),
		},
		{
			TaskID:      taskID,
			ExecutionID: "2025-10-27_120000",
			Success:     true,
			ExitReason:  "completed",
			StartTime:   time.Now().Add(-30 * time.Minute),
			EndTime:     time.Now().Add(-25 * time.Minute),
		},
	}

	// Save all executions
	for _, exec := range executions {
		if err := processor.saveExecutionMetadata(exec); err != nil {
			t.Fatalf("Failed to save execution %s: %v", exec.ExecutionID, err)
		}
	}

	// Load execution history
	history, err := processor.LoadExecutionHistory(taskID)
	if err != nil {
		t.Fatalf("LoadExecutionHistory failed: %v", err)
	}

	// Verify count
	if len(history) != len(executions) {
		t.Errorf("Expected %d executions, got %d", len(executions), len(history))
	}

	// Verify all executions are present
	foundIDs := make(map[string]bool)
	for _, h := range history {
		foundIDs[h.ExecutionID] = true
		if h.TaskID != taskID {
			t.Errorf("Expected TaskID %q, got %q", taskID, h.TaskID)
		}
	}

	for _, exec := range executions {
		if !foundIDs[exec.ExecutionID] {
			t.Errorf("Execution %s was not loaded", exec.ExecutionID)
		}
	}

	t.Logf("Successfully loaded %d executions for task %s", len(history), taskID)
}

// TestLoadAllExecutionHistory verifies loading history for all tasks
func TestLoadAllExecutionHistory(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	// Create executions for multiple tasks
	tasks := []struct {
		taskID      string
		executionID string
		success     bool
	}{
		{"task-001", "2025-10-27_100000", true},
		{"task-001", "2025-10-27_110000", false},
		{"task-002", "2025-10-27_120000", true},
		{"task-003", "2025-10-27_130000", true},
	}

	startTime := time.Now().Add(-4 * time.Hour)
	for i, tc := range tasks {
		exec := ExecutionHistory{
			TaskID:      tc.taskID,
			ExecutionID: tc.executionID,
			Success:     tc.success,
			ExitReason:  map[bool]string{true: "completed", false: "failed"}[tc.success],
			StartTime:   startTime.Add(time.Duration(i) * time.Hour),
			EndTime:     startTime.Add(time.Duration(i)*time.Hour + 5*time.Minute),
		}
		if err := processor.saveExecutionMetadata(exec); err != nil {
			t.Fatalf("Failed to save execution %s/%s: %v", tc.taskID, tc.executionID, err)
		}
	}

	// Load all execution history
	allHistory, err := processor.LoadAllExecutionHistory()
	if err != nil {
		t.Fatalf("LoadAllExecutionHistory failed: %v", err)
	}

	// Verify count
	if len(allHistory) != len(tasks) {
		t.Errorf("Expected %d total executions, got %d", len(tasks), len(allHistory))
	}

	// Verify executions are sorted by start time (most recent first)
	for i := 1; i < len(allHistory); i++ {
		if allHistory[i].StartTime.After(allHistory[i-1].StartTime) {
			t.Errorf("Execution history not sorted correctly: index %d is newer than %d", i, i-1)
		}
	}

	// Verify all tasks are represented
	taskCounts := make(map[string]int)
	for _, h := range allHistory {
		taskCounts[h.TaskID]++
	}

	if taskCounts["task-001"] != 2 {
		t.Errorf("Expected 2 executions for task-001, got %d", taskCounts["task-001"])
	}
	if taskCounts["task-002"] != 1 {
		t.Errorf("Expected 1 execution for task-002, got %d", taskCounts["task-002"])
	}
	if taskCounts["task-003"] != 1 {
		t.Errorf("Expected 1 execution for task-003, got %d", taskCounts["task-003"])
	}

	t.Logf("Successfully loaded %d executions across %d tasks", len(allHistory), len(taskCounts))
}

// TestLoadExecutionHistoryNoHistory verifies behavior when no history exists
func TestLoadExecutionHistoryNoHistory(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	// Load history for non-existent task
	history, err := processor.LoadExecutionHistory("non-existent-task")
	if err != nil {
		t.Fatalf("LoadExecutionHistory should not error on missing history: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d executions", len(history))
	}

	t.Logf("Correctly returned empty history for non-existent task")
}

// TestCleanupOldExecutions verifies that old executions are cleaned up
func TestCleanupOldExecutions(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	taskID := "test-task-cleanup"

	// Create executions with different ages
	oldExecution := ExecutionHistory{
		TaskID:      taskID,
		ExecutionID: "2025-10-01_100000",
		Success:     true,
		ExitReason:  "completed",
		StartTime:   time.Now().AddDate(0, 0, -40), // 40 days ago
		EndTime:     time.Now().AddDate(0, 0, -40),
	}

	recentExecution := ExecutionHistory{
		TaskID:      taskID,
		ExecutionID: "2025-10-27_100000",
		Success:     true,
		ExitReason:  "completed",
		StartTime:   time.Now().Add(-1 * time.Hour),
		EndTime:     time.Now(),
	}

	// Save both executions
	if err := processor.saveExecutionMetadata(oldExecution); err != nil {
		t.Fatalf("Failed to save old execution: %v", err)
	}
	if err := processor.saveExecutionMetadata(recentExecution); err != nil {
		t.Fatalf("Failed to save recent execution: %v", err)
	}

	// Verify both exist
	history, err := processor.LoadExecutionHistory(taskID)
	if err != nil {
		t.Fatalf("Failed to load initial history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("Expected 2 executions before cleanup, got %d", len(history))
	}

	// Clean up executions older than 30 days
	err = processor.CleanupOldExecutions(taskID, 30)
	if err != nil {
		t.Fatalf("CleanupOldExecutions failed: %v", err)
	}

	// Verify old execution was removed
	history, err = processor.LoadExecutionHistory(taskID)
	if err != nil {
		t.Fatalf("Failed to load history after cleanup: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 execution after cleanup, got %d", len(history))
	}

	if len(history) > 0 && history[0].ExecutionID != recentExecution.ExecutionID {
		t.Errorf("Expected recent execution %q to remain, got %q", recentExecution.ExecutionID, history[0].ExecutionID)
	}

	t.Logf("Successfully cleaned up old executions, kept recent execution")
}

// TestGetExecutionFilePath verifies path construction
func TestGetExecutionFilePath(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	taskID := "test-task-005"
	executionID := "2025-10-27_120000"
	filename := "prompt.txt"

	path := processor.GetExecutionFilePath(taskID, executionID, filename)

	expectedPath := filepath.Join(processor.taskLogsDir, taskID, "executions", executionID, filename)
	if path != expectedPath {
		t.Errorf("Expected path %q, got %q", expectedPath, path)
	}

	t.Logf("GetExecutionFilePath correctly constructed: %s", path)
}

func TestLatestExecutionOutputPathPrefersClean(t *testing.T) {
	processor, cleanup := setupTestExecutionProcessor(t)
	defer cleanup()

	taskID := "test-task-006"
	base := processor.taskLogsDir

	newer := ExecutionHistory{
		TaskID:          taskID,
		ExecutionID:     "2025-10-27_130000",
		StartTime:       time.Now(),
		CleanOutputPath: filepath.Join(taskID, "executions", "2025-10-27_130000", "clean_output.txt"),
		OutputPath:      filepath.Join(taskID, "executions", "2025-10-27_130000", "output.log"),
	}
	older := ExecutionHistory{
		TaskID:      taskID,
		ExecutionID: "2025-10-27_120000",
		StartTime:   time.Now().Add(-1 * time.Hour),
		OutputPath:  filepath.Join(taskID, "executions", "2025-10-27_120000", "output.log"),
	}

	if err := processor.saveExecutionMetadata(newer); err != nil {
		t.Fatalf("failed to persist newer execution: %v", err)
	}
	if err := processor.saveExecutionMetadata(older); err != nil {
		t.Fatalf("failed to persist older execution: %v", err)
	}

	latest := processor.LatestExecutionOutputPath(taskID)
	expected := filepath.Join(base, newer.CleanOutputPath)
	if latest != expected {
		t.Fatalf("expected latest output path %q, got %q", expected, latest)
	}

	// Remove clean path to ensure fallback to output.log
	newer.CleanOutputPath = ""
	if err := processor.saveExecutionMetadata(newer); err != nil {
		t.Fatalf("failed to update newer execution: %v", err)
	}
	latest = processor.LatestExecutionOutputPath(taskID)
	fallback := filepath.Join(base, newer.OutputPath)
	if latest != fallback {
		t.Fatalf("expected fallback output path %q, got %q", fallback, latest)
	}
}
