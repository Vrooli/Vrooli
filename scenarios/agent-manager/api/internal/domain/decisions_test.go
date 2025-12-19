package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// STATE TRANSITION TESTS
// =============================================================================

func TestTaskStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		from    TaskStatus
		to      TaskStatus
		wantOK  bool
		wantMsg string
	}{
		// Valid transitions
		{"queued to running", TaskStatusQueued, TaskStatusRunning, true, ""},
		{"queued to cancelled", TaskStatusQueued, TaskStatusCancelled, true, ""},
		{"running to needs_review", TaskStatusRunning, TaskStatusNeedsReview, true, ""},
		{"running to failed", TaskStatusRunning, TaskStatusFailed, true, ""},
		{"running to cancelled", TaskStatusRunning, TaskStatusCancelled, true, ""},
		{"needs_review to approved", TaskStatusNeedsReview, TaskStatusApproved, true, ""},
		{"needs_review to rejected", TaskStatusNeedsReview, TaskStatusRejected, true, ""},

		// Invalid transitions
		{"queued to approved", TaskStatusQueued, TaskStatusApproved, false, "transition not allowed"},
		{"running to approved", TaskStatusRunning, TaskStatusApproved, false, "transition not allowed"},
		{"approved to running", TaskStatusApproved, TaskStatusRunning, false, "terminal state"},
		{"rejected to running", TaskStatusRejected, TaskStatusRunning, false, "terminal state"},
		{"failed to running", TaskStatusFailed, TaskStatusRunning, false, "terminal state"},
		{"cancelled to running", TaskStatusCancelled, TaskStatusRunning, false, "terminal state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, msg := tt.from.CanTransitionTo(tt.to)
			if ok != tt.wantOK {
				t.Errorf("CanTransitionTo() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK && tt.wantMsg != "" && msg == "" {
				t.Errorf("CanTransitionTo() should return reason for denial")
			}
		})
	}
}

func TestRunStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   RunStatus
		to     RunStatus
		wantOK bool
	}{
		// Valid transitions
		{"pending to starting", RunStatusPending, RunStatusStarting, true},
		{"pending to cancelled", RunStatusPending, RunStatusCancelled, true},
		{"pending to failed", RunStatusPending, RunStatusFailed, true},
		{"starting to running", RunStatusStarting, RunStatusRunning, true},
		{"starting to failed", RunStatusStarting, RunStatusFailed, true},
		{"running to needs_review", RunStatusRunning, RunStatusNeedsReview, true},
		{"running to complete", RunStatusRunning, RunStatusComplete, true},
		{"running to failed", RunStatusRunning, RunStatusFailed, true},
		{"running to cancelled", RunStatusRunning, RunStatusCancelled, true},
		{"needs_review to complete", RunStatusNeedsReview, RunStatusComplete, true},
		{"needs_review to failed", RunStatusNeedsReview, RunStatusFailed, true},

		// Invalid transitions
		{"pending to complete", RunStatusPending, RunStatusComplete, false},
		{"complete to running", RunStatusComplete, RunStatusRunning, false},
		{"failed to running", RunStatusFailed, RunStatusRunning, false},
		{"cancelled to running", RunStatusCancelled, RunStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, _ := tt.from.CanTransitionTo(tt.to)
			if ok != tt.wantOK {
				t.Errorf("CanTransitionTo() = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

// =============================================================================
// CANCELLATION DECISION TESTS
// =============================================================================

func TestTask_IsCancellable(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   bool
	}{
		{TaskStatusQueued, true},
		{TaskStatusRunning, true},
		{TaskStatusNeedsReview, false},
		{TaskStatusApproved, false},
		{TaskStatusRejected, false},
		{TaskStatusFailed, false},
		{TaskStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			task := &Task{Status: tt.status}
			if got := task.IsCancellable(); got != tt.want {
				t.Errorf("IsCancellable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_IsStoppable(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   bool
	}{
		{RunStatusPending, false},
		{RunStatusStarting, true},
		{RunStatusRunning, true},
		{RunStatusNeedsReview, false},
		{RunStatusComplete, false},
		{RunStatusFailed, false},
		{RunStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			run := &Run{Status: tt.status}
			if got := run.IsStoppable(); got != tt.want {
				t.Errorf("IsStoppable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// APPROVAL DECISION TESTS
// =============================================================================

func TestRun_IsApprovable(t *testing.T) {
	sandboxID := mustParseUUID("12345678-1234-1234-1234-123456789abc")

	tests := []struct {
		name   string
		run    *Run
		wantOK bool
	}{
		{
			name: "valid - needs_review with sandbox",
			run: &Run{
				Status:        RunStatusNeedsReview,
				SandboxID:     &sandboxID,
				ApprovalState: ApprovalStatePending,
			},
			wantOK: true,
		},
		{
			name: "invalid - wrong status",
			run: &Run{
				Status:    RunStatusRunning,
				SandboxID: &sandboxID,
			},
			wantOK: false,
		},
		{
			name: "invalid - no sandbox",
			run: &Run{
				Status:    RunStatusNeedsReview,
				SandboxID: nil,
			},
			wantOK: false,
		},
		{
			name: "invalid - already approved",
			run: &Run{
				Status:        RunStatusNeedsReview,
				SandboxID:     &sandboxID,
				ApprovalState: ApprovalStateApproved,
			},
			wantOK: false,
		},
		{
			name: "invalid - already rejected",
			run: &Run{
				Status:        RunStatusNeedsReview,
				SandboxID:     &sandboxID,
				ApprovalState: ApprovalStateRejected,
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, reason := tt.run.IsApprovable()
			if ok != tt.wantOK {
				t.Errorf("IsApprovable() = %v (reason: %s), want %v", ok, reason, tt.wantOK)
			}
		})
	}
}

// =============================================================================
// RUN MODE DECISION TESTS
// =============================================================================

func TestDecideRunMode(t *testing.T) {
	sandboxed := RunModeSandboxed
	inPlace := RunModeInPlace

	tests := []struct {
		name                   string
		requestedMode          *RunMode
		forceInPlace           bool
		policyAllowsInPlace    bool
		profileRequiresSandbox bool
		wantMode               RunMode
		wantExplicit           bool
		wantPolicyOverride     bool
	}{
		{
			name:          "explicit sandboxed request",
			requestedMode: &sandboxed,
			wantMode:      RunModeSandboxed,
			wantExplicit:  true,
		},
		{
			name:          "explicit in_place request",
			requestedMode: &inPlace,
			wantMode:      RunModeInPlace,
			wantExplicit:  true,
		},
		{
			name:                "force in_place with policy permission",
			forceInPlace:        true,
			policyAllowsInPlace: true,
			wantMode:            RunModeInPlace,
			wantPolicyOverride:  true,
		},
		{
			name:                "force in_place without policy permission",
			forceInPlace:        true,
			policyAllowsInPlace: false,
			wantMode:            RunModeSandboxed, // Falls back to default
		},
		{
			name:                   "profile requires sandbox",
			profileRequiresSandbox: true,
			wantMode:               RunModeSandboxed,
		},
		{
			name:                "policy allows in_place but not forced",
			policyAllowsInPlace: true,
			wantMode:            RunModeSandboxed, // Defaults to safer option
		},
		{
			name:     "default is sandboxed",
			wantMode: RunModeSandboxed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideRunMode(
				tt.requestedMode,
				tt.forceInPlace,
				tt.policyAllowsInPlace,
				tt.profileRequiresSandbox,
			)

			if decision.Mode != tt.wantMode {
				t.Errorf("Mode = %v, want %v", decision.Mode, tt.wantMode)
			}
			if decision.ExplicitChoice != tt.wantExplicit {
				t.Errorf("ExplicitChoice = %v, want %v", decision.ExplicitChoice, tt.wantExplicit)
			}
			if decision.PolicyOverride != tt.wantPolicyOverride {
				t.Errorf("PolicyOverride = %v, want %v", decision.PolicyOverride, tt.wantPolicyOverride)
			}
			if decision.Reason == "" {
				t.Errorf("Reason should not be empty")
			}
		})
	}
}

// =============================================================================
// RESULT CLASSIFICATION TESTS
// =============================================================================

func TestClassifyRunOutcome(t *testing.T) {
	exitZero := 0
	exitOne := 1

	tests := []struct {
		name         string
		err          error
		exitCode     *int
		wasCancelled bool
		timedOut     bool
		want         RunOutcome
	}{
		{"success", nil, &exitZero, false, false, RunOutcomeSuccess},
		{"cancelled", nil, nil, true, false, RunOutcomeCancelled},
		{"timeout", nil, nil, false, true, RunOutcomeTimeout},
		{"exit error", nil, &exitOne, false, false, RunOutcomeExitError},
		{"exception", errors.New("boom"), nil, false, false, RunOutcomeException},
		{"cancelled takes priority", nil, &exitOne, true, true, RunOutcomeCancelled},
		{"timeout takes priority over exit", nil, &exitOne, false, true, RunOutcomeTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRunOutcome(tt.err, tt.exitCode, tt.wasCancelled, tt.timedOut)
			if got != tt.want {
				t.Errorf("ClassifyRunOutcome() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunOutcome_RequiresReview(t *testing.T) {
	tests := []struct {
		outcome RunOutcome
		want    bool
	}{
		{RunOutcomeSuccess, true},
		{RunOutcomeExitError, false},
		{RunOutcomeException, false},
		{RunOutcomeCancelled, false},
		{RunOutcomeTimeout, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.outcome), func(t *testing.T) {
			if got := tt.outcome.RequiresReview(); got != tt.want {
				t.Errorf("RequiresReview() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunOutcome_IsTerminalFailure(t *testing.T) {
	tests := []struct {
		outcome RunOutcome
		want    bool
	}{
		{RunOutcomeSuccess, false},
		{RunOutcomeCancelled, false},
		{RunOutcomeExitError, true},
		{RunOutcomeException, true},
		{RunOutcomeTimeout, true},
		{RunOutcomeSandboxFail, true},
		{RunOutcomeRunnerFail, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.outcome), func(t *testing.T) {
			if got := tt.outcome.IsTerminalFailure(); got != tt.want {
				t.Errorf("IsTerminalFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// SCOPE CONFLICT TESTS
// =============================================================================

func TestScopesOverlap(t *testing.T) {
	tests := []struct {
		name   string
		scopeA string
		scopeB string
		want   bool
	}{
		{"identical", "src/", "src/", true},
		{"identical normalized", "src", "/src/", true},
		{"parent-child", "src/", "src/foo", true},
		{"child-parent", "src/foo", "src/", true},
		{"deep nesting", "src/", "src/foo/bar/baz", true},
		{"siblings", "src/", "tests/", false},
		{"sibling files", "src/foo", "src/bar", false},
		{"root is ancestor", "/", "src/foo", true},
		{"root normalized", "", "src/foo", true},
		{"prefix but not ancestor", "src/fo", "src/foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScopesOverlap(tt.scopeA, tt.scopeB); got != tt.want {
				t.Errorf("ScopesOverlap(%q, %q) = %v, want %v", tt.scopeA, tt.scopeB, got, tt.want)
			}
		})
	}
}

// =============================================================================
// HELPERS
// =============================================================================

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
