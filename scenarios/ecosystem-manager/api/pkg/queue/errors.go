package queue

import "fmt"

// FinalizationError represents an error that occurred during task finalization
// This helps distinguish finalization failures from other types of errors
type FinalizationError struct {
	TaskID string // ID of the task that failed to finalize
	Phase  string // Phase where finalization failed: "success", "failure", "rate_limit", "timeout"
	Err    error  // Underlying error
}

func (e *FinalizationError) Error() string {
	return fmt.Sprintf("failed to finalize task %s during %s phase: %v", e.TaskID, e.Phase, e.Err)
}

func (e *FinalizationError) Unwrap() error {
	return e.Err
}

// NewFinalizationError creates a new finalization error
func NewFinalizationError(taskID, phase string, err error) *FinalizationError {
	return &FinalizationError{
		TaskID: taskID,
		Phase:  phase,
		Err:    err,
	}
}

// AgentTerminationError represents an error during agent process termination
type AgentTerminationError struct {
	TaskID   string // ID of the task whose agent failed to terminate
	AgentTag string // Agent tag identifier
	PID      int    // Process ID that failed to terminate
	Err      error  // Underlying error
}

func (e *AgentTerminationError) Error() string {
	return fmt.Sprintf("failed to terminate agent %s (pid %d) for task %s: %v", e.AgentTag, e.PID, e.TaskID, e.Err)
}

func (e *AgentTerminationError) Unwrap() error {
	return e.Err
}

// NewAgentTerminationError creates a new agent termination error
func NewAgentTerminationError(taskID, agentTag string, pid int, err error) *AgentTerminationError {
	return &AgentTerminationError{
		TaskID:   taskID,
		AgentTag: agentTag,
		PID:      pid,
		Err:      err,
	}
}

// ExecutionRegistryError represents an error in the execution registry
type ExecutionRegistryError struct {
	TaskID    string // ID of the task
	Operation string // Operation that failed: "register", "unregister", "lookup"
	Err       error  // Underlying error
}

func (e *ExecutionRegistryError) Error() string {
	return fmt.Sprintf("execution registry %s failed for task %s: %v", e.Operation, e.TaskID, e.Err)
}

func (e *ExecutionRegistryError) Unwrap() error {
	return e.Err
}

// NewExecutionRegistryError creates a new execution registry error
func NewExecutionRegistryError(taskID, operation string, err error) *ExecutionRegistryError {
	return &ExecutionRegistryError{
		TaskID:    taskID,
		Operation: operation,
		Err:       err,
	}
}
