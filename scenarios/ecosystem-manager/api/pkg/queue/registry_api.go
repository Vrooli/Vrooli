package queue

import (
	"time"
)

// ExecutionRegistryAPI defines the interface for tracking running task executions.
// This abstraction enables unit testing without managing real processes.
type ExecutionRegistryAPI interface {
	// ReserveExecution creates a placeholder execution entry for a task about to start.
	// This prevents race conditions with reconciliation logic.
	ReserveExecution(taskID, agentID string, startedAt time.Time)

	// RegisterRunID associates an agent-manager run ID with a task execution.
	RegisterRunID(taskID, runID string)

	// UnregisterExecution removes a task from the execution registry.
	UnregisterExecution(taskID string)

	// GetExecution retrieves the execution state for a task.
	GetExecution(taskID string) (*taskExecution, bool)

	// IsTaskRunning returns true if the task is currently tracked in executions.
	IsTaskRunning(taskID string) bool

	// ListRunningTaskIDs returns task IDs of all currently running processes.
	ListRunningTaskIDs() []string

	// ListRunningTaskIDsAsMap returns task IDs as a map for O(1) lookup.
	ListRunningTaskIDsAsMap() map[string]struct{}

	// GetRunIDForTask returns the agent-manager run ID for a task, empty if not found.
	GetRunIDForTask(taskID string) string

	// Count returns the number of registered executions.
	Count() int

	// GetAllExecutions returns a snapshot of all current executions for iteration.
	GetAllExecutions() []taskExecutionSnapshot

	// GetTimedOutExecutions returns snapshots of executions that have exceeded their timeout.
	GetTimedOutExecutions() []taskExecutionSnapshot

	// MarkTimedOut marks an execution as timed out (for watchdog use).
	MarkTimedOut(taskID string)

	// Clear removes all executions from the registry.
	Clear() int
}

// Compile-time assertion that ExecutionRegistry implements ExecutionRegistryAPI.
var _ ExecutionRegistryAPI = (*ExecutionRegistry)(nil)
