package domain

import (
	"errors"
	"testing"
	"time"

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
// REJECTION DECISION TESTS
// =============================================================================

func TestRun_IsRejectable(t *testing.T) {
	sandboxID := mustParseUUID("12345678-1234-1234-1234-123456789abc")

	tests := []struct {
		name   string
		run    *Run
		wantOK bool
	}{
		{
			name: "valid - needs_review with pending approval",
			run: &Run{
				Status:        RunStatusNeedsReview,
				SandboxID:     &sandboxID,
				ApprovalState: ApprovalStatePending,
			},
			wantOK: true,
		},
		{
			name: "valid - needs_review with approved state (can still reject)",
			run: &Run{
				Status:        RunStatusNeedsReview,
				SandboxID:     &sandboxID,
				ApprovalState: ApprovalStateApproved,
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
			ok, reason := tt.run.IsRejectable()
			if ok != tt.wantOK {
				t.Errorf("IsRejectable() = %v (reason: %s), want %v", ok, reason, tt.wantOK)
			}
		})
	}
}

// =============================================================================
// RESUMPTION DECISION TESTS
// =============================================================================

func TestDecideResumption(t *testing.T) {
	staleDuration := 5 * time.Minute

	tests := []struct {
		name       string
		run        *Run
		checkpoint *RunCheckpoint
		wantResume bool
		wantReason string
	}{
		{
			name: "can resume from executing phase",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusRunning,
				Phase:  RunPhaseExecuting,
			},
			checkpoint: nil,
			wantResume: true,
		},
		{
			name: "can resume from queued phase",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusPending,
				Phase:  RunPhaseQueued,
			},
			checkpoint: nil,
			wantResume: true,
		},
		{
			name: "cannot resume completed run",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusComplete,
				Phase:  RunPhaseCompleted,
			},
			checkpoint: nil,
			wantResume: false,
			wantReason: "complete",
		},
		{
			name: "cannot resume failed run",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusFailed,
				Phase:  RunPhaseExecuting,
			},
			checkpoint: nil,
			wantResume: false,
			wantReason: "failed",
		},
		{
			name: "cannot resume cancelled run",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusCancelled,
				Phase:  RunPhaseExecuting,
			},
			checkpoint: nil,
			wantResume: false,
			wantReason: "cancelled",
		},
		{
			name: "cannot resume from collecting_results phase",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusRunning,
				Phase:  RunPhaseCollectingResults,
			},
			checkpoint: nil,
			wantResume: false,
			wantReason: "does not support resumption",
		},
		{
			name: "uses checkpoint phase when available",
			run: &Run{
				ID:     uuid.New(),
				Status: RunStatusRunning,
				Phase:  RunPhaseInitializing,
			},
			checkpoint: &RunCheckpoint{
				Phase: RunPhaseExecuting,
			},
			wantResume: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideResumption(tt.run, tt.checkpoint, staleDuration)
			if decision.CanResume != tt.wantResume {
				t.Errorf("DecideResumption().CanResume = %v, want %v", decision.CanResume, tt.wantResume)
			}
			if !tt.wantResume && tt.wantReason != "" {
				if !containsStr(decision.Reason, tt.wantReason) {
					t.Errorf("DecideResumption().Reason = %q, should contain %q", decision.Reason, tt.wantReason)
				}
			}
		})
	}
}

func TestDecideResumption_SkippedPhases(t *testing.T) {
	staleDuration := 5 * time.Minute

	t.Run("executing phase skips earlier phases", func(t *testing.T) {
		run := &Run{
			ID:     uuid.New(),
			Status: RunStatusRunning,
			Phase:  RunPhaseExecuting,
		}
		decision := DecideResumption(run, nil, staleDuration)
		if !decision.CanResume {
			t.Fatal("Expected to be able to resume")
		}
		if len(decision.SkippedPhases) != 4 {
			t.Errorf("Expected 4 skipped phases, got %d: %v", len(decision.SkippedPhases), decision.SkippedPhases)
		}
	})

	t.Run("initializing phase skips only queued", func(t *testing.T) {
		run := &Run{
			ID:     uuid.New(),
			Status: RunStatusRunning,
			Phase:  RunPhaseInitializing,
		}
		decision := DecideResumption(run, nil, staleDuration)
		if !decision.CanResume {
			t.Fatal("Expected to be able to resume")
		}
		if len(decision.SkippedPhases) != 1 {
			t.Errorf("Expected 1 skipped phase, got %d: %v", len(decision.SkippedPhases), decision.SkippedPhases)
		}
	})
}

// =============================================================================
// STALE RUN DECISION TESTS
// =============================================================================

func TestDecideStaleRunAction(t *testing.T) {
	staleDuration := 5 * time.Minute
	maxRetries := 3

	t.Run("not stale - no action", func(t *testing.T) {
		recentTime := time.Now().Add(-1 * time.Minute)
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusRunning,
			Phase:         RunPhaseExecuting,
			LastHeartbeat: &recentTime,
		}
		decision := DecideStaleRunAction(run, nil, staleDuration, maxRetries)
		if decision.IsStale {
			t.Error("Recent run should not be stale")
		}
		if decision.Action != StaleRunActionNone {
			t.Errorf("Action = %v, want %v", decision.Action, StaleRunActionNone)
		}
	})

	t.Run("stale and resumable - resume action", func(t *testing.T) {
		oldTime := time.Now().Add(-10 * time.Minute)
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusRunning,
			Phase:         RunPhaseExecuting,
			LastHeartbeat: &oldTime,
		}
		decision := DecideStaleRunAction(run, nil, staleDuration, maxRetries)
		if !decision.IsStale {
			t.Error("Old run should be stale")
		}
		if decision.Action != StaleRunActionResume {
			t.Errorf("Action = %v, want %v", decision.Action, StaleRunActionResume)
		}
	})

	t.Run("stale with max retries exceeded - fail action", func(t *testing.T) {
		oldTime := time.Now().Add(-10 * time.Minute)
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusRunning,
			Phase:         RunPhaseExecuting,
			LastHeartbeat: &oldTime,
		}
		checkpoint := &RunCheckpoint{
			RetryCount: 5, // Exceeds maxRetries
		}
		decision := DecideStaleRunAction(run, checkpoint, staleDuration, maxRetries)
		if !decision.IsStale {
			t.Error("Old run should be stale")
		}
		if decision.Action != StaleRunActionFail {
			t.Errorf("Action = %v, want %v", decision.Action, StaleRunActionFail)
		}
	})

	t.Run("stale but not resumable - alert action", func(t *testing.T) {
		oldTime := time.Now().Add(-10 * time.Minute)
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusNeedsReview, // Not a resumable status
			Phase:         RunPhaseAwaitingReview,
			LastHeartbeat: &oldTime,
		}
		decision := DecideStaleRunAction(run, nil, staleDuration, maxRetries)
		if !decision.IsStale {
			t.Error("Old run should be stale")
		}
		if decision.Action != StaleRunActionAlert {
			t.Errorf("Action = %v, want %v", decision.Action, StaleRunActionAlert)
		}
	})
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

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
