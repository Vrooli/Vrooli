package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Domain Errors
// These errors represent domain-specific failure conditions.
// -----------------------------------------------------------------------------

// NotFoundError indicates a requested entity does not exist.
type NotFoundError struct {
	EntityType string
	ID         string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.EntityType, e.ID)
}

// NewNotFoundError creates a NotFoundError for the given entity.
func NewNotFoundError(entityType string, id uuid.UUID) *NotFoundError {
	return &NotFoundError{EntityType: entityType, ID: id.String()}
}

// ValidationError indicates invalid input data.
type ValidationError struct {
	Field   string
	Message string
	Hint    string
}

func (e *ValidationError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("validation error on %s: %s (hint: %s)", e.Field, e.Message, e.Hint)
	}
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// NewValidationError creates a ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// NewValidationErrorWithHint creates a ValidationError with a helpful hint.
func NewValidationErrorWithHint(field, message, hint string) *ValidationError {
	return &ValidationError{Field: field, Message: message, Hint: hint}
}

// StateError indicates an operation is not valid for the current state.
type StateError struct {
	EntityType   string
	CurrentState string
	Operation    string
	Reason       string
}

func (e *StateError) Error() string {
	return fmt.Sprintf("cannot %s %s in %s state: %s", e.Operation, e.EntityType, e.CurrentState, e.Reason)
}

// NewStateError creates a StateError.
func NewStateError(entityType, currentState, operation, reason string) *StateError {
	return &StateError{
		EntityType:   entityType,
		CurrentState: currentState,
		Operation:    operation,
		Reason:       reason,
	}
}

// ScopeConflictError indicates overlapping scope paths.
type ScopeConflictError struct {
	RequestedPath string
	ConflictsWith []ScopeConflict
}

func (e *ScopeConflictError) Error() string {
	return fmt.Sprintf("scope path %s conflicts with %d existing scope(s)", e.RequestedPath, len(e.ConflictsWith))
}

// ScopeConflict represents a single scope overlap.
type ScopeConflict struct {
	RunID     uuid.UUID
	ScopePath string
}

// PolicyViolationError indicates a policy rule was violated.
type PolicyViolationError struct {
	PolicyID   uuid.UUID
	PolicyName string
	Rule       string
	Message    string
}

func (e *PolicyViolationError) Error() string {
	return fmt.Sprintf("policy violation [%s]: %s - %s", e.PolicyName, e.Rule, e.Message)
}

// CapacityExceededError indicates resource limits have been reached.
type CapacityExceededError struct {
	Resource     string
	Current      int
	Maximum      int
	WaitEstimate string
}

func (e *CapacityExceededError) Error() string {
	return fmt.Sprintf("capacity exceeded for %s: %d/%d", e.Resource, e.Current, e.Maximum)
}

// RunnerError indicates a problem with the agent runner.
type RunnerError struct {
	RunnerType RunnerType
	Operation  string
	Cause      error
}

func (e *RunnerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("runner %s error during %s: %v", e.RunnerType, e.Operation, e.Cause)
	}
	return fmt.Sprintf("runner %s error during %s", e.RunnerType, e.Operation)
}

func (e *RunnerError) Unwrap() error {
	return e.Cause
}

// SandboxError indicates a problem with sandbox operations.
type SandboxError struct {
	SandboxID *uuid.UUID
	Operation string
	Cause     error
}

func (e *SandboxError) Error() string {
	if e.SandboxID != nil {
		return fmt.Sprintf("sandbox %s error during %s: %v", e.SandboxID, e.Operation, e.Cause)
	}
	return fmt.Sprintf("sandbox error during %s: %v", e.Operation, e.Cause)
}

func (e *SandboxError) Unwrap() error {
	return e.Cause
}
