package queue

import (
	"fmt"
	"log"
	"os/exec"
	"time"
)

// ProcessInfo contains runtime information about a running task process
type ProcessInfo struct {
	TaskID          string `json:"task_id"`
	ProcessID       int    `json:"process_id"`
	StartTime       string `json:"start_time"`       // RFC3339 timestamp
	Duration        string `json:"duration"`         // Human-readable (e.g., "5m30s")
	DurationSeconds int64  `json:"duration_seconds"` // Machine-readable seconds
	AgentID         string `json:"agent_id,omitempty"`
}

// reserveExecution creates a placeholder execution entry for a task that's about to start
func (qp *Processor) reserveExecution(taskID, agentID string, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	qp.executionsMu.Lock()
	if existing, ok := qp.executions[taskID]; ok {
		if agentID != "" {
			existing.agentTag = agentID
		}
		if existing.started.IsZero() {
			existing.started = startedAt
		}
	} else {
		qp.executions[taskID] = &taskExecution{
			taskID:   taskID,
			agentTag: agentID,
			started:  startedAt,
		}
	}
	qp.executionsMu.Unlock()
}

// registerExecution registers a running process for a task
func (qp *Processor) registerExecution(taskID, agentID string, cmd *exec.Cmd, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	qp.executionsMu.Lock()
	execState, exists := qp.executions[taskID]
	if !exists {
		execState = &taskExecution{taskID: taskID}
		qp.executions[taskID] = execState
	}
	if agentID != "" {
		execState.agentTag = agentID
	}
	execState.cmd = cmd
	if execState.started.IsZero() {
		execState.started = startedAt
	}
	qp.executionsMu.Unlock()

	if cmd != nil && cmd.Process != nil {
		log.Printf("Registered execution %d for task %s", cmd.Process.Pid, taskID)
	} else {
		log.Printf("Registered execution record for task %s (pid unknown)", taskID)
	}
}

// unregisterExecution removes a task from the execution registry
func (qp *Processor) unregisterExecution(taskID string) {
	qp.executionsMu.Lock()
	if exec, exists := qp.executions[taskID]; exists {
		log.Printf("Unregistered execution %d for task %s", exec.pid(), taskID)
		delete(qp.executions, taskID)
	}
	qp.executionsMu.Unlock()
}

// getExecution retrieves the execution state for a task
func (qp *Processor) getExecution(taskID string) (*taskExecution, bool) {
	qp.executionsMu.RLock()
	exec, exists := qp.executions[taskID]
	qp.executionsMu.RUnlock()
	return exec, exists
}

// ListRunningProcesses returns task IDs of all currently running processes
func (qp *Processor) ListRunningProcesses() []string {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	taskIDs := make([]string, 0, len(qp.executions))
	for taskID := range qp.executions {
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

// GetRunningProcessesInfo returns detailed information about all running processes
func (qp *Processor) GetRunningProcessesInfo() []ProcessInfo {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	processes := make([]ProcessInfo, 0, len(qp.executions))
	now := time.Now()

	for taskID, execState := range qp.executions {
		duration := now.Sub(execState.started)
		processes = append(processes, ProcessInfo{
			TaskID:          taskID,
			ProcessID:       execState.pid(),
			StartTime:       execState.started.Format(time.RFC3339),
			Duration:        duration.Round(time.Second).String(),
			DurationSeconds: int64(duration.Seconds()),
			AgentID:         execState.agentTag,
		})
	}

	return processes
}

// IsTaskRunning returns true if the task is currently tracked in executions
func (qp *Processor) IsTaskRunning(taskID string) bool {
	_, exists := qp.getExecution(taskID)
	return exists
}

// getInternalRunningTaskIDs returns task IDs tracked in the internal execution registry
func (qp *Processor) getInternalRunningTaskIDs() map[string]struct{} {
	qp.executionsMu.RLock()
	defer qp.executionsMu.RUnlock()

	ids := make(map[string]struct{}, len(qp.executions))
	for taskID := range qp.executions {
		ids[taskID] = struct{}{}
	}
	return ids
}

// TerminateRunningProcess terminates a running task process
func (qp *Processor) TerminateRunningProcess(taskID string) error {
	exec, exists := qp.getExecution(taskID)
	if !exists {
		return fmt.Errorf("no running process found for task %s", taskID)
	}

	agentIdentifier := exec.agentTag
	if agentIdentifier == "" {
		agentIdentifier = makeAgentTag(taskID)
	}

	pid := exec.pid()

	// Terminate the agent and cleanup
	qp.stopClaudeAgent(agentIdentifier, pid)
	qp.cleanupClaudeAgentRegistry()

	// Force kill the process group if still alive
	if qp.isProcessAlive(pid) {
		if err := KillProcessGroup(pid); err != nil {
			log.Printf("Warning: failed to kill process group for task %s (pid %d): %v", taskID, pid, err)
		}
	}

	qp.ResetTaskLogs(taskID)
	qp.unregisterExecution(taskID)

	return nil
}
