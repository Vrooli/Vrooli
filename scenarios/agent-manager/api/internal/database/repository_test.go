package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // SQLite driver for tests
)

// testBackend returns the test database backend from environment.
// Defaults to "sqlite" for fast, isolated tests.
func testBackend() string {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("AM_TEST_BACKEND")))
	if backend == "" {
		backend = strings.ToLower(strings.TrimSpace(os.Getenv("AM_DB_BACKEND")))
	}
	if backend == "" {
		backend = "sqlite"
	}
	return backend
}

// setupTestDB creates a fresh SQLite database for testing.
// Returns the DB wrapper and a cleanup function.
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	backend := testBackend()
	if backend != "sqlite" {
		t.Skipf("unsupported test backend %q (set AM_TEST_BACKEND=sqlite)", backend)
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agent-manager-test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
		dbPath,
	)

	sqlDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}

	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.PanicLevel) // Suppress logs during tests

	wrapped := &DB{
		DB:      sqlDB,
		log:     log,
		dialect: DialectSQLite,
	}
	if err := wrapped.initSchema(); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("init schema: %v", err)
	}

	return wrapped, func() {
		_ = sqlDB.Close()
	}
}

// ============================================================================
// Profile Repository Tests
// ============================================================================

func TestProfileCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	profile := &domain.AgentProfile{
		ID:                   uuid.New(),
		Name:                 "test-profile",
		Description:          "A test profile",
		RunnerType:           domain.RunnerTypeClaudeCode,
		Model:                "claude-sonnet-4-20250514",
		MaxTurns:             100,
		Timeout:              30 * time.Minute,
		AllowedTools:         []string{"read", "write"},
		DeniedTools:          []string{"bash"},
		SkipPermissionPrompt: true,
		RequiresSandbox:      true,
		RequiresApproval:     false,
		AllowedPaths:         []string{"/home/user"},
		DeniedPaths:          []string{"/etc"},
		CreatedBy:            "test-user",
	}

	// Create
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get by ID
	got, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != profile.Name {
		t.Errorf("expected name %q, got %q", profile.Name, got.Name)
	}
	if got.RunnerType != profile.RunnerType {
		t.Errorf("expected runner type %q, got %q", profile.RunnerType, got.RunnerType)
	}
	if len(got.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(got.AllowedTools))
	}

	// Get by name
	byName, err := repos.Profiles.GetByName(ctx, profile.Name)
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if byName == nil || byName.ID != profile.ID {
		t.Fatal("GetByName returned wrong profile")
	}

	// List
	profiles, err := repos.Profiles.List(ctx, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}

	// Update
	profile.Name = "renamed-profile"
	profile.Description = "Updated description"
	if err := repos.Profiles.Update(ctx, profile); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Name != "renamed-profile" {
		t.Errorf("expected updated name, got %q", got.Name)
	}

	// Delete
	if err := repos.Profiles.Delete(ctx, profile.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestProfileListPagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create 5 profiles
	for i := 0; i < 5; i++ {
		profile := &domain.AgentProfile{
			ID:          uuid.New(),
			Name:        fmt.Sprintf("profile-%d", i),
			Description: "Test",
			RunnerType:  domain.RunnerTypeClaudeCode,
		}
		if err := repos.Profiles.Create(ctx, profile); err != nil {
			t.Fatalf("Create profile %d: %v", i, err)
		}
		// Add delay to ensure distinct timestamps for ordering
		time.Sleep(10 * time.Millisecond)
	}

	// Test limit
	profiles, err := repos.Profiles.List(ctx, repository.ListFilter{Limit: 3})
	if err != nil {
		t.Fatalf("List with limit: %v", err)
	}
	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles with limit, got %d", len(profiles))
	}

	// Test offset
	profiles, err = repos.Profiles.List(ctx, repository.ListFilter{Limit: 10, Offset: 2})
	if err != nil {
		t.Fatalf("List with offset: %v", err)
	}
	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles with offset 2, got %d", len(profiles))
	}
}

// ============================================================================
// Task Repository Tests
// ============================================================================

func TestTaskCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "A task for testing",
		ScopePath:   "/home/user/project",
		ProjectRoot: "/home/user/project",
		Status:      domain.TaskStatusQueued,
		CreatedBy:   "test-user",
	}

	// Create
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repos.Tasks.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Title != task.Title {
		t.Errorf("expected title %q, got %q", task.Title, got.Title)
	}
	if got.Status != domain.TaskStatusQueued {
		t.Errorf("expected status queued, got %q", got.Status)
	}

	// List
	tasks, err := repos.Tasks.List(ctx, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	// ListByStatus
	task.Status = domain.TaskStatusRunning
	if err := repos.Tasks.Update(ctx, task); err != nil {
		t.Fatalf("Update status: %v", err)
	}

	runningTasks, err := repos.Tasks.ListByStatus(ctx, domain.TaskStatusRunning, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(runningTasks) != 1 {
		t.Errorf("expected 1 running task, got %d", len(runningTasks))
	}

	queuedTasks, err := repos.Tasks.ListByStatus(ctx, domain.TaskStatusQueued, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListByStatus queued: %v", err)
	}
	if len(queuedTasks) != 0 {
		t.Errorf("expected 0 queued tasks, got %d", len(queuedTasks))
	}

	// Update
	task.Title = "Updated Task Title"
	if err := repos.Tasks.Update(ctx, task); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Title != "Updated Task Title" {
		t.Errorf("expected updated title, got %q", got.Title)
	}

	// Delete
	if err := repos.Tasks.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Tasks.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

// ============================================================================
// Run Repository Tests
// ============================================================================

func TestRunCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a task first (runs reference tasks)
	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Parent Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	run := &domain.Run{
		ID:              uuid.New(),
		TaskID:          task.ID,
		Tag:             "test-run",
		RunMode:         domain.RunModeSandboxed,
		Status:          domain.RunStatusPending,
		Phase:           domain.RunPhaseQueued,
		ProgressPercent: 0,
		ApprovalState:   domain.ApprovalStateNone,
	}

	// Create
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Tag != run.Tag {
		t.Errorf("expected tag %q, got %q", run.Tag, got.Tag)
	}
	if got.Status != domain.RunStatusPending {
		t.Errorf("expected status pending, got %q", got.Status)
	}

	// List
	runs, err := repos.Runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 10}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	// ListByTask
	runsByTask, err := repos.Runs.ListByTask(ctx, task.ID, repository.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListByTask: %v", err)
	}
	if len(runsByTask) != 1 {
		t.Errorf("expected 1 run for task, got %d", len(runsByTask))
	}

	// CountByStatus
	count, err := repos.Runs.CountByStatus(ctx, domain.RunStatusPending)
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Update
	startedAt := time.Now()
	run.Status = domain.RunStatusRunning
	run.StartedAt = &startedAt
	run.ProgressPercent = 50
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("expected status running, got %q", got.Status)
	}
	if got.ProgressPercent != 50 {
		t.Errorf("expected progress 50, got %d", got.ProgressPercent)
	}

	// Delete
	if err := repos.Runs.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestRunListFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a task
	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Filter Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	// Create a profile
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "filter-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
	}
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile: %v", err)
	}

	// Create runs with different tags and statuses
	runs := []*domain.Run{
		{ID: uuid.New(), TaskID: task.ID, AgentProfileID: &profile.ID, Tag: "batch-1", Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-1"},
		{ID: uuid.New(), TaskID: task.ID, Tag: "batch-2", Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-2"},
		{ID: uuid.New(), TaskID: task.ID, Tag: "batch-1-sub", Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-3"},
	}
	for _, run := range runs {
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Filter by status
	completedRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		Status: func() *domain.RunStatus { s := domain.RunStatusComplete; return &s }(),
	})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if len(completedRuns) != 1 {
		t.Errorf("expected 1 completed run, got %d", len(completedRuns))
	}

	// Filter by tag prefix
	batchRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		TagPrefix: "batch-1",
	})
	if err != nil {
		t.Fatalf("List by tag prefix: %v", err)
	}
	if len(batchRuns) != 2 {
		t.Errorf("expected 2 runs with tag prefix 'batch-1', got %d", len(batchRuns))
	}

	// Filter by profile ID
	profileRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		AgentProfileID: &profile.ID,
	})
	if err != nil {
		t.Fatalf("List by profile: %v", err)
	}
	if len(profileRuns) != 1 {
		t.Errorf("expected 1 run with profile, got %d", len(profileRuns))
	}
}

// ============================================================================
// Event Repository Tests
// ============================================================================

func TestEventRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run first
	task := &domain.Task{ID: uuid.New(), Title: "Event Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	// Append events
	events := []*domain.RunEvent{
		{EventType: domain.EventTypeLog, Data: &domain.LogEventData{Message: "line 1", Level: "info"}},
		{EventType: domain.EventTypeLog, Data: &domain.LogEventData{Message: "line 2", Level: "info"}},
		{EventType: domain.EventTypeStatus, Data: &domain.StatusEventData{OldStatus: string(domain.RunStatusPending), NewStatus: string(domain.RunStatusRunning), Reason: "started"}},
	}
	if err := repos.Events.Append(ctx, run.ID, events...); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Get all events
	gotEvents, err := repos.Events.Get(ctx, run.ID, -1, 100)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(gotEvents) != 3 {
		t.Errorf("expected 3 events, got %d", len(gotEvents))
	}

	// Check sequence numbers
	for i, evt := range gotEvents {
		if evt.Sequence != int64(i) {
			t.Errorf("expected sequence %d, got %d", i, evt.Sequence)
		}
	}

	// Get events after sequence
	afterEvents, err := repos.Events.Get(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("Get after sequence: %v", err)
	}
	if len(afterEvents) != 2 {
		t.Errorf("expected 2 events after sequence 0, got %d", len(afterEvents))
	}

	// Get by type
	logEvents, err := repos.Events.GetByType(ctx, run.ID, []domain.RunEventType{domain.EventTypeLog}, 100)
	if err != nil {
		t.Fatalf("GetByType: %v", err)
	}
	if len(logEvents) != 2 {
		t.Errorf("expected 2 log events, got %d", len(logEvents))
	}

	// Count
	count, err := repos.Events.Count(ctx, run.ID)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	// Delete
	if err := repos.Events.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	count, err = repos.Events.Count(ctx, run.ID)
	if err != nil {
		t.Fatalf("Count after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 after delete, got %d", count)
	}
}

// ============================================================================
// Checkpoint Repository Tests
// ============================================================================

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
	profile := &domain.AgentProfile{ID: uuid.New(), Name: "complex-profile", RunnerType: domain.RunnerTypeClaudeCode}
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
		ResolvedConfig: &domain.RunConfig{
			MaxTurns:        100,
			AllowedTools:    []string{"read", "write", "bash"},
			DeniedTools:     []string{},
			RequiresSandbox: true,
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
}
