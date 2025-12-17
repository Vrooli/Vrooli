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
		return fmt.Sprintf("%s not found: %s. Verify the ID is correct and the %s hasn't been deleted.", e.Resource, e.ID, e.Resource)
	}
	return fmt.Sprintf("resource not found: %s. Verify the ID is correct.", e.ID)
}

func (e *NotFoundError) HTTPStatus() int {
	return http.StatusNotFound
}

func (e *NotFoundError) IsRetryable() bool {
	return false
}

// Hint returns actionable guidance for resolving this error.
func (e *NotFoundError) Hint() string {
	return fmt.Sprintf("Use GET /api/v1/sandboxes to list available sandboxes and their IDs.")
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
		return fmt.Sprintf("scope conflict: path overlaps with sandbox %s (scope: %s). Sandboxes cannot have overlapping paths.", c.ExistingID, c.ExistingScope)
	}
	return fmt.Sprintf("scope conflicts with %d existing sandboxes. Sandboxes cannot have overlapping paths.", len(e.Conflicts))
}

func (e *ScopeConflictError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *ScopeConflictError) IsRetryable() bool {
	return false // Conflicts are deterministic
}

// Hint returns actionable guidance for resolving this error.
func (e *ScopeConflictError) Hint() string {
	return "Either delete or stop the conflicting sandbox, or choose a non-overlapping scope path."
}

// ValidationError indicates input validation failed.
type ValidationError struct {
	Field   string
	Message string
	Hint    string // Optional hint for resolution
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) HTTPStatus() int {
	return http.StatusBadRequest
}

func (e *ValidationError) IsRetryable() bool {
	return false
}

// GetHint returns actionable guidance for resolving this error.
func (e *ValidationError) GetHint() string {
	if e.Hint != "" {
		return e.Hint
	}
	return "Check the API documentation for valid field values and formats."
}

// NewValidationError creates a ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// NewValidationErrorWithHint creates a ValidationError with a resolution hint.
func NewValidationErrorWithHint(field, message, hint string) *ValidationError {
	return &ValidationError{Field: field, Message: message, Hint: hint}
}

// StateError indicates an operation was attempted in an invalid state.
// This wraps InvalidTransitionError for HTTP response purposes.
type StateError struct {
	Message       string
	Wrapped       error
	CurrentStatus Status
	Operation     string
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

// Hint returns actionable guidance for resolving this error.
func (e *StateError) Hint() string {
	switch e.CurrentStatus {
	case StatusDeleted:
		return "This sandbox has been deleted and cannot be modified."
	case StatusApproved:
		return "This sandbox has already been approved. Create a new sandbox to make additional changes."
	case StatusRejected:
		return "This sandbox has been rejected. Create a new sandbox to try again."
	case StatusError:
		return "This sandbox is in an error state. Delete it and create a new one."
	default:
		return "Check the sandbox status with GET /api/v1/sandboxes/{id} and review valid state transitions."
	}
}

// NewStateError creates a StateError from an InvalidTransitionError.
func NewStateError(err *InvalidTransitionError) *StateError {
	return &StateError{
		Message:       err.Error(),
		Wrapped:       err,
		CurrentStatus: err.Current,
		Operation:     string(err.Attempted),
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

// Hint returns actionable guidance for resolving this error.
func (e *DriverError) Hint() string {
	switch e.Operation {
	case "mount":
		return "Check that overlayfs/fuse-overlayfs is available and the user has appropriate permissions. Check GET /api/v1/driver/info for driver status."
	case "unmount":
		return "Ensure no processes are using the sandbox workspace. The sandbox may need to be forcefully cleaned up."
	case "cleanup":
		return "The sandbox directories may need manual cleanup. Check the server logs for the specific path."
	default:
		return "Check the server logs for detailed error information. This operation may succeed on retry."
	}
}

// NewDriverError creates a DriverError.
func NewDriverError(operation string, err error) *DriverError {
	return &DriverError{
		Operation: operation,
		Message:   err.Error(),
		Wrapped:   err,
	}
}

// --- Idempotency and Concurrency Errors ---

// ConcurrentModificationError indicates an update conflicted with a concurrent change.
// This occurs when optimistic locking detects a version mismatch.
type ConcurrentModificationError struct {
	Resource        string
	ID              string
	ExpectedVersion int64
	ActualVersion   int64
}

func (e *ConcurrentModificationError) Error() string {
	return fmt.Sprintf("%s %s was modified by another operation (expected version %d, found %d). Re-fetch and retry.",
		e.Resource, e.ID, e.ExpectedVersion, e.ActualVersion)
}

func (e *ConcurrentModificationError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *ConcurrentModificationError) IsRetryable() bool {
	return true // Can retry after re-fetching the current state
}

// Hint returns actionable guidance for resolving this error.
func (e *ConcurrentModificationError) Hint() string {
	return "Fetch the latest sandbox state with GET /api/v1/sandboxes/{id}, then retry your operation with the current version."
}

// NewConcurrentModificationError creates a ConcurrentModificationError.
func NewConcurrentModificationError(id string, expected, actual int64) *ConcurrentModificationError {
	return &ConcurrentModificationError{
		Resource:        "sandbox",
		ID:              id,
		ExpectedVersion: expected,
		ActualVersion:   actual,
	}
}

// AlreadyExistsError indicates a duplicate creation was attempted.
// This is used when an idempotency key collision occurs but the parameters differ.
type AlreadyExistsError struct {
	Resource       string
	IdempotencyKey string
	ExistingID     string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists with idempotency key '%s' (existing ID: %s). Cannot create duplicate with same key but different parameters.",
		e.Resource, e.IdempotencyKey, e.ExistingID)
}

func (e *AlreadyExistsError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *AlreadyExistsError) IsRetryable() bool {
	return false // Idempotency key conflict is deterministic
}

// Hint returns actionable guidance for resolving this error.
func (e *AlreadyExistsError) Hint() string {
	return fmt.Sprintf("Use the existing sandbox %s, or generate a new idempotency key for a new sandbox.", e.ExistingID)
}

// NewAlreadyExistsError creates an AlreadyExistsError.
func NewAlreadyExistsError(idempotencyKey, existingID string) *AlreadyExistsError {
	return &AlreadyExistsError{
		Resource:       "sandbox",
		IdempotencyKey: idempotencyKey,
		ExistingID:     existingID,
	}
}

// IdempotentSuccessResult represents the result of an idempotent operation
// that was already completed. The WasNoOp field indicates whether this was
// a replay of an already-completed operation.
type IdempotentSuccessResult struct {
	WasNoOp       bool   `json:"wasNoOp"`
	Message       string `json:"message,omitempty"`
	PriorResultID string `json:"priorResultId,omitempty"` // ID of the result from the original operation
}
