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
	ErrCodeAgentRunBusy       ErrorCode = "V015" // Agent run is still in progress
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

// Convenience constructors for common validation errors

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
