// Package domain defines the core domain types for the Agent Inbox scenario.
// This file contains structured error types for consistent error handling.
//
// Error Design Principles:
//   - Errors are categorized by domain (validation, resource, integration, internal)
//   - Each error code has an explicit recovery hint for both users and agents
//   - Errors are machine-readable (codes) and human-readable (messages)
//   - HTTP status codes are derived from error categories, not stored in errors
package domain

import (
	"fmt"
)

// ErrorCategory groups errors by their nature for consistent handling.
type ErrorCategory string

const (
	// CategoryValidation covers input validation failures.
	// Recovery: User or agent should correct the input and retry.
	CategoryValidation ErrorCategory = "validation"

	// CategoryNotFound covers missing resources.
	// Recovery: Verify the resource ID exists before retrying.
	CategoryNotFound ErrorCategory = "not_found"

	// CategoryConflict covers state conflicts (e.g., duplicate names).
	// Recovery: Modify the conflicting data and retry.
	CategoryConflict ErrorCategory = "conflict"

	// CategoryDependency covers external service failures.
	// Recovery: Wait and retry, or check service availability.
	CategoryDependency ErrorCategory = "dependency"

	// CategoryConfiguration covers missing or invalid configuration.
	// Recovery: Check environment variables and service configuration.
	CategoryConfiguration ErrorCategory = "configuration"

	// CategoryInternal covers unexpected internal errors.
	// Recovery: Report the error; manual intervention may be needed.
	CategoryInternal ErrorCategory = "internal"
)

// ErrorCode provides machine-readable error identification.
// Codes are prefixed by category: V=validation, N=not_found, D=dependency, etc.
type ErrorCode string

// Validation errors (V prefix)
const (
	ErrCodeInvalidInput       ErrorCode = "V001"
	ErrCodeMissingField       ErrorCode = "V002"
	ErrCodeInvalidUUID        ErrorCode = "V003"
	ErrCodeInvalidRole        ErrorCode = "V004"
	ErrCodeInvalidViewMode    ErrorCode = "V005"
	ErrCodeEmptyContent       ErrorCode = "V006"
	ErrCodeMissingToolCallID  ErrorCode = "V007"
	ErrCodeInvalidJSON        ErrorCode = "V008"
	ErrCodeNoFieldsToUpdate   ErrorCode = "V009"
	ErrCodeInvalidColor       ErrorCode = "V010"
	ErrCodeNoMessagesInChat   ErrorCode = "V011"
	ErrCodeAgentNotInMode     ErrorCode = "V012" // Chat is not in agent mode
	ErrCodeAgentNoActiveRun   ErrorCode = "V013" // No active agent run
	ErrCodeAgentAlreadyActive ErrorCode = "V014" // Chat already in agent mode
)

// Not found errors (N prefix)
const (
	ErrCodeChatNotFound    ErrorCode = "N001"
	ErrCodeMessageNotFound ErrorCode = "N002"
	ErrCodeLabelNotFound   ErrorCode = "N003"
	ErrCodeToolNotFound    ErrorCode = "N004"
)

// Dependency errors (D prefix)
const (
	ErrCodeDatabaseUnavailable     ErrorCode = "D001"
	ErrCodeDatabaseQueryFailed     ErrorCode = "D002"
	ErrCodeOpenRouterUnavailable   ErrorCode = "D003"
	ErrCodeOpenRouterError         ErrorCode = "D004"
	ErrCodeOllamaUnavailable       ErrorCode = "D005"
	ErrCodeAgentManagerError       ErrorCode = "D006"
	ErrCodeToolExecutionFailed     ErrorCode = "D007"
	ErrCodeAgentManagerUnavailable ErrorCode = "D008" // Agent-manager service not reachable
	ErrCodeAgentRunNotFound        ErrorCode = "D009" // Run ID not found in agent-manager
	ErrCodeAgentProtoParseFailed   ErrorCode = "D010" // Proto response parse failure
)

// Configuration errors (C prefix)
const (
	ErrCodeMissingAPIKey     ErrorCode = "C001"
	ErrCodeInvalidConfig     ErrorCode = "C002"
	ErrCodeServiceNotEnabled ErrorCode = "C003"
)

// Internal errors (I prefix)
const (
	ErrCodeInternalError    ErrorCode = "I001"
	ErrCodeStreamingError   ErrorCode = "I002"
	ErrCodeSerializationErr ErrorCode = "I003"
)

// Async operation errors (A prefix)
const (
	ErrCodeAsyncOperationNotFound  ErrorCode = "A001"
	ErrCodeAsyncTrackingFailed     ErrorCode = "A002"
	ErrCodeAsyncCancellationFailed ErrorCode = "A003"
	ErrCodeAsyncNoCancellation     ErrorCode = "A004"
	ErrCodeAsyncTimeout            ErrorCode = "A005"
	ErrCodeAsyncAlreadyCompleted   ErrorCode = "A006"
)

// RecoveryAction suggests what the caller should do after an error.
type RecoveryAction string

const (
	// ActionRetry indicates the operation may succeed if retried.
	ActionRetry RecoveryAction = "retry"

	// ActionRetryWithBackoff indicates retry after exponential delay.
	ActionRetryWithBackoff RecoveryAction = "retry_with_backoff"

	// ActionCorrectInput indicates the user/agent should fix the input.
	ActionCorrectInput RecoveryAction = "correct_input"

	// ActionCheckConfiguration indicates configuration needs review.
	ActionCheckConfiguration RecoveryAction = "check_configuration"

	// ActionCheckDependency indicates an external service should be verified.
	ActionCheckDependency RecoveryAction = "check_dependency"

	// ActionEscalate indicates the error needs manual intervention.
	ActionEscalate RecoveryAction = "escalate"

	// ActionVerifyResource indicates the resource ID should be verified.
	ActionVerifyResource RecoveryAction = "verify_resource"

	// ActionNone indicates no recovery action is possible.
	ActionNone RecoveryAction = "none"
)

// AppError is a structured error with category, code, and recovery guidance.
// It implements the error interface and provides machine-readable fields.
type AppError struct {
	// Code is a machine-readable error identifier.
	Code ErrorCode `json:"code"`

	// Category groups this error for HTTP status mapping.
	Category ErrorCategory `json:"category"`

	// Message is a user-friendly error description.
	Message string `json:"message"`

	// Recovery suggests what the caller should do next.
	Recovery RecoveryAction `json:"recovery"`

	// Details provides additional context (optional).
	// This may contain field names, constraint violations, etc.
	Details map[string]interface{} `json:"details,omitempty"`

	// Cause is the underlying error (not serialized).
	Cause error `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithCause attaches an underlying error.
func (e *AppError) WithCause(err error) *AppError {
	e.Cause = err
	return e
}

// WithDetail adds a detail key-value pair.
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewError creates a new AppError with the given parameters.
func NewError(code ErrorCode, category ErrorCategory, message string, recovery RecoveryAction) *AppError {
	return &AppError{
		Code:     code,
		Category: category,
		Message:  message,
		Recovery: recovery,
	}
}

// Convenience constructors for common errors

// ErrInvalidInput creates a validation error for bad input.
func ErrInvalidInput(message string) *AppError {
	return NewError(ErrCodeInvalidInput, CategoryValidation, message, ActionCorrectInput)
}

// ErrMissingField creates a validation error for a required field.
func ErrMissingField(field string) *AppError {
	return NewError(ErrCodeMissingField, CategoryValidation,
		fmt.Sprintf("%s is required", field), ActionCorrectInput).
		WithDetail("field", field)
}

// ErrInvalidUUID creates a validation error for an invalid UUID.
func ErrInvalidUUID(field string) *AppError {
	return NewError(ErrCodeInvalidUUID, CategoryValidation,
		fmt.Sprintf("invalid %s format", field), ActionCorrectInput).
		WithDetail("field", field)
}

// ErrInvalidJSON creates a validation error for malformed JSON.
func ErrInvalidJSON() *AppError {
	return NewError(ErrCodeInvalidJSON, CategoryValidation,
		"invalid JSON in request body", ActionCorrectInput)
}

// ErrChatNotFound creates a not-found error for a missing chat.
func ErrChatNotFound(chatID string) *AppError {
	return NewError(ErrCodeChatNotFound, CategoryNotFound,
		"chat not found", ActionVerifyResource).
		WithDetail("chat_id", chatID)
}

// ErrLabelNotFound creates a not-found error for a missing label.
func ErrLabelNotFound(labelID string) *AppError {
	return NewError(ErrCodeLabelNotFound, CategoryNotFound,
		"label not found", ActionVerifyResource).
		WithDetail("label_id", labelID)
}

// ErrNoMessagesInChat creates a validation error when a chat has no messages.
func ErrNoMessagesInChat(chatID string) *AppError {
	return NewError(ErrCodeNoMessagesInChat, CategoryValidation,
		"no messages in chat to process", ActionCorrectInput).
		WithDetail("chat_id", chatID)
}

// ErrDatabaseError creates a dependency error for database failures.
func ErrDatabaseError(operation string, err error) *AppError {
	return NewError(ErrCodeDatabaseQueryFailed, CategoryDependency,
		fmt.Sprintf("database operation failed: %s", operation), ActionRetryWithBackoff).
		WithCause(err).WithDetail("operation", operation)
}

// ErrOpenRouterUnavailable creates a dependency error for OpenRouter failures.
func ErrOpenRouterUnavailable(err error) *AppError {
	return NewError(ErrCodeOpenRouterUnavailable, CategoryDependency,
		"AI service temporarily unavailable", ActionRetryWithBackoff).
		WithCause(err)
}

// ErrOpenRouterAPIError creates a dependency error for OpenRouter API errors.
func ErrOpenRouterAPIError(statusCode int, message string) *AppError {
	return NewError(ErrCodeOpenRouterError, CategoryDependency,
		fmt.Sprintf("AI service error: %s", message), ActionRetryWithBackoff).
		WithDetail("status_code", statusCode)
}

// ErrMissingAPIKey creates a configuration error for missing API keys.
func ErrMissingAPIKey(service string) *AppError {
	return NewError(ErrCodeMissingAPIKey, CategoryConfiguration,
		fmt.Sprintf("%s API key not configured", service), ActionCheckConfiguration).
		WithDetail("service", service)
}

// ErrToolNotFound creates a not-found error for unknown tools.
func ErrToolNotFound(toolName string) *AppError {
	return NewError(ErrCodeToolNotFound, CategoryNotFound,
		fmt.Sprintf("unknown tool: %s", toolName), ActionVerifyResource).
		WithDetail("tool_name", toolName)
}

// ErrToolExecutionFailed creates a dependency error for tool failures.
func ErrToolExecutionFailed(toolName string, err error) *AppError {
	return NewError(ErrCodeToolExecutionFailed, CategoryDependency,
		fmt.Sprintf("tool execution failed: %s", toolName), ActionRetryWithBackoff).
		WithCause(err).WithDetail("tool_name", toolName)
}

// ErrAgentManagerError creates a dependency error for agent-manager failures.
func ErrAgentManagerError(operation string, err error) *AppError {
	return NewError(ErrCodeAgentManagerError, CategoryDependency,
		fmt.Sprintf("agent manager error: %s", operation), ActionCheckDependency).
		WithCause(err).WithDetail("operation", operation)
}

// ErrServiceUnavailable creates a dependency error for unavailable services.
func ErrServiceUnavailable(service string) *AppError {
	return NewError(ErrCodeAgentManagerError, CategoryDependency,
		fmt.Sprintf("%s service is unavailable", service), ActionCheckDependency).
		WithDetail("service", service)
}

// ErrExternalService creates a dependency error for external service failures.
func ErrExternalService(service, message string) *AppError {
	return NewError(ErrCodeAgentManagerError, CategoryDependency,
		fmt.Sprintf("%s: %s", service, message), ActionRetryWithBackoff).
		WithDetail("service", service)
}

// ErrOllamaUnavailable creates a dependency error for Ollama failures.
// Note: Ollama is optional, so this may trigger graceful degradation.
func ErrOllamaUnavailable(err error) *AppError {
	return NewError(ErrCodeOllamaUnavailable, CategoryDependency,
		"naming service temporarily unavailable", ActionRetryWithBackoff).
		WithCause(err)
}

// ErrInternal creates an internal error for unexpected failures.
func ErrInternal(message string, err error) *AppError {
	return NewError(ErrCodeInternalError, CategoryInternal,
		message, ActionEscalate).
		WithCause(err)
}

// ErrStreamingError creates an internal error for streaming failures.
func ErrStreamingError(message string, err error) *AppError {
	return NewError(ErrCodeStreamingError, CategoryInternal,
		message, ActionRetry).
		WithCause(err)
}

// CategoryToHTTPStatus maps error categories to HTTP status codes.
// This centralizes the HTTP semantics decision.
func CategoryToHTTPStatus(category ErrorCategory) int {
	switch category {
	case CategoryValidation:
		return 400
	case CategoryNotFound:
		return 404
	case CategoryConflict:
		return 409
	case CategoryDependency:
		return 502
	case CategoryConfiguration:
		return 503
	case CategoryInternal:
		return 500
	default:
		return 500
	}
}

// IsRetryable returns true if the error suggests retrying may help.
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Recovery {
		case ActionRetry, ActionRetryWithBackoff:
			return true
		}
	}
	return false
}

// IsUserError returns true if the error was caused by user input.
func IsUserError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Category == CategoryValidation || appErr.Category == CategoryNotFound
	}
	return false
}

// Async operation error constructors

// ErrAsyncOperationNotFound creates a not-found error for missing async operations.
func ErrAsyncOperationNotFound(toolCallID string) *AppError {
	return NewError(ErrCodeAsyncOperationNotFound, CategoryNotFound,
		"async operation not found", ActionVerifyResource).
		WithDetail("tool_call_id", toolCallID)
}

// ErrAsyncTrackingFailed creates a dependency error when async tracking fails to start.
func ErrAsyncTrackingFailed(toolCallID string, err error) *AppError {
	return NewError(ErrCodeAsyncTrackingFailed, CategoryDependency,
		"failed to start async tracking", ActionRetryWithBackoff).
		WithCause(err).WithDetail("tool_call_id", toolCallID)
}

// ErrAsyncCancellationFailed creates a dependency error when cancellation fails.
func ErrAsyncCancellationFailed(toolCallID string, err error) *AppError {
	return NewError(ErrCodeAsyncCancellationFailed, CategoryDependency,
		"failed to cancel async operation", ActionRetry).
		WithCause(err).WithDetail("tool_call_id", toolCallID)
}

// ErrAsyncNoCancellation creates a validation error when cancellation is not supported.
func ErrAsyncNoCancellation(toolCallID string) *AppError {
	return NewError(ErrCodeAsyncNoCancellation, CategoryValidation,
		"async operation does not support cancellation", ActionNone).
		WithDetail("tool_call_id", toolCallID)
}

// ErrAsyncTimeout creates a dependency error when an async operation times out.
func ErrAsyncTimeout(toolCallID string, duration string) *AppError {
	return NewError(ErrCodeAsyncTimeout, CategoryDependency,
		fmt.Sprintf("async operation timed out after %s", duration), ActionRetry).
		WithDetail("tool_call_id", toolCallID).WithDetail("duration", duration)
}

// ErrAsyncAlreadyCompleted creates a conflict error when operating on a completed async op.
func ErrAsyncAlreadyCompleted(toolCallID string, status string) *AppError {
	return NewError(ErrCodeAsyncAlreadyCompleted, CategoryConflict,
		"async operation has already completed", ActionNone).
		WithDetail("tool_call_id", toolCallID).WithDetail("status", status)
}

// Agent mode error constructors
//
// These provide specific, recovery-distinct errors for agent mode operations.
// Each maps to a unique code so the UI can show targeted recovery guidance.

// ErrAgentNotInMode creates a validation error when the chat is not in agent mode.
func ErrAgentNotInMode(chatID string) *AppError {
	return NewError(ErrCodeAgentNotInMode, CategoryValidation,
		"chat is not in agent mode", ActionCorrectInput).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "This chat is in LLM mode. Switch to agent mode to use this feature.")
}

// ErrAgentNoActiveRun creates a validation error when there is no active agent run.
func ErrAgentNoActiveRun(chatID string) *AppError {
	return NewError(ErrCodeAgentNoActiveRun, CategoryValidation,
		"no active agent run", ActionCorrectInput).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "No agent is currently running. Start a new agent session first.")
}

// ErrAgentAlreadyActive creates a conflict error when agent mode is already active.
func ErrAgentAlreadyActive(chatID string) *AppError {
	return NewError(ErrCodeAgentAlreadyActive, CategoryConflict,
		"chat already has an active agent run", ActionCorrectInput).
		WithDetail("chat_id", chatID).
		WithDetail("user_message", "An agent is already running in this chat. Stop it first or send a follow-up message.")
}

// ErrAgentManagerUnavailable creates a dependency error when agent-manager is not reachable.
func ErrAgentManagerUnavailable() *AppError {
	return NewError(ErrCodeAgentManagerUnavailable, CategoryDependency,
		"agent-manager service is not available", ActionCheckDependency).
		WithDetail("service", "agent-manager").
		WithDetail("user_message", "The agent-manager service is not running. Please start it with: vrooli scenario start agent-manager")
}

// ErrAgentRunNotFound creates a dependency error when a run ID is not found in agent-manager.
func ErrAgentRunNotFound(runID string) *AppError {
	return NewError(ErrCodeAgentRunNotFound, CategoryDependency,
		"agent run not found in agent-manager", ActionCorrectInput).
		WithDetail("run_id", runID).
		WithDetail("user_message", "The agent run could not be found. It may have expired. Try starting a new agent session.")
}

// ErrAgentProtoParseFailed creates a dependency error when proto response parsing fails.
func ErrAgentProtoParseFailed(operation string, err error) *AppError {
	return NewError(ErrCodeAgentProtoParseFailed, CategoryDependency,
		fmt.Sprintf("failed to parse agent-manager response for %s", operation), ActionEscalate).
		WithCause(err).WithDetail("operation", operation)
}
