package database

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // SQLite driver for tests
)

func TestCheckpointRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run first
	task := &domain.Task{ID: uuid.New(), Title: "Checkpoint Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	checkpoint := &domain.RunCheckpoint{
		RunID:             run.ID,
		Phase:             domain.RunPhaseExecuting,
		StepWithinPhase:   5,
		WorkDir:           "/tmp/work",
		LastEventSequence: 10,
		LastHeartbeat:     time.Now(),
		RetryCount:        0,
		Metadata:          map[string]string{"key": "value"},
	}

	// Save (insert)
	if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get
	got, err := repos.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Phase != domain.RunPhaseExecuting {
		t.Errorf("expected phase executing, got %q", got.Phase)
	}
	if got.StepWithinPhase != 5 {
		t.Errorf("expected step 5, got %d", got.StepWithinPhase)
	}
	if got.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", got.Metadata)
	}

	// Save (upsert)
	checkpoint.StepWithinPhase = 10
	checkpoint.LastEventSequence = 20
	if err := repos.Checkpoints.Save(ctx, checkpoint); err != nil {
		t.Fatalf("Save update: %v", err)
	}

	got, err = repos.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.StepWithinPhase != 10 {
		t.Errorf("expected step 10, got %d", got.StepWithinPhase)
	}

	// Heartbeat
	if err := repos.Checkpoints.Heartbeat(ctx, run.ID); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	// ListStale - checkpoint is fresh, should not appear
	stale, err := repos.Checkpoints.ListStale(ctx, 1*time.Hour)
	if err != nil {
		t.Fatalf("ListStale: %v", err)
	}
	if len(stale) != 0 {
		t.Errorf("expected 0 stale checkpoints, got %d", len(stale))
	}

	// Delete
	if err := repos.Checkpoints.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Checkpoints.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// ============================================================================
// Idempotency Repository Tests
// ============================================================================

func TestIdempotencyRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	key := "test-idempotency-key"
	ttl := 1 * time.Hour

	// Check non-existent key
	record, err := repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if record != nil {
		t.Error("expected nil for non-existent key")
	}

	// Reserve
	reserved, err := repos.Idempotency.Reserve(ctx, key, ttl)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if reserved == nil {
		t.Fatal("Reserve returned nil")
	}
	if reserved.Status != domain.IdempotencyStatusPending {
		t.Errorf("expected pending status, got %q", reserved.Status)
	}

	// Check reserved key
	record, err = repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check reserved: %v", err)
	}
	if record == nil {
		t.Fatal("expected record for reserved key")
	}
	if record.Status != domain.IdempotencyStatusPending {
		t.Errorf("expected pending status, got %q", record.Status)
	}

	// Complete
	entityID := uuid.New()
	if err := repos.Idempotency.Complete(ctx, key, entityID, "run", []byte(`{"result": "ok"}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	record, err = repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check completed: %v", err)
	}
	if record.Status != domain.IdempotencyStatusComplete {
		t.Errorf("expected complete status, got %q", record.Status)
	}
	if record.EntityID == nil || *record.EntityID != entityID {
		t.Errorf("expected entity ID %s, got %v", entityID, record.EntityID)
	}

	// Fail a different key
	failKey := "fail-key"
	if _, err := repos.Idempotency.Reserve(ctx, failKey, ttl); err != nil {
		t.Fatalf("Reserve fail key: %v", err)
	}
	if err := repos.Idempotency.Fail(ctx, failKey); err != nil {
		t.Fatalf("Fail: %v", err)
	}

	record, err = repos.Idempotency.Check(ctx, failKey)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if record.Status != domain.IdempotencyStatusFailed {
		t.Errorf("expected failed status, got %q", record.Status)
	}
}

func TestIdempotencyExpiration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Reserve with very short TTL (already expired)
	key := "expired-key"
	ttl := -1 * time.Hour // Negative TTL = already expired
	if _, err := repos.Idempotency.Reserve(ctx, key, ttl); err != nil {
		t.Fatalf("Reserve: %v", err)
	}

	// Check should return nil for expired key
	record, err := repos.Idempotency.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check expired: %v", err)
	}
	if record != nil {
		t.Error("expected nil for expired key")
	}

	// Cleanup expired
	count, err := repos.Idempotency.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 cleaned up, got %d", count)
	}
}

// ============================================================================
// Policy Repository Tests
// ============================================================================

func TestPolicyCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	policy := &domain.Policy{
		ID:           uuid.New(),
		Name:         "test-policy",
		Description:  "A test policy",
		Priority:     100,
		ScopePattern: "/home/*",
		Rules: domain.PolicyRules{
			RequireSandbox:    func() *bool { b := true; return &b }(),
			RequireApproval:   func() *bool { b := false; return &b }(),
			MaxConcurrentRuns: func() *int { n := 5; return &n }(),
			AllowedRunners:    []domain.RunnerType{domain.RunnerTypeClaudeCode},
		},
		Enabled:   true,
		CreatedBy: "test-user",
	}

	// Create
	if err := repos.Policies.Create(ctx, policy); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repos.Policies.Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != policy.Name {
		t.Errorf("expected name %q, got %q", policy.Name, got.Name)
	}
	if got.Priority != 100 {
		t.Errorf("expected priority 100, got %d", got.Priority)
	}
	if len(got.Rules.AllowedRunners) != 1 {
		t.Errorf("expected 1 allowed runner, got %d", len(got.Rules.AllowedRunners))
	}
	if got.Rules.RequireSandbox == nil || !*got.Rules.RequireSandbox {
		t.Error("expected RequireSandbox to be true")
	}

	// List
	policies, err := repos.Policies.List(ctx, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}

	// ListEnabled
	enabledPolicies, err := repos.Policies.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(enabledPolicies) != 1 {
		t.Errorf("expected 1 enabled policy, got %d", len(enabledPolicies))
	}

	// Update - disable
	policy.Enabled = false
	if err := repos.Policies.Update(ctx, policy); err != nil {
		t.Fatalf("Update: %v", err)
	}

	enabledPolicies, err = repos.Policies.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled after disable: %v", err)
	}
	if len(enabledPolicies) != 0 {
		t.Errorf("expected 0 enabled policies, got %d", len(enabledPolicies))
	}

	// Delete
	if err := repos.Policies.Delete(ctx, policy.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Policies.Get(ctx, policy.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// ============================================================================
// Lock Repository Tests
// ============================================================================

func TestLockRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run first
	task := &domain.Task{ID: uuid.New(), Title: "Lock Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	lock := &domain.ScopeLock{
		ID:          uuid.New(),
		RunID:       run.ID,
		ScopePath:   "/home/user/project",
		ProjectRoot: "/home/user",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	// Acquire
	if err := repos.Locks.Acquire(ctx, lock); err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Check
	locks, err := repos.Locks.Check(ctx, lock.ScopePath, lock.ProjectRoot)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(locks) != 1 {
		t.Errorf("expected 1 lock, got %d", len(locks))
	}

	// Refresh
	newExpiry := time.Now().Add(2 * time.Hour).Unix()
	if err := repos.Locks.Refresh(ctx, lock.ID, newExpiry); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	// Release
	if err := repos.Locks.Release(ctx, lock.ID); err != nil {
		t.Fatalf("Release: %v", err)
	}

	locks, err = repos.Locks.Check(ctx, lock.ScopePath, lock.ProjectRoot)
	if err != nil {
		t.Fatalf("Check after release: %v", err)
	}
	if len(locks) != 0 {
		t.Errorf("expected 0 locks after release, got %d", len(locks))
	}
}

func TestLockReleaseByRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run
	task := &domain.Task{ID: uuid.New(), Title: "Lock Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	// Create multiple locks for the same run
	for i := 0; i < 3; i++ {
		lock := &domain.ScopeLock{
			RunID:       run.ID,
			ScopePath:   fmt.Sprintf("/scope/%d", i),
			ProjectRoot: "/project",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}
		if err := repos.Locks.Acquire(ctx, lock); err != nil {
			t.Fatalf("Acquire lock %d: %v", i, err)
		}
	}

	// Release all by run
	if err := repos.Locks.ReleaseByRun(ctx, run.ID); err != nil {
		t.Fatalf("ReleaseByRun: %v", err)
	}

	// Check all scopes are unlocked
	for i := 0; i < 3; i++ {
		locks, err := repos.Locks.Check(ctx, fmt.Sprintf("/scope/%d", i), "/project")
		if err != nil {
			t.Fatalf("Check scope %d: %v", i, err)
		}
		if len(locks) != 0 {
			t.Errorf("expected 0 locks for scope %d, got %d", i, len(locks))
		}
	}
}

func TestLockCleanupExpired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run
	task := &domain.Task{ID: uuid.New(), Title: "Lock Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	// Create an expired lock
	expiredLock := &domain.ScopeLock{
		RunID:       run.ID,
		ScopePath:   "/expired",
		ProjectRoot: "/project",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Already expired
	}
	if err := repos.Locks.Acquire(ctx, expiredLock); err != nil {
		t.Fatalf("Acquire expired lock: %v", err)
	}

	// Create a valid lock
	validLock := &domain.ScopeLock{
		RunID:       run.ID,
		ScopePath:   "/valid",
		ProjectRoot: "/project",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	if err := repos.Locks.Acquire(ctx, validLock); err != nil {
		t.Fatalf("Acquire valid lock: %v", err)
	}

	// Cleanup expired
	count, err := repos.Locks.CleanupExpired(ctx)
	if err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 cleaned up, got %d", count)
	}

	// Valid lock should still exist
	locks, err := repos.Locks.Check(ctx, "/valid", "/project")
	if err != nil {
		t.Fatalf("Check valid: %v", err)
	}
	if len(locks) != 1 {
		t.Errorf("expected valid lock to remain, got %d", len(locks))
	}

	// Expired lock should be gone (Check filters by expires_at > now)
	locks, err = repos.Locks.Check(ctx, "/expired", "/project")
	if err != nil {
		t.Fatalf("Check expired: %v", err)
	}
	if len(locks) != 0 {
		t.Errorf("expected expired lock to be gone, got %d", len(locks))
	}
}

// ============================================================================
// JSONB Type Tests
// ============================================================================

func TestJSONBTypes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Test complex task with context attachments
	attachment1 := domain.ContextAttachment{
		Type:    "file",
		Path:    "/path/to/file.txt",
		Content: "file content",
	}
	attachment2 := domain.ContextAttachment{
		Type:    "url",
		Path:    "https://example.com",
		Content: "",
	}

	task := &domain.Task{
		ID:                 uuid.New(),
		Title:              "JSONB Test Task",
		ScopePath:          "/test",
		Status:             domain.TaskStatusQueued,
		PhasePromptIDs:     []uuid.UUID{uuid.New(), uuid.New()},
		ContextAttachments: []domain.ContextAttachment{attachment1, attachment2},
	}

	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task with attachments: %v", err)
	}

	got, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}

	if len(got.PhasePromptIDs) != 2 {
		t.Errorf("expected 2 phase prompt IDs, got %d", len(got.PhasePromptIDs))
	}

	if len(got.ContextAttachments) != 2 {
		t.Errorf("expected 2 context attachments, got %d", len(got.ContextAttachments))
	}

	if got.ContextAttachments[0].Type != "file" {
		t.Errorf("expected attachment type 'file', got %q", got.ContextAttachments[0].Type)
	}
}

func TestRunWithComplexFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task
	task := &domain.Task{ID: uuid.New(), Title: "Complex Run Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// Create profile
	profile := &domain.AgentProfile{ID: uuid.New(), Name: "complex-profile", ProfileKey: "complex-profile", RoleRef: "code.default"}
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile: %v", err)
	}

	sandboxID := uuid.New()
	startedAt := time.Now()
	exitCode := 0

	run := &domain.Run{
		ID:              uuid.New(),
		TaskID:          task.ID,
		AgentProfileID:  &profile.ID,
		SandboxID:       &sandboxID,
		Tag:             "complex-run",
		RunMode:         domain.RunModeSandboxed,
		Status:          domain.RunStatusComplete,
		Phase:           domain.RunPhaseCompleted,
		StartedAt:       &startedAt,
		ProgressPercent: 100,
		ApprovalState:   domain.ApprovalStateApproved,
		ExitCode:        &exitCode,
		Summary: &domain.RunSummary{
			Description:   "Completed successfully",
			FilesModified: []string{"file1.go", "file2.go", "file3.go"},
			FilesCreated:  []string{"new_file.go"},
			TokensUsed:    1000,
			TurnsUsed:     10,
			CostEstimate:  0.05,
		},
		Result: &domain.RunResult{
			FinalOutput: "Completed successfully",
			Selection: domain.FinalOutputSelection{
				Status:              domain.FinalOutputSelectionSelected,
				SelectedCandidateID: "candidate-1",
				Rule:                "unique_terminal_main_assistant",
				AlgorithmVersion:    domain.FinalOutputResolverVersion,
			},
			Candidates: []domain.FinalOutputCandidate{{ID: "candidate-1", Content: "Completed successfully", Terminal: true, EvidenceTier: 3}},
			Success:    true,
			Structured: &domain.StructuredResult{
				Status: domain.StructuredResultSuccess, SpecKind: domain.ResultSpecKindClassification,
				SchemaDigest: "sha256:result-schema", Value: json.RawMessage(`"complete"`),
				Method: "whole_document", SourceCandidateID: "candidate-1",
			},
		},
		ResolvedConfig: &domain.RunConfig{
			MaxTurns:     100,
			AllowedTools: []string{"read", "write", "bash"},
			DeniedTools:  []string{},
			ResultSpec: &domain.ResultSpec{
				Version: "result-spec/v1", Kind: domain.ResultSpecKindClassification,
				Schema: json.RawMessage(`{"enum":["complete"],"type":"string"}`), SchemaDigest: "sha256:result-schema",
				ExtractionMode: domain.StructuredExtractionDeterministic,
			},
			PolicySnapshot: &domain.ExecutionPolicySnapshot{
				CatalogDigest: "sha256:test-revision",
				RoleRef:       "code.smart",
				Candidates: []domain.ExecutionCandidate{{
					RunnerType:    domain.RunnerTypeCodex,
					SelectionType: domain.ModelSelectionTypeModel,
					Model:         "gpt-test",
				}},
				SelectedIndex: 0,
				SelectedCandidate: domain.ExecutionCandidate{
					RunnerType:    domain.RunnerTypeCodex,
					SelectionType: domain.ModelSelectionTypeModel,
					Model:         "gpt-test",
				},
				Explanation: domain.PolicyResolutionExplanation{
					Source:  "named_policy",
					Summary: "repository round trip",
				},
			},
		},
		DiffPath:       "/path/to/diff",
		LogPath:        "/path/to/log",
		ChangedFiles:   3,
		TotalSizeBytes: 1024,
	}

	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get run: %v", err)
	}

	// Verify nullable UUID fields
	if got.AgentProfileID == nil || *got.AgentProfileID != profile.ID {
		t.Errorf("expected agent profile ID %s, got %v", profile.ID, got.AgentProfileID)
	}
	if got.SandboxID == nil || *got.SandboxID != sandboxID {
		t.Errorf("expected sandbox ID %s, got %v", sandboxID, got.SandboxID)
	}

	// Verify nullable time fields
	if got.StartedAt == nil {
		t.Error("expected started_at to be set")
	}

	// Verify exit code
	if got.ExitCode == nil || *got.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %v", got.ExitCode)
	}

	// Verify summary JSONB
	if got.Summary == nil {
		t.Fatal("expected summary to be set")
	}
	if len(got.Summary.FilesModified) != 3 {
		t.Errorf("expected 3 files modified, got %d", len(got.Summary.FilesModified))
	}
	if got.Summary.TokensUsed != 1000 {
		t.Errorf("expected 1000 tokens used, got %d", got.Summary.TokensUsed)
	}
	if got.Result == nil || got.Result.Selection.Status != domain.FinalOutputSelectionSelected || got.Result.FinalOutput != "Completed successfully" {
		t.Fatalf("run result round trip = %#v", got.Result)
	}
	if got.Result.Structured == nil || got.Result.Structured.Status != domain.StructuredResultSuccess || string(got.Result.Structured.Value) != `"complete"` {
		t.Fatalf("structured result round trip = %#v", got.Result.Structured)
	}

	// Verify resolved config JSONB
	if got.ResolvedConfig == nil {
		t.Fatal("expected resolved config to be set")
	}
	if got.ResolvedConfig.MaxTurns != 100 {
		t.Errorf("expected max turns 100, got %d", got.ResolvedConfig.MaxTurns)
	}
	if len(got.ResolvedConfig.AllowedTools) != 3 {
		t.Errorf("expected 3 allowed tools, got %d", len(got.ResolvedConfig.AllowedTools))
	}
	if got.ResolvedConfig.PolicySnapshot == nil {
		t.Fatal("expected policy snapshot to be persisted")
	}
	if got.ResolvedConfig.ResultSpec == nil || got.ResolvedConfig.ResultSpec.SchemaDigest != "sha256:result-schema" {
		t.Fatalf("result spec round trip = %+v", got.ResolvedConfig.ResultSpec)
	}
	if got.ResolvedConfig.PolicySnapshot.CatalogDigest != "sha256:test-revision" ||
		got.ResolvedConfig.PolicySnapshot.SelectedCandidate.Model != "gpt-test" {
		t.Fatalf("policy snapshot round trip = %+v", got.ResolvedConfig.PolicySnapshot)
	}
}

// ============================================================================
// Stats Repository Time Scanning Tests
// ============================================================================

func TestStatsGetTimeSeries_DailyBucketParsesTimestamp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Stats Task",
		ScopePath: "/stats",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "stats-time-series",
		Status:        domain.RunStatusComplete,
		Phase:         domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStateApproved,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-sonnet-4-20250514",
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	now := time.Now()
	filter := repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now.Add(24 * time.Hour),
		},
	}

	buckets, err := repos.Stats.GetTimeSeries(ctx, filter, 24*time.Hour)
	if err != nil {
		t.Fatalf("GetTimeSeries: %v", err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected at least one time-series bucket")
	}
	if buckets[0].Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp in first bucket")
	}
}

func TestStatsUsageQueries_ParseCreatedAtTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Usage Stats Task",
		ScopePath: "/stats/usage",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "stats-usage",
		Status:        domain.RunStatusComplete,
		Phase:         domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStateApproved,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-sonnet-4-20250514",
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	toolCall := domain.NewToolCallEvent(run.ID, "read", "call-1", map[string]interface{}{"path": "README.md"})
	toolResult := domain.NewToolResultEvent(run.ID, "read", "call-1", "ok", nil)
	if err := repos.Events.Append(ctx, run.ID, toolCall, toolResult); err != nil {
		t.Fatalf("Append events: %v", err)
	}

	now := time.Now()
	filter := repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now.Add(24 * time.Hour),
		},
	}

	modelRuns, err := repos.Stats.GetModelRunUsage(ctx, filter, "claude-sonnet-4-20250514", 10)
	if err != nil {
		t.Fatalf("GetModelRunUsage: %v", err)
	}
	if len(modelRuns) == 0 {
		t.Fatal("expected model usage rows")
	}
	if modelRuns[0].CreatedAt.IsZero() {
		t.Fatal("expected model usage created_at to be parsed")
	}

	toolRuns, err := repos.Stats.GetToolRunUsage(ctx, filter, "read", 10)
	if err != nil {
		t.Fatalf("GetToolRunUsage: %v", err)
	}
	if len(toolRuns) == 0 {
		t.Fatal("expected tool usage rows")
	}
	if toolRuns[0].CreatedAt.IsZero() {
		t.Fatal("expected tool usage created_at to be parsed")
	}
}

func TestStatsGetErrorPatterns_ParsesLastSeenTimestamp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Error Stats Task",
		ScopePath: "/stats/errors",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "stats-errors",
		Status:        domain.RunStatusFailed,
		Phase:         domain.RunPhaseCompleted,
		ApprovalState: domain.ApprovalStateNone,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	errEvent := domain.NewErrorEvent(run.ID, "TOOL_FAILED", "tool execution failed", true)
	if err := repos.Events.Append(ctx, run.ID, errEvent); err != nil {
		t.Fatalf("Append error event: %v", err)
	}

	now := time.Now()
	filter := repository.StatsFilter{
		Window: repository.StatsTimeWindow{
			Start: now.Add(-24 * time.Hour),
			End:   now.Add(24 * time.Hour),
		},
	}

	patterns, err := repos.Stats.GetErrorPatterns(ctx, filter, 10)
	if err != nil {
		t.Fatalf("GetErrorPatterns: %v", err)
	}
	if len(patterns) == 0 {
		t.Fatal("expected error patterns rows")
	}
	if patterns[0].LastSeen.IsZero() {
		t.Fatal("expected non-zero last_seen timestamp")
	}
}

func TestIdempotencyReserveReclaimsExpiredAndFailed(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// A live pending reservation blocks a second reserve.
	live := "live-key"
	if _, err := repos.Idempotency.Reserve(ctx, live, time.Hour); err != nil {
		t.Fatalf("Reserve live: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, live, time.Hour); err == nil {
		t.Fatal("second reserve on a live key must fail")
	}

	// A completed reservation stays held even after a reserve attempt.
	done := "done-key"
	if _, err := repos.Idempotency.Reserve(ctx, done, time.Hour); err != nil {
		t.Fatalf("Reserve done: %v", err)
	}
	if err := repos.Idempotency.Complete(ctx, done, uuid.New(), "run", nil); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, done, time.Hour); err == nil {
		t.Fatal("reserve on a completed key must fail")
	}

	// An expired pending reservation (crash between Reserve and Complete) is
	// reclaimed by the next reserve instead of wedging the key forever.
	expired := "expired-key"
	if _, err := repos.Idempotency.Reserve(ctx, expired, -time.Minute); err != nil {
		t.Fatalf("Reserve expired: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, expired, time.Hour); err != nil {
		t.Fatalf("reserve must reclaim an expired reservation: %v", err)
	}
	record, err := repos.Idempotency.Check(ctx, expired)
	if err != nil || record == nil || record.Status != domain.IdempotencyStatusPending {
		t.Fatalf("reclaimed key should be live pending, got %+v err=%v", record, err)
	}

	// A failed reservation is reclaimed (the documented allow-retry path).
	failed := "failed-key"
	if _, err := repos.Idempotency.Reserve(ctx, failed, time.Hour); err != nil {
		t.Fatalf("Reserve failed-key: %v", err)
	}
	if err := repos.Idempotency.Fail(ctx, failed); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if _, err := repos.Idempotency.Reserve(ctx, failed, time.Hour); err != nil {
		t.Fatalf("reserve must reclaim a failed reservation: %v", err)
	}
}
