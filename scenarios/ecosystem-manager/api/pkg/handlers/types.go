package handlers

// NOTE: This file previously imported queue types which caused a boundary violation.
// Now we use either inline types or the api/dto package for clean separation.

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
