package dto

import "time"

// QueueStatusResponse represents the current state of the queue processor.
type QueueStatusResponse struct {
	IsActive           bool      `json:"is_active"`
	IsPaused           bool      `json:"is_paused"`
	IsRateLimitPaused  bool      `json:"is_rate_limit_paused"`
	RateLimitResumeAt  string    `json:"rate_limit_resume_at,omitempty"`
	PendingCount       int       `json:"pending_count"`
	InProgressCount    int       `json:"in_progress_count"`
	RunningProcesses   int       `json:"running_processes"`
	AvailableSlots     int       `json:"available_slots"`
	MaxSlots           int       `json:"max_slots"`
	LastProcessedAt    time.Time `json:"last_processed_at,omitempty"`
	CooldownSeconds    int       `json:"cooldown_seconds"`
	TaskTimeoutMinutes int       `json:"task_timeout_minutes"`
}

// ProcessListResponse represents a list of running processes.
type ProcessListResponse struct {
	Processes []ProcessInfo `json:"processes"`
	Count     int           `json:"count"`
	Timestamp int64         `json:"timestamp"`
}

// QueueTriggerResponse represents the result of triggering queue processing.
type QueueTriggerResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Status    any    `json:"status,omitempty"` // Include refreshed status if available
}

// QueuePauseResponse represents the result of pausing/resuming the queue.
type QueuePauseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Paused  bool   `json:"paused"`
}

// ResumeDiagnosticsResponse provides diagnostic information for troubleshooting.
type ResumeDiagnosticsResponse struct {
	Success     bool           `json:"success"`
	Diagnostics map[string]any `json:"diagnostics"`
	GeneratedAt string         `json:"generated_at"`
}

// ResetResponse represents the result of a reset operation.
type ResetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Summary string `json:"summary,omitempty"`
}
