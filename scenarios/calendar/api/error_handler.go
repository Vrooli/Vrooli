package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	// Client errors (4xx)
	ErrorCodeBadRequest      ErrorCode = "BAD_REQUEST"
	ErrorCodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden       ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound        ErrorCode = "NOT_FOUND"
	ErrorCodeConflict        ErrorCode = "CONFLICT"
	ErrorCodeValidationError ErrorCode = "VALIDATION_ERROR"
	ErrorCodeRateLimit       ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Server errors (5xx)
	ErrorCodeInternalServer       ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrorCodeServiceUnavailable   ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeTimeout              ErrorCode = "TIMEOUT"
	ErrorCodeDatabaseError        ErrorCode = "DATABASE_ERROR"
	ErrorCodeExternalServiceError ErrorCode = "EXTERNAL_SERVICE_ERROR"
)

// APIError represents a structured API error response
type APIError struct {
	Code      ErrorCode   `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
	Path      string      `json:"path,omitempty"`
}

// Error implements the error interface
func (e APIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ValidationError represents validation errors for request data
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

// ErrorResponse represents the complete error response structure
type ErrorResponse struct {
	Error       APIError          `json:"error"`
	Validations []ValidationError `json:"validations,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
}

// NewAPIError creates a new APIError with current timestamp
func NewAPIError(code ErrorCode, message string, details interface{}) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: getCurrentTimestamp(),
	}
}

// NewValidationError creates a validation error
func NewValidationError(field, value, message string) ValidationError {
	return ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	includeStackTrace bool
	logErrors         bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(includeStackTrace, logErrors bool) *ErrorHandler {
	return &ErrorHandler{
		includeStackTrace: includeStackTrace,
		logErrors:         logErrors,
	}
}

// HandleError processes an error and sends an appropriate HTTP response
func (eh *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var apiError *APIError
	var validationErrors []ValidationError
	var statusCode int

	switch e := err.(type) {
	case *APIError:
		apiError = e
		statusCode = eh.getStatusCodeForErrorCode(e.Code)
	case ValidationErrors:
		apiError = NewAPIError(ErrorCodeValidationError, "Validation failed", nil)
		validationErrors = []ValidationError(e)
		statusCode = http.StatusBadRequest
	default:
		// Wrap unknown errors
		apiError = eh.wrapUnknownError(err)
		statusCode = http.StatusInternalServerError
	}

	// Add request context
	apiError.Path = r.URL.Path
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		apiError.RequestID = requestID
	}

	// Log error if enabled
	if eh.logErrors {
		eh.logError(r, apiError, err)
	}

	// Create response
	errorResponse := ErrorResponse{
		Error:       *apiError,
		Validations: validationErrors,
	}

	// Add debug context in development
	if eh.includeStackTrace && statusCode >= 500 {
		errorResponse.Context = map[string]string{
			"go_version": runtime.Version(),
			"stack":      getStackTrace(),
		}
	}

	// Send response
	eh.sendErrorResponse(w, statusCode, errorResponse)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var messages []string
	for _, v := range ve {
		messages = append(messages, fmt.Sprintf("%s: %s", v.Field, v.Message))
	}
	return "Validation errors: " + strings.Join(messages, ", ")
}

// Common error constructors
func BadRequestError(message string, details interface{}) *APIError {
	return NewAPIError(ErrorCodeBadRequest, message, details)
}

func UnauthorizedError(message string) *APIError {
	return NewAPIError(ErrorCodeUnauthorized, message, nil)
}

func ForbiddenError(message string) *APIError {
	return NewAPIError(ErrorCodeForbidden, message, nil)
}

func NotFoundError(resource string) *APIError {
	return NewAPIError(ErrorCodeNotFound, fmt.Sprintf("%s not found", resource), nil)
}

func ConflictError(message string, details interface{}) *APIError {
	return NewAPIError(ErrorCodeConflict, message, details)
}

func InternalServerError(message string, details interface{}) *APIError {
	return NewAPIError(ErrorCodeInternalServer, message, details)
}

func DatabaseError(operation string, err error) *APIError {
	return NewAPIError(ErrorCodeDatabaseError,
		fmt.Sprintf("Database %s failed", operation),
		map[string]string{"original_error": err.Error()})
}

func ExternalServiceError(service string, err error) *APIError {
	return NewAPIError(ErrorCodeExternalServiceError,
		fmt.Sprintf("External service %s unavailable", service),
		map[string]string{"service": service, "error": err.Error()})
}

func ServiceUnavailableError(service string) *APIError {
	return NewAPIError(ErrorCodeServiceUnavailable,
		fmt.Sprintf("Service %s is temporarily unavailable", service), nil)
}

// Helper methods
func (eh *ErrorHandler) getStatusCodeForErrorCode(code ErrorCode) int {
	switch code {
	case ErrorCodeBadRequest, ErrorCodeValidationError:
		return http.StatusBadRequest
	case ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeNotFound:
		return http.StatusNotFound
	case ErrorCodeConflict:
		return http.StatusConflict
	case ErrorCodeRateLimit:
		return http.StatusTooManyRequests
	case ErrorCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func (eh *ErrorHandler) wrapUnknownError(err error) *APIError {
	message := "An unexpected error occurred"
	details := map[string]string{"type": "unknown_error"}

	if eh.includeStackTrace {
		details["original_error"] = err.Error()
	}

	return NewAPIError(ErrorCodeInternalServer, message, details)
}

func (eh *ErrorHandler) logError(r *http.Request, apiError *APIError, originalErr error) {
	log.Printf("[ERROR] %s %s - %s: %s",
		r.Method, r.URL.Path, apiError.Code, apiError.Message)

	if originalErr != nil && originalErr.Error() != apiError.Message {
		log.Printf("[ERROR] Original error: %v", originalErr)
	}

	// Log additional context
	if apiError.RequestID != "" {
		log.Printf("[ERROR] Request ID: %s", apiError.RequestID)
	}

	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		log.Printf("[ERROR] User Agent: %s", userAgent)
	}
}

func (eh *ErrorHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, errorResponse ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		// Fallback to plain text
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Internal server error")
	}
}

// Utility functions
func getCurrentTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z07:00")
}

func getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// Middleware for panic recovery
func RecoveryMiddleware(eh *ErrorHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("[PANIC] %v", err)

					var apiError *APIError
					if e, ok := err.(error); ok {
						apiError = InternalServerError("Server panic occurred", map[string]string{
							"panic": e.Error(),
						})
					} else {
						apiError = InternalServerError("Server panic occurred", map[string]interface{}{
							"panic": err,
						})
					}

					eh.HandleError(w, r, apiError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Request validation helpers
func ValidateRequired(field, value string) *ValidationError {
	if value == "" {
		ve := NewValidationError(field, value, "This field is required")
		return &ve
	}
	return nil
}

func ValidateEmail(field, email string) *ValidationError {
	if email == "" {
		return nil // Use ValidateRequired for required check
	}

	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		ve := NewValidationError(field, email, "Invalid email format")
		return &ve
	}
	return nil
}

func ValidateStringLength(field, value string, minLen, maxLen int) *ValidationError {
	if len(value) < minLen {
		ve := NewValidationError(field, value, fmt.Sprintf("Must be at least %d characters", minLen))
		return &ve
	}

	if maxLen > 0 && len(value) > maxLen {
		ve := NewValidationError(field, value, fmt.Sprintf("Must be no more than %d characters", maxLen))
		return &ve
	}

	return nil
}
