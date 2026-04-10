package dto

import (
	"time"

	"github.com/ecosystem-manager/api/pkg/insights"
)

// InsightGenerateRequest represents a request to generate an insight report.
type InsightGenerateRequest struct {
	Limit        int    `json:"limit,omitempty"`
	StatusFilter string `json:"status_filter,omitempty"`
	CustomPrompt string `json:"custom_prompt,omitempty"`
}

// InsightReportResponse wraps an insight report with API metadata.
type InsightReportResponse struct {
	Report    insights.InsightReport `json:"report"`
	TaskID    string                 `json:"task_id"`
	Generated time.Time              `json:"generated_at"`
}

// InsightReportListResponse represents a list of insight reports.
type InsightReportListResponse struct {
	TaskID  string                   `json:"task_id"`
	Reports []insights.InsightReport `json:"reports"`
	Count   int                      `json:"count"`
}

// InsightPromptPreviewResponse represents a preview of the prompt to be used.
type InsightPromptPreviewResponse struct {
	TaskID       string `json:"task_id"`
	Prompt       string `json:"prompt"`
	Limit        int    `json:"limit"`
	StatusFilter string `json:"status_filter"`
	Executions   int    `json:"executions"`
}

// SuggestionStatusUpdateRequest represents a request to update suggestion status.
type SuggestionStatusUpdateRequest struct {
	Status string `json:"status" validate:"required,oneof=pending applied rejected superseded"`
}

// SuggestionStatusUpdateResponse represents the result of updating suggestion status.
type SuggestionStatusUpdateResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	TaskID       string `json:"task_id"`
	ReportID     string `json:"report_id"`
	SuggestionID string `json:"suggestion_id"`
	Status       string `json:"status"`
}

// SuggestionApplyRequest represents a request to apply a suggestion.
type SuggestionApplyRequest struct {
	DryRun bool `json:"dry_run"`
}

// SuggestionApplyResponse represents the result of applying a suggestion.
type SuggestionApplyResponse struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`
	BackupPath   string   `json:"backup_path,omitempty"`
	DryRun       bool     `json:"dry_run"`
	Error        string   `json:"error,omitempty"`
}

// SystemInsightsResponse represents system-wide insights data.
type SystemInsightsResponse struct {
	TimeWindow struct {
		Since time.Time `json:"since"`
		Days  int       `json:"days"`
	} `json:"time_window"`
	Summary struct {
		TotalReports       int            `json:"total_reports"`
		UniqueTasks        int            `json:"unique_tasks"`
		TotalExecutions    int            `json:"total_executions"`
		PatternsByType     map[string]int `json:"patterns_by_type"`
		SuggestionsByType  map[string]int `json:"suggestions_by_type"`
	} `json:"summary"`
	Reports []insights.InsightReport `json:"reports"`
}

// SystemInsightReportResponse wraps a system insight report.
type SystemInsightReportResponse struct {
	Message  string                      `json:"message"`
	ReportID string                      `json:"report_id"`
	Report   insights.SystemInsightReport `json:"report"`
}
