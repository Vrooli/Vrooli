package errors

import (
	"context"
	"fmt"
)

// ContextKey represents keys used in error context
type ContextKey string

const (
	// Context keys for error tracking
	RequestIDKey    ContextKey = "request_id"
	UserIDKey       ContextKey = "user_id"
	OperationKey    ContextKey = "operation"
	ResourceIDKey   ContextKey = "resource_id"
	WorkflowIDKey   ContextKey = "workflow_id"
)

// ContextualError wraps an error with additional context information
type ContextualError struct {
	*AppError
	Context map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *ContextualError) Error() string {
	if len(e.Context) > 0 {
		return fmt.Sprintf("%s [context: %+v]", e.AppError.Error(), e.Context)
	}
	return e.AppError.Error()
}

// Unwrap returns the underlying AppError
func (e *ContextualError) Unwrap() error {
	return e.AppError
}

// WithContext adds context to an error
func WithContext(err error, ctx context.Context) error {
	// If it's already an AppError, enhance it with context
	if appErr, ok := AsAppError(err); ok {
		contextual := &ContextualError{
			AppError: appErr,
			Context:  extractContext(ctx),
		}
		return contextual
	}
	
	// Convert regular error to AppError with context
	appErr := NewInternalError(err.Error(), err)
	contextual := &ContextualError{
		AppError: appErr,
		Context:  extractContext(ctx),
	}
	return contextual
}

// WithOperation adds operation context to an error
func WithOperation(err error, operation string) *AppError {
	if appErr, ok := AsAppError(err); ok {
		return appErr.WithDetail("operation", operation)
	}
	
	return NewInternalError(err.Error(), err).WithDetail("operation", operation)
}

// WithResource adds resource context to an error
func WithResource(err error, resource string, id interface{}) *AppError {
	if appErr, ok := AsAppError(err); ok {
		return appErr.WithDetail("resource", resource).WithDetail("resource_id", id)
	}
	
	return NewInternalError(err.Error(), err).
		WithDetail("resource", resource).
		WithDetail("resource_id", id)
}

// WrapDatabase wraps database errors with consistent handling
func WrapDatabase(err error, operation string) error {
	if err == nil {
		return nil
	}
	
	// Check for common database error patterns
	errMsg := err.Error()
	
	if contains(errMsg, "no rows") {
		return NewNotFoundError("resource", "unknown").
			WithCause(err).
			WithDetail("operation", operation)
	}
	
	if contains(errMsg, "duplicate key") || contains(errMsg, "unique constraint") {
		return NewConflictError("resource already exists").
			WithCause(err).
			WithDetail("operation", operation)
	}
	
	if contains(errMsg, "foreign key") {
		return NewValidationError("invalid reference").
			WithCause(err).
			WithDetail("operation", operation)
	}
	
	// Generic database error
	return NewDatabaseError(fmt.Sprintf("database operation failed: %s", operation), err)
}

// WrapExternalAPI wraps external API errors
func WrapExternalAPI(err error, service string, operation string) error {
	if err == nil {
		return nil
	}
	
	return NewExternalAPIError(service, err).
		WithDetail("operation", operation)
}

// WrapValidation wraps validation errors with field information
func WrapValidation(err error, field string, value interface{}) error {
	if err == nil {
		return nil
	}
	
	if appErr, ok := AsAppError(err); ok && appErr.Type == ValidationError {
		return appErr.WithDetail(field, value)
	}
	
	return NewValidationError(err.Error()).
		WithDetail("field", field).
		WithDetail("value", value)
}

// Helper functions

// extractContext extracts relevant information from context
func extractContext(ctx context.Context) map[string]interface{} {
	contextMap := make(map[string]interface{})
	
	// Extract common context values
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		contextMap["request_id"] = requestID
	}
	
	if userID := ctx.Value(UserIDKey); userID != nil {
		contextMap["user_id"] = userID
	}
	
	if operation := ctx.Value(OperationKey); operation != nil {
		contextMap["operation"] = operation
	}
	
	if resourceID := ctx.Value(ResourceIDKey); resourceID != nil {
		contextMap["resource_id"] = resourceID
	}
	
	if workflowID := ctx.Value(WorkflowIDKey); workflowID != nil {
		contextMap["workflow_id"] = workflowID
	}
	
	return contextMap
}

// contains checks if string contains substring (simple implementation)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

// containsSubstring is a simple substring check
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Chain allows chaining error operations
func Chain(err error, operations ...func(error) error) error {
	if err == nil {
		return nil
	}
	
	result := err
	for _, op := range operations {
		result = op(result)
	}
	return result
}

// Context builders for common scenarios

// DatabaseContext creates context for database operations
func DatabaseContext(ctx context.Context, operation string, resource string, id interface{}) context.Context {
	ctx = context.WithValue(ctx, OperationKey, operation)
	ctx = context.WithValue(ctx, ResourceIDKey, fmt.Sprintf("%s:%v", resource, id))
	return ctx
}

// APIContext creates context for API operations
func APIContext(ctx context.Context, operation string, workflowID string) context.Context {
	ctx = context.WithValue(ctx, OperationKey, operation)
	if workflowID != "" {
		ctx = context.WithValue(ctx, WorkflowIDKey, workflowID)
	}
	return ctx
}