// Package tasks provides a unified task orchestration system for agent-based
// pipeline investigation and fix workflows.
//
// This package implements the "screaming architecture" pattern where the folder
// and file names clearly communicate the system's purpose:
//   - tasks/investigate/: Investigation task handler
//   - tasks/fix/: Fix task handler with iterative loop
//   - tasks/shared/: Reusable context attachment builders and shared types
package tasks

import (
	"context"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/tasks/shared"
)

// Re-export shared types for convenience
type (
	TaskInput    = shared.TaskInput
	PromptResult = shared.PromptResult
	AgentResult  = shared.AgentResult
)

// TaskHandler defines the contract for task-type-specific handlers.
// Implementations handle the full lifecycle of a task type.
type TaskHandler interface {
	// TaskType returns the type this handler processes.
	TaskType() domain.TaskType

	// BuildPromptAndContext creates the prompt and attachments for agent execution.
	// This should be a pure function for easy testing.
	BuildPromptAndContext(ctx context.Context, input TaskInput) (PromptResult, error)

	// AgentTag returns the tag to use for agent-manager runs.
	AgentTag() string

	// ShouldContinue determines if an iterative task should continue.
	// Returns (continue, reason). For non-iterative tasks, always returns (false, "").
	ShouldContinue(ctx context.Context, task *domain.Investigation, result *AgentResult) (bool, string)
}

// ProgressBroadcaster defines the interface for broadcasting progress events.
type ProgressBroadcaster interface {
	BroadcastInvestigation(pipelineID, invID, eventType string, progress float64, message string)
}

// ProgressEvent types for task execution.
const (
	// Common events
	EventTaskStarted   = "task_started"
	EventTaskProgress  = "task_progress"
	EventTaskCompleted = "task_completed"
	EventTaskFailed    = "task_failed"
	EventTaskCancelled = "task_cancelled"
	EventTaskStopped   = "task_stopped"

	// Investigation-specific events
	EventInvestigationStarted   = "investigation_started"
	EventInvestigationProgress  = "investigation_progress"
	EventInvestigationCompleted = "investigation_completed"
	EventInvestigationFailed    = "investigation_failed"

	// Fix-specific events
	EventFixStarted           = "fix_started"
	EventFixProgress          = "fix_progress"
	EventFixIterationStarted  = "fix_iteration_started"
	EventFixDiagnosis         = "fix_diagnosis"
	EventFixChangesApplied    = "fix_changes_applied"
	EventFixRebuildTriggered  = "fix_rebuild_triggered"
	EventFixVerifyStarted     = "fix_verify_started"
	EventFixVerifyResult      = "fix_verify_result"
	EventFixIterationComplete = "fix_iteration_completed"
	EventFixLoopCompleted     = "fix_loop_completed"
	EventFixCompleted         = "fix_completed"
	EventFixFailed            = "fix_failed"
)

// Re-export fix loop termination statuses from shared.
const (
	FixStatusSuccess       = shared.FixStatusSuccess
	FixStatusMaxIterations = shared.FixStatusMaxIterations
	FixStatusAgentGaveUp   = shared.FixStatusAgentGaveUp
	FixStatusUserStopped   = shared.FixStatusUserStopped
	FixStatusTimeout       = shared.FixStatusTimeout
)

// HandlerRegistry holds registered task handlers.
type HandlerRegistry struct {
	handlers map[domain.TaskType]TaskHandler
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[domain.TaskType]TaskHandler),
	}
}

// Register adds a handler to the registry.
func (r *HandlerRegistry) Register(h TaskHandler) {
	r.handlers[h.TaskType()] = h
}

// Get retrieves a handler by task type.
func (r *HandlerRegistry) Get(taskType domain.TaskType) (TaskHandler, bool) {
	h, ok := r.handlers[taskType]
	return h, ok
}
