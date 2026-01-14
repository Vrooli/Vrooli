package queue

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
)

// insightProcess tracks a running insight generation process
type insightProcess struct {
	taskID    string
	taskTitle string
	started   time.Time
}

var (
	// Global registry for insight processes
	insightProcessesMu sync.RWMutex
	insightProcesses   = make(map[string]*insightProcess)
)

// ProcessInfo contains runtime information about a running task process
type ProcessInfo struct {
	TaskID          string `json:"task_id"`
	ProcessID       int    `json:"process_id"`
	ProcessType     string `json:"process_type"`         // "task" or "insight"
	TaskTitle       string `json:"task_title,omitempty"` // Optional task title for UI display
	StartTime       string `json:"start_time"`           // RFC3339 timestamp
	Duration        string `json:"duration"`             // Human-readable (e.g., "5m30s")
	DurationSeconds int64  `json:"duration_seconds"`     // Machine-readable seconds
	AgentID         string `json:"agent_id,omitempty"`
	TimeoutAt       string `json:"timeout_at,omitempty"`     // RFC3339 timestamp when timeout occurs
	TimedOut        bool   `json:"timed_out"`                // Whether task has exceeded timeout
	TimeRemaining   string `json:"time_remaining,omitempty"` // Human-readable time until timeout
}

// reserveExecution creates a placeholder execution entry for a task that's about to start
func (qp *Processor) reserveExecution(taskID, agentID string, startedAt time.Time) {
	qp.registry.ReserveExecution(taskID, agentID, startedAt)
}

// registerExecution registers a running process for a task
func (qp *Processor) registerExecution(taskID, agentID string, cmd *exec.Cmd, startedAt time.Time) {
	qp.registry.RegisterExecution(taskID, agentID, cmd, startedAt)
}

// registerRunID associates an agent-manager run ID with a task execution
func (qp *Processor) registerRunID(taskID, runID string) {
	qp.registry.RegisterRunID(taskID, runID)
}

// unregisterExecution removes a task from the execution registry
func (qp *Processor) unregisterExecution(taskID string) {
	qp.registry.UnregisterExecution(taskID)
}

// getExecution retrieves the execution state for a task
func (qp *Processor) getExecution(taskID string) (*taskExecution, bool) {
	return qp.registry.GetExecution(taskID)
}

// ListRunningProcesses returns task IDs of all currently running processes
func (qp *Processor) ListRunningProcesses() []string {
	return qp.registry.ListRunningTaskIDs()
}

// GetRunningProcessesInfo returns detailed information about all running processes
func (qp *Processor) GetRunningProcessesInfo() []ProcessInfo {
	executions := qp.registry.GetAllExecutions()
	now := time.Now()

	processes := make([]ProcessInfo, 0, len(executions))

	// Add task executions
	for _, exec := range executions {
		duration := now.Sub(exec.Started)
		info := ProcessInfo{
			TaskID:          exec.TaskID,
			ProcessID:       exec.PID,
			ProcessType:     "task",
			StartTime:       exec.Started.Format(time.RFC3339),
			Duration:        duration.Round(time.Second).String(),
			DurationSeconds: int64(duration.Seconds()),
			AgentID:         exec.AgentTag,
			TimedOut:        exec.TimedOut,
		}

		// Try to get task title from storage
		if qp.storage != nil {
			if task, _, err := qp.storage.GetTaskByID(exec.TaskID); err == nil && task != nil {
				info.TaskTitle = task.Title
			}
		}

		// Add timeout information if available
		if !exec.TimeoutAt.IsZero() {
			info.TimeoutAt = exec.TimeoutAt.Format(time.RFC3339)
			timeRemaining := exec.TimeoutAt.Sub(now)
			if timeRemaining > 0 {
				info.TimeRemaining = timeRemaining.Round(time.Second).String()
			} else {
				info.TimeRemaining = "exceeded"
			}
		}

		processes = append(processes, info)
	}

	// Add insight generation processes
	insightProcessesMu.RLock()
	for taskID, insightProc := range insightProcesses {
		duration := now.Sub(insightProc.started)
		info := ProcessInfo{
			TaskID:          taskID,
			ProcessID:       0, // No actual process ID for HTTP-based insights
			ProcessType:     "insight",
			TaskTitle:       insightProc.taskTitle,
			StartTime:       insightProc.started.Format(time.RFC3339),
			Duration:        duration.Round(time.Second).String(),
			DurationSeconds: int64(duration.Seconds()),
			AgentID:         "insight-generator",
		}
		processes = append(processes, info)
	}
	insightProcessesMu.RUnlock()

	return processes
}

// RegisterInsightProcess registers a running insight generation process
func RegisterInsightProcess(taskID, taskTitle string) {
	insightProcessesMu.Lock()
	defer insightProcessesMu.Unlock()

	insightProcesses[taskID] = &insightProcess{
		taskID:    taskID,
		taskTitle: taskTitle,
		started:   time.Now(),
	}

	log.Printf("Registered insight generation process for task %s", taskID)
}

// UnregisterInsightProcess removes an insight generation process from the registry
func UnregisterInsightProcess(taskID string) {
	insightProcessesMu.Lock()
	defer insightProcessesMu.Unlock()

	if _, exists := insightProcesses[taskID]; exists {
		delete(insightProcesses, taskID)
		log.Printf("Unregistered insight generation process for task %s", taskID)
	}
}

// IsTaskRunning returns true if the task is currently tracked in executions
func (qp *Processor) IsTaskRunning(taskID string) bool {
	return qp.registry.IsTaskRunning(taskID)
}

// TerminateRunningProcess terminates a running task process via agent-manager
func (qp *Processor) TerminateRunningProcess(taskID string) error {
	if !qp.registry.IsTaskRunning(taskID) {
		return fmt.Errorf("no running process found for task %s", taskID)
	}

	// Stop via agent-manager
	if err := qp.stopRunViaAgentManager(taskID); err != nil {
		log.Printf("Warning: agent-manager stop reported error for task %s: %v", taskID, err)
		// Continue with cleanup even if termination had issues
	}

	qp.ResetTaskLogs(taskID)
	qp.registry.UnregisterExecution(taskID)

	return nil
}
