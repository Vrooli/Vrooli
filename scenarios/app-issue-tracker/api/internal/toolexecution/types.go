// Package toolexecution implements the Tool Execution Protocol for app-issue-tracker.
//
// This package provides the server-side implementation of the Tool Execution Protocol,
// allowing AI agents to execute app-issue-tracker tools via a standardized HTTP API.
//
// Protocol Contract:
//
//	Request:  POST /api/v1/tools/execute
//	          { "tool_name": "list_issues", "arguments": {...} }
//
//	Response: { "success": true, "result": {...}, "is_async": false }
//	     or:  { "success": true, "is_async": true, "run_id": "...", "status": "pending" }
//	     or:  { "success": false, "error": "...", "code": "..." }
package toolexecution

// ExecuteRequest is the request body for tool execution.
type ExecuteRequest struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ExecutionResult is the standardized response for tool execution.
type ExecutionResult struct {
	// Success indicates whether the tool executed successfully
	Success bool `json:"success"`

	// Result contains the tool output (for successful executions)
	Result interface{} `json:"result,omitempty"`

	// Error contains the error message (for failed executions)
	Error string `json:"error,omitempty"`

	// Code is a machine-readable error code (for failed executions)
	// Common codes: unknown_tool, invalid_args, not_found, conflict, internal_error
	Code string `json:"code,omitempty"`

	// IsAsync indicates whether this is a long-running operation
	IsAsync bool `json:"is_async"`

	// RunID is the unique identifier for async operations (for polling status)
	RunID string `json:"run_id,omitempty"`

	// Status is the current status for async operations (pending, running, completed, failed)
	Status string `json:"status,omitempty"`
}

// Error codes used in ExecutionResult.Code
const (
	CodeUnknownTool   = "unknown_tool"
	CodeInvalidArgs   = "invalid_args"
	CodeNotFound      = "not_found"
	CodeConflict      = "conflict"
	CodeInternalError = "internal_error"
	CodeRateLimited   = "rate_limited"
)

// Async status values used in ExecutionResult.Status
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusActive    = "active"
)

// Helper functions for creating common responses

// SuccessResult creates a successful synchronous result.
func SuccessResult(result interface{}) *ExecutionResult {
	return &ExecutionResult{
		Success: true,
		Result:  result,
		IsAsync: false,
	}
}

// AsyncResult creates a successful async result with a run ID.
func AsyncResult(result interface{}, runID string) *ExecutionResult {
	return &ExecutionResult{
		Success: true,
		Result:  result,
		IsAsync: true,
		RunID:   runID,
		Status:  StatusPending,
	}
}

// ErrorResult creates an error result.
func ErrorResult(message, code string) *ExecutionResult {
	return &ExecutionResult{
		Success: false,
		Error:   message,
		Code:    code,
		IsAsync: false,
	}
}
