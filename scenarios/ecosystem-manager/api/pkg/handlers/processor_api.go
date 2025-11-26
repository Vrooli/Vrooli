package handlers

import "github.com/ecosystem-manager/api/pkg/queue"

// ProcessorAPI captures the processor surface TaskHandlers require.
type ProcessorAPI interface {
	TerminateRunningProcess(taskID string) error
	ForceStartTask(taskID string, allowOverflow bool) error
	StartTaskIfSlotAvailable(taskID string) error
	Wake()
	GetRunningProcessesInfo() []queue.ProcessInfo
	AutoSteerIntegration() *queue.AutoSteerIntegration
	GetTaskLogs(taskID string, afterSeq int64) (entries []queue.LogEntry, nextSeq int64, running bool, agentID string, completed bool, processID int)
	LoadExecutionHistory(taskID string) ([]queue.ExecutionHistory, error)
	LoadAllExecutionHistory() ([]queue.ExecutionHistory, error)
	GetExecutionFilePath(taskID, executionID, filename string) string
}
