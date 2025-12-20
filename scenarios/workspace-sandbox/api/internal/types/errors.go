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

// Details returns structured information about the conflicts for API responses.
func (e *ScopeConflictError) Details() map[string]interface{} {
	conflicts := make([]map[string]interface{}, len(e.Conflicts))
	for i, c := range e.Conflicts {
		conflicts[i] = map[string]interface{}{
			"sandboxId":    c.ExistingID,
			"scope":        c.ExistingScope,
			"conflictType": string(c.ConflictType),
		}
	}
	return map[string]interface{}{
		"conflicts": conflicts,
	}
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

// --- Conflict Detection Errors (OT-P2-002) ---

// RepoChangedError indicates the canonical repository has changed since sandbox creation.
// This is detected when the current commit hash differs from BaseCommitHash.
type RepoChangedError struct {
	SandboxID      string
	BaseCommitHash string // Commit hash at sandbox creation
	CurrentHash    string // Current commit hash in canonical repo
	AffectedFiles  []string
}

func (e *RepoChangedError) Error() string {
	return fmt.Sprintf("canonical repository has changed since sandbox %s was created (base: %s, current: %s). Patch application may fail or produce unexpected results.",
		e.SandboxID, truncateHash(e.BaseCommitHash), truncateHash(e.CurrentHash))
}

func (e *RepoChangedError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *RepoChangedError) IsRetryable() bool {
	return false // Requires manual intervention or sandbox recreation
}

// Hint returns actionable guidance for resolving this error.
func (e *RepoChangedError) Hint() string {
	return "The canonical repository has new commits since this sandbox was created. Options: " +
		"(1) Review and force-apply the patch if changes don't overlap, " +
		"(2) Regenerate the diff against the current state, " +
		"(3) Create a new sandbox from the current repo state and re-apply your changes."
}

// HasAffectedFiles returns true if specific files are known to be affected by the conflict.
func (e *RepoChangedError) HasAffectedFiles() bool {
	return len(e.AffectedFiles) > 0
}

// NewRepoChangedError creates a RepoChangedError.
func NewRepoChangedError(sandboxID, baseHash, currentHash string) *RepoChangedError {
	return &RepoChangedError{
		SandboxID:      sandboxID,
		BaseCommitHash: baseHash,
		CurrentHash:    currentHash,
	}
}

// NewRepoChangedErrorWithFiles creates a RepoChangedError with affected file information.
func NewRepoChangedErrorWithFiles(sandboxID, baseHash, currentHash string, files []string) *RepoChangedError {
	return &RepoChangedError{
		SandboxID:      sandboxID,
		BaseCommitHash: baseHash,
		CurrentHash:    currentHash,
		AffectedFiles:  files,
	}
}

// PatchConflictError indicates patch application failed due to conflicts.
type PatchConflictError struct {
	SandboxID       string
	ConflictingFile string
	RejectFile      string // Path to .rej file if generated
	Details         string
}

func (e *PatchConflictError) Error() string {
	if e.ConflictingFile != "" {
		return fmt.Sprintf("patch conflict in file '%s' for sandbox %s: %s", e.ConflictingFile, e.SandboxID, e.Details)
	}
	return fmt.Sprintf("patch conflict for sandbox %s: %s", e.SandboxID, e.Details)
}

func (e *PatchConflictError) HTTPStatus() int {
	return http.StatusConflict
}

func (e *PatchConflictError) IsRetryable() bool {
	return false // Requires manual resolution
}

// Hint returns actionable guidance for resolving this error.
func (e *PatchConflictError) Hint() string {
	if e.RejectFile != "" {
		return fmt.Sprintf("The patch could not be applied cleanly. Review the reject file at %s to see what couldn't be applied. "+
			"Options: (1) Manually resolve conflicts and retry, (2) Create a new sandbox with the latest repo state.", e.RejectFile)
	}
	return "The patch could not be applied cleanly. Create a new sandbox with the latest repo state and re-apply your changes."
}

// NewPatchConflictError creates a PatchConflictError.
func NewPatchConflictError(sandboxID, details string) *PatchConflictError {
	return &PatchConflictError{
		SandboxID: sandboxID,
		Details:   details,
	}
}

// NewPatchConflictErrorForFile creates a PatchConflictError for a specific file.
func NewPatchConflictErrorForFile(sandboxID, file, rejectFile, details string) *PatchConflictError {
	return &PatchConflictError{
		SandboxID:       sandboxID,
		ConflictingFile: file,
		RejectFile:      rejectFile,
		Details:         details,
	}
}

// truncateHash returns a shortened version of a git hash for display.
func truncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}
