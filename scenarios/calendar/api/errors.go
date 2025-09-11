package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// ErrorCode represents a standardized error code
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden           ErrorCode = "FORBIDDEN"
	ErrCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrCodeConflict            ErrorCode = "CONFLICT"
	ErrCodeValidationFailed    ErrorCode = "VALIDATION_FAILED"
	ErrCodeRateLimited         ErrorCode = "RATE_LIMITED"

	// Server errors (5xx)
	ErrCodeInternalServer      ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeDatabaseError       ErrorCode = "DATABASE_ERROR"
	ErrCodeExternalServiceError ErrorCode = "EXTERNAL_SERVICE_ERROR"
	ErrCodeTimeout             ErrorCode = "TIMEOUT"

	// Calendar specific errors
	ErrCodeEventConflict       ErrorCode = "EVENT_CONFLICT"
	ErrCodeInvalidTimeRange    ErrorCode = "INVALID_TIME_RANGE"
	ErrCodeRecurrenceError     ErrorCode = "RECURRENCE_ERROR"
	ErrCodeReminderError       ErrorCode = "REMINDER_ERROR"
	ErrCodeSchedulingError     ErrorCode = "SCHEDULING_ERROR"
)

// CalendarError represents a structured error in the calendar system
type CalendarError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Fields     map[string]string      `json:"fields,omitempty"`
	Cause      error                  `json:"-"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	Component  string                 `json:"component,omitempty"`
	Function   string                 `json:"function,omitempty"`
	File       string                 `json:"file,omitempty"`
	Line       int                    `json:"line,omitempty"`
	StackTrace []string               `json:"stack_trace,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface
func (e *CalendarError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *CalendarError) Unwrap() error {
	return e.Cause
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *CalendarError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeBadRequest, ErrCodeValidationFailed, ErrCodeInvalidTimeRange:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeConflict, ErrCodeEventConflict:
		return http.StatusConflict
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// ErrorBuilder provides a fluent interface for building CalendarError instances
type ErrorBuilder struct {
	error *CalendarError
}

// NewError creates a new ErrorBuilder
func NewError(code ErrorCode, message string) *ErrorBuilder {
	// Get caller information
	_, file, line, _ := runtime.Caller(1)
	function := getFunctionName(2)

	err := &CalendarError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Function:  function,
		File:      getShortFilename(file),
		Line:      line,
		Fields:    make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	return &ErrorBuilder{error: err}
}

// WithDetails adds additional details to the error
func (b *ErrorBuilder) WithDetails(details string) *ErrorBuilder {
	b.error.Details = details
	return b
}

// WithDetailsf adds formatted details to the error
func (b *ErrorBuilder) WithDetailsf(format string, args ...interface{}) *ErrorBuilder {
	b.error.Details = fmt.Sprintf(format, args...)
	return b
}

// WithCause adds the underlying cause error
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.error.Cause = cause
	return b
}

// WithField adds a validation field error
func (b *ErrorBuilder) WithField(field, message string) *ErrorBuilder {
	b.error.Fields[field] = message
	return b
}

// WithFields adds multiple validation field errors
func (b *ErrorBuilder) WithFields(fields map[string]string) *ErrorBuilder {
	for k, v := range fields {
		b.error.Fields[k] = v
	}
	return b
}

// WithRequestID adds request context
func (b *ErrorBuilder) WithRequestID(requestID string) *ErrorBuilder {
	b.error.RequestID = requestID
	return b
}

// WithUserID adds user context
func (b *ErrorBuilder) WithUserID(userID string) *ErrorBuilder {
	b.error.UserID = userID
	return b
}

// WithComponent adds component context
func (b *ErrorBuilder) WithComponent(component string) *ErrorBuilder {
	b.error.Component = component
	return b
}

// WithMetadata adds custom metadata
func (b *ErrorBuilder) WithMetadata(key string, value interface{}) *ErrorBuilder {
	b.error.Metadata[key] = value
	return b
}

// WithStackTrace captures the full stack trace
func (b *ErrorBuilder) WithStackTrace() *ErrorBuilder {
	b.error.StackTrace = captureStackTrace(2)
	return b
}

// Build returns the constructed CalendarError
func (b *ErrorBuilder) Build() *CalendarError {
	return b.error
}

// Common error constructors

// NewValidationError creates a validation error with field details
func NewValidationError(message string, fields map[string]string) *CalendarError {
	return NewError(ErrCodeValidationFailed, message).
		WithFields(fields).
		Build()
}

// NewDatabaseError creates a database error
func NewDatabaseError(operation string, cause error) *CalendarError {
	return NewError(ErrCodeDatabaseError, "Database operation failed").
		WithDetailsf("Failed to %s", operation).
		WithCause(cause).
		WithComponent("database").
		Build()
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(details string) *CalendarError {
	return NewError(ErrCodeUnauthorized, "Authentication failed").
		WithDetails(details).
		WithComponent("auth").
		Build()
}

// NewEventConflictError creates an event scheduling conflict error
func NewEventConflictError(eventTitle string, conflictTime time.Time) *CalendarError {
	return NewError(ErrCodeEventConflict, "Event scheduling conflict").
		WithDetailsf("Event '%s' conflicts with existing event at %s", eventTitle, conflictTime.Format(time.RFC3339)).
		WithComponent("scheduler").
		WithMetadata("conflict_time", conflictTime).
		WithMetadata("event_title", eventTitle).
		Build()
}

// NewTimeRangeError creates an invalid time range error
func NewTimeRangeError(startTime, endTime time.Time) *CalendarError {
	return NewError(ErrCodeInvalidTimeRange, "Invalid time range").
		WithDetailsf("Start time %s must be before end time %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)).
		WithComponent("validation").
		WithMetadata("start_time", startTime).
		WithMetadata("end_time", endTime).
		Build()
}

// NewExternalServiceError creates an external service error
func NewExternalServiceError(serviceName string, cause error) *CalendarError {
	return NewError(ErrCodeExternalServiceError, "External service error").
		WithDetailsf("Failed to communicate with %s", serviceName).
		WithCause(cause).
		WithComponent("external").
		WithMetadata("service", serviceName).
		Build()
}

// Error response structure for HTTP API
type ErrorResponse struct {
	Error     *CalendarError `json:"error"`
	Success   bool           `json:"success"`
	Timestamp time.Time      `json:"timestamp"`
}

// SendErrorResponse sends a structured error response to the client
func SendErrorResponse(w http.ResponseWriter, calendarErr *CalendarError) {
	// Log the error
	logLevel := Error
	if calendarErr.HTTPStatus() < 500 {
		logLevel = Warn // Client errors are warnings
	}

	logBuilder := logLevel("http").
		Message("HTTP error response").
		WithError(calendarErr).
		WithField("error_code", string(calendarErr.Code)).
		WithField("http_status", calendarErr.HTTPStatus())

	if calendarErr.RequestID != "" {
		logBuilder = logBuilder.WithRequest(calendarErr.RequestID)
	}
	if calendarErr.UserID != "" {
		logBuilder = logBuilder.WithUser(calendarErr.UserID)
	}
	if calendarErr.Component != "" {
		logBuilder = logBuilder.WithField("component", calendarErr.Component)
	}

	logBuilder.Log()

	// Prepare response
	response := ErrorResponse{
		Error:     calendarErr,
		Success:   false,
		Timestamp: time.Now().UTC(),
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(calendarErr.HTTPStatus())

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback if JSON encoding fails
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		Error("http").
			Message("Failed to encode error response").
			WithError(err).
			Log()
	}
}

// HandlePanic recovers from panics and sends appropriate error responses
func HandlePanic(w http.ResponseWriter, r *http.Request) {
	if recovered := recover(); recovered != nil {
		var err error
		switch x := recovered.(type) {
		case string:
			err = fmt.Errorf(x)
		case error:
			err = x
		default:
			err = fmt.Errorf("unknown panic: %v", recovered)
		}

		panicErr := NewError(ErrCodeInternalServer, "Internal server error").
			WithDetails("An unexpected error occurred").
			WithCause(err).
			WithComponent("panic_handler").
			WithStackTrace().
			Build()

		Fatal("panic").
			Message("Panic recovered").
			WithError(err).
			WithHTTP(r.Method, r.URL.Path, 500).
			WithField("panic_value", fmt.Sprintf("%v", recovered)).
			Log()

		SendErrorResponse(w, panicErr)
	}
}

// WrapHandler wraps an HTTP handler with panic recovery
func WrapHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer HandlePanic(w, r)
		handler(w, r)
	}
}

// Helper functions

func captureStackTrace(skip int) []string {
	var trace []string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		var funcName string
		if fn != nil {
			funcName = fn.Name()
		}

		trace = append(trace, fmt.Sprintf("%s:%d %s", getShortFilename(file), line, funcName))
		
		// Limit stack trace depth
		if len(trace) >= 10 {
			break
		}
	}
	return trace
}

// IsCalendarError checks if an error is a CalendarError
func IsCalendarError(err error) bool {
	_, ok := err.(*CalendarError)
	return ok
}

// AsCalendarError converts an error to CalendarError if possible
func AsCalendarError(err error) (*CalendarError, bool) {
	if calErr, ok := err.(*CalendarError); ok {
		return calErr, true
	}
	return nil, false
}

// WrapError wraps a generic error as a CalendarError
func WrapError(err error, code ErrorCode, message string) *CalendarError {
	return NewError(code, message).
		WithCause(err).
		Build()
}

// Error middleware for HTTP handlers
func ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				HandlePanic(w, r)
			}
		}()

		next.ServeHTTP(w, r)
	})
}