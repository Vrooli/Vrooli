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
	ErrCodeInvalidInput      ErrorCode = "V001"
	ErrCodeMissingField      ErrorCode = "V002"
	ErrCodeInvalidUUID       ErrorCode = "V003"
	ErrCodeInvalidRole       ErrorCode = "V004"
	ErrCodeInvalidViewMode   ErrorCode = "V005"
	ErrCodeEmptyContent      ErrorCode = "V006"
	ErrCodeMissingToolCallID ErrorCode = "V007"
	ErrCodeInvalidJSON       ErrorCode = "V008"
	ErrCodeNoFieldsToUpdate  ErrorCode = "V009"
	ErrCodeInvalidColor      ErrorCode = "V010"
	ErrCodeNoMessagesInChat  ErrorCode = "V011"
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
	ErrCodeDatabaseUnavailable   ErrorCode = "D001"
	ErrCodeDatabaseQueryFailed   ErrorCode = "D002"
	ErrCodeOpenRouterUnavailable ErrorCode = "D003"
	ErrCodeOpenRouterError       ErrorCode = "D004"
	ErrCodeOllamaUnavailable     ErrorCode = "D005"
	ErrCodeAgentManagerError     ErrorCode = "D006"
	ErrCodeToolExecutionFailed   ErrorCode = "D007"
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
