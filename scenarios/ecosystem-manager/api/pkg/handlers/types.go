package handlers

import "github.com/ecosystem-manager/api/pkg/queue"

// QueueStatusResponse represents the response for queue status endpoint
type QueueStatusResponse struct {
	ProcessorActive         bool                   `json:"processor_active"`
	SettingsActive          bool                   `json:"settings_active"`
	MaintenanceState        string                 `json:"maintenance_state"`
	RateLimitInfo           RateLimitInfoResponse  `json:"rate_limit_info"`
	MaxConcurrent           int                    `json:"max_concurrent"`
	ExecutingCount          int                    `json:"executing_count"`
	RunningCount            int                    `json:"running_count"`
	AvailableSlots          int                    `json:"available_slots"`
	PendingCount            int                    `json:"pending_count"`
	InProgressCount         int                    `json:"in_progress_count"`
	CompletedCount          int                    `json:"completed_count"`
	FailedCount             int                    `json:"failed_count"`
	CompletedFinalizedCount int                    `json:"completed_finalized_count"`
	FailedBlockedCount      int                    `json:"failed_blocked_count"`
	ArchivedCount           int                    `json:"archived_count"`
	ReviewCount             int                    `json:"review_count"`
	ReadyInProgress         int                    `json:"ready_in_progress"`
	RefreshInterval         int                    `json:"refresh_interval"`
	ProcessorRunning        bool                   `json:"processor_running"`
	Timestamp               int64                  `json:"timestamp"`
	LastProcessedAt         interface{}            `json:"last_processed_at,omitempty"` // string or nil
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
	Success             bool                      `json:"success"`
	Message             string                    `json:"message"`
	State               string                    `json:"state"`
	Status              QueueStatusResponse       `json:"status"`
	ResumeResetSummary  *queue.ResumeResetSummary `json:"resume_reset_summary,omitempty"`
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
	Success      bool                      `json:"success"`
	Diagnostics  queue.ResumeDiagnostics   `json:"diagnostics"`
	GeneratedAt  string                    `json:"generated_at"`
}
