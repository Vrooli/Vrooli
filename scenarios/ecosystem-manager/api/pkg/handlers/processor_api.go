package handlers

import (
	"time"

	"github.com/ecosystem-manager/api/pkg/insights"
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

	// Insight-related methods
	LoadInsightReports(taskID string) ([]insights.InsightReport, error)
	LoadInsightReport(taskID, reportID string) (*insights.InsightReport, error)
	SaveInsightReport(report insights.InsightReport) error
	UpdateSuggestionStatus(taskID, reportID, suggestionID, status string) error
	LoadAllInsightReports(sinceTime time.Time) ([]insights.InsightReport, error)
	BuildInsightPrompt(taskID string, limit int, statusFilter string) (string, error)
	GenerateInsightReportForTask(taskID string, limit int, statusFilter string) (*insights.InsightReport, error)
	GenerateInsightReportWithCustomPrompt(taskID string, limit int, statusFilter string, customPrompt string) (*insights.InsightReport, error)
	GenerateSystemInsightReport(sinceTime time.Time) (*insights.SystemInsightReport, error)
}
