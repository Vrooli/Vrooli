package queue

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/autosteer"
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

// appendManualSteeringSection attaches a manual steering mode when provided, otherwise falls back to Progress.
func (qp *Processor) appendManualSteeringSection(prompt string, steerMode autosteer.SteerMode) string {
	if qp.assembler == nil {
		return prompt
	}

	mode := steerMode
	if !mode.IsValid() {
		mode = autosteer.ModeProgress
	}

	// Prefer the orchestrator's cached prompt enhancer when available.
	if qp.autoSteerIntegration != nil {
		if orchestrator := qp.autoSteerIntegration.ExecutionOrchestrator(); orchestrator != nil {
			if section := strings.TrimSpace(orchestrator.GenerateModeSection(mode)); section != "" {
				return autosteer.InjectSteeringSection(prompt, section)
			}
		}
	}

	phasesDir := filepath.Join(qp.assembler.PromptsDir, "phases")
	section := strings.TrimSpace(autosteer.NewPromptEnhancer(phasesDir).GenerateModeSection(mode))
	if section == "" {
		return autosteer.InjectSteeringSection(prompt, "")
	}
	return autosteer.InjectSteeringSection(prompt, section)
}

// enrichSteeringMetadata captures the steering context used for an execution so we can analyze prompts later.
func (qp *Processor) enrichSteeringMetadata(task *tasks.TaskItem, history *ExecutionHistory) {
	if task == nil || history == nil {
		return
	}

	history.SteeringSource = "none"

	// Capture Auto Steer state when active.
	if qp.autoSteerIntegration != nil && strings.TrimSpace(task.AutoSteerProfileID) != "" {
		orchestrator := qp.autoSteerIntegration.ExecutionOrchestrator()
		if orchestrator != nil {
			if state, err := orchestrator.GetExecutionState(task.ID); err == nil && state != nil {
				history.AutoSteerProfileID = state.ProfileID
				history.AutoSteerIteration = state.AutoSteerIteration
				history.SteerPhaseIndex = state.CurrentPhaseIndex + 1
				history.SteerPhaseIteration = state.CurrentPhaseIteration + 1
				history.SteeringSource = "auto_steer"
			} else if err != nil {
				log.Printf("Warning: Failed to capture Auto Steer state for task %s: %v", task.ID, err)
			}

			if mode, err := orchestrator.GetCurrentMode(task.ID); err == nil {
				if mode != "" {
					history.SteerMode = string(mode)
					if history.SteeringSource == "none" {
						history.SteeringSource = "auto_steer"
					}
				}
			} else {
				log.Printf("Warning: Failed to capture Auto Steer mode for task %s: %v", task.ID, err)
			}
		}
	}

	// For improver tasks without Auto Steer we still inject steering guidance.
	if history.SteeringSource == "none" && task.Type == "scenario" && task.Operation == "improver" {
		mode := autosteer.SteerMode(strings.ToLower(strings.TrimSpace(task.SteerMode)))
		if mode.IsValid() {
			history.SteerMode = string(mode)
			history.SteeringSource = "manual_mode"
		} else {
			history.SteerMode = string(autosteer.ModeProgress)
			history.SteeringSource = "default_progress"
		}
		history.SteerPhaseIndex = 1
		history.SteerPhaseIteration = 1
	}
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
