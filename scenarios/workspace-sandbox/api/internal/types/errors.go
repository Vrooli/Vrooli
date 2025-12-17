// Package types provides shared types for workspace sandboxes.
// This file contains domain error types that represent business-level failures.
package types

import (
	"fmt"
	"net/http"
)

// --- Domain Error Interface ---

// DomainError represents an error that has business meaning and can be
// mapped to an HTTP status code. This interface enables consistent
// error handling across the API.
type DomainError interface {
	error
	// HTTPStatus returns the appropriate HTTP status code for this error.
	HTTPStatus() int
	// IsRetryable indicates if the operation could succeed on retry.
	IsRetryable() bool
}

// --- Concrete Domain Errors ---

// NotFoundError indicates a requested resource was not found.
type NotFoundError struct {
	Resource string // e.g., "sandbox", "file"
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
	}
	return fmt.Sprintf("not found: %s", e.ID)
}

func (e *NotFoundError) HTTPStatus() int {
	return http.StatusNotFound
}

func (e *NotFoundError) IsRetryable() bool {
	return false
}

// NewNotFoundError creates a NotFoundError for a sandbox.
func NewNotFoundError(id string) *NotFoundError {
	return &NotFoundError{Resource: "sandbox", ID: id}
}

// ScopeConflictError indicates a scope path conflict with existing sandboxes.
type ScopeConflictError struct {
	Conflicts []PathConflict
}

func (e *ScopeConflictError) Error() string {
	if len(e.Conflicts) == 1 {
		c := e.Conflicts[0]
		return fmt.Sprintf("scope conflict with sandbox %s (scope: %s)", c.ExistingID, c.ExistingScope)
	}
	return fmt.Sprintf("scope conflicts with %d existing sandboxes", len(e.Conflicts))
}

func (e *ScopeConflictError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *ScopeConflictError) IsRetryable() bool {
	return false // Conflicts are deterministic
}

// ValidationError indicates input validation failed.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) HTTPStatus() int {
	return http.StatusBadRequest
}

func (e *ValidationError) IsRetryable() bool {
	return false
}

// NewValidationError creates a ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// StateError indicates an operation was attempted in an invalid state.
// This wraps InvalidTransitionError for HTTP response purposes.
type StateError struct {
	Message string
	Wrapped error
}

func (e *StateError) Error() string {
	return e.Message
}

func (e *StateError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *StateError) IsRetryable() bool {
	return false
}

func (e *StateError) Unwrap() error {
	return e.Wrapped
}

// NewStateError creates a StateError from an InvalidTransitionError.
func NewStateError(err *InvalidTransitionError) *StateError {
	return &StateError{
		Message: err.Error(),
		Wrapped: err,
	}
}

// DriverError indicates an error from the filesystem driver.
type DriverError struct {
	Operation string // e.g., "mount", "unmount", "cleanup"
	Message   string
	Wrapped   error
}

func (e *DriverError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("driver %s failed: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("driver error: %s", e.Message)
}

func (e *DriverError) HTTPStatus() int {
	return http.StatusInternalServerError
}

func (e *DriverError) IsRetryable() bool {
	return true // Driver operations might succeed on retry
}

func (e *DriverError) Unwrap() error {
	return e.Wrapped
}

// NewDriverError creates a DriverError.
func NewDriverError(operation string, err error) *DriverError {
	return &DriverError{
		Operation: operation,
		Message:   err.Error(),
		Wrapped:   err,
	}
}
