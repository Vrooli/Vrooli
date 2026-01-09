package handlers

import (
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
)

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

	// Insight-related methods
	LoadInsightReports(taskID string) ([]queue.InsightReport, error)
	LoadInsightReport(taskID, reportID string) (*queue.InsightReport, error)
	SaveInsightReport(report queue.InsightReport) error
	UpdateSuggestionStatus(taskID, reportID, suggestionID, status string) error
	LoadAllInsightReports(sinceTime time.Time) ([]queue.InsightReport, error)
	BuildInsightPrompt(taskID string, limit int, statusFilter string) (string, error)
	GenerateInsightReportForTask(taskID string, limit int, statusFilter string) (*queue.InsightReport, error)
	GenerateInsightReportWithCustomPrompt(taskID string, limit int, statusFilter string, customPrompt string) (*queue.InsightReport, error)
	GenerateSystemInsightReport(sinceTime time.Time) (*queue.SystemInsightReport, error)
}
