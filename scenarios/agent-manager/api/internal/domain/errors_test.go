package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ERROR CODE TESTS
// =============================================================================

func TestErrorCode_Category(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrCodeNotFoundTask, "NOT"},
		{ErrCodeValidationField, "VALIDATION"},
		{ErrCodeStateTransition, "STATE"},
		{ErrCodePolicySandbox, "POLICY"},
		{ErrCodeCapacityRuns, "CAPACITY"},
		{ErrCodeRunnerUnavailable, "RUNNER"},
		{ErrCodeSandboxCreate, "SANDBOX"},
		{ErrCodeDatabaseConnection, "DATABASE"},
		{ErrCodeConfigInvalid, "CONFIG"},
		{ErrCodeInternal, "INTERNAL"},
		{ErrorCode(""), ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.Category())
		})
	}
}

// =============================================================================
// NOT FOUND ERROR TESTS
// =============================================================================

func TestNotFoundError(t *testing.T) {
	id := uuid.New()
	err := NewNotFoundError("Task", id)

	assert.ErrorContains(t, err, "Task not found")
	assert.ErrorContains(t, err, id.String())
	assert.Equal(t, ErrCodeNotFoundTask, err.Code())
	assert.Equal(t, RecoveryFixInput, err.Recovery())
	assert.False(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "Task")
	assert.Equal(t, "Task", err.Details()["entity_type"])
	assert.Equal(t, id.String(), err.Details()["entity_id"])
}

func TestNotFoundError_CodeMapping(t *testing.T) {
	tests := []struct {
		entityType string
		expected   ErrorCode
	}{
		{"Task", ErrCodeNotFoundTask},
		{"Run", ErrCodeNotFoundRun},
		{"AgentProfile", ErrCodeNotFoundProfile},
		{"Policy", ErrCodeNotFoundPolicy},
		{"Sandbox", ErrCodeNotFoundSandbox},
		{"Widget", ErrorCode("NOT_FOUND_WIDGET")}, // unknown type falls back
	}
	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			err := NewNotFoundError(tt.entityType, uuid.New())
			assert.Equal(t, tt.expected, err.Code())
		})
	}
}

func TestNewNotFoundErrorWithID(t *testing.T) {
	err := NewNotFoundErrorWithID("Runner", "claude-code")
	assert.Equal(t, "Runner", err.EntityType)
	assert.Equal(t, "claude-code", err.ID)
	assert.Equal(t, ErrCodeNotFoundRunner, err.Code())
}

// =============================================================================
// VALIDATION ERROR TESTS
// =============================================================================

func TestValidationError(t *testing.T) {
	err := NewValidationError("name", "must not be empty")

	assert.ErrorContains(t, err, "validation error on name")
	assert.ErrorContains(t, err, "must not be empty")
	assert.NotContains(t, err.Error(), "hint")
	assert.Equal(t, ErrCodeValidationField, err.Code())
	assert.Equal(t, RecoveryFixInput, err.Recovery())
	assert.False(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "name")
	assert.Equal(t, "name", err.Details()["field"])
}

func TestValidationErrorWithHint(t *testing.T) {
	err := NewValidationErrorWithHint("email", "invalid format", "use user@example.com")

	assert.Contains(t, err.Error(), "hint: use user@example.com")
	assert.Contains(t, err.UserMessage(), "use user@example.com")
	assert.Equal(t, "use user@example.com", err.Details()["hint"])
}

func TestValidationErrorWithCode(t *testing.T) {
	err := NewValidationErrorWithCode("range", "out of bounds", ErrCodeValidationRange)
	assert.Equal(t, ErrCodeValidationRange, err.Code())
}

// =============================================================================
// STATE ERROR TESTS
// =============================================================================

func TestStateError_Terminal(t *testing.T) {
	err := NewStateError("Run", string(RunStatusComplete), "stop", "already completed")

	assert.ErrorContains(t, err, "cannot stop Run in complete state")
	assert.Equal(t, ErrCodeStateTerminal, err.Code())
	assert.Equal(t, RecoveryNone, err.Recovery())
	assert.False(t, err.Retryable())
	assert.True(t, err.IsTerminal)
	assert.Contains(t, err.UserMessage(), "completed")
}

func TestStateError_NonTerminal(t *testing.T) {
	err := NewStateError("Run", string(RunStatusRunning), "delete", "run is active")

	assert.Equal(t, ErrCodeStateTransition, err.Code())
	assert.Equal(t, RecoveryWait, err.Recovery())
	assert.True(t, err.Retryable())
	assert.False(t, err.IsTerminal)
	assert.Contains(t, err.UserMessage(), "running")
}

func TestStateErrorWithID(t *testing.T) {
	err := NewStateErrorWithID("Task", "abc-123", string(TaskStatusApproved), "edit", "task is approved")

	assert.Equal(t, "abc-123", err.EntityID)
	assert.Equal(t, "abc-123", err.Details()["entity_id"])
	assert.True(t, err.IsTerminal) // TaskStatusApproved is terminal
}

func TestIsTerminalState(t *testing.T) {
	// Task terminal states
	for _, s := range []TaskStatus{TaskStatusApproved, TaskStatusRejected, TaskStatusFailed, TaskStatusCancelled} {
		assert.True(t, isTerminalState("Task", string(s)), "Task %s should be terminal", s)
	}
	// Task non-terminal states
	for _, s := range []TaskStatus{TaskStatusQueued, TaskStatusRunning, TaskStatusNeedsReview} {
		assert.False(t, isTerminalState("Task", string(s)), "Task %s should not be terminal", s)
	}
	// Run terminal states
	for _, s := range []RunStatus{RunStatusComplete, RunStatusFailed, RunStatusCancelled} {
		assert.True(t, isTerminalState("Run", string(s)), "Run %s should be terminal", s)
	}
	// Run non-terminal states
	for _, s := range []RunStatus{RunStatusPending, RunStatusStarting, RunStatusRunning, RunStatusNeedsReview} {
		assert.False(t, isTerminalState("Run", string(s)), "Run %s should not be terminal", s)
	}
	// Unknown entity type
	assert.False(t, isTerminalState("Widget", "done"))
}

// =============================================================================
// SCOPE CONFLICT ERROR TESTS
// =============================================================================

func TestScopeConflictError(t *testing.T) {
	runID := uuid.New()
	err := &ScopeConflictError{
		RequestedPath: "/home/user/project",
		ProjectRoot:   "/home/user",
		ConflictsWith: []ScopeConflict{
			{RunID: runID, ScopePath: "/home/user/project/src"},
		},
		WaitEstimate: 30 * time.Second,
	}

	assert.ErrorContains(t, err, "scope path /home/user/project conflicts with 1 existing scope(s)")
	assert.Equal(t, ErrCodeCapacityScope, err.Code())
	assert.Equal(t, RecoveryWait, err.Recovery())
	assert.True(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "30s")
	details := err.Details()
	assert.Equal(t, "/home/user/project", details["requested_path"])
	conflicts := details["conflicts"].([]map[string]string)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, runID.String(), conflicts[0]["run_id"])
}

func TestScopeConflictError_NoWaitEstimate(t *testing.T) {
	err := &ScopeConflictError{
		RequestedPath: "/project",
		ConflictsWith: []ScopeConflict{},
	}
	assert.Contains(t, err.UserMessage(), "Please wait")
	assert.NotContains(t, err.UserMessage(), "Estimated")
}

// =============================================================================
// POLICY VIOLATION ERROR TESTS
// =============================================================================

func TestPolicyViolationError_CodeMapping(t *testing.T) {
	tests := []struct {
		rule     string
		expected ErrorCode
	}{
		{"sandbox_required", ErrCodePolicySandbox},
		{"approval_required", ErrCodePolicyApproval},
		{"runner_denied", ErrCodePolicyRunner},
		{"scope_denied", ErrCodePolicyScope},
		{"custom_rule", ErrCodePolicyLimit},
	}
	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			err := &PolicyViolationError{Rule: tt.rule, PolicyName: "test"}
			assert.Equal(t, tt.expected, err.Code())
		})
	}
}

func TestPolicyViolationError_Overrideable(t *testing.T) {
	err := &PolicyViolationError{
		PolicyName:   "sandbox-policy",
		Rule:         "sandbox_required",
		Message:      "sandbox is required",
		Overrideable: true,
	}
	assert.Equal(t, RecoveryEscalate, err.Recovery())
	assert.Contains(t, err.UserMessage(), "administrator may override")
}

func TestPolicyViolationError_NotOverrideable(t *testing.T) {
	err := &PolicyViolationError{
		PolicyName: "strict",
		Rule:       "scope_denied",
		Message:    "no access",
	}
	assert.Equal(t, RecoveryUseAlternative, err.Recovery())
	assert.False(t, err.Retryable())
	assert.NotContains(t, err.UserMessage(), "override")
}

// =============================================================================
// CAPACITY EXCEEDED ERROR TESTS
// =============================================================================

func TestCapacityExceededError_CodeMapping(t *testing.T) {
	tests := []struct {
		resource string
		expected ErrorCode
	}{
		{"runs", ErrCodeCapacityRuns},
		{"concurrent_runs", ErrCodeCapacityRuns},
		{"scope_locks", ErrCodeCapacityScope},
		{"storage", ErrCodeCapacityStorage},
		{"memory", ErrCodeCapacityMemory},
		{"unknown", ErrCodeCapacityRuns},
	}
	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			err := &CapacityExceededError{Resource: tt.resource, Current: 5, Maximum: 5}
			assert.Equal(t, tt.expected, err.Code())
		})
	}
}

func TestCapacityExceededError_Fields(t *testing.T) {
	err := &CapacityExceededError{
		Resource:     "runs",
		Current:      3,
		Maximum:      3,
		WaitEstimate: 10 * time.Second,
		Scope:        "project-a",
	}

	assert.ErrorContains(t, err, "capacity exceeded for runs: 3/3")
	assert.Equal(t, RecoveryRetryBackoff, err.Recovery())
	assert.True(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "10s")
	assert.Equal(t, "project-a", err.Details()["scope"])
}

func TestCapacityExceededError_NoWaitEstimate(t *testing.T) {
	err := &CapacityExceededError{Resource: "runs", Current: 1, Maximum: 1}
	assert.Contains(t, err.UserMessage(), "try again shortly")
}

// =============================================================================
// RUNNER ERROR TESTS
// =============================================================================

func TestRunnerError_CodeMapping(t *testing.T) {
	tests := []struct {
		operation string
		expected  ErrorCode
	}{
		{"execute", ErrCodeRunnerExecution},
		{"execution", ErrCodeRunnerExecution},
		{"connect", ErrCodeRunnerCommunication},
		{"communication", ErrCodeRunnerCommunication},
		{"timeout", ErrCodeRunnerTimeout},
		{"probe", ErrCodeRunnerUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := &RunnerError{RunnerType: RunnerTypeClaudeCode, Operation: tt.operation}
			assert.Equal(t, tt.expected, err.Code())
		})
	}
}

func TestRunnerError_WithAlternative(t *testing.T) {
	err := &RunnerError{
		RunnerType:  RunnerTypeClaudeCode,
		Operation:   "execute",
		Alternative: "codex",
	}
	assert.Equal(t, RecoveryUseAlternative, err.Recovery())
	assert.Contains(t, err.UserMessage(), "codex")
}

func TestRunnerError_Transient(t *testing.T) {
	cause := errors.New("connection refused")
	err := &RunnerError{
		RunnerType:  RunnerTypeClaudeCode,
		Operation:   "connect",
		Cause:       cause,
		IsTransient: true,
	}
	assert.True(t, err.Retryable())
	assert.Equal(t, RecoveryRetryBackoff, err.Recovery())
	assert.Contains(t, err.UserMessage(), "temporarily unavailable")
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Details()["cause"], "connection refused")
}

func TestRunnerError_Permanent(t *testing.T) {
	err := &RunnerError{
		RunnerType: RunnerTypeClaudeCode,
		Operation:  "execute",
	}
	assert.False(t, err.Retryable())
	assert.Equal(t, RecoveryEscalate, err.Recovery())
	assert.Contains(t, err.UserMessage(), "check runner configuration")
}

func TestRunnerError_NilCause(t *testing.T) {
	err := &RunnerError{RunnerType: RunnerTypeClaudeCode, Operation: "execute"}
	assert.Equal(t, "runner claude-code error during execute", err.Error())
}

// =============================================================================
// SANDBOX ERROR TESTS
// =============================================================================

func TestSandboxError_CodeMapping(t *testing.T) {
	tests := []struct {
		operation string
		expected  ErrorCode
	}{
		{"create", ErrCodeSandboxCreate},
		{"approve", ErrCodeSandboxApprove},
		{"apply", ErrCodeSandboxApprove},
		{"reject", ErrCodeSandboxReject},
		{"sync", ErrCodeSandboxOperation},
	}
	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			err := &SandboxError{Operation: tt.operation}
			assert.Equal(t, tt.expected, err.Code())
		})
	}
}

func TestSandboxError_CanRetry(t *testing.T) {
	err := &SandboxError{Operation: "create", CanRetry: true}
	assert.Equal(t, RecoveryRetryImmediate, err.Recovery())
	assert.True(t, err.Retryable())
}

func TestSandboxError_Transient(t *testing.T) {
	err := &SandboxError{Operation: "create", IsTransient: true}
	assert.Equal(t, RecoveryRetryBackoff, err.Recovery())
	assert.True(t, err.Retryable())
}

func TestSandboxError_Permanent(t *testing.T) {
	err := &SandboxError{Operation: "create"}
	assert.Equal(t, RecoveryEscalate, err.Recovery())
	assert.False(t, err.Retryable())
}

func TestSandboxError_WithID(t *testing.T) {
	id := uuid.New()
	cause := errors.New("timeout")
	err := &SandboxError{
		SandboxID: &id,
		Operation: "approve",
		Cause:     cause,
		ExtraDetails: map[string]interface{}{
			"conflict_count": 2,
		},
	}
	assert.Contains(t, err.Error(), id.String())
	assert.ErrorIs(t, err, cause)
	details := err.Details()
	assert.Equal(t, id.String(), details["sandbox_id"])
	assert.Equal(t, 2, details["conflict_count"])
}

func TestSandboxError_NilID(t *testing.T) {
	err := &SandboxError{Operation: "create", Cause: errors.New("fail")}
	assert.Contains(t, err.Error(), "sandbox error during create")
	assert.NotContains(t, err.Error(), "<nil>")
}

func TestSandboxError_UserMessages(t *testing.T) {
	assert.Contains(t, (&SandboxError{Operation: "create"}).UserMessage(), "sandbox service")
	assert.Contains(t, (&SandboxError{Operation: "approve"}).UserMessage(), "apply changes")
	assert.Contains(t, (&SandboxError{Operation: "apply"}).UserMessage(), "apply changes")
	assert.Contains(t, (&SandboxError{Operation: "reject"}).UserMessage(), "discard")
	assert.Contains(t, (&SandboxError{Operation: "sync"}).UserMessage(), "sandbox operation failed")
}

// =============================================================================
// DATABASE ERROR TESTS
// =============================================================================

func TestDatabaseError_Transient(t *testing.T) {
	cause := errors.New("connection lost")
	err := &DatabaseError{
		Operation:   "insert",
		EntityType:  "Task",
		EntityID:    "abc",
		Cause:       cause,
		IsTransient: true,
	}

	assert.ErrorContains(t, err, "database error during insert of Task abc")
	assert.Equal(t, ErrCodeDatabaseConnection, err.Code())
	assert.Equal(t, RecoveryRetryBackoff, err.Recovery())
	assert.True(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "temporarily unavailable")
	assert.ErrorIs(t, err, cause)
}

func TestDatabaseError_Permanent(t *testing.T) {
	err := &DatabaseError{
		Operation: "query",
		Cause:     errors.New("syntax error"),
	}

	assert.Equal(t, ErrCodeDatabaseQuery, err.Code())
	assert.Equal(t, RecoveryEscalate, err.Recovery())
	assert.False(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "contact support")
}

func TestDatabaseError_NoEntityID(t *testing.T) {
	err := &DatabaseError{Operation: "migrate"}
	assert.Equal(t, "database error during migrate: <nil>", err.Error())
	_, hasEntityID := err.Details()["entity_id"]
	assert.False(t, hasEntityID)
}

// =============================================================================
// CONFIG ERROR TESTS
// =============================================================================

func TestConfigMissingError(t *testing.T) {
	cause := errors.New("file not found")
	err := NewConfigMissingError("database_url", "required for startup", cause)

	assert.ErrorContains(t, err, "config error for database_url")
	assert.Equal(t, ErrCodeConfigMissing, err.Code())
	assert.Equal(t, RecoveryFixInput, err.Recovery())
	assert.False(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "missing")
	assert.ErrorIs(t, err, cause)
}

func TestConfigInvalidError(t *testing.T) {
	err := NewConfigInvalidError("max_turns", "must be positive", nil)

	assert.Equal(t, ErrCodeConfigInvalid, err.Code())
	assert.Contains(t, err.UserMessage(), "invalid")
	_, hasCause := err.Details()["cause"]
	assert.False(t, hasCause) // nil cause should not be in details
}

func TestConfigError_NoSetting(t *testing.T) {
	err := &ConfigError{Message: "general failure"}
	assert.Equal(t, "config error: general failure", err.Error())
	_, hasSetting := err.Details()["setting"]
	assert.False(t, hasSetting)
}

// =============================================================================
// INTERNAL ERROR TESTS
// =============================================================================

func TestInternalError(t *testing.T) {
	cause := errors.New("nil pointer")
	err := NewInternalError("unexpected state", cause)

	assert.ErrorContains(t, err, "unexpected state: nil pointer")
	assert.Equal(t, ErrCodeInternal, err.Code())
	assert.Equal(t, RecoveryEscalate, err.Recovery())
	assert.False(t, err.Retryable())
	assert.Contains(t, err.UserMessage(), "unexpected error")
	assert.ErrorIs(t, err, cause)
}

func TestInternalError_MessageOnly(t *testing.T) {
	err := &InternalError{Message: "bad state"}
	assert.Equal(t, "bad state", err.Error())
}

func TestInternalError_CauseOnly(t *testing.T) {
	err := &InternalError{Cause: errors.New("boom")}
	assert.Equal(t, "boom", err.Error())
}

func TestInternalError_Empty(t *testing.T) {
	err := &InternalError{}
	assert.Equal(t, "internal error", err.Error())
}

func TestInternalError_CustomCodeTag(t *testing.T) {
	err := &InternalError{CodeTag: ErrCodeInternalPanic, Message: "recovered panic"}
	assert.Equal(t, ErrCodeInternalPanic, err.Code())
}

// =============================================================================
// AsDomainError TESTS
// =============================================================================

func TestAsDomainError_NilInput(t *testing.T) {
	de := AsDomainError(nil)
	require.NotNil(t, de)
	assert.Equal(t, ErrCodeInternal, de.Code())
	assert.Contains(t, de.Error(), "nil error")
}

func TestAsDomainError_DomainError(t *testing.T) {
	original := NewValidationError("field", "bad")
	de := AsDomainError(original)
	assert.Equal(t, original, de)
}

func TestAsDomainError_PlainError(t *testing.T) {
	plain := errors.New("something went wrong")
	de := AsDomainError(plain)
	assert.Equal(t, ErrCodeInternal, de.Code())
	assert.Contains(t, de.Error(), "internal error")
}

// =============================================================================
// ToErrorResponse TESTS
// =============================================================================

func TestToErrorResponse(t *testing.T) {
	err := NewNotFoundError("Task", uuid.New())
	resp := ToErrorResponse(err, "req-123")

	assert.Equal(t, ErrCodeNotFoundTask, resp.Code)
	assert.Contains(t, resp.Message, "Task not found")
	assert.Contains(t, resp.UserMessage, "Task")
	assert.Equal(t, RecoveryFixInput, resp.Recovery)
	assert.False(t, resp.Retryable)
	assert.Equal(t, "req-123", resp.RequestID)
	assert.NotEmpty(t, resp.Details)
}

func TestToErrorResponse_PlainError(t *testing.T) {
	resp := ToErrorResponse(errors.New("oops"), "req-456")
	assert.Equal(t, ErrCodeInternal, resp.Code)
	assert.Equal(t, "req-456", resp.RequestID)
}

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestIsRetryable(t *testing.T) {
	assert.False(t, IsRetryable(NewValidationError("f", "m")))
	assert.True(t, IsRetryable(&CapacityExceededError{Resource: "runs"}))
	assert.False(t, IsRetryable(errors.New("plain")))
}

func TestGetRecoveryAction(t *testing.T) {
	assert.Equal(t, RecoveryFixInput, GetRecoveryAction(NewValidationError("f", "m")))
	assert.Equal(t, RecoveryRetryBackoff, GetRecoveryAction(&CapacityExceededError{Resource: "runs"}))
	assert.Equal(t, RecoveryEscalate, GetRecoveryAction(errors.New("plain")))
}

func TestGetErrorCode(t *testing.T) {
	assert.Equal(t, ErrCodeValidationField, GetErrorCode(NewValidationError("f", "m")))
	assert.Equal(t, ErrCodeInternal, GetErrorCode(errors.New("plain")))
}

// =============================================================================
// DOMAIN ERROR INTERFACE COMPLIANCE
// =============================================================================

func TestDomainErrorInterfaceCompliance(t *testing.T) {
	// Verify all error types implement DomainError
	var _ DomainError = &NotFoundError{}
	var _ DomainError = &ValidationError{}
	var _ DomainError = &StateError{}
	var _ DomainError = &ScopeConflictError{}
	var _ DomainError = &PolicyViolationError{}
	var _ DomainError = &CapacityExceededError{}
	var _ DomainError = &RunnerError{}
	var _ DomainError = &SandboxError{}
	var _ DomainError = &DatabaseError{}
	var _ DomainError = &ConfigError{}
	var _ DomainError = &InternalError{}
}
