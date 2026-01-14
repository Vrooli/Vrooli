package queue

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// cleanupTaskWithVerifiedAgentRemoval is the unified cleanup handler used after task finalization.
// With agent-manager integration, cleanup is simplified: the cleanup function calls agentSvc.StopRun()
// which handles all termination logic. We just call cleanup and unregister.
func (qp *Processor) cleanupTaskWithVerifiedAgentRemoval(taskID string, cleanupFunc func(), ctx string, keepTracking bool) {
	if ctx != "" {
		log.Printf("Task %s cleanup (%s)", taskID, ctx)
	}

	// Call the cleanup function (calls agentSvc.StopRun via closure)
	cleanupFunc()

	// Unregister from execution tracking
	qp.unregisterExecution(taskID)
}

// completeTaskCleanupExplicit handles cleanup after successful finalization
func (qp *Processor) completeTaskCleanupExplicit(taskID string, cleanupFunc func()) {
	qp.cleanupTaskWithVerifiedAgentRemoval(taskID, cleanupFunc, "task finalization success", true)
}

// cleanupAgentAfterFinalizationFailure handles cleanup when finalization fails
func (qp *Processor) cleanupAgentAfterFinalizationFailure(taskID string, cleanupFunc func(), context string) {
	qp.cleanupTaskWithVerifiedAgentRemoval(taskID, cleanupFunc, context, true)
}

// executeTask executes a single task by delegating to ExecutionManager.
// This is a thin wrapper that handles Processor-specific concerns like rate limiting.
func (qp *Processor) executeTask(task tasks.TaskItem) {
	if qp.executionManager == nil {
		log.Printf("CRITICAL: ExecutionManager not available for task %s", task.ID)
		systemlog.Errorf("ExecutionManager not initialized - cannot execute task %s", task.ID)
		return
	}

	result, err := qp.executionManager.ExecuteTask(qp.ctx, task)
	if err != nil {
		log.Printf("Task %s execution error: %v", task.ID, err)
	}

	// Handle rate limiting at Processor level (scheduling concern)
	if result != nil && result.RateLimited && result.RetryAfter > 0 {
		qp.handleRateLimitPause(result.RetryAfter)
	}
}

// detectMaxTurnsExceeded checks if the output indicates MAX_TURNS was exceeded.
// This is a shared utility used by both Processor and ExecutionManager.

func detectMaxTurnsExceeded(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "max turns") && strings.Contains(lower, "reached")
}

// fileExistsAndNotEmpty checks if a file exists and has content.
// This is a shared utility used by ExecutionManager.
func fileExistsAndNotEmpty(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// isValidClaudeConfig checks if a byte slice contains valid Claude config JSON.
// This is a shared utility used by ExecutionManager.
func isValidClaudeConfig(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}

	var parsed map[string]any
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return false
	}

	return true
}

// broadcastUpdate sends updates to all connected WebSocket clients
func (qp *Processor) broadcastUpdate(updateType string, data any) {
	// Send the typed update directly, not wrapped in another object
	// The WebSocket manager will wrap it properly
	select {
	case qp.broadcast <- map[string]any{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}:
		log.Printf("Broadcast %s update for task", updateType)
	default:
		log.Printf("Warning: WebSocket broadcast channel full, dropping update")
	}
}

// getScenarioNameFromTask extracts the scenario name from a task
// For scenario-related tasks, returns the target scenario name
func getScenarioNameFromTask(task *tasks.TaskItem) string {
	if task.Type == "scenario" && task.Target != "" {
		return task.Target
	}
	// For multiple targets, use the first one
	if len(task.Targets) > 0 {
		return task.Targets[0]
	}
	// Fallback: try to extract from title or use task ID
	return task.Target
}

// GetScenarioNameFromTask is an exported helper to ensure consistent scenario name derivation.
func GetScenarioNameFromTask(task *tasks.TaskItem) string {
	return getScenarioNameFromTask(task)
}
