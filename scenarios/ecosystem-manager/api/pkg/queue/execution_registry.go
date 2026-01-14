package queue

import (
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
)

// ExecutionRegistry manages the tracking of running task executions.
// It provides thread-safe operations for reserving, registering, and
// unregistering task executions with timeout tracking.
type ExecutionRegistry struct {
	executions   map[string]*taskExecution
	executionsMu sync.RWMutex
}

// NewExecutionRegistry creates a new ExecutionRegistry
func NewExecutionRegistry() *ExecutionRegistry {
	return &ExecutionRegistry{
		executions: make(map[string]*taskExecution),
	}
}

// ReserveExecution creates a placeholder execution entry for a task that's about to start.
// This should be called before the actual execution begins to prevent race conditions
// with reconciliation logic.
func (r *ExecutionRegistry) ReserveExecution(taskID, agentID string, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	// Get current timeout setting
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	timeoutAt := startedAt.Add(timeoutDuration)

	r.executionsMu.Lock()
	defer r.executionsMu.Unlock()

	if existing, ok := r.executions[taskID]; ok {
		if agentID != "" {
			existing.agentTag = agentID
		}
		if existing.started.IsZero() {
			existing.started = startedAt
			existing.timeoutAt = timeoutAt
		}
	} else {
		r.executions[taskID] = &taskExecution{
			taskID:    taskID,
			agentTag:  agentID,
			started:   startedAt,
			timeoutAt: timeoutAt,
		}
	}
}

// RegisterExecution registers a running process for a task.
// This updates an existing reservation or creates a new entry with the given process.
func (r *ExecutionRegistry) RegisterExecution(taskID, agentID string, cmd *exec.Cmd, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	// Get current timeout setting
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	timeoutAt := startedAt.Add(timeoutDuration)

	r.executionsMu.Lock()
	defer r.executionsMu.Unlock()

	execState, exists := r.executions[taskID]
	if !exists {
		execState = &taskExecution{taskID: taskID}
		r.executions[taskID] = execState
	}
	if agentID != "" {
		execState.agentTag = agentID
	}
	execState.cmd = cmd
	if execState.started.IsZero() {
		execState.started = startedAt
		execState.timeoutAt = timeoutAt
	}

	// Defensive nil check for logging
	if cmd != nil && cmd.Process != nil {
		log.Printf("Registered execution %d for task %s (timeout at %s)", cmd.Process.Pid, taskID, timeoutAt.Format(time.RFC3339))
	} else {
		log.Printf("Registered execution record for task %s (pid unknown, timeout at %s)", taskID, timeoutAt.Format(time.RFC3339))
	}
}

// RegisterRunID associates an agent-manager run ID with a task execution
func (r *ExecutionRegistry) RegisterRunID(taskID, runID string) {
	r.executionsMu.Lock()
	defer r.executionsMu.Unlock()

	if execState, exists := r.executions[taskID]; exists {
		execState.runID = runID
	}
}

// UnregisterExecution removes a task from the execution registry
func (r *ExecutionRegistry) UnregisterExecution(taskID string) {
	r.executionsMu.Lock()
	defer r.executionsMu.Unlock()

	if exec, exists := r.executions[taskID]; exists {
		log.Printf("Unregistered execution %d for task %s", exec.pid(), taskID)
		delete(r.executions, taskID)
	}
}

// GetExecution retrieves the execution state for a task
func (r *ExecutionRegistry) GetExecution(taskID string) (*taskExecution, bool) {
	r.executionsMu.RLock()
	exec, exists := r.executions[taskID]
	r.executionsMu.RUnlock()
	return exec, exists
}

// IsTaskRunning returns true if the task is currently tracked in executions
func (r *ExecutionRegistry) IsTaskRunning(taskID string) bool {
	_, exists := r.GetExecution(taskID)
	return exists
}

// ListRunningTaskIDs returns task IDs of all currently running processes
func (r *ExecutionRegistry) ListRunningTaskIDs() []string {
	r.executionsMu.RLock()
	defer r.executionsMu.RUnlock()

	taskIDs := make([]string, 0, len(r.executions))
	for taskID := range r.executions {
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

// GetRunIDForTask returns the agent-manager run ID for a task, empty if not found
func (r *ExecutionRegistry) GetRunIDForTask(taskID string) string {
	r.executionsMu.RLock()
	defer r.executionsMu.RUnlock()
	if exec, ok := r.executions[taskID]; ok {
		return exec.getRunID()
	}
	return ""
}

// Count returns the number of registered executions
func (r *ExecutionRegistry) Count() int {
	r.executionsMu.RLock()
	defer r.executionsMu.RUnlock()
	return len(r.executions)
}

// GetAllExecutions returns a snapshot of all current executions for iteration.
// The returned slice contains copies to prevent data races.
func (r *ExecutionRegistry) GetAllExecutions() []taskExecutionSnapshot {
	r.executionsMu.RLock()
	defer r.executionsMu.RUnlock()

	snapshots := make([]taskExecutionSnapshot, 0, len(r.executions))
	for taskID, exec := range r.executions {
		snapshots = append(snapshots, taskExecutionSnapshot{
			TaskID:    taskID,
			AgentTag:  exec.agentTag,
			RunID:     exec.runID,
			Started:   exec.started,
			TimeoutAt: exec.timeoutAt,
			TimedOut:  exec.isTimedOut(),
			PID:       exec.pid(),
		})
	}
	return snapshots
}

// GetTimedOutExecutions returns snapshots of executions that have exceeded their timeout
func (r *ExecutionRegistry) GetTimedOutExecutions() []taskExecutionSnapshot {
	r.executionsMu.RLock()
	defer r.executionsMu.RUnlock()

	timedOut := make([]taskExecutionSnapshot, 0)
	for taskID, exec := range r.executions {
		if exec.isTimedOut() {
			timedOut = append(timedOut, taskExecutionSnapshot{
				TaskID:    taskID,
				AgentTag:  exec.agentTag,
				RunID:     exec.runID,
				Started:   exec.started,
				TimeoutAt: exec.timeoutAt,
				TimedOut:  true,
				PID:       exec.pid(),
			})
		}
	}
	return timedOut
}

// MarkTimedOut marks an execution as timed out (for watchdog use)
func (r *ExecutionRegistry) MarkTimedOut(taskID string) {
	r.executionsMu.Lock()
	defer r.executionsMu.Unlock()
	if exec, exists := r.executions[taskID]; exists {
		exec.timedOut = true
	}
}

// taskExecutionSnapshot is an immutable copy of execution state for external use
type taskExecutionSnapshot struct {
	TaskID    string
	AgentTag  string
	RunID     string
	Started   time.Time
	TimeoutAt time.Time
	TimedOut  bool
	PID       int
}

// ListRunningTaskIDsAsMap returns task IDs as a map for O(1) lookup
func (r *ExecutionRegistry) ListRunningTaskIDsAsMap() map[string]struct{} {
	r.executionsMu.RLock()
	defer r.executionsMu.RUnlock()

	result := make(map[string]struct{}, len(r.executions))
	for taskID := range r.executions {
		result[taskID] = struct{}{}
	}
	return result
}

// Clear removes all executions from the registry
func (r *ExecutionRegistry) Clear() int {
	r.executionsMu.Lock()
	defer r.executionsMu.Unlock()

	count := len(r.executions)
	r.executions = make(map[string]*taskExecution)
	return count
}
