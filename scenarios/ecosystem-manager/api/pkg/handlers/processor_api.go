package handlers

import (
	"github.com/ecosystem-manager/api/pkg/queue"
)

// ProcessorAPI captures the processor surface used by handlers.
// Both TaskHandlers and QueueHandlers depend on this interface for testability.
type ProcessorAPI interface {
	// Lifecycle management
	Start()
	Stop()
	Pause()
	ResumeWithReset() queue.ResumeResetSummary
	ResumeWithoutReset()
	Wake()

	// Queue status and diagnostics
	GetQueueStatus() map[string]any
	GetResumeDiagnostics() queue.ResumeDiagnostics
	ProcessQueue()

	// Task execution control
	TerminateRunningProcess(taskID string) error
	ForceStartTask(taskID string, allowOverflow bool) error
	StartTaskIfSlotAvailable(taskID string) error
	GetRunningProcessesInfo() []queue.ProcessInfo

	// Task logs
	GetTaskLogs(taskID string, afterSeq int64) (entries []queue.LogEntry, nextSeq int64, running bool, agentID string, completed bool, processID int)
	ResetTaskLogs(taskID string)

	// Rate limiting
	ResetRateLimitPause()

	// Auto Steer integration
	AutoSteerIntegration() *queue.AutoSteerIntegration

	// Execution history
	LoadExecutionHistory(taskID string) ([]queue.ExecutionHistory, error)
	LoadAllExecutionHistory() ([]queue.ExecutionHistory, error)
	GetExecutionFilePath(taskID, executionID, filename string) string
}
