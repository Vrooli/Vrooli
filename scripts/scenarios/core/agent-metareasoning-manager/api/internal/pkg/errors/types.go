package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// Business logic errors
	ValidationError     ErrorType = "validation_error"
	NotFoundError      ErrorType = "not_found"
	ConflictError      ErrorType = "conflict"
	UnauthorizedError  ErrorType = "unauthorized"
	ForbiddenError     ErrorType = "forbidden"
	
	// Infrastructure errors
	DatabaseError      ErrorType = "database_error"
	ExternalAPIError   ErrorType = "external_api_error"
	NetworkError       ErrorType = "network_error"
	
	// System errors
	InternalError      ErrorType = "internal_error"
	ConfigurationError ErrorType = "configuration_error"
	TimeoutError       ErrorType = "timeout_error"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	StatusCode int                    `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause sets the underlying cause
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// HTTP status code mapping
func (t ErrorType) StatusCode() int {
	switch t {
	case ValidationError:
		return http.StatusBadRequest
	case NotFoundError:
		return http.StatusNotFound
	case ConflictError:
		return http.StatusConflict
	case UnauthorizedError:
		return http.StatusUnauthorized
	case ForbiddenError:
		return http.StatusForbidden
	case DatabaseError, ExternalAPIError, NetworkError:
		return http.StatusServiceUnavailable
	case TimeoutError:
		return http.StatusRequestTimeout
	case ConfigurationError, InternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Constructor functions for common error types

// NewValidationError creates a validation error
func NewValidationError(message string, details ...map[string]interface{}) *AppError {
	err := &AppError{
		Type:       ValidationError,
		Message:    message,
		StatusCode: ValidationError.StatusCode(),
	}
	
	if len(details) > 0 {
		err.Details = details[0]
	}
	
	return err
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string, id interface{}) *AppError {
	return &AppError{
		Type:       NotFoundError,
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: NotFoundError.StatusCode(),
		Details:    map[string]interface{}{"resource": resource, "id": id},
	}
}

// NewConflictError creates a conflict error
func NewConflictError(message string, details ...map[string]interface{}) *AppError {
	err := &AppError{
		Type:       ConflictError,
		Message:    message,
		StatusCode: ConflictError.StatusCode(),
	}
	
	if len(details) > 0 {
		err.Details = details[0]
	}
	
	return err
}

// NewDatabaseError creates a database error
func NewDatabaseError(message string, cause error) *AppError {
	return &AppError{
		Type:       DatabaseError,
		Message:    message,
		Cause:      cause,
		StatusCode: DatabaseError.StatusCode(),
	}
}

// NewExternalAPIError creates an external API error
func NewExternalAPIError(service string, cause error) *AppError {
	return &AppError{
		Type:       ExternalAPIError,
		Message:    fmt.Sprintf("external service %s error", service),
		Cause:      cause,
		StatusCode: ExternalAPIError.StatusCode(),
		Details:    map[string]interface{}{"service": service},
	}
}

// NewInternalError creates an internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:       InternalError,
		Message:    message,
		Cause:      cause,
		StatusCode: InternalError.StatusCode(),
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, duration interface{}) *AppError {
	return &AppError{
		Type:       TimeoutError,
		Message:    fmt.Sprintf("operation %s timed out", operation),
		StatusCode: TimeoutError.StatusCode(),
		Details:    map[string]interface{}{"operation": operation, "duration": duration},
	}
}

// Error checking helpers

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// AsAppError converts an error to AppError if possible
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	if appErr, ok := AsAppError(err); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// IsType checks if error is of specific type
func IsType(err error, errType ErrorType) bool {
	if appErr, ok := AsAppError(err); ok {
		return appErr.Type == errType
	}
	return false
}