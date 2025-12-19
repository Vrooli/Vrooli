package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// RUN HELPER METHOD TESTS
// =============================================================================

func TestRun_GetTag(t *testing.T) {
	runID := uuid.New()

	tests := []struct {
		name     string
		run      *Run
		expected string
	}{
		{
			name:     "returns custom tag when set",
			run:      &Run{ID: runID, Tag: "my-custom-tag"},
			expected: "my-custom-tag",
		},
		{
			name:     "returns run ID when tag empty",
			run:      &Run{ID: runID, Tag: ""},
			expected: runID.String(),
		},
		{
			name:     "returns run ID for nil tag equivalent",
			run:      &Run{ID: runID},
			expected: runID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.run.GetTag()
			if got != tt.expected {
				t.Errorf("GetTag() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRun_IsResumable(t *testing.T) {
	tests := []struct {
		name   string
		run    *Run
		expect bool
	}{
		{
			name:   "resumable when pending with queued phase",
			run:    &Run{Status: RunStatusPending, Phase: RunPhaseQueued},
			expect: true,
		},
		{
			name:   "resumable when running with executing phase",
			run:    &Run{Status: RunStatusRunning, Phase: RunPhaseExecuting},
			expect: true,
		},
		{
			name:   "not resumable when complete",
			run:    &Run{Status: RunStatusComplete, Phase: RunPhaseCompleted},
			expect: false,
		},
		{
			name:   "not resumable when failed",
			run:    &Run{Status: RunStatusFailed, Phase: RunPhaseExecuting},
			expect: false,
		},
		{
			name:   "not resumable when cancelled",
			run:    &Run{Status: RunStatusCancelled, Phase: RunPhaseExecuting},
			expect: false,
		},
		{
			name:   "not resumable in awaiting review phase",
			run:    &Run{Status: RunStatusNeedsReview, Phase: RunPhaseAwaitingReview},
			expect: false,
		},
		{
			name:   "not resumable in applying phase",
			run:    &Run{Status: RunStatusRunning, Phase: RunPhaseApplying},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.run.IsResumable()
			if got != tt.expect {
				t.Errorf("IsResumable() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestRun_IsStale(t *testing.T) {
	now := time.Now()
	staleDuration := 5 * time.Minute

	tests := []struct {
		name   string
		run    *Run
		expect bool
	}{
		{
			name: "not stale when heartbeat is recent",
			run: &Run{
				LastHeartbeat: timePtr(now.Add(-1 * time.Minute)),
			},
			expect: false,
		},
		{
			name: "stale when heartbeat is old",
			run: &Run{
				LastHeartbeat: timePtr(now.Add(-10 * time.Minute)),
			},
			expect: true,
		},
		{
			name: "uses started time when no heartbeat",
			run: &Run{
				StartedAt:     timePtr(now.Add(-10 * time.Minute)),
				LastHeartbeat: nil,
			},
			expect: true,
		},
		{
			name: "not stale when started recently and no heartbeat",
			run: &Run{
				StartedAt:     timePtr(now.Add(-1 * time.Minute)),
				LastHeartbeat: nil,
			},
			expect: false,
		},
		{
			name: "not stale when never started",
			run: &Run{
				StartedAt:     nil,
				LastHeartbeat: nil,
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.run.IsStale(staleDuration)
			if got != tt.expect {
				t.Errorf("IsStale() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestRun_UpdateProgress(t *testing.T) {
	run := &Run{
		ID:     uuid.New(),
		Phase:  RunPhaseQueued,
		Status: RunStatusPending,
	}

	before := time.Now()
	run.UpdateProgress(RunPhaseExecuting, 50)
	after := time.Now()

	if run.Phase != RunPhaseExecuting {
		t.Errorf("Phase = %s, want %s", run.Phase, RunPhaseExecuting)
	}

	if run.ProgressPercent != 50 {
		t.Errorf("ProgressPercent = %d, want 50", run.ProgressPercent)
	}

	if run.LastHeartbeat == nil {
		t.Fatal("LastHeartbeat is nil")
	}

	if run.LastHeartbeat.Before(before) || run.LastHeartbeat.After(after) {
		t.Error("LastHeartbeat not set to current time")
	}

	if run.UpdatedAt.Before(before) || run.UpdatedAt.After(after) {
		t.Error("UpdatedAt not set to current time")
	}
}

// =============================================================================
// RUN CONFIG TESTS
// =============================================================================

func TestDefaultRunConfig(t *testing.T) {
	cfg := DefaultRunConfig()

	if cfg == nil {
		t.Fatal("DefaultRunConfig() returned nil")
	}

	if cfg.RunnerType != RunnerTypeClaudeCode {
		t.Errorf("RunnerType = %s, want %s", cfg.RunnerType, RunnerTypeClaudeCode)
	}

	if cfg.MaxTurns != 30 {
		t.Errorf("MaxTurns = %d, want 30", cfg.MaxTurns)
	}

	if cfg.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m", cfg.Timeout)
	}

	if !cfg.RequiresSandbox {
		t.Error("RequiresSandbox should be true by default")
	}

	if !cfg.RequiresApproval {
		t.Error("RequiresApproval should be true by default")
	}
}

func TestRunConfig_ApplyProfile(t *testing.T) {
	t.Run("applies all fields from profile", func(t *testing.T) {
		cfg := &RunConfig{}
		profile := &AgentProfile{
			RunnerType:           RunnerTypeCodex,
			Model:                "opus",
			MaxTurns:             100,
			Timeout:              time.Hour,
			AllowedTools:         []string{"Read", "Write"},
			DeniedTools:          []string{"Bash"},
			SkipPermissionPrompt: true,
			RequiresSandbox:      false,
			RequiresApproval:     false,
			AllowedPaths:         []string{"/src"},
			DeniedPaths:          []string{"/secrets"},
		}

		cfg.ApplyProfile(profile)

		if cfg.RunnerType != RunnerTypeCodex {
			t.Errorf("RunnerType = %s, want %s", cfg.RunnerType, RunnerTypeCodex)
		}
		if cfg.Model != "opus" {
			t.Errorf("Model = %s, want opus", cfg.Model)
		}
		if cfg.MaxTurns != 100 {
			t.Errorf("MaxTurns = %d, want 100", cfg.MaxTurns)
		}
		if cfg.Timeout != time.Hour {
			t.Errorf("Timeout = %v, want 1h", cfg.Timeout)
		}
		if len(cfg.AllowedTools) != 2 {
			t.Errorf("AllowedTools length = %d, want 2", len(cfg.AllowedTools))
		}
		if len(cfg.DeniedTools) != 1 {
			t.Errorf("DeniedTools length = %d, want 1", len(cfg.DeniedTools))
		}
		if !cfg.SkipPermissionPrompt {
			t.Error("SkipPermissionPrompt should be true")
		}
		if cfg.RequiresSandbox {
			t.Error("RequiresSandbox should be false")
		}
		if cfg.RequiresApproval {
			t.Error("RequiresApproval should be false")
		}
	})

	t.Run("handles nil profile safely", func(t *testing.T) {
		cfg := &RunConfig{
			RunnerType: RunnerTypeClaudeCode,
			MaxTurns:   30,
		}

		cfg.ApplyProfile(nil)

		// Should remain unchanged
		if cfg.RunnerType != RunnerTypeClaudeCode {
			t.Errorf("RunnerType = %s, want %s", cfg.RunnerType, RunnerTypeClaudeCode)
		}
		if cfg.MaxTurns != 30 {
			t.Errorf("MaxTurns = %d, want 30", cfg.MaxTurns)
		}
	})
}

// =============================================================================
// EVENT CONSTRUCTOR TESTS
// =============================================================================

func TestNewLogEvent(t *testing.T) {
	runID := uuid.New()
	event := NewLogEvent(runID, "info", "Starting execution")

	if event.ID == uuid.Nil {
		t.Error("event ID should be set")
	}
	if event.RunID != runID {
		t.Errorf("RunID = %s, want %s", event.RunID, runID)
	}
	if event.EventType != EventTypeLog {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeLog)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	data, ok := event.Data.(*LogEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *LogEventData", event.Data)
	}
	if data.Level != "info" {
		t.Errorf("Level = %s, want info", data.Level)
	}
	if data.Message != "Starting execution" {
		t.Errorf("Message = %s, want 'Starting execution'", data.Message)
	}
}

func TestNewMessageEvent(t *testing.T) {
	runID := uuid.New()
	event := NewMessageEvent(runID, "assistant", "Hello, how can I help?")

	if event.EventType != EventTypeMessage {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeMessage)
	}

	data, ok := event.Data.(*MessageEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *MessageEventData", event.Data)
	}
	if data.Role != "assistant" {
		t.Errorf("Role = %s, want assistant", data.Role)
	}
	if data.Content != "Hello, how can I help?" {
		t.Errorf("Content = %s, want 'Hello, how can I help?'", data.Content)
	}
}

func TestNewToolCallEvent(t *testing.T) {
	runID := uuid.New()
	input := map[string]interface{}{
		"path":  "/home/user/file.txt",
		"limit": 100,
	}
	event := NewToolCallEvent(runID, "Read", input)

	if event.EventType != EventTypeToolCall {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeToolCall)
	}

	data, ok := event.Data.(*ToolCallEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *ToolCallEventData", event.Data)
	}
	if data.ToolName != "Read" {
		t.Errorf("ToolName = %s, want Read", data.ToolName)
	}
	if data.Input["path"] != "/home/user/file.txt" {
		t.Errorf("Input[path] = %v, want /home/user/file.txt", data.Input["path"])
	}
}

func TestNewToolResultEvent(t *testing.T) {
	runID := uuid.New()

	t.Run("successful result", func(t *testing.T) {
		event := NewToolResultEvent(runID, "Read", "file contents here", nil)

		data, ok := event.Data.(*ToolResultEventData)
		if !ok {
			t.Fatalf("Data type = %T, want *ToolResultEventData", event.Data)
		}
		if !data.Success {
			t.Error("Success should be true for nil error")
		}
		if data.Output != "file contents here" {
			t.Errorf("Output = %s, want 'file contents here'", data.Output)
		}
		if data.Error != "" {
			t.Errorf("Error = %s, want empty", data.Error)
		}
	})

	t.Run("failed result", func(t *testing.T) {
		event := NewToolResultEvent(runID, "Write", "", errors.New("permission denied"))

		data, ok := event.Data.(*ToolResultEventData)
		if !ok {
			t.Fatalf("Data type = %T, want *ToolResultEventData", event.Data)
		}
		if data.Success {
			t.Error("Success should be false for error")
		}
		if data.Error != "permission denied" {
			t.Errorf("Error = %s, want 'permission denied'", data.Error)
		}
	})
}

func TestNewStatusEvent(t *testing.T) {
	runID := uuid.New()
	event := NewStatusEvent(runID, "pending", "running", "Execution started")

	if event.EventType != EventTypeStatus {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeStatus)
	}

	data, ok := event.Data.(*StatusEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *StatusEventData", event.Data)
	}
	if data.OldStatus != "pending" {
		t.Errorf("OldStatus = %s, want pending", data.OldStatus)
	}
	if data.NewStatus != "running" {
		t.Errorf("NewStatus = %s, want running", data.NewStatus)
	}
	if data.Reason != "Execution started" {
		t.Errorf("Reason = %s, want 'Execution started'", data.Reason)
	}
}

func TestNewMetricEvent(t *testing.T) {
	runID := uuid.New()
	event := NewMetricEvent(runID, "tokens_used", 1500.0, "tokens")

	if event.EventType != EventTypeMetric {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeMetric)
	}

	data, ok := event.Data.(*MetricEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *MetricEventData", event.Data)
	}
	if data.Name != "tokens_used" {
		t.Errorf("Name = %s, want tokens_used", data.Name)
	}
	if data.Value != 1500.0 {
		t.Errorf("Value = %f, want 1500.0", data.Value)
	}
	if data.Unit != "tokens" {
		t.Errorf("Unit = %s, want tokens", data.Unit)
	}
}

func TestNewArtifactEvent(t *testing.T) {
	runID := uuid.New()
	event := NewArtifactEvent(runID, "diff", "/tmp/run-123/changes.diff", 2048)

	if event.EventType != EventTypeArtifact {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeArtifact)
	}

	data, ok := event.Data.(*ArtifactEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *ArtifactEventData", event.Data)
	}
	if data.Type != "diff" {
		t.Errorf("Type = %s, want diff", data.Type)
	}
	if data.Path != "/tmp/run-123/changes.diff" {
		t.Errorf("Path = %s, want /tmp/run-123/changes.diff", data.Path)
	}
	if data.Size != 2048 {
		t.Errorf("Size = %d, want 2048", data.Size)
	}
}

func TestNewErrorEvent(t *testing.T) {
	runID := uuid.New()
	event := NewErrorEvent(runID, "RUNNER_UNAVAILABLE", "Claude Code is not installed", true)

	if event.EventType != EventTypeError {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeError)
	}

	data, ok := event.Data.(*ErrorEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *ErrorEventData", event.Data)
	}
	if data.Code != "RUNNER_UNAVAILABLE" {
		t.Errorf("Code = %s, want RUNNER_UNAVAILABLE", data.Code)
	}
	if data.Message != "Claude Code is not installed" {
		t.Errorf("Message = %s, want 'Claude Code is not installed'", data.Message)
	}
	if !data.Retryable {
		t.Error("Retryable should be true")
	}
}

func TestNewRateLimitEvent(t *testing.T) {
	runID := uuid.New()
	resetTime := time.Now().Add(1 * time.Hour)
	event := NewRateLimitEvent(runID, "5_hour", "Usage limit reached", &resetTime, 3600)

	data, ok := event.Data.(*RateLimitEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *RateLimitEventData", event.Data)
	}
	if data.LimitType != "5_hour" {
		t.Errorf("LimitType = %s, want 5_hour", data.LimitType)
	}
	if data.Message != "Usage limit reached" {
		t.Errorf("Message = %s, want 'Usage limit reached'", data.Message)
	}
	if data.ResetTime == nil {
		t.Error("ResetTime should not be nil")
	}
	if data.RetryAfter != 3600 {
		t.Errorf("RetryAfter = %d, want 3600", data.RetryAfter)
	}
}

func TestNewCostEvent(t *testing.T) {
	runID := uuid.New()
	event := NewCostEvent(runID, 5000, 1000, 0.15)

	data, ok := event.Data.(*CostEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *CostEventData", event.Data)
	}
	if data.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want 5000", data.InputTokens)
	}
	if data.OutputTokens != 1000 {
		t.Errorf("OutputTokens = %d, want 1000", data.OutputTokens)
	}
	if data.TotalCostUSD != 0.15 {
		t.Errorf("TotalCostUSD = %f, want 0.15", data.TotalCostUSD)
	}
}

func TestNewProgressEvent(t *testing.T) {
	runID := uuid.New()
	event := NewProgressEvent(runID, RunPhaseExecuting, 50, "Running tests")

	data, ok := event.Data.(*ProgressEventData)
	if !ok {
		t.Fatalf("Data type = %T, want *ProgressEventData", event.Data)
	}
	if data.Phase != RunPhaseExecuting {
		t.Errorf("Phase = %s, want %s", data.Phase, RunPhaseExecuting)
	}
	if data.PercentComplete != 50 {
		t.Errorf("PercentComplete = %d, want 50", data.PercentComplete)
	}
	if data.CurrentAction != "Running tests" {
		t.Errorf("CurrentAction = %s, want 'Running tests'", data.CurrentAction)
	}
}

// =============================================================================
// RUN PHASE TESTS
// =============================================================================

func TestRunPhase_CanResumeFromPhase(t *testing.T) {
	tests := []struct {
		phase  RunPhase
		expect bool
	}{
		{RunPhaseQueued, true},
		{RunPhaseInitializing, true},
		{RunPhaseSandboxCreating, true},
		{RunPhaseRunnerAcquiring, true},
		{RunPhaseExecuting, true},
		{RunPhaseCollectingResults, false},
		{RunPhaseAwaitingReview, false},
		{RunPhaseApplying, false},
		{RunPhaseCleaningUp, false},
		{RunPhaseCompleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			got := tt.phase.CanResumeFromPhase()
			if got != tt.expect {
				t.Errorf("CanResumeFromPhase() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestRunPhase_IsTerminal(t *testing.T) {
	tests := []struct {
		phase  RunPhase
		expect bool
	}{
		{RunPhaseQueued, false},
		{RunPhaseExecuting, false},
		{RunPhaseAwaitingReview, false},
		{RunPhaseCompleted, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			got := tt.phase.IsTerminal()
			if got != tt.expect {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestRunPhase_Description(t *testing.T) {
	tests := []struct {
		phase       RunPhase
		expectEmpty bool
	}{
		{RunPhaseQueued, false},
		{RunPhaseInitializing, false},
		{RunPhaseSandboxCreating, false},
		{RunPhaseRunnerAcquiring, false},
		{RunPhaseExecuting, false},
		{RunPhaseCollectingResults, false},
		{RunPhaseAwaitingReview, false},
		{RunPhaseApplying, false},
		{RunPhaseCleaningUp, false},
		{RunPhaseCompleted, false},
		{RunPhase("unknown"), false}, // Should return "Unknown phase"
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			got := tt.phase.Description()
			if (got == "") != tt.expectEmpty {
				t.Errorf("Description() empty = %v, want empty = %v", got == "", tt.expectEmpty)
			}
		})
	}

	// Test specific descriptions
	if desc := RunPhaseQueued.Description(); desc != "Waiting to start" {
		t.Errorf("RunPhaseQueued.Description() = %s, want 'Waiting to start'", desc)
	}
	if desc := RunPhaseExecuting.Description(); desc != "Agent is executing" {
		t.Errorf("RunPhaseExecuting.Description() = %s, want 'Agent is executing'", desc)
	}
}

func TestPhaseToProgress(t *testing.T) {
	tests := []struct {
		phase    RunPhase
		expected int
	}{
		{RunPhaseQueued, 0},
		{RunPhaseInitializing, 5},
		{RunPhaseSandboxCreating, 15},
		{RunPhaseRunnerAcquiring, 25},
		{RunPhaseExecuting, 50},
		{RunPhaseCollectingResults, 85},
		{RunPhaseAwaitingReview, 90},
		{RunPhaseApplying, 95},
		{RunPhaseCleaningUp, 98},
		{RunPhaseCompleted, 100},
		{RunPhase("unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			got := PhaseToProgress(tt.phase)
			if got != tt.expected {
				t.Errorf("PhaseToProgress() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// RUN CHECKPOINT TESTS
// =============================================================================

func TestNewCheckpoint(t *testing.T) {
	runID := uuid.New()
	before := time.Now()
	cp := NewCheckpoint(runID, RunPhaseInitializing)
	after := time.Now()

	if cp.RunID != runID {
		t.Errorf("RunID = %s, want %s", cp.RunID, runID)
	}
	if cp.Phase != RunPhaseInitializing {
		t.Errorf("Phase = %s, want %s", cp.Phase, RunPhaseInitializing)
	}
	if cp.LastHeartbeat.Before(before) || cp.LastHeartbeat.After(after) {
		t.Error("LastHeartbeat not set correctly")
	}
	if cp.SavedAt.Before(before) || cp.SavedAt.After(after) {
		t.Error("SavedAt not set correctly")
	}
	if cp.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestRunCheckpoint_Update(t *testing.T) {
	runID := uuid.New()
	sandboxID := uuid.New()
	lockID := uuid.New()

	original := &RunCheckpoint{
		RunID:             runID,
		Phase:             RunPhaseInitializing,
		StepWithinPhase:   0,
		SandboxID:         &sandboxID,
		WorkDir:           "/tmp/sandbox",
		LockID:            &lockID,
		LastEventSequence: 10,
		RetryCount:        1,
		Metadata:          map[string]string{"key": "value"},
	}

	updated := original.Update(RunPhaseExecuting, 5)

	// Check updated fields
	if updated.Phase != RunPhaseExecuting {
		t.Errorf("Phase = %s, want %s", updated.Phase, RunPhaseExecuting)
	}
	if updated.StepWithinPhase != 5 {
		t.Errorf("StepWithinPhase = %d, want 5", updated.StepWithinPhase)
	}

	// Check preserved fields
	if updated.RunID != runID {
		t.Errorf("RunID = %s, want %s", updated.RunID, runID)
	}
	if *updated.SandboxID != sandboxID {
		t.Error("SandboxID should be preserved")
	}
	if updated.WorkDir != "/tmp/sandbox" {
		t.Errorf("WorkDir = %s, want /tmp/sandbox", updated.WorkDir)
	}
	if *updated.LockID != lockID {
		t.Error("LockID should be preserved")
	}
	if updated.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", updated.RetryCount)
	}

	// Check original is unchanged
	if original.Phase != RunPhaseInitializing {
		t.Error("Original checkpoint should be unchanged")
	}
}

func TestRunCheckpoint_WithSandbox(t *testing.T) {
	cp := NewCheckpoint(uuid.New(), RunPhaseQueued)
	sandboxID := uuid.New()
	workDir := "/tmp/sandbox-123"

	updated := cp.WithSandbox(sandboxID, workDir)

	if *updated.SandboxID != sandboxID {
		t.Errorf("SandboxID = %s, want %s", *updated.SandboxID, sandboxID)
	}
	if updated.WorkDir != workDir {
		t.Errorf("WorkDir = %s, want %s", updated.WorkDir, workDir)
	}

	// Original should be unchanged
	if cp.SandboxID != nil {
		t.Error("Original checkpoint should be unchanged")
	}
}

func TestRunCheckpoint_WithLock(t *testing.T) {
	cp := NewCheckpoint(uuid.New(), RunPhaseQueued)
	lockID := uuid.New()

	updated := cp.WithLock(lockID)

	if *updated.LockID != lockID {
		t.Errorf("LockID = %s, want %s", *updated.LockID, lockID)
	}

	// Original should be unchanged
	if cp.LockID != nil {
		t.Error("Original checkpoint should be unchanged")
	}
}

func TestRunCheckpoint_WithEventSequence(t *testing.T) {
	cp := NewCheckpoint(uuid.New(), RunPhaseQueued)

	updated := cp.WithEventSequence(100)

	if updated.LastEventSequence != 100 {
		t.Errorf("LastEventSequence = %d, want 100", updated.LastEventSequence)
	}

	// Original should be unchanged
	if cp.LastEventSequence != 0 {
		t.Error("Original checkpoint should be unchanged")
	}
}

func TestRunCheckpoint_IncrementRetry(t *testing.T) {
	cp := NewCheckpoint(uuid.New(), RunPhaseQueued)

	updated1 := cp.IncrementRetry()
	if updated1.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", updated1.RetryCount)
	}

	updated2 := updated1.IncrementRetry()
	if updated2.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", updated2.RetryCount)
	}

	// Original should be unchanged
	if cp.RetryCount != 0 {
		t.Error("Original checkpoint should be unchanged")
	}
}

// =============================================================================
// HEARTBEAT CONFIG TESTS
// =============================================================================

func TestDefaultHeartbeatConfig(t *testing.T) {
	cfg := DefaultHeartbeatConfig()

	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}
	if cfg.Timeout != 2*time.Minute {
		t.Errorf("Timeout = %v, want 2m", cfg.Timeout)
	}
	if cfg.MaxMissedBeats != 3 {
		t.Errorf("MaxMissedBeats = %d, want 3", cfg.MaxMissedBeats)
	}
}

// =============================================================================
// LEGACY RUN EVENT DATA TESTS
// =============================================================================

func TestRunEventData_EventType(t *testing.T) {
	tests := []struct {
		name     string
		data     RunEventData
		expected RunEventType
	}{
		{
			name:     "log event by level",
			data:     RunEventData{Level: "info"},
			expected: EventTypeLog,
		},
		{
			name:     "log event by message without role",
			data:     RunEventData{Message: "Hello"},
			expected: EventTypeLog,
		},
		{
			name:     "message event by role",
			data:     RunEventData{Role: "assistant", Content: "Hello"},
			expected: EventTypeMessage,
		},
		{
			name:     "tool call event",
			data:     RunEventData{ToolName: "Read", ToolInput: map[string]interface{}{}},
			expected: EventTypeToolCall,
		},
		{
			name:     "tool result event with output",
			data:     RunEventData{ToolOutput: "file contents"},
			expected: EventTypeToolResult,
		},
		{
			name:     "tool result event with error",
			data:     RunEventData{ToolError: "not found"},
			expected: EventTypeToolResult,
		},
		{
			name:     "status event",
			data:     RunEventData{OldStatus: "pending", NewStatus: "running"},
			expected: EventTypeStatus,
		},
		{
			name:     "metric event",
			data:     RunEventData{MetricName: "tokens", MetricValue: 100},
			expected: EventTypeMetric,
		},
		{
			name:     "artifact event",
			data:     RunEventData{ArtifactType: "diff"},
			expected: EventTypeArtifact,
		},
		{
			name:     "error event by code",
			data:     RunEventData{ErrorCode: "TIMEOUT"},
			expected: EventTypeError,
		},
		{
			name:     "error event by message",
			data:     RunEventData{ErrorMessage: "Something failed"},
			expected: EventTypeError,
		},
		{
			name:     "empty defaults to log",
			data:     RunEventData{},
			expected: EventTypeLog,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.data.EventType()
			if got != tt.expected {
				t.Errorf("EventType() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestRunEventData_ToTypedPayload(t *testing.T) {
	t.Run("converts log event", func(t *testing.T) {
		data := RunEventData{Level: "warn", Message: "Warning message"}
		payload := data.ToTypedPayload()

		logData, ok := payload.(*LogEventData)
		if !ok {
			t.Fatalf("expected *LogEventData, got %T", payload)
		}
		if logData.Level != "warn" {
			t.Errorf("Level = %s, want warn", logData.Level)
		}
		if logData.Message != "Warning message" {
			t.Errorf("Message = %s, want 'Warning message'", logData.Message)
		}
	})

	t.Run("converts message event", func(t *testing.T) {
		data := RunEventData{Role: "user", Content: "Hello"}
		payload := data.ToTypedPayload()

		msgData, ok := payload.(*MessageEventData)
		if !ok {
			t.Fatalf("expected *MessageEventData, got %T", payload)
		}
		if msgData.Role != "user" {
			t.Errorf("Role = %s, want user", msgData.Role)
		}
	})

	t.Run("converts tool call event", func(t *testing.T) {
		input := map[string]interface{}{"path": "/test"}
		data := RunEventData{ToolName: "Read", ToolInput: input}
		payload := data.ToTypedPayload()

		toolData, ok := payload.(*ToolCallEventData)
		if !ok {
			t.Fatalf("expected *ToolCallEventData, got %T", payload)
		}
		if toolData.ToolName != "Read" {
			t.Errorf("ToolName = %s, want Read", toolData.ToolName)
		}
	})

	t.Run("converts tool result with error", func(t *testing.T) {
		data := RunEventData{ToolError: "not found"}
		payload := data.ToTypedPayload()

		resultData, ok := payload.(*ToolResultEventData)
		if !ok {
			t.Fatalf("expected *ToolResultEventData, got %T", payload)
		}
		if resultData.Success {
			t.Error("Success should be false when error is present")
		}
	})
}

// =============================================================================
// EVENT PAYLOAD INTERFACE TESTS
// =============================================================================

func TestEventPayload_Interface(t *testing.T) {
	payloads := []EventPayload{
		&LogEventData{Level: "info", Message: "test"},
		&MessageEventData{Role: "user", Content: "test"},
		&ToolCallEventData{ToolName: "Read", Input: nil},
		&ToolResultEventData{ToolName: "Read", Success: true},
		&StatusEventData{OldStatus: "pending", NewStatus: "running"},
		&MetricEventData{Name: "tokens", Value: 100},
		&ArtifactEventData{Type: "diff", Path: "/tmp/diff"},
		&ErrorEventData{Code: "ERROR", Message: "test"},
		&RateLimitEventData{LimitType: "5_hour"},
		&CostEventData{InputTokens: 100, OutputTokens: 50},
		&ProgressEventData{Phase: RunPhaseExecuting},
		RunEventData{Level: "info"}, // Legacy type
	}

	expectedTypes := []RunEventType{
		EventTypeLog,
		EventTypeMessage,
		EventTypeToolCall,
		EventTypeToolResult,
		EventTypeStatus,
		EventTypeMetric,
		EventTypeArtifact,
		EventTypeError,
		EventTypeError, // RateLimitEventData returns EventTypeError
		EventTypeMetric, // CostEventData returns EventTypeMetric
		EventTypeStatus, // ProgressEventData returns EventTypeStatus
		EventTypeLog,   // Legacy defaults
	}

	for i, payload := range payloads {
		if payload.EventType() != expectedTypes[i] {
			t.Errorf("payload %d: EventType() = %s, want %s", i, payload.EventType(), expectedTypes[i])
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func timePtr(t time.Time) *time.Time {
	return &t
}
