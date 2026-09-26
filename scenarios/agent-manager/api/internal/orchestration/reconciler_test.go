package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

type reconcilerWorkflowRecoveryStub struct{ err error }

func (s reconcilerWorkflowRecoveryStub) RecoverWorkflowExecutions(context.Context) error {
	return s.err
}

type reconcilerWaitingRecoveryStub struct{ err error }

func (s reconcilerWaitingRecoveryStub) ReconcileUnarmedWorkflowWaits(context.Context, time.Duration, time.Duration) error {
	return s.err
}

type retentionStoreStub struct {
	cutoff  time.Time
	limit   int
	deleted int
}

func (s *retentionStoreStub) DeleteBefore(_ context.Context, cutoff time.Time, limit int) (int, error) {
	s.cutoff, s.limit = cutoff, limit
	return s.deleted, nil
}

func TestReconcilerEventRetentionUsesLeverAndBoundedBatch(t *testing.T) {
	store := &retentionStoreStub{deleted: 7}
	levers := config.DefaultLevers()
	levers.Storage.EventRetentionDays = 3
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	reconciler := NewReconciler(nil, nil, WithReconcilerLevers(levers), WithReconcilerEventRetention(store))
	reconciler.clock = func() time.Time { return now }
	deleted, err := reconciler.cleanupExpiredEvents(context.Background())
	if err != nil || deleted != 7 {
		t.Fatalf("cleanupExpiredEvents = %d, %v", deleted, err)
	}
	if store.limit != eventRetentionBatchSize {
		t.Fatalf("retention batch = %d, want %d", store.limit, eventRetentionBatchSize)
	}
	if want := now.Add(-3 * 24 * time.Hour); !store.cutoff.Equal(want) {
		t.Fatalf("retention cutoff = %s, want %s", store.cutoff, want)
	}
}

func TestReconcilerArtifactRetentionUsesLeverAndBoundedBatch(t *testing.T) {
	store := &retentionStoreStub{deleted: 4}
	levers := config.DefaultLevers()
	levers.Storage.ArtifactRetentionDays = 9
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	reconciler := NewReconciler(nil, nil, WithReconcilerLevers(levers), WithReconcilerArtifactRetention(store))
	reconciler.clock = func() time.Time { return now }
	deleted, err := reconciler.cleanupExpiredArtifacts(context.Background())
	if err != nil || deleted != 4 {
		t.Fatalf("cleanupExpiredArtifacts = %d, %v", deleted, err)
	}
	if store.limit != eventRetentionBatchSize || !store.cutoff.Equal(now.Add(-9*24*time.Hour)) {
		t.Fatalf("artifact retention input = cutoff %s limit %d", store.cutoff, store.limit)
	}
}

// =============================================================================
// RECONCILER CONFIG TESTS
// =============================================================================

func TestDefaultReconcilerConfig(t *testing.T) {
	cfg := DefaultReconcilerConfig()

	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}

	if cfg.StaleThreshold != 5*time.Minute {
		t.Errorf("StaleThreshold = %v, want 5m", cfg.StaleThreshold)
	}

	if cfg.MaxRecoveryAge != 10*time.Minute {
		t.Errorf("MaxRecoveryAge = %v, want 10m", cfg.MaxRecoveryAge)
	}

	if cfg.OrphanGracePeriod != 10*time.Minute {
		t.Errorf("OrphanGracePeriod = %v, want 10m", cfg.OrphanGracePeriod)
	}

	if cfg.MaxStaleRuns != 10 {
		t.Errorf("MaxStaleRuns = %d, want 10", cfg.MaxStaleRuns)
	}
	if cfg.PendingThreshold != 5*time.Minute {
		t.Errorf("PendingThreshold = %v, want 5m", cfg.PendingThreshold)
	}

	// Production defaults - always kill orphans and auto-recover
	if !cfg.KillOrphans {
		t.Error("KillOrphans should be true by default (production mode)")
	}

	if !cfg.AutoRecover {
		t.Error("AutoRecover should be true by default (production mode)")
	}
}

func TestReconcilerConfig_CustomValues(t *testing.T) {
	cfg := ReconcilerConfig{
		Interval:          1 * time.Minute,
		StaleThreshold:    5 * time.Minute,
		OrphanGracePeriod: 10 * time.Minute,
		MaxStaleRuns:      20,
		KillOrphans:       true,
		AutoRecover:       true,
	}

	if cfg.Interval != 1*time.Minute {
		t.Errorf("Interval = %v, want 1m", cfg.Interval)
	}
	if cfg.StaleThreshold != 5*time.Minute {
		t.Errorf("StaleThreshold = %v, want 5m", cfg.StaleThreshold)
	}
	if cfg.OrphanGracePeriod != 10*time.Minute {
		t.Errorf("OrphanGracePeriod = %v, want 10m", cfg.OrphanGracePeriod)
	}
	if cfg.MaxStaleRuns != 20 {
		t.Errorf("MaxStaleRuns = %d, want 20", cfg.MaxStaleRuns)
	}
	if !cfg.KillOrphans {
		t.Error("KillOrphans should be true")
	}
	if !cfg.AutoRecover {
		t.Error("AutoRecover should be true")
	}
}

func TestReconcilerRunOnceReapsAgedPendingAndKeepsOtherRecoveryWorkIndependent(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "stranded pending", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerEvents(eventStore),
		WithReconcilerConfig(ReconcilerConfig{StaleThreshold: time.Minute, PendingThreshold: time.Nanosecond, OrphanGracePeriod: 24 * time.Hour, MaxStaleRuns: 5}),
		WithReconcilerWorkflowRecovery(reconcilerWorkflowRecoveryStub{err: errors.New("workflow backend unavailable")}),
		WithReconcilerWorkflowWaitingLiveness(reconcilerWaitingRecoveryStub{}),
	)
	stats := reconciler.RunOnce(ctx)
	if stats.RunsChecked != 1 || stats.StaleRuns != 1 || stats.WorkflowRecoveryRuns != 0 || len(stats.Errors) != 1 {
		t.Fatalf("reconcile stats=%+v", stats)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored == nil || stored.Status != domain.RunStatusFailed || stored.ErrorMsg == "" {
		t.Fatalf("pending run after reconcile=%+v err=%v", stored, err)
	}
	events, err := eventStore.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(events) == 0 || events[len(events)-1].EventType != domain.EventTypeLog {
		t.Fatalf("pending reap evidence=%+v err=%v", events, err)
	}
}

func TestReconcilerRunOnceRecordsIndependentWorkflowRecoverySuccess(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerConfig(ReconcilerConfig{OrphanGracePeriod: 24 * time.Hour}),
		WithReconcilerWorkflowRecovery(reconcilerWorkflowRecoveryStub{}),
		WithReconcilerWorkflowWaitingLiveness(reconcilerWaitingRecoveryStub{err: errors.New("wait liveness failure")}),
	)
	stats := reconciler.RunOnce(context.Background())
	if stats.WorkflowRecoveryRuns != 1 || len(stats.Errors) != 1 || stats.RunsChecked != 0 {
		t.Fatalf("independent recovery stats=%+v", stats)
	}
}

func TestReconcilerRunOnceSynchronizesApprovedAndRejectedSandboxReviews(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "review sync", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	approvedSandbox, rejectedSandbox := uuid.New(), uuid.New()
	approved := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &approvedSandbox, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview}
	rejected := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &rejectedSandbox, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview}
	for _, run := range []*domain.Run{approved, rejected} {
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	provider := mocks.NewFakeSandboxProvider()
	provider.GetFunc = func(_ context.Context, id uuid.UUID) (*sandbox.Sandbox, error) {
		status := sandbox.SandboxStatusApproved
		if id == rejectedSandbox {
			status = sandbox.SandboxStatusRejected
		}
		return &sandbox.Sandbox{ID: id, Status: status}, nil
	}
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerSandbox(provider),
		WithReconcilerConfig(ReconcilerConfig{OrphanGracePeriod: 24 * time.Hour}),
	)
	stats := reconciler.RunOnce(ctx)
	if stats.ReviewChecked != 2 || stats.ReviewSynced != 2 {
		t.Fatalf("review sync stats=%+v", stats)
	}
	approvedStored, err := repos.Runs.Get(ctx, approved.ID)
	if err != nil || approvedStored.Status != domain.RunStatusComplete || approvedStored.ApprovalState != domain.ApprovalStateApproved || approvedStored.ApprovedBy != "workspace-sandbox-sync" {
		t.Fatalf("approved run=%+v err=%v", approvedStored, err)
	}
	rejectedStored, err := repos.Runs.Get(ctx, rejected.ID)
	if err != nil || rejectedStored.Status != domain.RunStatusFailed || rejectedStored.ApprovalState != domain.ApprovalStateRejected || rejectedStored.ApprovedBy != "workspace-sandbox-sync" {
		t.Fatalf("rejected run=%+v err=%v", rejectedStored, err)
	}
}

func TestReconcilerRunOnceFailsStaleRunWhoseProcessHasExited(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "stale run", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	lastHeartbeat := time.Now().Add(-time.Hour)
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "missing-process-" + uuid.NewString(), Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, LastHeartbeat: &lastHeartbeat}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatal(err)
	}
	reconciler := NewReconciler(repos.Runs, nil, WithReconcilerConfig(ReconcilerConfig{StaleThreshold: time.Nanosecond, OrphanGracePeriod: 24 * time.Hour, MaxRecoveryAge: 0}))
	stats := reconciler.RunOnce(ctx)
	if stats.RunsChecked != 1 || stats.StaleRuns != 1 {
		t.Fatalf("stale-run stats=%+v", stats)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || stored.Status != domain.RunStatusFailed || stored.ErrorMsg == "" {
		t.Fatalf("stale run=%+v err=%v", stored, err)
	}
}

// =============================================================================
// RECONCILE STATS TESTS
// =============================================================================

func TestReconcileStats_ZeroValues(t *testing.T) {
	stats := ReconcileStats{}

	if !stats.Timestamp.IsZero() {
		t.Error("Timestamp should be zero value")
	}
	if stats.Duration != 0 {
		t.Errorf("Duration = %v, want 0", stats.Duration)
	}
	if stats.RunsChecked != 0 {
		t.Errorf("RunsChecked = %d, want 0", stats.RunsChecked)
	}
	if stats.StaleRuns != 0 {
		t.Errorf("StaleRuns = %d, want 0", stats.StaleRuns)
	}
	if stats.OrphansFound != 0 {
		t.Errorf("OrphansFound = %d, want 0", stats.OrphansFound)
	}
	if stats.RunsRecovered != 0 {
		t.Errorf("RunsRecovered = %d, want 0", stats.RunsRecovered)
	}
	if stats.OrphansKilled != 0 {
		t.Errorf("OrphansKilled = %d, want 0", stats.OrphansKilled)
	}
	if len(stats.Errors) > 0 {
		t.Error("Errors should be empty")
	}
}

func TestReconcileStats_WithData(t *testing.T) {
	now := time.Now()
	stats := ReconcileStats{
		Timestamp:     now,
		Duration:      500 * time.Millisecond,
		RunsChecked:   100,
		StaleRuns:     5,
		OrphansFound:  3,
		RunsRecovered: 2,
		OrphansKilled: 1,
		Errors:        []string{"error 1", "error 2"},
	}

	if stats.Timestamp != now {
		t.Error("Timestamp not set correctly")
	}
	if stats.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", stats.Duration)
	}
	if stats.RunsChecked != 100 {
		t.Errorf("RunsChecked = %d, want 100", stats.RunsChecked)
	}
	if stats.StaleRuns != 5 {
		t.Errorf("StaleRuns = %d, want 5", stats.StaleRuns)
	}
	if stats.OrphansFound != 3 {
		t.Errorf("OrphansFound = %d, want 3", stats.OrphansFound)
	}
	if stats.RunsRecovered != 2 {
		t.Errorf("RunsRecovered = %d, want 2", stats.RunsRecovered)
	}
	if stats.OrphansKilled != 1 {
		t.Errorf("OrphansKilled = %d, want 1", stats.OrphansKilled)
	}
	if len(stats.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(stats.Errors))
	}
}

// =============================================================================
// ORPHAN PROCESS TESTS
// =============================================================================

func TestOrphanProcess_Fields(t *testing.T) {
	now := time.Now()
	orphan := OrphanProcess{
		PID:       12345,
		Tag:       "test-run-abc123",
		Command:   "claude-code run --tag test-run-abc123",
		StartTime: now,
	}

	if orphan.PID != 12345 {
		t.Errorf("PID = %d, want 12345", orphan.PID)
	}
	if orphan.Tag != "test-run-abc123" {
		t.Errorf("Tag = %s, want test-run-abc123", orphan.Tag)
	}
	if orphan.Command != "claude-code run --tag test-run-abc123" {
		t.Errorf("Command = %s", orphan.Command)
	}
	if orphan.StartTime != now {
		t.Error("StartTime not set correctly")
	}
}

// =============================================================================
// EXTRACT TAG FROM COMMAND TESTS
// =============================================================================

func TestExtractTagFromCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "tag with space separator",
			command:  "claude-code run --tag my-tag-123 -p test",
			expected: "my-tag-123",
		},
		{
			name:     "tag with equals separator",
			command:  "claude-code run --tag=my-tag-456 -p test",
			expected: "my-tag-456",
		},
		{
			name:     "UUID tag",
			command:  "resource-claude-code run --tag 123e4567-e89b-12d3-a456-426614174000",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "no tag argument",
			command:  "claude-code run -p test",
			expected: "",
		},
		{
			name:     "empty command",
			command:  "",
			expected: "",
		},
		{
			name:     "tag at end of command",
			command:  "resource-claude-code run --verbose --tag final-tag",
			expected: "final-tag",
		},
		{
			name:     "tag with complex value",
			command:  "resource-claude-code run --tag ecosystem-task-12345-run-1",
			expected: "ecosystem-task-12345-run-1",
		},
		{
			name:     "CLAUDE_CODE_AGENT_TAG env prefix",
			command:  "env CLAUDE_CODE_AGENT_TAG=my-cc-tag-789 /usr/local/bin/resource-claude-code run -",
			expected: "my-cc-tag-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTagFromCommand(tt.command)
			if got != tt.expected {
				t.Errorf("extractTagFromCommand(%q) = %q, want %q", tt.command, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// LOOKS LIKE AGENT MANAGER TAG TESTS
// =============================================================================

func TestLooksLikeAgentManagerTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected bool
	}{
		// Valid UUIDs
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", true},

		// Known prefixes
		{"heartbeat-director-swarm-director-2026-03-16T21-12-05Z", true},
		{"ecosystem-task-123", true},
		{"test-genie-run-456", true},
		{"agent-manager-task-789", true},
		{"run-12345", true},

		// Not agent-manager tags
		{"random-tag", false},
		{"user-process", false},
		{"my-script", false},
		{"", false},
		{"abc123", false},
		{"process-12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := looksLikeAgentManagerTag(tt.tag)
			if got != tt.expected {
				t.Errorf("looksLikeAgentManagerTag(%q) = %v, want %v", tt.tag, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// NEW RECONCILER TESTS
// =============================================================================

func TestNewReconciler(t *testing.T) {
	rec := NewReconciler(nil, nil)

	if rec == nil {
		t.Fatal("NewReconciler returned nil")
	}

	// Should use defaults
	if rec.config.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", rec.config.Interval)
	}

	if rec.running {
		t.Error("should not be running initially")
	}
}

func TestNewReconciler_WithConfig(t *testing.T) {
	customCfg := ReconcilerConfig{
		Interval:       1 * time.Minute,
		StaleThreshold: 10 * time.Minute,
		KillOrphans:    true,
	}

	rec := NewReconciler(nil, nil, WithReconcilerConfig(customCfg))

	if rec.config.Interval != 1*time.Minute {
		t.Errorf("Interval = %v, want 1m", rec.config.Interval)
	}
	if rec.config.StaleThreshold != 10*time.Minute {
		t.Errorf("StaleThreshold = %v, want 10m", rec.config.StaleThreshold)
	}
	if !rec.config.KillOrphans {
		t.Error("KillOrphans should be true")
	}
}

func TestReconciler_IsRunning_Initial(t *testing.T) {
	rec := NewReconciler(nil, nil)

	if rec.IsRunning() {
		t.Error("should not be running initially")
	}
}

func TestReconciler_LastStats_Initial(t *testing.T) {
	rec := NewReconciler(nil, nil)
	stats := rec.LastStats()

	if !stats.Timestamp.IsZero() {
		t.Error("initial stats should have zero timestamp")
	}
	if stats.RunsChecked != 0 {
		t.Error("initial stats should have zero runs checked")
	}
}

// =============================================================================
// RECONCILER OPTION TESTS
// =============================================================================

func TestWithReconcilerConfig(t *testing.T) {
	cfg := ReconcilerConfig{
		Interval:    5 * time.Minute,
		KillOrphans: true,
	}

	rec := &Reconciler{}
	opt := WithReconcilerConfig(cfg)
	opt(rec)

	if rec.config.Interval != 5*time.Minute {
		t.Errorf("Interval = %v, want 5m", rec.config.Interval)
	}
	if !rec.config.KillOrphans {
		t.Error("KillOrphans should be true")
	}
}

// =============================================================================
// SCAN FOR PROCESS TESTS
// =============================================================================

func TestScanForProcess_NoMatchForNonRunnerProcesses(t *testing.T) {
	// Regression test: scanForProcess must NOT match non-runner processes
	// that happen to contain the tag string in their command line.
	// This was the root cause of stale runs never being marked as failed —
	// child processes (bash wrappers, tee, cleanup handlers) inherited the
	// tag via environment and were falsely detected as "alive" by the old
	// broad "pgrep -f <tag>" approach.
	rec := NewReconciler(nil, nil)

	// Use a unique tag that won't match any real process
	tag := "test-nonexistent-tag-for-regression-" + t.Name()

	if rec.scanForProcess(tag) {
		t.Error("scanForProcess should return false for a tag with no matching runner process")
	}
}

func TestScanRunnerProcessByTag_VerifiesRunnerName(t *testing.T) {
	// Ensure that scanRunnerProcessByTag only matches processes that are
	// known runner executables, not arbitrary processes
	rec := NewReconciler(nil, nil)

	tag := "test-verify-runner-" + t.Name()

	// Should not find any process since no real runners are running with this tag
	if rec.scanRunnerProcessByTag("claude", tag) {
		t.Error("should not find claude process with test tag")
	}
	if rec.scanRunnerProcessByTag("codex", tag) {
		t.Error("should not find codex process with test tag")
	}
	if rec.scanRunnerProcessByTag("opencode", tag) {
		t.Error("should not find opencode process with test tag")
	}
}

// =============================================================================
// RECONCILER STOP BEFORE START TESTS
// =============================================================================

func TestReconciler_Stop_NotRunning(t *testing.T) {
	rec := NewReconciler(nil, nil)

	// Should not error when stopping a non-running reconciler
	err := rec.Stop()
	if err != nil {
		t.Errorf("Stop() returned error for non-running reconciler: %v", err)
	}
}
