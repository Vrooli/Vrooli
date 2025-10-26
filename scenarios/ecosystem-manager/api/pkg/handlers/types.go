package handlers

import "github.com/ecosystem-manager/api/pkg/queue"

// QueueStatusResponse represents the response for queue status endpoint
type QueueStatusResponse struct {
	ProcessorActive         bool                  `json:"processor_active"`
	SettingsActive          bool                  `json:"settings_active"`
	MaintenanceState        string                `json:"maintenance_state"`
	RateLimitInfo           RateLimitInfoResponse `json:"rate_limit_info"`
	MaxConcurrent           int                   `json:"max_concurrent"`
	ExecutingCount          int                   `json:"executing_count"`
	RunningCount            int                   `json:"running_count"`
	AvailableSlots          int                   `json:"available_slots"`
	PendingCount            int                   `json:"pending_count"`
	InProgressCount         int                   `json:"in_progress_count"`
	CompletedCount          int                   `json:"completed_count"`
	FailedCount             int                   `json:"failed_count"`
	CompletedFinalizedCount int                   `json:"completed_finalized_count"`
	FailedBlockedCount      int                   `json:"failed_blocked_count"`
	ArchivedCount           int                   `json:"archived_count"`
	ReviewCount             int                   `json:"review_count"`
	ReadyInProgress         int                   `json:"ready_in_progress"`
	RefreshInterval         int                   `json:"refresh_interval"`
	ProcessorRunning        bool                  `json:"processor_running"`
	Timestamp               int64                 `json:"timestamp"`
	LastProcessedAt         any                   `json:"last_processed_at,omitempty"` // string or nil
}

// RateLimitInfoResponse represents rate limit pause information
type RateLimitInfoResponse struct {
	Paused        bool   `json:"paused"`
	PauseUntil    string `json:"pause_until,omitempty"`
	RemainingSecs int    `json:"remaining_secs,omitempty"`
}

// QueueTriggerResponse represents the response for triggering queue processing
type QueueTriggerResponse struct {
	Success   bool                `json:"success"`
	Message   string              `json:"message"`
	Timestamp int64               `json:"timestamp"`
	Status    QueueStatusResponse `json:"status"`
}

// ProcessesListResponse represents the response for running processes list
type ProcessesListResponse struct {
	Processes []queue.ProcessInfo `json:"processes"`
	Count     int                 `json:"count"`
	Timestamp int64               `json:"timestamp"`
}

// ProcessTerminateResponse represents the response for process termination
type ProcessTerminateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	TaskID  string `json:"task_id"`
}

// MaintenanceStateResponse represents the response for maintenance state changes
type MaintenanceStateResponse struct {
	Success            bool                      `json:"success"`
	Message            string                    `json:"message"`
	State              string                    `json:"state"`
	Status             QueueStatusResponse       `json:"status"`
	ResumeResetSummary *queue.ResumeResetSummary `json:"resume_reset_summary,omitempty"`
}

// SimpleSuccessResponse represents a simple success/error response
type SimpleSuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// RateLimitResetResponse represents the response for rate limit reset
type RateLimitResetResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Status  QueueStatusResponse `json:"status"`
}

// ResumeDiagnosticsResponse represents the response for resume diagnostics
type ResumeDiagnosticsResponse struct {
	Success     bool                    `json:"success"`
	Diagnostics queue.ResumeDiagnostics `json:"diagnostics"`
	GeneratedAt string                  `json:"generated_at"`
}

// TaskCreateErrorItem represents an error that occurred during task creation
type TaskCreateErrorItem struct {
	Target string `json:"target"`
	Error  string `json:"error"`
}

// TaskCreateSkippedItem represents a skipped task during batch creation
type TaskCreateSkippedItem struct {
	Target string `json:"target"`
	Reason string `json:"reason"`
}

// TaskCreateResponse represents the response for task creation (batch or single)
type TaskCreateResponse struct {
	Success bool                    `json:"success"`
	Created []any                   `json:"created"` // TaskItem slice
	Skipped []TaskCreateSkippedItem `json:"skipped,omitempty"`
	Errors  []TaskCreateErrorItem   `json:"errors,omitempty"`
}

// TaskGetResponse represents the response for getting a task
type TaskGetResponse struct {
	Success bool `json:"success"`
	Task    any  `json:"task"` // TaskItem
}

// TaskListResponse represents the response for task list
type TaskListResponse struct {
	Tasks []any `json:"tasks"` // []TaskItem
	Count int   `json:"count"`
}

// TaskLogsResponse represents the response for task logs
type TaskLogsResponse struct {
	TaskID       string `json:"task_id"`
	AgentID      string `json:"agent_id"`
	ProcessID    int    `json:"process_id"`
	Running      bool   `json:"running"`
	Completed    bool   `json:"completed"`
	NextSequence int    `json:"next_sequence"`
	Entries      []any  `json:"entries"` // Log entries
	Timestamp    int64  `json:"timestamp"`
}

// TaskUpdateResponse represents the response for task update
type TaskUpdateResponse struct {
	Success bool `json:"success"`
	Task    any  `json:"task"` // TaskItem
}

// TaskPromptResponse represents the response for task prompt details
type TaskPromptResponse struct {
	TaskID          string `json:"task_id"`
	Operation       string `json:"operation"`
	PromptSections  []any  `json:"prompt_sections"` // Prompt sections
	OperationConfig any    `json:"operation_config"`
	TaskDetails     any    `json:"task_details"` // TaskItem
}

// AssembledPromptResponse represents the response for assembled prompt
type AssembledPromptResponse struct {
	TaskID           string `json:"task_id"`
	Operation        string `json:"operation"`
	Prompt           string `json:"prompt"`
	PromptLength     int    `json:"prompt_length"`
	PromptCached     bool   `json:"prompt_cached"`
	SectionsDetailed []any  `json:"sections_detailed"` // Section details
	OperationConfig  any    `json:"operation_config"`
	TaskStatus       string `json:"task_status"`
}

// PromptViewerResponse represents the response for prompt viewer (no task)
type PromptViewerResponse struct {
	TaskType         string  `json:"task_type"`
	Operation        string  `json:"operation"`
	Title            string  `json:"title"`
	Sections         []any   `json:"sections"`
	SectionCount     int     `json:"section_count"`
	SectionsDetailed []any   `json:"sections_detailed"`
	PromptSize       int     `json:"prompt_size"`
	PromptSizeKB     string  `json:"prompt_size_kb"`
	PromptSizeMB     string  `json:"prompt_size_mb"`
	WarningLarge     bool    `json:"warning_large,omitempty"`
	Prompt           *string `json:"prompt,omitempty"`
}

// SettingsResponse represents the response for settings endpoints
type SettingsResponse struct {
	Success            bool                      `json:"success"`
	Settings           any                       `json:"settings"` // settings.Settings
	Message            string                    `json:"message,omitempty"`
	ResumeResetSummary *queue.ResumeResetSummary `json:"resume_reset_summary,omitempty"`
}

// ModelsResponse represents the response for available models
type ModelsResponse struct {
	Success  bool     `json:"success"`
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
	Error    string   `json:"error,omitempty"`
}

// RecyclerPreviewResponse represents the response for recycler prompt preview
type RecyclerPreviewResponse struct {
	Success  bool   `json:"success"`
	Result   any    `json:"result"` // Summarizer result
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Error    string `json:"error,omitempty"`
}

// LogsResponse represents the response for system logs
type LogsResponse struct {
	Success bool  `json:"success"`
	Entries []any `json:"entries"` // Log entries
}
