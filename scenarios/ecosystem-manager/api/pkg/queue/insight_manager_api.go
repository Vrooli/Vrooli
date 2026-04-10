package queue

import (
	"time"

	"github.com/ecosystem-manager/api/pkg/insights"
)

// InsightManagerAPI defines the interface for insight report management.
// This separates insight generation and persistence from task execution.
type InsightManagerAPI interface {
	// SaveInsightReport persists an insight report to disk.
	SaveInsightReport(report insights.InsightReport) error

	// LoadInsightReports loads all insight reports for a task.
	// Results are sorted by generated time (newest first).
	LoadInsightReports(taskID string) ([]insights.InsightReport, error)

	// LoadInsightReport loads a specific insight report.
	LoadInsightReport(taskID, reportID string) (*insights.InsightReport, error)

	// LoadAllInsightReports loads all insight reports across all tasks since a given time.
	LoadAllInsightReports(sinceTime time.Time) ([]insights.InsightReport, error)

	// UpdateSuggestionStatus updates the status of a specific suggestion.
	UpdateSuggestionStatus(taskID, reportID, suggestionID, status string) error

	// GenerateSystemInsightReport analyzes all insight reports and generates system-wide insights.
	GenerateSystemInsightReport(sinceTime time.Time) (*insights.SystemInsightReport, error)

	// GenerateInsightReportForTask analyzes execution history and generates an insight report.
	GenerateInsightReportForTask(taskID string, limit int, statusFilter string) (*insights.InsightReport, error)

	// BuildInsightPrompt builds and returns the prompt that would be used for insight generation.
	// Useful for preview/editing workflows.
	BuildInsightPrompt(taskID string, limit int, statusFilter string) (string, error)

	// GenerateInsightReportWithCustomPrompt generates an insight report using a custom prompt.
	GenerateInsightReportWithCustomPrompt(taskID string, limit int, statusFilter string, customPrompt string) (*insights.InsightReport, error)
}

// Verify InsightManager implements the interface at compile time.
var _ InsightManagerAPI = (*InsightManager)(nil)
