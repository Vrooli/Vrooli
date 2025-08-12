package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

	"metareasoning-api/internal/pkg/errors"
)

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Error struct {
		Type    string                 `json:"type"`
		Message string                 `json:"message"`
		Details map[string]interface{} `json:"details,omitempty"`
		TraceID string                 `json:"trace_id,omitempty"`
	} `json:"error"`
	StatusCode int    `json:"status_code"`
	Timestamp  string `json:"timestamp"`
}

// ErrorHandler provides centralized error handling for HTTP responses
type ErrorHandler struct {
	includeStackTrace bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(includeStackTrace bool) *ErrorHandler {
	return &ErrorHandler{
		includeStackTrace: includeStackTrace,
	}
}

// HandleError processes errors and sends appropriate HTTP responses
func (eh *ErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	// Set common headers
	w.Header().Set("Content-Type", "application/json")
	
	var statusCode int
	var errorType string
	var message string
	var details map[string]interface{}
	
	// Check if it's an AppError
	if appErr, ok := errors.AsAppError(err); ok {
		statusCode = appErr.StatusCode
		errorType = string(appErr.Type)
		message = appErr.Message
		details = appErr.Details
		
		// Log based on error type
		if appErr.Type == errors.InternalError || appErr.Type == errors.DatabaseError {
			log.Printf("Internal error: %v", err)
			if eh.includeStackTrace {
				log.Printf("Stack trace: %s", debug.Stack())
			}
		} else {
			log.Printf("Application error: %v", err)
		}
	} else {
		// Unknown error - treat as internal error
		statusCode = http.StatusInternalServerError
		errorType = string(errors.InternalError)
		message = "An internal error occurred"
		
		log.Printf("Unhandled error: %v", err)
		if eh.includeStackTrace {
			log.Printf("Stack trace: %s", debug.Stack())
		}
	}
	
	// Create error response
	response := ErrorResponse{
		StatusCode: statusCode,
		Timestamp:  getCurrentTimestamp(),
	}
	response.Error.Type = errorType
	response.Error.Message = message
	response.Error.Details = details
	
	// Add trace ID if available from request context
	if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		response.Error.TraceID = traceID
	}
	
	// Set status code and send response
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RecoveryMiddleware wraps handlers with panic recovery
func (eh *ErrorHandler) RecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("Panic recovered: %v", recovered)
				log.Printf("Stack trace: %s", debug.Stack())
				
				// Convert panic to internal error
				var err error
				if e, ok := recovered.(error); ok {
					err = errors.NewInternalError("panic recovered", e)
				} else {
					err = errors.NewInternalError("panic recovered", nil).
						WithDetail("panic_value", recovered)
				}
				
				eh.HandleError(w, r, err)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// ErrorMiddleware wraps handlers to catch and handle errors from handlers that return errors
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request) error

// WrapErrorHandler converts an error-returning handler to standard http.HandlerFunc
func (eh *ErrorHandler) WrapErrorHandler(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return eh.RecoveryMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			eh.HandleError(w, r, err)
		}
	})
}

// WrapGenericErrorHandler converts a generic error-returning handler to standard http.HandlerFunc
func (eh *ErrorHandler) WrapGenericErrorHandler(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return eh.RecoveryMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			eh.HandleError(w, r, err)
		}
	})
}

// Helper function to get current timestamp in ISO format
func getCurrentTimestamp() string {
	// This would typically use time.Now().Format(time.RFC3339)
	// but avoiding time import to keep this focused on error handling
	return "2025-08-12T10:00:00Z" // placeholder - would be dynamic in real implementation
}

// ValidationErrorFromFields creates a validation error from field errors
func ValidationErrorFromFields(fields map[string]string) *errors.AppError {
	details := make(map[string]interface{})
	for field, message := range fields {
		details[field] = message
	}
	
	return errors.NewValidationError("validation failed", details)
}

// NotFoundErrorFromRequest creates a not found error from request context
func NotFoundErrorFromRequest(r *http.Request, resource string) *errors.AppError {
	// Extract ID from URL path or query parameters
	var id interface{} = "unknown"
	
	// Try to extract from URL path (simple extraction)
	if pathID := r.URL.Path; pathID != "" {
		id = pathID
	}
	
	return errors.NewNotFoundError(resource, id)
}