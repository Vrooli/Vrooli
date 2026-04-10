// Package domain defines the core domain types for the Agent Inbox scenario.
// This file contains error constructors for AI/external service operations.
package domain

import "fmt"

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

// ErrOllamaUnavailable creates a dependency error for Ollama failures.
// Note: Ollama is optional, so this may trigger graceful degradation.
func ErrOllamaUnavailable(err error) *AppError {
	return NewError(ErrCodeOllamaUnavailable, CategoryDependency,
		"naming service temporarily unavailable", ActionRetryWithBackoff).
		WithCause(err)
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

// ErrAgentManagerError creates a dependency error for agent-manager failures.
func ErrAgentManagerError(operation string, err error) *AppError {
	return NewError(ErrCodeAgentManagerError, CategoryDependency,
		fmt.Sprintf("agent manager error: %s", operation), ActionCheckDependency).
		WithCause(err).WithDetail("operation", operation)
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
