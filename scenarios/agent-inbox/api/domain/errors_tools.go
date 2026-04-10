// Package domain defines the core domain types for the Agent Inbox scenario.
// This file contains error constructors for tool and async operations.
package domain

import "fmt"

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
