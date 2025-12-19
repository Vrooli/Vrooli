package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// INVARIANT CHECKING TESTS
// =============================================================================

func TestRun_CheckRunSandboxInvariant(t *testing.T) {
	// Set to log mode so we don't panic during tests
	oldMode := invariantMode
	SetInvariantMode(InvariantModeLog)
	defer SetInvariantMode(oldMode)

	sandboxID := uuid.New()

	tests := []struct {
		name string
		run  *Run
		want bool
	}{
		{
			name: "in_place mode - no sandbox required",
			run: &Run{
				ID:      uuid.New(),
				RunMode: RunModeInPlace,
				Phase:   RunPhaseExecuting,
			},
			want: true,
		},
		{
			name: "sandboxed mode - before sandbox creation",
			run: &Run{
				ID:        uuid.New(),
				RunMode:   RunModeSandboxed,
				Phase:     RunPhaseQueued,
				SandboxID: nil,
			},
			want: true,
		},
		{
			name: "sandboxed mode - during sandbox creation",
			run: &Run{
				ID:        uuid.New(),
				RunMode:   RunModeSandboxed,
				Phase:     RunPhaseSandboxCreating,
				SandboxID: nil,
			},
			want: true,
		},
		{
			name: "sandboxed mode - after sandbox creation with ID",
			run: &Run{
				ID:        uuid.New(),
				RunMode:   RunModeSandboxed,
				Phase:     RunPhaseExecuting,
				SandboxID: &sandboxID,
			},
			want: true,
		},
		{
			name: "sandboxed mode - after sandbox creation without ID (violation)",
			run: &Run{
				ID:        uuid.New(),
				RunMode:   RunModeSandboxed,
				Phase:     RunPhaseExecuting,
				SandboxID: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.run.CheckRunSandboxInvariant(); got != tt.want {
				t.Errorf("CheckRunSandboxInvariant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_CheckApprovalStateInvariant(t *testing.T) {
	oldMode := invariantMode
	SetInvariantMode(InvariantModeLog)
	defer SetInvariantMode(oldMode)

	tests := []struct {
		name string
		run  *Run
		want bool
	}{
		{
			name: "pending status with none approval - valid",
			run: &Run{
				ID:            uuid.New(),
				Status:        RunStatusPending,
				ApprovalState: ApprovalStateNone,
			},
			want: true,
		},
		{
			name: "running status with none approval - valid",
			run: &Run{
				ID:            uuid.New(),
				Status:        RunStatusRunning,
				ApprovalState: ApprovalStateNone,
			},
			want: true,
		},
		{
			name: "needs_review with pending approval - valid",
			run: &Run{
				ID:            uuid.New(),
				Status:        RunStatusNeedsReview,
				ApprovalState: ApprovalStatePending,
			},
			want: true,
		},
		{
			name: "needs_review with approved by set - valid",
			run: &Run{
				ID:            uuid.New(),
				Status:        RunStatusNeedsReview,
				ApprovalState: ApprovalStateApproved,
				ApprovedBy:    "user@example.com",
			},
			want: true,
		},
		{
			name: "approved without approvedBy - violation",
			run: &Run{
				ID:            uuid.New(),
				Status:        RunStatusNeedsReview,
				ApprovalState: ApprovalStateApproved,
				ApprovedBy:    "",
			},
			want: false,
		},
		{
			name: "running status with approval state - violation",
			run: &Run{
				ID:            uuid.New(),
				Status:        RunStatusRunning,
				ApprovalState: ApprovalStatePending,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.run.CheckApprovalStateInvariant(); got != tt.want {
				t.Errorf("CheckApprovalStateInvariant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_CheckTerminalTransition(t *testing.T) {
	oldMode := invariantMode
	SetInvariantMode(InvariantModeLog)
	defer SetInvariantMode(oldMode)

	tests := []struct {
		name      string
		current   RunStatus
		proposed  RunStatus
		wantValid bool
	}{
		{"pending to running - valid", RunStatusPending, RunStatusRunning, true},
		{"running to complete - valid", RunStatusRunning, RunStatusComplete, true},
		{"complete to running - invalid", RunStatusComplete, RunStatusRunning, false},
		{"failed to pending - invalid", RunStatusFailed, RunStatusPending, false},
		{"cancelled to running - invalid", RunStatusCancelled, RunStatusRunning, false},
		{"complete to failed - valid (terminal to terminal)", RunStatusComplete, RunStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{ID: uuid.New(), Status: tt.current}
			if got := run.CheckTerminalTransition(tt.proposed); got != tt.wantValid {
				t.Errorf("CheckTerminalTransition() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestRun_CheckPhaseTransition(t *testing.T) {
	oldMode := invariantMode
	SetInvariantMode(InvariantModeLog)
	defer SetInvariantMode(oldMode)

	tests := []struct {
		name      string
		current   RunPhase
		proposed  RunPhase
		wantValid bool
	}{
		{"queued to initializing - valid", RunPhaseQueued, RunPhaseInitializing, true},
		{"initializing to sandbox - valid", RunPhaseInitializing, RunPhaseSandboxCreating, true},
		{"executing to collecting - valid", RunPhaseExecuting, RunPhaseCollectingResults, true},
		{"executing to queued - invalid (backward)", RunPhaseExecuting, RunPhaseQueued, false},
		{"sandbox to initializing - invalid (backward)", RunPhaseSandboxCreating, RunPhaseInitializing, false},
		{"same phase - valid (retry)", RunPhaseExecuting, RunPhaseExecuting, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{ID: uuid.New(), Phase: tt.current}
			if got := run.CheckPhaseTransition(tt.proposed); got != tt.wantValid {
				t.Errorf("CheckPhaseTransition() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

// =============================================================================
// SAFE ACCESSOR TESTS
// =============================================================================

func TestRun_SafeAccessors(t *testing.T) {
	now := time.Now()
	sandboxID := uuid.New()
	checkpointID := uuid.New()
	exitCode := 0

	t.Run("SafeSandboxID with nil", func(t *testing.T) {
		run := &Run{}
		if got := run.SafeSandboxID(); got != uuid.Nil {
			t.Errorf("SafeSandboxID() = %v, want uuid.Nil", got)
		}
	})

	t.Run("SafeSandboxID with value", func(t *testing.T) {
		run := &Run{SandboxID: &sandboxID}
		if got := run.SafeSandboxID(); got != sandboxID {
			t.Errorf("SafeSandboxID() = %v, want %v", got, sandboxID)
		}
	})

	t.Run("HasSandbox false", func(t *testing.T) {
		run := &Run{}
		if run.HasSandbox() {
			t.Error("HasSandbox() = true, want false")
		}
	})

	t.Run("HasSandbox true", func(t *testing.T) {
		run := &Run{SandboxID: &sandboxID}
		if !run.HasSandbox() {
			t.Error("HasSandbox() = false, want true")
		}
	})

	t.Run("SafeStartedAt with nil", func(t *testing.T) {
		run := &Run{}
		if !run.SafeStartedAt().IsZero() {
			t.Error("SafeStartedAt() should be zero time when nil")
		}
	})

	t.Run("SafeStartedAt with value", func(t *testing.T) {
		run := &Run{StartedAt: &now}
		if got := run.SafeStartedAt(); !got.Equal(now) {
			t.Errorf("SafeStartedAt() = %v, want %v", got, now)
		}
	})

	t.Run("SafeExitCode with nil", func(t *testing.T) {
		run := &Run{}
		if got := run.SafeExitCode(); got != -1 {
			t.Errorf("SafeExitCode() = %v, want -1", got)
		}
	})

	t.Run("SafeExitCode with value", func(t *testing.T) {
		run := &Run{ExitCode: &exitCode}
		if got := run.SafeExitCode(); got != 0 {
			t.Errorf("SafeExitCode() = %v, want 0", got)
		}
	})

	t.Run("SafeLastCheckpointID with nil", func(t *testing.T) {
		run := &Run{}
		if got := run.SafeLastCheckpointID(); got != uuid.Nil {
			t.Errorf("SafeLastCheckpointID() = %v, want uuid.Nil", got)
		}
	})

	t.Run("SafeLastCheckpointID with value", func(t *testing.T) {
		run := &Run{LastCheckpointID: &checkpointID}
		if got := run.SafeLastCheckpointID(); got != checkpointID {
			t.Errorf("SafeLastCheckpointID() = %v, want %v", got, checkpointID)
		}
	})

	t.Run("Duration not started", func(t *testing.T) {
		run := &Run{}
		if got := run.Duration(); got != 0 {
			t.Errorf("Duration() = %v, want 0", got)
		}
	})

	t.Run("Duration completed", func(t *testing.T) {
		start := now.Add(-5 * time.Minute)
		run := &Run{StartedAt: &start, EndedAt: &now}
		got := run.Duration()
		if got < 4*time.Minute || got > 6*time.Minute {
			t.Errorf("Duration() = %v, want ~5 minutes", got)
		}
	})
}

func TestRunCheckpoint_SafeAccessors(t *testing.T) {
	sandboxID := uuid.New()
	lockID := uuid.New()

	t.Run("SafeSandboxID with nil", func(t *testing.T) {
		cp := &RunCheckpoint{}
		if got := cp.SafeSandboxID(); got != uuid.Nil {
			t.Errorf("SafeSandboxID() = %v, want uuid.Nil", got)
		}
	})

	t.Run("SafeSandboxID with value", func(t *testing.T) {
		cp := &RunCheckpoint{SandboxID: &sandboxID}
		if got := cp.SafeSandboxID(); got != sandboxID {
			t.Errorf("SafeSandboxID() = %v, want %v", got, sandboxID)
		}
	})

	t.Run("SafeLockID with nil", func(t *testing.T) {
		cp := &RunCheckpoint{}
		if got := cp.SafeLockID(); got != uuid.Nil {
			t.Errorf("SafeLockID() = %v, want uuid.Nil", got)
		}
	})

	t.Run("SafeLockID with value", func(t *testing.T) {
		cp := &RunCheckpoint{LockID: &lockID}
		if got := cp.SafeLockID(); got != lockID {
			t.Errorf("SafeLockID() = %v, want %v", got, lockID)
		}
	})
}

// =============================================================================
// STATE ASSERTION TESTS
// =============================================================================

func TestRun_Assertions(t *testing.T) {
	sandboxID := uuid.New()

	t.Run("AssertCanStart - pending", func(t *testing.T) {
		run := &Run{Status: RunStatusPending}
		if err := run.AssertCanStart(); err != nil {
			t.Errorf("AssertCanStart() error = %v, want nil", err)
		}
	})

	t.Run("AssertCanStart - running", func(t *testing.T) {
		run := &Run{Status: RunStatusRunning}
		if err := run.AssertCanStart(); err == nil {
			t.Error("AssertCanStart() should error for running status")
		}
	})

	t.Run("AssertCanStop - running", func(t *testing.T) {
		run := &Run{Status: RunStatusRunning}
		if err := run.AssertCanStop(); err != nil {
			t.Errorf("AssertCanStop() error = %v, want nil", err)
		}
	})

	t.Run("AssertCanStop - pending", func(t *testing.T) {
		run := &Run{Status: RunStatusPending}
		if err := run.AssertCanStop(); err == nil {
			t.Error("AssertCanStop() should error for pending status")
		}
	})

	t.Run("AssertCanApprove - needs_review with sandbox", func(t *testing.T) {
		run := &Run{
			Status:        RunStatusNeedsReview,
			SandboxID:     &sandboxID,
			ApprovalState: ApprovalStatePending,
		}
		if err := run.AssertCanApprove(); err != nil {
			t.Errorf("AssertCanApprove() error = %v, want nil", err)
		}
	})

	t.Run("AssertCanApprove - running", func(t *testing.T) {
		run := &Run{Status: RunStatusRunning}
		if err := run.AssertCanApprove(); err == nil {
			t.Error("AssertCanApprove() should error for running status")
		}
	})

	t.Run("AssertCanResume - stalled run", func(t *testing.T) {
		run := &Run{
			Status: RunStatusRunning,
			Phase:  RunPhaseExecuting,
		}
		if err := run.AssertCanResume(); err != nil {
			t.Errorf("AssertCanResume() error = %v, want nil", err)
		}
	})

	t.Run("AssertCanResume - completed run", func(t *testing.T) {
		run := &Run{
			Status: RunStatusComplete,
			Phase:  RunPhaseCompleted,
		}
		if err := run.AssertCanResume(); err == nil {
			t.Error("AssertCanResume() should error for completed run")
		}
	})
}

func TestTask_AssertCanCancel(t *testing.T) {
	t.Run("queued", func(t *testing.T) {
		task := &Task{Status: TaskStatusQueued}
		if err := task.AssertCanCancel(); err != nil {
			t.Errorf("AssertCanCancel() error = %v, want nil", err)
		}
	})

	t.Run("running", func(t *testing.T) {
		task := &Task{Status: TaskStatusRunning}
		if err := task.AssertCanCancel(); err != nil {
			t.Errorf("AssertCanCancel() error = %v, want nil", err)
		}
	})

	t.Run("approved", func(t *testing.T) {
		task := &Task{Status: TaskStatusApproved}
		if err := task.AssertCanCancel(); err == nil {
			t.Error("AssertCanCancel() should error for approved status")
		}
	})
}

// =============================================================================
// LIFECYCLE STATE TESTS
// =============================================================================

func TestRun_GetLifecycleState(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   LifecycleState
	}{
		{RunStatusPending, LifecycleStateNew},
		{RunStatusStarting, LifecycleStateActive},
		{RunStatusRunning, LifecycleStateActive},
		{RunStatusNeedsReview, LifecycleStateReviewable},
		{RunStatusComplete, LifecycleStateFinished},
		{RunStatusFailed, LifecycleStateFinished},
		{RunStatusCancelled, LifecycleStateFinished},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			run := &Run{Status: tt.status}
			if got := run.GetLifecycleState(); got != tt.want {
				t.Errorf("GetLifecycleState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_CanReceiveEvents(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   bool
	}{
		{RunStatusPending, false},
		{RunStatusStarting, true},
		{RunStatusRunning, true},
		{RunStatusNeedsReview, true},
		{RunStatusComplete, false},
		{RunStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			run := &Run{Status: tt.status}
			if got := run.CanReceiveEvents(); got != tt.want {
				t.Errorf("CanReceiveEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRun_CanReceiveHeartbeats(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   bool
	}{
		{RunStatusPending, false},
		{RunStatusStarting, true},
		{RunStatusRunning, true},
		{RunStatusNeedsReview, false},
		{RunStatusComplete, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			run := &Run{Status: tt.status}
			if got := run.CanReceiveHeartbeats(); got != tt.want {
				t.Errorf("CanReceiveHeartbeats() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// SCOPE CONFLICT TESTS
// =============================================================================

func TestClassifyScopeConflict(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		existing  string
		want      string
	}{
		{"exact match", "src/", "src/", "exact"},
		{"ancestor", "src/", "src/foo/bar", "ancestor"},
		{"descendant", "src/foo/bar", "src/", "descendant"},
		{"no conflict", "src/", "tests/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyScopeConflict(tt.requested, tt.existing); got != tt.want {
				t.Errorf("ClassifyScopeConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// PHASE ORDINAL TESTS
// =============================================================================

func TestPhaseOrdinal(t *testing.T) {
	// Verify phases are in strictly increasing order
	phases := []RunPhase{
		RunPhaseQueued,
		RunPhaseInitializing,
		RunPhaseSandboxCreating,
		RunPhaseRunnerAcquiring,
		RunPhaseExecuting,
		RunPhaseCollectingResults,
		RunPhaseAwaitingReview,
		RunPhaseApplying,
		RunPhaseCleaningUp,
		RunPhaseCompleted,
	}

	for i := 0; i < len(phases)-1; i++ {
		current := phaseOrdinal(phases[i])
		next := phaseOrdinal(phases[i+1])
		if current >= next {
			t.Errorf("phase %s (ord=%d) should be < phase %s (ord=%d)",
				phases[i], current, phases[i+1], next)
		}
	}
}

// =============================================================================
// INVARIANT MODE TESTS
// =============================================================================

func TestInvariantMode(t *testing.T) {
	oldMode := invariantMode

	t.Run("log mode doesn't panic", func(t *testing.T) {
		SetInvariantMode(InvariantModeLog)
		defer func() {
			if r := recover(); r != nil {
				t.Error("Log mode should not panic")
			}
		}()

		run := &Run{
			ID:        uuid.New(),
			RunMode:   RunModeSandboxed,
			Phase:     RunPhaseExecuting,
			SandboxID: nil, // Violation
		}
		run.CheckRunSandboxInvariant()
	})

	t.Run("disabled mode doesn't panic", func(t *testing.T) {
		SetInvariantMode(InvariantModeDisabled)
		defer func() {
			if r := recover(); r != nil {
				t.Error("Disabled mode should not panic")
			}
		}()

		run := &Run{
			ID:        uuid.New(),
			RunMode:   RunModeSandboxed,
			Phase:     RunPhaseExecuting,
			SandboxID: nil, // Violation
		}
		run.CheckRunSandboxInvariant()
	})

	// Restore
	SetInvariantMode(oldMode)
}
