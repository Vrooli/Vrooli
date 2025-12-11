package agents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantCategory FailureCategory
		wantCode     FailureCode
		wantRetry    bool
	}{
		{
			name:         "Nil error returns no failure",
			err:          nil,
			wantCategory: FailureCategoryNone,
			wantCode:     "",
			wantRetry:    false,
		},
		{
			name:         "Context canceled is transient",
			err:          context.Canceled,
			wantCategory: FailureCategoryTransient,
			wantCode:     FailureCodeContextCanceled,
			wantRetry:    true,
		},
		{
			name:         "Deadline exceeded is transient",
			err:          context.DeadlineExceeded,
			wantCategory: FailureCategoryTransient,
			wantCode:     FailureCodeDatabaseTimeout,
			wantRetry:    true,
		},
		{
			name:         "SQL no rows is resource failure",
			err:          sql.ErrNoRows,
			wantCategory: FailureCategoryResource,
			wantCode:     FailureCodeAgentNotFound,
			wantRetry:    false,
		},
		{
			name:         "SQL conn done is transient",
			err:          sql.ErrConnDone,
			wantCategory: FailureCategoryTransient,
			wantCode:     FailureCodeDatabaseConnection,
			wantRetry:    true,
		},
		{
			name:         "Scope conflict error",
			err:          errors.New("scope conflict with active agents"),
			wantCategory: FailureCategoryConflict,
			wantCode:     FailureCodeScopeConflict,
			wantRetry:    false,
		},
		{
			name:         "Duplicate agent error",
			err:          errors.New("duplicate agent: identical prompt already running"),
			wantCategory: FailureCategoryConflict,
			wantCode:     FailureCodeDuplicateAgent,
			wantRetry:    false,
		},
		{
			name:         "Agent not found error",
			err:          errors.New("agent not found: abc123"),
			wantCategory: FailureCategoryResource,
			wantCode:     FailureCodeAgentNotFound,
			wantRetry:    false,
		},
		{
			name:         "Agent not running error",
			err:          errors.New("agent xyz is not running (status: completed)"),
			wantCategory: FailureCategoryResource,
			wantCode:     FailureCodeAgentAlreadyStopped,
			wantRetry:    false,
		},
		{
			name:         "Lock acquisition error",
			err:          errors.New("acquire lock for path api: lock already held"),
			wantCategory: FailureCategoryTransient,
			wantCode:     FailureCodeLockAcquisitionFail,
			wantRetry:    true,
		},
		{
			name:         "Unknown error is internal",
			err:          errors.New("something completely unexpected"),
			wantCategory: FailureCategoryInternal,
			wantCode:     FailureCodeUnexpected,
			wantRetry:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)

			if result.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", result.Category, tt.wantCategory)
			}

			if result.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", result.Code, tt.wantCode)
			}

			if result.Retryable != tt.wantRetry {
				t.Errorf("Retryable = %v, want %v", result.Retryable, tt.wantRetry)
			}

			// Non-nil errors should have messages and internal details
			if tt.err != nil {
				if result.Message == "" {
					t.Error("Expected non-empty Message for error")
				}
				if result.InternalDetails == "" && tt.wantCategory != FailureCategoryNone {
					t.Error("Expected non-empty InternalDetails for error")
				}
			}
		})
	}
}

func TestFailureInfo_Methods(t *testing.T) {
	t.Run("Error interface", func(t *testing.T) {
		f := FailureInfo{Message: "test error"}
		if f.Error() != "test error" {
			t.Errorf("Error() = %q, want %q", f.Error(), "test error")
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		retryable := FailureInfo{Retryable: true}
		notRetryable := FailureInfo{Retryable: false}

		if !retryable.IsRetryable() {
			t.Error("Expected IsRetryable() to return true")
		}
		if notRetryable.IsRetryable() {
			t.Error("Expected IsRetryable() to return false")
		}
	})

	t.Run("WithInternalDetails preserves original", func(t *testing.T) {
		original := FailureInfo{Message: "test", InternalDetails: "original"}
		modified := original.WithInternalDetails("modified")

		if original.InternalDetails != "original" {
			t.Error("Original was mutated")
		}
		if modified.InternalDetails != "modified" {
			t.Errorf("Modified has wrong details: %s", modified.InternalDetails)
		}
	})

	t.Run("WithRecoveryHint preserves original", func(t *testing.T) {
		original := FailureInfo{Message: "test", RecoveryHint: "original"}
		modified := original.WithRecoveryHint("modified")

		if original.RecoveryHint != "original" {
			t.Error("Original was mutated")
		}
		if modified.RecoveryHint != "modified" {
			t.Errorf("Modified has wrong hint: %s", modified.RecoveryHint)
		}
	})
}

func TestFailureInfo_Formatting(t *testing.T) {
	t.Run("FormatForUser excludes internal details", func(t *testing.T) {
		f := FailureInfo{
			Message:         "Something went wrong",
			RecoveryHint:    "Please try again",
			InternalDetails: "secret SQL query",
		}

		userMsg := f.FormatForUser()
		if containsStr(userMsg, "secret SQL") {
			t.Error("FormatForUser leaked internal details")
		}
		if !containsStr(userMsg, "Something went wrong") {
			t.Error("FormatForUser missing message")
		}
		if !containsStr(userMsg, "Please try again") {
			t.Error("FormatForUser missing recovery hint")
		}
	})

	t.Run("FormatForLog includes internal details", func(t *testing.T) {
		f := FailureInfo{
			Category:        FailureCategoryInternal,
			Code:            FailureCodeUnexpected,
			Message:         "Something went wrong",
			InternalDetails: "detailed stack trace here",
		}

		logMsg := f.FormatForLog()
		if !containsStr(logMsg, "detailed stack trace") {
			t.Error("FormatForLog missing internal details")
		}
		if !containsStr(logMsg, "category=internal") {
			t.Error("FormatForLog missing category")
		}
	})

	t.Run("ToMap excludes internal details", func(t *testing.T) {
		f := FailureInfo{
			Category:          FailureCategoryTransient,
			Code:              FailureCodeDatabaseTimeout,
			Message:           "Timeout",
			Retryable:         true,
			RetryAfterSeconds: 5,
			RecoveryHint:      "Try again",
			InternalDetails:   "secret",
		}

		m := f.ToMap()

		if _, ok := m["internalDetails"]; ok {
			t.Error("ToMap included internal details")
		}
		if m["category"] != "transient" {
			t.Errorf("category = %v, want transient", m["category"])
		}
		if m["code"] != "database_timeout" {
			t.Errorf("code = %v, want database_timeout", m["code"])
		}
		if m["retryable"] != true {
			t.Errorf("retryable = %v, want true", m["retryable"])
		}
		if m["retryAfterSeconds"] != 5 {
			t.Errorf("retryAfterSeconds = %v, want 5", m["retryAfterSeconds"])
		}
	})
}

func TestDecideOnContainmentFailure(t *testing.T) {
	tests := []struct {
		name           string
		failure        FailureInfo
		wantContinue   bool
		wantHasWarning bool
	}{
		{
			name: "Docker unavailable allows continuation",
			failure: FailureInfo{
				Code: FailureCodeDockerUnavailable,
			},
			wantContinue:   true,
			wantHasWarning: true,
		},
		{
			name: "Containment failed allows continuation",
			failure: FailureInfo{
				Code: FailureCodeContainmentFailed,
			},
			wantContinue:   true,
			wantHasWarning: true,
		},
		{
			name: "Unknown error allows continuation",
			failure: FailureInfo{
				Code: FailureCodeUnexpected,
			},
			wantContinue:   true,
			wantHasWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideOnContainmentFailure(tt.failure)

			if decision.ShouldContinue != tt.wantContinue {
				t.Errorf("ShouldContinue = %v, want %v", decision.ShouldContinue, tt.wantContinue)
			}

			hasWarning := decision.WarningForUser != ""
			if hasWarning != tt.wantHasWarning {
				t.Errorf("HasWarning = %v, want %v", hasWarning, tt.wantHasWarning)
			}

			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}

			if decision.OriginalFailure == nil {
				t.Error("Expected OriginalFailure to be set")
			}
		})
	}
}

func TestDecideOnHeartbeatFailure(t *testing.T) {
	failure := FailureInfo{Code: FailureCodeHeartbeatFailed}
	maxFailures := 3

	tests := []struct {
		name               string
		consecutiveFailures int
		wantContinue       bool
	}{
		{
			name:               "First failure continues",
			consecutiveFailures: 1,
			wantContinue:       true,
		},
		{
			name:               "Second failure continues",
			consecutiveFailures: 2,
			wantContinue:       true,
		},
		{
			name:               "At max failures stops",
			consecutiveFailures: 3,
			wantContinue:       false,
		},
		{
			name:               "Over max failures stops",
			consecutiveFailures: 5,
			wantContinue:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideOnHeartbeatFailure(tt.consecutiveFailures, maxFailures, failure)

			if decision.ShouldContinue != tt.wantContinue {
				t.Errorf("ShouldContinue = %v, want %v", decision.ShouldContinue, tt.wantContinue)
			}

			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}
		})
	}
}

func TestDecideOnSessionTrackingFailure(t *testing.T) {
	failure := FailureInfo{
		Code:    FailureCodeUnexpected,
		Message: "session tracking failed",
	}

	decision := DecideOnSessionTrackingFailure(failure)

	if !decision.ShouldContinue {
		t.Error("Session tracking failure should allow continuation")
	}

	// Session tracking is internal - no user warning
	if decision.WarningForUser != "" {
		t.Error("Expected no user warning for session tracking failure")
	}
}

func TestDecideOnRegistrationFailure(t *testing.T) {
	tests := []struct {
		name         string
		failure      FailureInfo
		wantContinue bool
		wantWarning  bool
	}{
		{
			name: "Scope conflict stops execution",
			failure: FailureInfo{
				Code:    FailureCodeScopeConflict,
				Message: "scope conflict",
			},
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name: "Duplicate agent stops execution",
			failure: FailureInfo{
				Code:    FailureCodeDuplicateAgent,
				Message: "duplicate agent",
			},
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name: "Database connection error stops with transient message",
			failure: FailureInfo{
				Code:    FailureCodeDatabaseConnection,
				Message: "db connection failed",
			},
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name: "Database timeout stops with transient message",
			failure: FailureInfo{
				Code:    FailureCodeDatabaseTimeout,
				Message: "db timeout",
			},
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name: "Unknown failure stops execution",
			failure: FailureInfo{
				Code:    FailureCodeUnexpected,
				Message: "something unexpected",
			},
			wantContinue: false,
			wantWarning:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideOnRegistrationFailure(tt.failure)

			if decision.ShouldContinue != tt.wantContinue {
				t.Errorf("ShouldContinue = %v, want %v", decision.ShouldContinue, tt.wantContinue)
			}

			hasWarning := decision.WarningForUser != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("HasWarning = %v, want %v", hasWarning, tt.wantWarning)
			}

			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}

			if decision.OriginalFailure == nil {
				t.Error("Expected OriginalFailure to be set")
			}
		})
	}
}

func TestDecideOnProcessFailure(t *testing.T) {
	tests := []struct {
		name         string
		failure      FailureInfo
		wantContinue bool
		wantWarning  bool
	}{
		{
			name:         "Timeout suggests breaking task into smaller pieces",
			failure:      NewProcessTimeoutFailure(300),
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name:         "Canceled is normal user action",
			failure:      NewProcessCanceledFailure(),
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name:         "Start failed suggests checking installation",
			failure:      NewProcessStartFailure(fmt.Errorf("exec not found")),
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name:         "Pipe failed suggests retry",
			failure:      NewProcessPipeFailure("stdout", fmt.Errorf("pipe error")),
			wantContinue: false,
			wantWarning:  true,
		},
		{
			name:         "Exit error suggests checking output",
			failure:      NewProcessExitFailure(fmt.Errorf("exit 1"), "module not found"),
			wantContinue: false,
			wantWarning:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideOnProcessFailure(tt.failure)

			if decision.ShouldContinue != tt.wantContinue {
				t.Errorf("ShouldContinue = %v, want %v", decision.ShouldContinue, tt.wantContinue)
			}

			hasWarning := decision.WarningForUser != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("HasWarning = %v, want %v", hasWarning, tt.wantWarning)
			}

			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}
		})
	}
}

func TestDecideOnOrphanCleanupFailure(t *testing.T) {
	failure := FailureInfo{
		Code:    FailureCodeUnexpected,
		Message: "cleanup failed",
	}

	decision := DecideOnOrphanCleanupFailure(failure, 5, 2)

	if !decision.ShouldContinue {
		t.Error("Orphan cleanup failure should allow continuation")
	}

	// Orphan cleanup is internal - no user warning
	if decision.WarningForUser != "" {
		t.Error("Expected no user warning for orphan cleanup failure")
	}

	// Reason should mention counts
	if !containsStr(decision.Reason, "5") || !containsStr(decision.Reason, "2") {
		t.Errorf("Reason should mention orphan counts: %s", decision.Reason)
	}
}

func TestDecisionWarningsContainActionableGuidance(t *testing.T) {
	// These tests verify that user-facing warnings contain helpful, actionable guidance
	// rather than just technical error descriptions

	t.Run("Registration scope conflict mentions waiting or stopping", func(t *testing.T) {
		failure := FailureInfo{
			Code:    FailureCodeScopeConflict,
			Message: "scope conflict",
		}
		decision := DecideOnRegistrationFailure(failure)
		if !containsStr(decision.WarningForUser, "wait") && !containsStr(decision.WarningForUser, "stop") {
			t.Errorf("Scope conflict warning should mention waiting or stopping: %s", decision.WarningForUser)
		}
	})

	t.Run("Registration duplicate mentions monitoring or waiting", func(t *testing.T) {
		failure := FailureInfo{
			Code:    FailureCodeDuplicateAgent,
			Message: "duplicate agent",
		}
		decision := DecideOnRegistrationFailure(failure)
		if !containsStr(decision.WarningForUser, "monitor") && !containsStr(decision.WarningForUser, "wait") {
			t.Errorf("Duplicate warning should mention monitoring or waiting: %s", decision.WarningForUser)
		}
	})

	t.Run("Process timeout mentions breaking into smaller tasks", func(t *testing.T) {
		failure := NewProcessTimeoutFailure(300)
		decision := DecideOnProcessFailure(failure)
		if !containsStr(decision.WarningForUser, "smaller") {
			t.Errorf("Timeout warning should suggest breaking into smaller tasks: %s", decision.WarningForUser)
		}
	})

	t.Run("Process start failure mentions checking installation", func(t *testing.T) {
		failure := NewProcessStartFailure(fmt.Errorf("exec not found"))
		decision := DecideOnProcessFailure(failure)
		if !containsStr(decision.WarningForUser, "install") {
			t.Errorf("Start failure should mention checking installation: %s", decision.WarningForUser)
		}
	})

	t.Run("Process canceled mentions spawning new agent", func(t *testing.T) {
		failure := NewProcessCanceledFailure()
		decision := DecideOnProcessFailure(failure)
		if !containsStr(decision.WarningForUser, "spawn") {
			t.Errorf("Canceled warning should mention spawning new agent: %s", decision.WarningForUser)
		}
	})
}

func TestNewFailureFactories(t *testing.T) {
	t.Run("NewValidationFailure", func(t *testing.T) {
		f := NewValidationFailure(FailureCodeInvalidPrompt, "Invalid prompt")
		if f.Category != FailureCategoryValidation {
			t.Errorf("Category = %v, want validation", f.Category)
		}
		if f.Code != FailureCodeInvalidPrompt {
			t.Errorf("Code = %v, want invalid_prompt", f.Code)
		}
	})

	t.Run("NewConflictFailure", func(t *testing.T) {
		f := NewConflictFailure(FailureCodeScopeConflict, "Scope conflict")
		if f.Category != FailureCategoryConflict {
			t.Errorf("Category = %v, want conflict", f.Category)
		}
	})

	t.Run("NewResourceFailure", func(t *testing.T) {
		f := NewResourceFailure(FailureCodeDockerUnavailable, "Docker not running", "Install Docker")
		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.RecoveryHint != "Install Docker" {
			t.Errorf("RecoveryHint = %v, want 'Install Docker'", f.RecoveryHint)
		}
	})

	t.Run("NewTransientFailure", func(t *testing.T) {
		f := NewTransientFailure(FailureCodeDatabaseTimeout, "Timeout", 5)
		if f.Category != FailureCategoryTransient {
			t.Errorf("Category = %v, want transient", f.Category)
		}
		if !f.Retryable {
			t.Error("Expected Retryable to be true")
		}
		if f.RetryAfterSeconds != 5 {
			t.Errorf("RetryAfterSeconds = %v, want 5", f.RetryAfterSeconds)
		}
	})

	t.Run("NewInternalFailure", func(t *testing.T) {
		f := NewInternalFailure(FailureCodeUnexpected, "Unexpected error")
		if f.Category != FailureCategoryInternal {
			t.Errorf("Category = %v, want internal", f.Category)
		}
		if f.Code != FailureCodeUnexpected {
			t.Errorf("Code = %v, want unexpected_error", f.Code)
		}
		if f.RecoveryHint == "" {
			t.Error("Expected non-empty RecoveryHint for internal failure")
		}
	})
}

func TestProcessExecutionFailureFactories(t *testing.T) {
	t.Run("NewProcessStartFailure", func(t *testing.T) {
		err := errors.New("exec: executable not found")
		f := NewProcessStartFailure(err)

		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.Code != FailureCodeProcessStartFailed {
			t.Errorf("Code = %v, want process_start_failed", f.Code)
		}
		if f.RecoveryHint == "" {
			t.Error("Expected non-empty RecoveryHint")
		}
		if !containsStr(f.InternalDetails, "executable not found") {
			t.Error("InternalDetails should contain original error")
		}
	})

	t.Run("NewProcessPipeFailure", func(t *testing.T) {
		err := errors.New("pipe creation failed")
		f := NewProcessPipeFailure("stdout", err)

		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.Code != FailureCodeProcessPipeFailed {
			t.Errorf("Code = %v, want process_pipe_failed", f.Code)
		}
		if !containsStr(f.InternalDetails, "stdout") {
			t.Error("InternalDetails should mention pipe type")
		}
	})

	t.Run("NewProcessTimeoutFailure", func(t *testing.T) {
		f := NewProcessTimeoutFailure(300)

		if f.Category != FailureCategoryTransient {
			t.Errorf("Category = %v, want transient", f.Category)
		}
		if f.Code != FailureCodeProcessTimeout {
			t.Errorf("Code = %v, want process_timeout", f.Code)
		}
		if !f.Retryable {
			t.Error("Timeout should be retryable")
		}
		if !containsStr(f.RecoveryHint, "300") {
			t.Error("RecoveryHint should mention the timeout value")
		}
	})

	t.Run("NewProcessCanceledFailure", func(t *testing.T) {
		f := NewProcessCanceledFailure()

		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.Code != FailureCodeProcessCanceled {
			t.Errorf("Code = %v, want process_canceled", f.Code)
		}
		if f.RecoveryHint == "" {
			t.Error("Expected non-empty RecoveryHint")
		}
	})

	t.Run("NewProcessExitFailure with short output", func(t *testing.T) {
		err := errors.New("exit status 1")
		f := NewProcessExitFailure(err, "Error: module not found")

		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.Code != FailureCodeProcessExitError {
			t.Errorf("Code = %v, want process_exit_error", f.Code)
		}
		if !containsStr(f.Message, "module not found") {
			t.Error("Message should include short output")
		}
	})

	t.Run("NewProcessExitFailure with long output truncates", func(t *testing.T) {
		err := errors.New("exit status 1")
		longOutput := "Error: " + string(make([]byte, 300)) // Long output
		f := NewProcessExitFailure(err, longOutput)

		// Message should be truncated
		if len(f.Message) > 250 {
			t.Error("Message should be truncated for long output")
		}
		// But InternalDetails should have full output
		if !containsStr(f.InternalDetails, longOutput) {
			t.Error("InternalDetails should contain full output")
		}
	})

	t.Run("NewRegistrationFailure", func(t *testing.T) {
		err := errors.New("database constraint violation")
		f := NewRegistrationFailure(err)

		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.Code != FailureCodeRegistrationFailed {
			t.Errorf("Code = %v, want registration_failed", f.Code)
		}
		if !containsStr(f.InternalDetails, "constraint violation") {
			t.Error("InternalDetails should contain original error")
		}
	})

	t.Run("NewContainmentPrepFailure", func(t *testing.T) {
		err := errors.New("image pull failed")
		f := NewContainmentPrepFailure("docker", err)

		if f.Category != FailureCategoryResource {
			t.Errorf("Category = %v, want resource", f.Category)
		}
		if f.Code != FailureCodeContainmentPrepFailed {
			t.Errorf("Code = %v, want containment_prep_failed", f.Code)
		}
		if !containsStr(f.InternalDetails, "docker") {
			t.Error("InternalDetails should mention provider")
		}
		if !containsStr(f.InternalDetails, "image pull failed") {
			t.Error("InternalDetails should contain original error")
		}
	})
}

func TestToHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		failure  FailureInfo
		wantCode int
	}{
		{
			name:     "None category returns 200",
			failure:  FailureInfo{Category: FailureCategoryNone},
			wantCode: 200,
		},
		{
			name:     "Validation returns 400",
			failure:  FailureInfo{Category: FailureCategoryValidation, Code: FailureCodeInvalidPrompt},
			wantCode: 400,
		},
		{
			name:     "Conflict returns 409",
			failure:  FailureInfo{Category: FailureCategoryConflict, Code: FailureCodeScopeConflict},
			wantCode: 409,
		},
		{
			name:     "AgentNotFound returns 404",
			failure:  FailureInfo{Category: FailureCategoryResource, Code: FailureCodeAgentNotFound},
			wantCode: 404,
		},
		{
			name:     "ScenarioNotFound returns 404",
			failure:  FailureInfo{Category: FailureCategoryResource, Code: FailureCodeScenarioNotFound},
			wantCode: 404,
		},
		{
			name:     "AgentAlreadyStopped returns 400",
			failure:  FailureInfo{Category: FailureCategoryResource, Code: FailureCodeAgentAlreadyStopped},
			wantCode: 400,
		},
		{
			name:     "Other resource failures return 503",
			failure:  FailureInfo{Category: FailureCategoryResource, Code: FailureCodeDockerUnavailable},
			wantCode: 503,
		},
		{
			name:     "ProcessStartFailed returns 503",
			failure:  FailureInfo{Category: FailureCategoryResource, Code: FailureCodeProcessStartFailed},
			wantCode: 503,
		},
		{
			name:     "Transient returns 503",
			failure:  FailureInfo{Category: FailureCategoryTransient, Code: FailureCodeDatabaseTimeout},
			wantCode: 503,
		},
		{
			name:     "Internal returns 500",
			failure:  FailureInfo{Category: FailureCategoryInternal, Code: FailureCodeUnexpected},
			wantCode: 500,
		},
		{
			name:     "Unknown category returns 500",
			failure:  FailureInfo{Category: "unknown"},
			wantCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.failure.ToHTTPStatus()
			if got != tt.wantCode {
				t.Errorf("ToHTTPStatus() = %v, want %v", got, tt.wantCode)
			}
		})
	}
}

func TestToLogFields(t *testing.T) {
	f := FailureInfo{
		Category:          FailureCategoryTransient,
		Code:              FailureCodeDatabaseTimeout,
		Message:           "Operation timed out",
		Retryable:         true,
		RetryAfterSeconds: 5,
		RecoveryHint:      "Try again later",
		InternalDetails:   "connection pool exhausted",
	}

	fields := f.ToLogFields()

	if fields["failure_category"] != "transient" {
		t.Errorf("failure_category = %v, want transient", fields["failure_category"])
	}
	if fields["failure_code"] != "database_timeout" {
		t.Errorf("failure_code = %v, want database_timeout", fields["failure_code"])
	}
	if fields["failure_message"] != "Operation timed out" {
		t.Errorf("failure_message = %v", fields["failure_message"])
	}
	if fields["failure_retryable"] != true {
		t.Errorf("failure_retryable = %v, want true", fields["failure_retryable"])
	}
	if fields["failure_internal_details"] != "connection pool exhausted" {
		t.Errorf("failure_internal_details = %v", fields["failure_internal_details"])
	}
	if fields["failure_recovery_hint"] != "Try again later" {
		t.Errorf("failure_recovery_hint = %v", fields["failure_recovery_hint"])
	}
	if fields["failure_retry_after_seconds"] != 5 {
		t.Errorf("failure_retry_after_seconds = %v, want 5", fields["failure_retry_after_seconds"])
	}
}

func TestToLogFields_OmitsEmptyValues(t *testing.T) {
	f := FailureInfo{
		Category:  FailureCategoryValidation,
		Code:      FailureCodeInvalidPrompt,
		Message:   "Invalid prompt",
		Retryable: false,
		// No InternalDetails, RecoveryHint, or RetryAfterSeconds
	}

	fields := f.ToLogFields()

	// These should not be present
	if _, ok := fields["failure_internal_details"]; ok {
		t.Error("Empty InternalDetails should not be in log fields")
	}
	if _, ok := fields["failure_recovery_hint"]; ok {
		t.Error("Empty RecoveryHint should not be in log fields")
	}
	if _, ok := fields["failure_retry_after_seconds"]; ok {
		t.Error("Zero RetryAfterSeconds should not be in log fields")
	}
}

// Helper for string contains check - renamed to avoid conflict with service_test.go
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
