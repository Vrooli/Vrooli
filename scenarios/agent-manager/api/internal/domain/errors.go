package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// ERROR CODE TAXONOMY
// =============================================================================
// Unified error codes enable:
// - Machine-readable error classification
// - Consistent client-side handling
// - Log aggregation and alerting
// - API documentation generation

// ErrorCode is a machine-readable error identifier.
// Format: CATEGORY_SPECIFIC (e.g., "NOT_FOUND_TASK", "VALIDATION_FIELD")
type ErrorCode string

const (
	// --- Not Found Errors (404) ---
	ErrCodeNotFoundTask       ErrorCode = "NOT_FOUND_TASK"
	ErrCodeNotFoundRun        ErrorCode = "NOT_FOUND_RUN"
	ErrCodeNotFoundProfile    ErrorCode = "NOT_FOUND_PROFILE"
	ErrCodeNotFoundPolicy     ErrorCode = "NOT_FOUND_POLICY"
	ErrCodeNotFoundSandbox    ErrorCode = "NOT_FOUND_SANDBOX"
	ErrCodeNotFoundRunner     ErrorCode = "NOT_FOUND_RUNNER"
	ErrCodeNotFoundEvent      ErrorCode = "NOT_FOUND_EVENT"
	ErrCodeNotFoundArtifact   ErrorCode = "NOT_FOUND_ARTIFACT"

	// --- Validation Errors (400) ---
	ErrCodeValidationField    ErrorCode = "VALIDATION_FIELD"
	ErrCodeValidationFormat   ErrorCode = "VALIDATION_FORMAT"
	ErrCodeValidationRange    ErrorCode = "VALIDATION_RANGE"
	ErrCodeValidationRequired ErrorCode = "VALIDATION_REQUIRED"
	ErrCodeValidationConflict ErrorCode = "VALIDATION_CONFLICT"

	// --- State Errors (409) ---
	ErrCodeStateTransition   ErrorCode = "STATE_TRANSITION_INVALID"
	ErrCodeStateTerminal     ErrorCode = "STATE_TERMINAL"
	ErrCodeStateNotReady     ErrorCode = "STATE_NOT_READY"
	ErrCodeStateConcurrency  ErrorCode = "STATE_CONCURRENCY"

	// --- Policy Errors (403) ---
	ErrCodePolicySandbox      ErrorCode = "POLICY_SANDBOX_REQUIRED"
	ErrCodePolicyApproval     ErrorCode = "POLICY_APPROVAL_REQUIRED"
	ErrCodePolicyRunner       ErrorCode = "POLICY_RUNNER_DENIED"
	ErrCodePolicyScope        ErrorCode = "POLICY_SCOPE_DENIED"
	ErrCodePolicyLimit        ErrorCode = "POLICY_LIMIT_EXCEEDED"

	// --- Capacity Errors (503) ---
	ErrCodeCapacityRuns       ErrorCode = "CAPACITY_MAX_RUNS"
	ErrCodeCapacityScope      ErrorCode = "CAPACITY_SCOPE_LOCKED"
	ErrCodeCapacityStorage    ErrorCode = "CAPACITY_STORAGE"
	ErrCodeCapacityMemory     ErrorCode = "CAPACITY_MEMORY"

	// --- Infrastructure Errors (500/503) ---
	ErrCodeRunnerUnavailable  ErrorCode = "RUNNER_UNAVAILABLE"
	ErrCodeRunnerTimeout      ErrorCode = "RUNNER_TIMEOUT"
	ErrCodeRunnerExecution    ErrorCode = "RUNNER_EXECUTION"
	ErrCodeRunnerCommunication ErrorCode = "RUNNER_COMMUNICATION"
	ErrCodeSandboxCreate      ErrorCode = "SANDBOX_CREATE"
	ErrCodeSandboxApprove     ErrorCode = "SANDBOX_APPROVE"
	ErrCodeSandboxReject      ErrorCode = "SANDBOX_REJECT"
	ErrCodeSandboxOperation   ErrorCode = "SANDBOX_OPERATION"
	ErrCodeDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrCodeDatabaseQuery      ErrorCode = "DATABASE_QUERY"
	ErrCodeConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrCodeConfigMissing      ErrorCode = "CONFIG_MISSING"

	// --- Internal Errors (500) ---
	ErrCodeInternal          ErrorCode = "INTERNAL"
	ErrCodeInternalPanic     ErrorCode = "INTERNAL_PANIC"
	ErrCodeInternalAssertion ErrorCode = "INTERNAL_ASSERTION"
)

// Category returns the high-level error category.
func (c ErrorCode) Category() string {
	parts := strings.SplitN(string(c), "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "UNKNOWN"
}

// =============================================================================
// RECOVERY ACTIONS
// =============================================================================
// Each error type includes explicit recovery guidance for both humans and agents.

// RecoveryAction describes what the caller can do to recover from an error.
type RecoveryAction string

const (
	// RecoveryRetryImmediate - Retry the operation immediately (transient error)
	RecoveryRetryImmediate RecoveryAction = "retry_immediate"

	// RecoveryRetryBackoff - Retry with exponential backoff
	RecoveryRetryBackoff RecoveryAction = "retry_backoff"

	// RecoveryWait - Wait for a condition to change, then retry
	RecoveryWait RecoveryAction = "wait"

	// RecoveryFixInput - Correct the input and retry
	RecoveryFixInput RecoveryAction = "fix_input"

	// RecoveryUseAlternative - Try a different approach or resource
	RecoveryUseAlternative RecoveryAction = "use_alternative"

	// RecoveryEscalate - Human intervention or escalation needed
	RecoveryEscalate RecoveryAction = "escalate"

	// RecoveryAbort - Stop and preserve state; no automatic recovery
	RecoveryAbort RecoveryAction = "abort"

	// RecoveryNone - No recovery possible; terminal error
	RecoveryNone RecoveryAction = "none"
)

// =============================================================================
// BASE ERROR INTERFACE
// =============================================================================
// All domain errors implement DomainError for consistent handling.

// DomainError provides structured error information for consistent handling.
type DomainError interface {
	error

	// Code returns the machine-readable error code
	Code() ErrorCode

	// Recovery returns the recommended recovery action
	Recovery() RecoveryAction

	// Retryable returns whether this error can be retried
	Retryable() bool

	// UserMessage returns a user-friendly message (no internal details)
	UserMessage() string

	// Details returns structured error context for logging/debugging
	Details() map[string]interface{}
}

// =============================================================================
// NOT FOUND ERROR
// =============================================================================

// NotFoundError indicates a requested entity does not exist.
type NotFoundError struct {
	EntityType string
	ID         string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.EntityType, e.ID)
}

func (e *NotFoundError) Code() ErrorCode {
	switch e.EntityType {
	case "Task":
		return ErrCodeNotFoundTask
	case "Run":
		return ErrCodeNotFoundRun
	case "AgentProfile":
		return ErrCodeNotFoundProfile
	case "Policy":
		return ErrCodeNotFoundPolicy
	case "Sandbox":
		return ErrCodeNotFoundSandbox
	default:
		return ErrorCode("NOT_FOUND_" + strings.ToUpper(e.EntityType))
	}
}

func (e *NotFoundError) Recovery() RecoveryAction {
	return RecoveryFixInput
}

func (e *NotFoundError) Retryable() bool {
	return false // The entity doesn't exist; retrying won't help
}

func (e *NotFoundError) UserMessage() string {
	return fmt.Sprintf("The requested %s could not be found. Please verify the ID is correct.", e.EntityType)
}

func (e *NotFoundError) Details() map[string]interface{} {
	return map[string]interface{}{
		"entity_type": e.EntityType,
		"entity_id":   e.ID,
	}
}

// NewNotFoundError creates a NotFoundError for the given entity.
func NewNotFoundError(entityType string, id uuid.UUID) *NotFoundError {
	return &NotFoundError{EntityType: entityType, ID: id.String()}
}

// =============================================================================
// VALIDATION ERROR
// =============================================================================

// ValidationError indicates invalid input data.
type ValidationError struct {
	Field     string
	Message   string
	Hint      string
	code      ErrorCode // optional override
}

func (e *ValidationError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("validation error on %s: %s (hint: %s)", e.Field, e.Message, e.Hint)
	}
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Code() ErrorCode {
	if e.code != "" {
		return e.code
	}
	return ErrCodeValidationField
}

func (e *ValidationError) Recovery() RecoveryAction {
	return RecoveryFixInput
}

func (e *ValidationError) Retryable() bool {
	return false // Invalid input won't change on retry
}

func (e *ValidationError) UserMessage() string {
	if e.Hint != "" {
		return fmt.Sprintf("Invalid value for %s: %s. %s", e.Field, e.Message, e.Hint)
	}
	return fmt.Sprintf("Invalid value for %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Details() map[string]interface{} {
	d := map[string]interface{}{
		"field":   e.Field,
		"message": e.Message,
	}
	if e.Hint != "" {
		d["hint"] = e.Hint
	}
	return d
}

// NewValidationError creates a ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// NewValidationErrorWithHint creates a ValidationError with a helpful hint.
func NewValidationErrorWithHint(field, message, hint string) *ValidationError {
	return &ValidationError{Field: field, Message: message, Hint: hint}
}

// NewValidationErrorWithCode creates a ValidationError with a specific code.
func NewValidationErrorWithCode(field, message string, code ErrorCode) *ValidationError {
	return &ValidationError{Field: field, Message: message, code: code}
}

// =============================================================================
// STATE ERROR
// =============================================================================

// StateError indicates an operation is not valid for the current state.
type StateError struct {
	EntityType   string
	EntityID     string // optional, for context
	CurrentState string
	Operation    string
	Reason       string
	IsTerminal   bool // true if the entity is in a terminal state
}

func (e *StateError) Error() string {
	return fmt.Sprintf("cannot %s %s in %s state: %s", e.Operation, e.EntityType, e.CurrentState, e.Reason)
}

func (e *StateError) Code() ErrorCode {
	if e.IsTerminal {
		return ErrCodeStateTerminal
	}
	return ErrCodeStateTransition
}

func (e *StateError) Recovery() RecoveryAction {
	if e.IsTerminal {
		return RecoveryNone // Terminal states can't be changed
	}
	return RecoveryWait // Non-terminal: wait for state to change
}

func (e *StateError) Retryable() bool {
	return !e.IsTerminal // Non-terminal states may become valid
}

func (e *StateError) UserMessage() string {
	if e.IsTerminal {
		return fmt.Sprintf("This %s has completed and cannot be modified.", e.EntityType)
	}
	return fmt.Sprintf("Cannot %s this %s while it is %s. Please wait or cancel.", e.Operation, e.EntityType, e.CurrentState)
}

func (e *StateError) Details() map[string]interface{} {
	d := map[string]interface{}{
		"entity_type":   e.EntityType,
		"current_state": e.CurrentState,
		"operation":     e.Operation,
		"reason":        e.Reason,
		"is_terminal":   e.IsTerminal,
	}
	if e.EntityID != "" {
		d["entity_id"] = e.EntityID
	}
	return d
}

// NewStateError creates a StateError.
func NewStateError(entityType, currentState, operation, reason string) *StateError {
	isTerminal := isTerminalState(entityType, currentState)
	return &StateError{
		EntityType:   entityType,
		CurrentState: currentState,
		Operation:    operation,
		Reason:       reason,
		IsTerminal:   isTerminal,
	}
}

// NewStateErrorWithID creates a StateError with entity ID context.
func NewStateErrorWithID(entityType, entityID, currentState, operation, reason string) *StateError {
	e := NewStateError(entityType, currentState, operation, reason)
	e.EntityID = entityID
	return e
}

func isTerminalState(entityType, state string) bool {
	switch entityType {
	case "Task":
		return state == string(TaskStatusApproved) ||
			state == string(TaskStatusRejected) ||
			state == string(TaskStatusFailed) ||
			state == string(TaskStatusCancelled)
	case "Run":
		return state == string(RunStatusComplete) ||
			state == string(RunStatusFailed) ||
			state == string(RunStatusCancelled)
	default:
		return false
	}
}

// =============================================================================
// SCOPE CONFLICT ERROR
// =============================================================================

// ScopeConflictError indicates overlapping scope paths prevent execution.
type ScopeConflictError struct {
	RequestedPath string
	ProjectRoot   string
	ConflictsWith []ScopeConflict
	WaitEstimate  time.Duration // estimated wait time if known
}

func (e *ScopeConflictError) Error() string {
	return fmt.Sprintf("scope path %s conflicts with %d existing scope(s)", e.RequestedPath, len(e.ConflictsWith))
}

func (e *ScopeConflictError) Code() ErrorCode {
	return ErrCodeCapacityScope
}

func (e *ScopeConflictError) Recovery() RecoveryAction {
	return RecoveryWait // Wait for conflicting runs to complete
}

func (e *ScopeConflictError) Retryable() bool {
	return true // Locks are released when runs complete
}

func (e *ScopeConflictError) UserMessage() string {
	if e.WaitEstimate > 0 {
		return fmt.Sprintf("Another run is using this scope. Estimated wait: %s", e.WaitEstimate.Round(time.Second))
	}
	return "Another run is currently using this scope. Please wait for it to complete."
}

func (e *ScopeConflictError) Details() map[string]interface{} {
	conflicts := make([]map[string]string, len(e.ConflictsWith))
	for i, c := range e.ConflictsWith {
		conflicts[i] = map[string]string{
			"run_id":     c.RunID.String(),
			"scope_path": c.ScopePath,
		}
	}
	return map[string]interface{}{
		"requested_path": e.RequestedPath,
		"project_root":   e.ProjectRoot,
		"conflicts":      conflicts,
		"wait_estimate":  e.WaitEstimate.String(),
	}
}

// ScopeConflict represents a single scope overlap.
type ScopeConflict struct {
	RunID     uuid.UUID
	ScopePath string
}

// =============================================================================
// POLICY VIOLATION ERROR
// =============================================================================

// PolicyViolationError indicates a policy rule was violated.
type PolicyViolationError struct {
	PolicyID     uuid.UUID
	PolicyName   string
	Rule         string
	Message      string
	RequiredBy   string // what requires this policy (e.g., "security", "operator")
	Overrideable bool   // whether an admin can override
}

func (e *PolicyViolationError) Error() string {
	return fmt.Sprintf("policy violation [%s]: %s - %s", e.PolicyName, e.Rule, e.Message)
}

func (e *PolicyViolationError) Code() ErrorCode {
	switch e.Rule {
	case "sandbox_required":
		return ErrCodePolicySandbox
	case "approval_required":
		return ErrCodePolicyApproval
	case "runner_denied":
		return ErrCodePolicyRunner
	case "scope_denied":
		return ErrCodePolicyScope
	default:
		return ErrCodePolicyLimit
	}
}

func (e *PolicyViolationError) Recovery() RecoveryAction {
	if e.Overrideable {
		return RecoveryEscalate // Admin can override
	}
	return RecoveryUseAlternative // Use different settings/scope
}

func (e *PolicyViolationError) Retryable() bool {
	return false // Policy won't change on retry
}

func (e *PolicyViolationError) UserMessage() string {
	if e.Overrideable {
		return fmt.Sprintf("Policy '%s' prevents this action: %s. An administrator may override.", e.PolicyName, e.Message)
	}
	return fmt.Sprintf("Policy '%s' prevents this action: %s", e.PolicyName, e.Message)
}

func (e *PolicyViolationError) Details() map[string]interface{} {
	return map[string]interface{}{
		"policy_id":    e.PolicyID.String(),
		"policy_name":  e.PolicyName,
		"rule":         e.Rule,
		"message":      e.Message,
		"required_by":  e.RequiredBy,
		"overrideable": e.Overrideable,
	}
}

// =============================================================================
// CAPACITY EXCEEDED ERROR
// =============================================================================

// CapacityExceededError indicates resource limits have been reached.
type CapacityExceededError struct {
	Resource     string
	Current      int
	Maximum      int
	WaitEstimate time.Duration
	Scope        string // optional: scope or context for the limit
}

func (e *CapacityExceededError) Error() string {
	return fmt.Sprintf("capacity exceeded for %s: %d/%d", e.Resource, e.Current, e.Maximum)
}

func (e *CapacityExceededError) Code() ErrorCode {
	switch e.Resource {
	case "runs", "concurrent_runs":
		return ErrCodeCapacityRuns
	case "scope_locks":
		return ErrCodeCapacityScope
	case "storage":
		return ErrCodeCapacityStorage
	case "memory":
		return ErrCodeCapacityMemory
	default:
		return ErrCodeCapacityRuns
	}
}

func (e *CapacityExceededError) Recovery() RecoveryAction {
	return RecoveryRetryBackoff // Wait with backoff for capacity
}

func (e *CapacityExceededError) Retryable() bool {
	return true // Capacity may be freed
}

func (e *CapacityExceededError) UserMessage() string {
	if e.WaitEstimate > 0 {
		return fmt.Sprintf("System is at capacity (%d/%d %s). Estimated wait: %s",
			e.Current, e.Maximum, e.Resource, e.WaitEstimate.Round(time.Second))
	}
	return fmt.Sprintf("System is at capacity (%d/%d %s). Please try again shortly.", e.Current, e.Maximum, e.Resource)
}

func (e *CapacityExceededError) Details() map[string]interface{} {
	d := map[string]interface{}{
		"resource":      e.Resource,
		"current":       e.Current,
		"maximum":       e.Maximum,
		"wait_estimate": e.WaitEstimate.String(),
	}
	if e.Scope != "" {
		d["scope"] = e.Scope
	}
	return d
}

// =============================================================================
// RUNNER ERROR
// =============================================================================

// RunnerError indicates a problem with the agent runner.
type RunnerError struct {
	RunnerType   RunnerType
	Operation    string
	Cause        error
	IsTransient  bool   // true for timeouts, connection issues
	Alternative  string // alternative runner if available
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

func (e *RunnerError) Code() ErrorCode {
	switch e.Operation {
	case "execute", "execution":
		return ErrCodeRunnerExecution
	case "connect", "communication":
		return ErrCodeRunnerCommunication
	case "timeout":
		return ErrCodeRunnerTimeout
	default:
		return ErrCodeRunnerUnavailable
	}
}

func (e *RunnerError) Recovery() RecoveryAction {
	if e.Alternative != "" {
		return RecoveryUseAlternative
	}
	if e.IsTransient {
		return RecoveryRetryBackoff
	}
	return RecoveryEscalate
}

func (e *RunnerError) Retryable() bool {
	return e.IsTransient
}

func (e *RunnerError) UserMessage() string {
	if e.Alternative != "" {
		return fmt.Sprintf("Runner %s is unavailable. You can try using %s instead.", e.RunnerType, e.Alternative)
	}
	if e.IsTransient {
		return fmt.Sprintf("Runner %s is temporarily unavailable. Please try again.", e.RunnerType)
	}
	return fmt.Sprintf("Runner %s encountered an error. Please check runner configuration.", e.RunnerType)
}

func (e *RunnerError) Details() map[string]interface{} {
	d := map[string]interface{}{
		"runner_type":  string(e.RunnerType),
		"operation":    e.Operation,
		"is_transient": e.IsTransient,
	}
	if e.Cause != nil {
		d["cause"] = e.Cause.Error()
	}
	if e.Alternative != "" {
		d["alternative"] = e.Alternative
	}
	return d
}

// =============================================================================
// SANDBOX ERROR
// =============================================================================

// SandboxError indicates a problem with sandbox operations.
type SandboxError struct {
	SandboxID   *uuid.UUID
	Operation   string
	Cause       error
	IsTransient bool // true for connection issues
	CanRetry    bool // true if the operation can be safely retried
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

func (e *SandboxError) Code() ErrorCode {
	switch e.Operation {
	case "create":
		return ErrCodeSandboxCreate
	case "approve", "apply":
		return ErrCodeSandboxApprove
	case "reject":
		return ErrCodeSandboxReject
	default:
		return ErrCodeSandboxOperation
	}
}

func (e *SandboxError) Recovery() RecoveryAction {
	if e.CanRetry {
		return RecoveryRetryImmediate
	}
	if e.IsTransient {
		return RecoveryRetryBackoff
	}
	return RecoveryEscalate
}

func (e *SandboxError) Retryable() bool {
	return e.CanRetry || e.IsTransient
}

func (e *SandboxError) UserMessage() string {
	switch e.Operation {
	case "create":
		return "Unable to create isolated workspace. The sandbox service may be unavailable."
	case "approve", "apply":
		return "Unable to apply changes. Please verify the sandbox still exists and try again."
	case "reject":
		return "Unable to discard changes. Please try again."
	default:
		return "A sandbox operation failed. Please try again."
	}
}

func (e *SandboxError) Details() map[string]interface{} {
	d := map[string]interface{}{
		"operation":    e.Operation,
		"is_transient": e.IsTransient,
		"can_retry":    e.CanRetry,
	}
	if e.SandboxID != nil {
		d["sandbox_id"] = e.SandboxID.String()
	}
	if e.Cause != nil {
		d["cause"] = e.Cause.Error()
	}
	return d
}

// =============================================================================
// DATABASE ERROR
// =============================================================================

// DatabaseError indicates a persistence layer failure.
type DatabaseError struct {
	Operation   string
	EntityType  string
	EntityID    string
	Cause       error
	IsTransient bool
}

func (e *DatabaseError) Error() string {
	if e.EntityID != "" {
		return fmt.Sprintf("database error during %s of %s %s: %v", e.Operation, e.EntityType, e.EntityID, e.Cause)
	}
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Cause)
}

func (e *DatabaseError) Unwrap() error {
	return e.Cause
}

func (e *DatabaseError) Code() ErrorCode {
	if e.IsTransient {
		return ErrCodeDatabaseConnection
	}
	return ErrCodeDatabaseQuery
}

func (e *DatabaseError) Recovery() RecoveryAction {
	if e.IsTransient {
		return RecoveryRetryBackoff
	}
	return RecoveryEscalate
}

func (e *DatabaseError) Retryable() bool {
	return e.IsTransient
}

func (e *DatabaseError) UserMessage() string {
	if e.IsTransient {
		return "Database temporarily unavailable. Please try again."
	}
	return "A database error occurred. Please contact support if this persists."
}

func (e *DatabaseError) Details() map[string]interface{} {
	d := map[string]interface{}{
		"operation":    e.Operation,
		"is_transient": e.IsTransient,
	}
	if e.EntityType != "" {
		d["entity_type"] = e.EntityType
	}
	if e.EntityID != "" {
		d["entity_id"] = e.EntityID
	}
	if e.Cause != nil {
		d["cause"] = e.Cause.Error()
	}
	return d
}

// =============================================================================
// ERROR RESPONSE TYPE (for HTTP responses)
// =============================================================================

// ErrorResponse is the standard API error response format.
type ErrorResponse struct {
	Code        ErrorCode              `json:"code"`
	Message     string                 `json:"message"`
	UserMessage string                 `json:"userMessage,omitempty"`
	Recovery    RecoveryAction         `json:"recovery,omitempty"`
	Retryable   bool                   `json:"retryable"`
	Details     map[string]interface{} `json:"details,omitempty"`
	RequestID   string                 `json:"requestId,omitempty"`
}

// ToErrorResponse converts a DomainError to an ErrorResponse.
func ToErrorResponse(err error, requestID string) ErrorResponse {
	if de, ok := err.(DomainError); ok {
		return ErrorResponse{
			Code:        de.Code(),
			Message:     err.Error(),
			UserMessage: de.UserMessage(),
			Recovery:    de.Recovery(),
			Retryable:   de.Retryable(),
			Details:     de.Details(),
			RequestID:   requestID,
		}
	}
	// Fallback for non-domain errors
	return ErrorResponse{
		Code:        ErrCodeInternal,
		Message:     err.Error(),
		UserMessage: "An unexpected error occurred. Please try again.",
		Recovery:    RecoveryEscalate,
		Retryable:   false,
		RequestID:   requestID,
	}
}

// =============================================================================
// HELPER: Check if an error is retryable
// =============================================================================

// IsRetryable returns true if the error can be retried.
func IsRetryable(err error) bool {
	if de, ok := err.(DomainError); ok {
		return de.Retryable()
	}
	return false
}

// GetRecoveryAction returns the recommended recovery action for an error.
func GetRecoveryAction(err error) RecoveryAction {
	if de, ok := err.(DomainError); ok {
		return de.Recovery()
	}
	return RecoveryEscalate
}

// GetErrorCode returns the error code for an error.
func GetErrorCode(err error) ErrorCode {
	if de, ok := err.(DomainError); ok {
		return de.Code()
	}
	return ErrCodeInternal
}
