package queue

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
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
	ProcessType     string `json:"process_type"`             // "task" or "insight"
	TaskTitle       string `json:"task_title,omitempty"`     // Optional task title for UI display
	StartTime       string `json:"start_time"`               // RFC3339 timestamp
	Duration        string `json:"duration"`                 // Human-readable (e.g., "5m30s")
	DurationSeconds int64  `json:"duration_seconds"`         // Machine-readable seconds
	AgentID         string `json:"agent_id,omitempty"`
	TimeoutAt       string `json:"timeout_at,omitempty"`     // RFC3339 timestamp when timeout occurs
	TimedOut        bool   `json:"timed_out"`                // Whether task has exceeded timeout
	TimeRemaining   string `json:"time_remaining,omitempty"` // Human-readable time until timeout
}

// reserveExecution creates a placeholder execution entry for a task that's about to start
func (qp *Processor) reserveExecution(taskID, agentID string, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	// Get current timeout setting
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	timeoutAt := startedAt.Add(timeoutDuration)

	qp.executionsMu.Lock()
	defer qp.executionsMu.Unlock()

	if existing, ok := qp.executions[taskID]; ok {
		if agentID != "" {
			existing.agentTag = agentID
		}
		if existing.started.IsZero() {
			existing.started = startedAt
			existing.timeoutAt = timeoutAt
		}
	} else {
		qp.executions[taskID] = &taskExecution{
			taskID:    taskID,
			agentTag:  agentID,
			started:   startedAt,
			timeoutAt: timeoutAt,
		}
	}
}

// registerExecution registers a running process for a task
func (qp *Processor) registerExecution(taskID, agentID string, cmd *exec.Cmd, startedAt time.Time) {
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	// Get current timeout setting
	currentSettings := settings.GetSettings()
	timeoutDuration := time.Duration(currentSettings.TaskTimeout) * time.Minute
	timeoutAt := startedAt.Add(timeoutDuration)

	qp.executionsMu.Lock()
	defer qp.executionsMu.Unlock()

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
		execState.timeoutAt = timeoutAt
	}

	// Defensive nil check for logging
	if cmd != nil && cmd.Process != nil {
		log.Printf("Registered execution %d for task %s (timeout at %s)", cmd.Process.Pid, taskID, timeoutAt.Format(time.RFC3339))
	} else {
		log.Printf("Registered execution record for task %s (pid unknown, timeout at %s)", taskID, timeoutAt.Format(time.RFC3339))
	}
}

// unregisterExecution removes a task from the execution registry
func (qp *Processor) unregisterExecution(taskID string) {
	qp.executionsMu.Lock()
	defer qp.executionsMu.Unlock()

	if exec, exists := qp.executions[taskID]; exists {
		log.Printf("Unregistered execution %d for task %s", exec.pid(), taskID)
		delete(qp.executions, taskID)
	}
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

	// Add task executions
	for taskID, execState := range qp.executions {
		duration := now.Sub(execState.started)
		info := ProcessInfo{
			TaskID:          taskID,
			ProcessID:       execState.pid(),
			ProcessType:     "task", // Default to task type
			StartTime:       execState.started.Format(time.RFC3339),
			Duration:        duration.Round(time.Second).String(),
			DurationSeconds: int64(duration.Seconds()),
			AgentID:         execState.agentTag,
			TimedOut:        execState.isTimedOut(),
		}

		// Try to get task title from storage
		if qp.storage != nil {
			if task, _, err := qp.storage.GetTaskByID(taskID); err == nil && task != nil {
				info.TaskTitle = task.Title
			}
		}

		// Add timeout information if available
		if !execState.timeoutAt.IsZero() {
			info.TimeoutAt = execState.timeoutAt.Format(time.RFC3339)
			timeRemaining := execState.timeoutAt.Sub(now)
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

	// Use canonical termination function for consistency
	if err := qp.terminateAgent(taskID, agentIdentifier, pid); err != nil {
		log.Printf("Warning: agent termination reported error for task %s: %v", taskID, err)
		// Continue with cleanup even if termination had issues
	}

	qp.ResetTaskLogs(taskID)
	qp.unregisterExecution(taskID)

	return nil
}
