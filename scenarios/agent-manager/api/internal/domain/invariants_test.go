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

	t.Run("HasSandbox false with nil UUID", func(t *testing.T) {
		nilID := uuid.Nil
		run := &Run{SandboxID: &nilID}
		if run.HasSandbox() {
			t.Error("HasSandbox() = true, want false for nil UUID")
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

	t.Run("SafeEndedAt with nil", func(t *testing.T) {
		run := &Run{}
		if !run.SafeEndedAt().IsZero() {
			t.Error("SafeEndedAt() should be zero time when nil")
		}
	})

	t.Run("SafeEndedAt with value", func(t *testing.T) {
		run := &Run{EndedAt: &now}
		if got := run.SafeEndedAt(); !got.Equal(now) {
			t.Errorf("SafeEndedAt() = %v, want %v", got, now)
		}
	})

	t.Run("SafeLastHeartbeat with nil", func(t *testing.T) {
		run := &Run{}
		if !run.SafeLastHeartbeat().IsZero() {
			t.Error("SafeLastHeartbeat() should be zero time when nil")
		}
	})

	t.Run("SafeLastHeartbeat with value", func(t *testing.T) {
		run := &Run{LastHeartbeat: &now}
		if got := run.SafeLastHeartbeat(); !got.Equal(now) {
			t.Errorf("SafeLastHeartbeat() = %v, want %v", got, now)
		}
	})

	t.Run("SafeApprovedAt with nil", func(t *testing.T) {
		run := &Run{}
		if !run.SafeApprovedAt().IsZero() {
			t.Error("SafeApprovedAt() should be zero time when nil")
		}
	})

	t.Run("SafeApprovedAt with value", func(t *testing.T) {
		run := &Run{ApprovedAt: &now}
		if got := run.SafeApprovedAt(); !got.Equal(now) {
			t.Errorf("SafeApprovedAt() = %v, want %v", got, now)
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

	t.Run("SafeExitCode with non-zero value", func(t *testing.T) {
		nonZeroCode := 127
		run := &Run{ExitCode: &nonZeroCode}
		if got := run.SafeExitCode(); got != 127 {
			t.Errorf("SafeExitCode() = %v, want 127", got)
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

	t.Run("Duration still running", func(t *testing.T) {
		start := time.Now().Add(-2 * time.Minute)
		run := &Run{StartedAt: &start, EndedAt: nil}
		got := run.Duration()
		// Should return elapsed time since start (approximately 2 minutes)
		if got < 1*time.Minute || got > 3*time.Minute {
			t.Errorf("Duration() = %v, want ~2 minutes for still running", got)
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

func TestIdempotencyRecord_SafeEntityID(t *testing.T) {
	entityID := uuid.New()

	t.Run("SafeEntityID with nil", func(t *testing.T) {
		record := &IdempotencyRecord{}
		if got := record.SafeEntityID(); got != uuid.Nil {
			t.Errorf("SafeEntityID() = %v, want uuid.Nil", got)
		}
	})

	t.Run("SafeEntityID with value", func(t *testing.T) {
		record := &IdempotencyRecord{EntityID: &entityID}
		if got := record.SafeEntityID(); got != entityID {
			t.Errorf("SafeEntityID() = %v, want %v", got, entityID)
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

	t.Run("AssertCanReject - needs_review", func(t *testing.T) {
		run := &Run{
			Status:        RunStatusNeedsReview,
			ApprovalState: ApprovalStatePending,
		}
		if err := run.AssertCanReject(); err != nil {
			t.Errorf("AssertCanReject() error = %v, want nil", err)
		}
	})

	t.Run("AssertCanReject - running", func(t *testing.T) {
		run := &Run{Status: RunStatusRunning}
		if err := run.AssertCanReject(); err == nil {
			t.Error("AssertCanReject() should error for running status")
		}
	})

	t.Run("AssertCanReject - already rejected", func(t *testing.T) {
		run := &Run{
			Status:        RunStatusNeedsReview,
			ApprovalState: ApprovalStateRejected,
		}
		if err := run.AssertCanReject(); err == nil {
			t.Error("AssertCanReject() should error for already rejected run")
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

// =============================================================================
// TASK TERMINAL TRANSITION TESTS
// =============================================================================

func TestTask_CheckTerminalTransition(t *testing.T) {
	oldMode := invariantMode
	SetInvariantMode(InvariantModeLog)
	defer SetInvariantMode(oldMode)

	tests := []struct {
		name      string
		current   TaskStatus
		proposed  TaskStatus
		wantValid bool
	}{
		{"queued to running - valid", TaskStatusQueued, TaskStatusRunning, true},
		{"running to needs_review - valid", TaskStatusRunning, TaskStatusNeedsReview, true},
		{"approved to running - invalid (terminal)", TaskStatusApproved, TaskStatusRunning, false},
		{"rejected to queued - invalid (terminal)", TaskStatusRejected, TaskStatusQueued, false},
		{"failed to running - invalid (terminal)", TaskStatusFailed, TaskStatusRunning, false},
		{"cancelled to running - invalid (terminal)", TaskStatusCancelled, TaskStatusRunning, false},
		{"approved to failed - valid (terminal to terminal)", TaskStatusApproved, TaskStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{ID: uuid.New(), Status: tt.current}
			if got := task.CheckTerminalTransition(tt.proposed); got != tt.wantValid {
				t.Errorf("CheckTerminalTransition() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

// =============================================================================
// CHECK ALL INVARIANTS TESTS
// =============================================================================

func TestRun_CheckAllInvariants(t *testing.T) {
	sandboxID := uuid.New()

	t.Run("valid run - no violations", func(t *testing.T) {
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusNeedsReview,
			RunMode:       RunModeSandboxed,
			Phase:         RunPhaseAwaitingReview,
			SandboxID:     &sandboxID,
			ApprovalState: ApprovalStatePending,
		}
		violations := run.CheckAllInvariants()
		if len(violations) != 0 {
			t.Errorf("CheckAllInvariants() found %d violations, want 0", len(violations))
		}
	})

	t.Run("sandbox invariant violation", func(t *testing.T) {
		run := &Run{
			ID:        uuid.New(),
			Status:    RunStatusRunning,
			RunMode:   RunModeSandboxed,
			Phase:     RunPhaseExecuting,
			SandboxID: nil, // Violation: sandboxed mode past sandbox creation without ID
		}
		violations := run.CheckAllInvariants()
		if len(violations) == 0 {
			t.Error("CheckAllInvariants() should detect sandbox invariant violation")
		}
	})

	t.Run("approval state invariant violation", func(t *testing.T) {
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusRunning, // Not needs_review
			ApprovalState: ApprovalStatePending,
		}
		violations := run.CheckAllInvariants()
		if len(violations) == 0 {
			t.Error("CheckAllInvariants() should detect approval state invariant violation")
		}
	})

	t.Run("in_place mode - no sandbox violation", func(t *testing.T) {
		run := &Run{
			ID:            uuid.New(),
			Status:        RunStatusRunning,
			RunMode:       RunModeInPlace,
			Phase:         RunPhaseExecuting,
			SandboxID:     nil, // Not a violation for in_place mode
			ApprovalState: ApprovalStateNone,
		}
		violations := run.CheckAllInvariants()
		if len(violations) != 0 {
			t.Errorf("CheckAllInvariants() found %d violations for in_place mode, want 0", len(violations))
		}
	})
}

// =============================================================================
// INVARIANT VIOLATION ERROR TESTS
// =============================================================================

func TestInvariantViolationError(t *testing.T) {
	err := &InvariantViolationError{
		Invariant:   "INV_TEST",
		Description: "test description",
		Entity:      "Run",
		EntityID:    "abc-123",
		Details:     "extra details",
	}

	errString := err.Error()
	if errString == "" {
		t.Error("Error() returned empty string")
	}
	if !contains(errString, "INV_TEST") {
		t.Error("Error() should contain invariant name")
	}
	if !contains(errString, "Run") {
		t.Error("Error() should contain entity type")
	}
	if !contains(errString, "abc-123") {
		t.Error("Error() should contain entity ID")
	}
	if !contains(errString, "test description") {
		t.Error("Error() should contain description")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// PHASE ORDINAL EDGE CASE TESTS
// =============================================================================

func TestPhaseOrdinal_UnknownPhase(t *testing.T) {
	// Test that unknown phase returns 0
	unknownPhase := RunPhase("unknown_phase")
	if got := phaseOrdinal(unknownPhase); got != 0 {
		t.Errorf("phaseOrdinal(unknown) = %d, want 0", got)
	}
}
