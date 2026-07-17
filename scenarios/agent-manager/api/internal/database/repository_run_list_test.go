package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// makeFullRun creates a domain.Run with all fields populated (including heavy ones).
// profileID should be a valid agent_profile_id from the database (or nil to skip).
func makeFullRun(taskID uuid.UUID, profileID *uuid.UUID) *domain.Run {
	sandboxID := uuid.New()
	checkpointID := uuid.New()
	now := time.Now()
	exitCode := 0
	return &domain.Run{
		ID:               uuid.New(),
		TaskID:           taskID,
		AgentProfileID:   profileID,
		Tag:              "test-full",
		SandboxID:        &sandboxID,
		RunMode:          domain.RunModeInPlace,
		Status:           domain.RunStatusComplete,
		StartedAt:        &now,
		EndedAt:          &now,
		Phase:            domain.RunPhaseExecuting,
		LastCheckpointID: &checkpointID,
		LastHeartbeat:    &now,
		ProgressPercent:  100,
		IdempotencyKey:   "idem-key-123",
		Summary: &domain.RunSummary{
			Description:   "Test summary description",
			FilesModified: []string{"a.go", "b.go"},
			TokensUsed:    5000,
			TurnsUsed:     10,
			CostEstimate:  0.42,
		},
		ErrorMsg:      "",
		ExitCode:      &exitCode,
		ApprovalState: domain.ApprovalStateNone,
		ApprovedBy:    "admin",
		ApprovedAt:    &now,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-sonnet-4-20250514",
		},
		DiffPath:       "/diffs/test.diff",
		LogPath:        "/logs/test.log",
		ChangedFiles:   2,
		TotalSizeBytes: 1024,
		SandboxConfig: &domain.SandboxConfig{
			NoLock: true,
		},
		SessionID:    "session-abc",
		SourceRunIDs: []uuid.UUID{uuid.New()},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// TestList_OmitsHeavyFields inserts a run with all fields set, then verifies
// that List() returns nil for the heavy fields that are pruned from list queries.
func TestList_OmitsHeavyFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, db.log)
	ctx := context.Background()

	// Create a task first (for the foreign key)
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "heavy-fields-test",
		Description: "A task for heavy field test",
		ScopePath:   "src/",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	run := makeFullRun(task.ID, nil)
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	runs, err := repos.Runs.List(ctx, repository.RunListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	got := runs[0]
	if got.Summary != nil {
		t.Errorf("List() should omit Summary, got %+v", got.Summary)
	}
	if got.ResolvedConfig != nil {
		t.Errorf("List() should omit ResolvedConfig, got %+v", got.ResolvedConfig)
	}
	if got.SandboxConfig != nil {
		t.Errorf("List() should omit SandboxConfig, got %+v", got.SandboxConfig)
	}
	if got.SandboxID != nil {
		t.Errorf("List() should omit SandboxID, got %v", got.SandboxID)
	}
	if got.DiffPath != "" {
		t.Errorf("List() should omit DiffPath, got %q", got.DiffPath)
	}
	if got.LogPath != "" {
		t.Errorf("List() should omit LogPath, got %q", got.LogPath)
	}
	if got.IdempotencyKey != "" {
		t.Errorf("List() should omit IdempotencyKey, got %q", got.IdempotencyKey)
	}
	if got.LastCheckpointID != nil {
		t.Errorf("List() should omit LastCheckpointID, got %v", got.LastCheckpointID)
	}
	// LastHeartbeat is intentionally included in List() — the reconciler
	// depends on it to detect stale runs.
	if got.ApprovedBy != "" {
		t.Errorf("List() should omit ApprovedBy, got %q", got.ApprovedBy)
	}
	if got.ApprovedAt != nil {
		t.Errorf("List() should omit ApprovedAt, got %v", got.ApprovedAt)
	}
}

// TestList_PopulatesLightFields verifies all non-pruned columns are correctly populated.
func TestList_PopulatesLightFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, db.log)
	ctx := context.Background()

	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "light-fields-test",
		Description: "A task for light field test",
		ScopePath:   "src/",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	run := makeFullRun(task.ID, nil)
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	runs, err := repos.Runs.List(ctx, repository.RunListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	got := runs[0]
	if got.ID != run.ID {
		t.Errorf("ID: want %s, got %s", run.ID, got.ID)
	}
	if got.TaskID != run.TaskID {
		t.Errorf("TaskID: want %s, got %s", run.TaskID, got.TaskID)
	}
	if (got.AgentProfileID == nil) != (run.AgentProfileID == nil) {
		t.Errorf("AgentProfileID nil mismatch: want %v, got %v", run.AgentProfileID, got.AgentProfileID)
	} else if got.AgentProfileID != nil && *got.AgentProfileID != *run.AgentProfileID {
		t.Errorf("AgentProfileID: want %s, got %s", *run.AgentProfileID, *got.AgentProfileID)
	}
	if got.Tag != run.Tag {
		t.Errorf("Tag: want %q, got %q", run.Tag, got.Tag)
	}
	if got.RunMode != run.RunMode {
		t.Errorf("RunMode: want %s, got %s", run.RunMode, got.RunMode)
	}
	if got.Status != run.Status {
		t.Errorf("Status: want %s, got %s", run.Status, got.Status)
	}
	if got.Phase != run.Phase {
		t.Errorf("Phase: want %s, got %s", run.Phase, got.Phase)
	}
	if got.ProgressPercent != run.ProgressPercent {
		t.Errorf("ProgressPercent: want %d, got %d", run.ProgressPercent, got.ProgressPercent)
	}
	if got.ApprovalState != run.ApprovalState {
		t.Errorf("ApprovalState: want %s, got %s", run.ApprovalState, got.ApprovalState)
	}
	if got.ChangedFiles != run.ChangedFiles {
		t.Errorf("ChangedFiles: want %d, got %d", run.ChangedFiles, got.ChangedFiles)
	}
	if got.TotalSizeBytes != run.TotalSizeBytes {
		t.Errorf("TotalSizeBytes: want %d, got %d", run.TotalSizeBytes, got.TotalSizeBytes)
	}
	if got.SessionID != run.SessionID {
		t.Errorf("SessionID: want %q, got %q", run.SessionID, got.SessionID)
	}
	if got.ExitCode == nil || *got.ExitCode != *run.ExitCode {
		t.Errorf("ExitCode mismatch: want %v, got %v", run.ExitCode, got.ExitCode)
	}
}

// TestList_PromptPreviewStillWorks verifies that the prompt preview JOIN is
// still functional with the pruned column set.
func TestList_PromptPreviewStillWorks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, db.log)
	ctx := context.Background()

	taskDesc := "This is a detailed task description for preview testing"
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "preview-test",
		Description: taskDesc,
		ScopePath:   "src/",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	run := &domain.Run{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Tag:       "preview-test",
		RunMode:   domain.RunModeInPlace,
		Status:    domain.RunStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	runs, err := repos.Runs.List(ctx, repository.RunListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	if runs[0].PromptPreview != taskDesc {
		t.Errorf("PromptPreview: want %q, got %q", taskDesc, runs[0].PromptPreview)
	}
}

// TestGet_StillReturnsFullFields is a regression guard ensuring Get() still
// returns all heavy fields that List() now omits.
func TestGet_StillReturnsFullFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, db.log)
	ctx := context.Background()

	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "get-full-test",
		Description: "A task for get full test",
		ScopePath:   "src/",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	run := makeFullRun(task.ID, nil)
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}

	if got.Summary == nil {
		t.Error("Get() should return Summary")
	}
	if got.ResolvedConfig == nil {
		t.Error("Get() should return ResolvedConfig")
	}
	if got.SandboxConfig == nil {
		t.Error("Get() should return SandboxConfig")
	}
	if got.SandboxID == nil {
		t.Error("Get() should return SandboxID")
	}
	if got.DiffPath != run.DiffPath {
		t.Errorf("Get() DiffPath: want %q, got %q", run.DiffPath, got.DiffPath)
	}
	if got.LogPath != run.LogPath {
		t.Errorf("Get() LogPath: want %q, got %q", run.LogPath, got.LogPath)
	}
	if got.IdempotencyKey != run.IdempotencyKey {
		t.Errorf("Get() IdempotencyKey: want %q, got %q", run.IdempotencyKey, got.IdempotencyKey)
	}
	if got.ApprovedBy != run.ApprovedBy {
		t.Errorf("Get() ApprovedBy: want %q, got %q", run.ApprovedBy, got.ApprovedBy)
	}
}
