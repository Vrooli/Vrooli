package handlers

import "github.com/ecosystem-manager/api/pkg/queue"

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

// SimpleSuccessResponse represents a simple success/error response
type SimpleSuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ResumeDiagnosticsResponse represents the response for resume diagnostics
type ResumeDiagnosticsResponse struct {
	Success     bool                    `json:"success"`
	Diagnostics queue.ResumeDiagnostics `json:"diagnostics"`
	GeneratedAt string                  `json:"generated_at"`
}
