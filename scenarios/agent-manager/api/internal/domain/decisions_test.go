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

		// Continuation reactivation: a finished run with a preserved SessionID
		// can be continued back to running (see CanContinueRun / ContinueRun).
		// These edges have always been exercised at runtime and are now declared
		// in runTransitions so CanTransitionTo enforcement does not reject them.
		{"needs_review to running", RunStatusNeedsReview, RunStatusRunning, true},
		{"complete to running", RunStatusComplete, RunStatusRunning, true},
		{"failed to running", RunStatusFailed, RunStatusRunning, true},
		{"cancelled to running", RunStatusCancelled, RunStatusRunning, true},

		// Invalid transitions
		{"pending to complete", RunStatusPending, RunStatusComplete, false},
		{"complete to needs_review", RunStatusComplete, RunStatusNeedsReview, false},
		{"failed to complete", RunStatusFailed, RunStatusComplete, false},
		{"running to pending", RunStatusRunning, RunStatusPending, false},
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

// TestDeriveRunMode is the regression gate for the silent-sandbox-bypass
// class of bug. SandboxConfig.Mode is the single source of truth; every
// mode except Off must produce RunModeSandboxed, and a nil config must
// not silently default to Sandboxed (the orchestrator always populates
// a non-nil cfg via DefaultSandboxConfig before calling DeriveRunMode,
// so nil here legitimately means "no sandbox at all").
func TestDeriveRunMode(t *testing.T) {
	tests := []struct {
		name string
		cfg  *SandboxConfig
		want RunMode
	}{
		{
			name: "nil config → in-place (caller did not request a sandbox)",
			cfg:  nil,
			want: RunModeInPlace,
		},
		{
			name: "Mode=Off → in-place",
			cfg:  &SandboxConfig{Mode: SandboxModeOff},
			want: RunModeInPlace,
		},
		{
			name: "Mode=Tracking → sandboxed (host execution + tracking)",
			cfg:  &SandboxConfig{Mode: SandboxModeTracking},
			want: RunModeSandboxed,
		},
		{
			name: "Mode=Protected → sandboxed (production default)",
			cfg:  &SandboxConfig{Mode: SandboxModeProtected},
			want: RunModeSandboxed,
		},
		{
			name: "Mode=Unspecified (zero-value) → sandboxed via Effective→Tracking",
			cfg:  &SandboxConfig{},
			want: RunModeSandboxed,
		},
		{
			name: "DefaultSandboxConfig produces sandboxed",
			cfg:  DefaultSandboxConfig(),
			want: RunModeSandboxed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveRunMode(tt.cfg)
			if got != tt.want {
				t.Errorf("DeriveRunMode(%+v) = %q, want %q", tt.cfg, got, tt.want)
			}
		})
	}
}

// TestSandboxModeAtLeast covers the strictness ordering used by the
// orchestrator to enforce policy-declared minimum sandbox modes:
// Off (0) < Tracking (1) < Protected (2). SandboxModeUnspecified
// normalises to Tracking via Effective().
func TestSandboxModeAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		mode     SandboxMode
		required SandboxMode
		want     bool
	}{
		{"protected satisfies tracking", SandboxModeProtected, SandboxModeTracking, true},
		{"protected satisfies protected", SandboxModeProtected, SandboxModeProtected, true},
		{"tracking does not satisfy protected", SandboxModeTracking, SandboxModeProtected, false},
		{"tracking satisfies tracking", SandboxModeTracking, SandboxModeTracking, true},
		{"tracking satisfies off", SandboxModeTracking, SandboxModeOff, true},
		{"off does not satisfy tracking", SandboxModeOff, SandboxModeTracking, false},
		{"off does not satisfy protected", SandboxModeOff, SandboxModeProtected, false},
		{"unspecified equals tracking", SandboxModeUnspecified, SandboxModeTracking, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.AtLeast(tt.required)
			if got != tt.want {
				t.Errorf("%q.AtLeast(%q) = %v, want %v", tt.mode, tt.required, got, tt.want)
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
// LIVENESS POLICY TESTS
// =============================================================================

func TestRunStatus_LivenessPolicy(t *testing.T) {
	tests := []struct {
		status           RunStatus
		scanned          bool
		expectsHeartbeat bool
		expectsProcess   bool
		staleAction      StaleRunAction
	}{
		// Pre-start and resting/terminal states are not scanned for liveness.
		{RunStatusPending, false, false, false, StaleRunActionNone},
		{RunStatusNeedsReview, false, false, false, StaleRunActionNone},
		{RunStatusComplete, false, false, false, StaleRunActionNone},
		{RunStatusFailed, false, false, false, StaleRunActionNone},
		{RunStatusCancelled, false, false, false, StaleRunActionNone},
		// Active states expect a live executor + process and get recover-or-kill.
		{RunStatusStarting, true, true, true, StaleRunActionResume},
		{RunStatusRunning, true, true, true, StaleRunActionResume},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			p := tt.status.LivenessPolicy()
			if p.Scanned != tt.scanned {
				t.Errorf("Scanned = %v, want %v", p.Scanned, tt.scanned)
			}
			if p.ExpectsHeartbeat != tt.expectsHeartbeat {
				t.Errorf("ExpectsHeartbeat = %v, want %v", p.ExpectsHeartbeat, tt.expectsHeartbeat)
			}
			if p.ExpectsProcess != tt.expectsProcess {
				t.Errorf("ExpectsProcess = %v, want %v", p.ExpectsProcess, tt.expectsProcess)
			}
			if p.StaleAction != tt.staleAction {
				t.Errorf("StaleAction = %v, want %v", p.StaleAction, tt.staleAction)
			}
		})
	}
}

// TestRunStatus_LivenessPolicy_UnknownStatusSafeDefault verifies an
// unrecognised/free-text status gets the inert zero-value policy (not scanned),
// so the reconciler never accidentally acts on a status it does not understand.
func TestRunStatus_LivenessPolicy_UnknownStatusSafeDefault(t *testing.T) {
	p := RunStatus("some-future-status").LivenessPolicy()
	if p.Scanned || p.ExpectsHeartbeat || p.ExpectsProcess || p.StaleAction != "" {
		t.Errorf("unknown status should map to inert zero-value policy, got %+v", p)
	}
}

// TestLivenessScannedStatuses pins the exact set + order the reconciler lists
// each cycle. The reconciler refactor (Phase 1) preserved running then starting;
// Phase 2 added parked, which is scanned (for restart recovery / TTL) but — by
// its LivenessPolicy — never heartbeat-reaped or orphan-killed. Order follows
// orderedRunStatuses: running, starting, then parked.
func TestLivenessScannedStatuses(t *testing.T) {
	got := LivenessScannedStatuses()
	want := []RunStatus{RunStatusRunning, RunStatusStarting, RunStatusParked}
	if len(got) != len(want) {
		t.Fatalf("scanned statuses = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("scanned[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	// Every scanned status must actually be marked Scanned in the table.
	for _, s := range got {
		if !s.LivenessPolicy().Scanned {
			t.Errorf("%v returned by LivenessScannedStatuses but Scanned=false", s)
		}
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

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
