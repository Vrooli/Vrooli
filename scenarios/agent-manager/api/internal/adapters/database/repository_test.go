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

// setupTestDB creates a fresh SQLite database for testing.
// Returns the DB wrapper and a cleanup function.
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

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
		DB:  sqlDB,
		log: log,
	}
	if err := wrapped.initSchema(); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("init schema: %v", err)
	}

	return wrapped, func() {
		_ = sqlDB.Close()
	}
}

func TestInitSchemaCreatesRoleOnlyProfileTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	var roleRefColumns int
	if err := db.Get(&roleRefColumns, "SELECT COUNT(*) FROM pragma_table_info('agent_profiles') WHERE name = 'role_ref'"); err != nil {
		t.Fatalf("inspect role_ref: %v", err)
	}
	if roleRefColumns != 1 {
		t.Fatalf("role_ref columns = %d, want 1", roleRefColumns)
	}

	for _, legacyColumn := range []string{"runner_type", "model", "policy_ref", "model_preset", "fallback_runner_types"} {
		var count int
		if err := db.Get(&count, "SELECT COUNT(*) FROM pragma_table_info('agent_profiles') WHERE name = ?", legacyColumn); err != nil {
			t.Fatalf("inspect %s: %v", legacyColumn, err)
		}
		if count != 0 {
			t.Fatalf("legacy column %s unexpectedly present", legacyColumn)
		}
	}
}

func TestDataDirPrefersCanonicalStorageOverLegacyFallbackEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SQLITE_DATABASE_PATH", filepath.Join(home, "legacy-sqlite-root"))
	t.Setenv("VROOLI_DATA", filepath.Join(home, "legacy-vrooli-data"))
	t.Setenv("AM_SQLITE_PATH", "")
	// Exercise the HOME-derived default, not a host-level canonical override
	// inherited by the test process (for example from web-console).
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("VROOLI_DATA_ROOT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("VROOLI_VARIANT", "")

	got := DataDir()
	want := filepath.Join(home, ".vrooli", "data", "vrooli", "agent-manager")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestSQLiteDSNUsesCanonicalPathWithoutLegacyMigration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AM_SQLITE_PATH", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SQLITE_DATABASE_PATH", filepath.Join(home, "legacy-sqlite-root"))
	t.Setenv("VROOLI_DATA", filepath.Join(home, "legacy-vrooli-data"))
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("VROOLI_DATA_ROOT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("VROOLI_VARIANT", "")

	dsn, err := sqliteDSN(nil)
	if err != nil {
		t.Fatalf("sqliteDSN() error = %v", err)
	}
	wantPath := filepath.Join(home, ".vrooli", "data", "vrooli", "agent-manager", "agent-manager.db")
	if !strings.Contains(dsn, wantPath) {
		t.Fatalf("sqliteDSN() = %q, want path containing %q", dsn, wantPath)
	}
	if _, err := os.Stat(filepath.Dir(wantPath)); err != nil {
		t.Fatalf("expected canonical sqlite dir at %s: %v", filepath.Dir(wantPath), err)
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
		ID:          uuid.New(),
		Name:        "test-profile",
		ProfileKey:  "test-profile",
		Description: "A test profile",

		MaxTurns:              100,
		Timeout:               30 * time.Minute,
		Effort:                domain.EffortHigh,
		AllowedTools:          []string{"read", "write"},
		DeniedTools:           []string{"bash"},
		ToolRestrictionPolicy: domain.ToolRestrictionPolicyAdvisory,
		SkipPermissionPrompt:  true,
		AllowedPaths:          []string{"/home/user"},
		DeniedPaths:           []string{"/etc"},
		CreatedBy:             "test-user", RoleRef:

		// Create
		"code.default",
	}

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
	if got.RoleRef != profile.RoleRef {
		t.Errorf("expected role ref %q, got %q", profile.RoleRef, got.RoleRef)
	}
	if got.Effort != profile.Effort {
		t.Errorf("expected effort %q, got %q", profile.Effort, got.Effort)
	}
	if len(got.AllowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(got.AllowedTools))
	}
	if got.ToolRestrictionPolicy != domain.ToolRestrictionPolicyAdvisory {
		t.Errorf("tool restriction policy = %q", got.ToolRestrictionPolicy)
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
			ProfileKey:  fmt.Sprintf("profile-%d", i),
			Description: "Test", RoleRef: "code.default",
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
// Profile Feature Flags & Extra Flags Persistence Tests
// ============================================================================

func TestProfileWithFeatureFlags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a profile with features enabled
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "features-profile",
		ProfileKey: "features-profile",

		Features: domain.FeatureFlags{EnableBrowser: true}, RoleRef: "code.default",
	}

	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if !got.Features.EnableBrowser {
		t.Error("expected Features.EnableBrowser to be true")
	}

	// Update: disable feature
	profile.Features.EnableBrowser = false
	if err := repos.Profiles.Update(ctx, profile); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Features.EnableBrowser {
		t.Error("expected Features.EnableBrowser to be false after update")
	}
}

func TestProfileWithZeroFeatureFlags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a profile with zero (default) features
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "zero-features-profile",
		ProfileKey: "zero-features-profile",

		Features: domain.FeatureFlags{}, RoleRef: // Zero value
		"code.default",
	}

	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Features.EnableBrowser {
		t.Error("expected Features.EnableBrowser to be false for zero-value profile")
	}
}

func TestProfileWithExtraFlags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a profile with extra flags
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "extra-flags-profile",
		ProfileKey: "extra-flags-profile",

		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--verbose", "--allowedTools=Read,Write"},
			domain.RunnerTypeCodex:      []string{"--verbose"},
		}, RoleRef: "code.default",
	}

	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}

	// Verify extra flags round-trip
	if len(got.ExtraFlags) != 2 {
		t.Fatalf("expected 2 runner types in ExtraFlags, got %d", len(got.ExtraFlags))
	}

	ccFlags, ok := got.ExtraFlags[domain.RunnerTypeClaudeCode]
	if !ok {
		t.Fatal("missing claude-code in ExtraFlags")
	}
	if len(ccFlags) != 2 {
		t.Errorf("expected 2 claude-code flags, got %d", len(ccFlags))
	}
	if len(ccFlags) >= 1 && ccFlags[0] != "--verbose" {
		t.Errorf("expected first flag '--verbose', got %q", ccFlags[0])
	}
	if len(ccFlags) >= 2 && ccFlags[1] != "--allowedTools=Read,Write" {
		t.Errorf("expected second flag '--allowedTools=Read,Write', got %q", ccFlags[1])
	}

	codexFlags, ok := got.ExtraFlags[domain.RunnerTypeCodex]
	if !ok {
		t.Fatal("missing codex in ExtraFlags")
	}
	if len(codexFlags) != 1 {
		t.Errorf("expected 1 codex flag, got %d", len(codexFlags))
	}
}

func TestProfileWithNilExtraFlags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a profile with nil extra flags
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "nil-extras-profile",
		ProfileKey: "nil-extras-profile",

		ExtraFlags: nil, RoleRef: "code.default",
	}

	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}

	// Nil or empty should round-trip as nil/empty
	if len(got.ExtraFlags) != 0 {
		t.Errorf("expected nil/empty ExtraFlags, got %v", got.ExtraFlags)
	}
}

func TestProfileWithFeaturesAndExtraFlags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create a profile with both features and extra flags
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "full-flags-profile",
		ProfileKey: "full-flags-profile",

		Features: domain.FeatureFlags{EnableBrowser: true},
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--verbose"},
		}, RoleRef: "code.default",
	}

	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Profiles.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}

	if !got.Features.EnableBrowser {
		t.Error("expected Features.EnableBrowser to be true")
	}
	if len(got.ExtraFlags) != 1 {
		t.Errorf("expected 1 runner type in ExtraFlags, got %d", len(got.ExtraFlags))
	}
	if flags, ok := got.ExtraFlags[domain.RunnerTypeClaudeCode]; !ok || len(flags) != 1 || flags[0] != "--verbose" {
		t.Errorf("expected ExtraFlags[claude-code] = [--verbose], got %v", got.ExtraFlags)
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

// TestRunCustomEnvRoundTrip proves Run.CustomEnv persists through the JSON
// column so the continue/wake path can re-inject it. Before Phase 0 there was
// no column at all, so a continued turn could never recover custom env.
func TestRunCustomEnvRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

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
		ID:            uuid.New(),
		TaskID:        task.ID,
		RunMode:       domain.RunModeSandboxed,
		Status:        domain.RunStatusPending,
		Phase:         domain.RunPhaseQueued,
		ApprovalState: domain.ApprovalStateNone,
		CustomEnv: map[string]string{
			"VROOLI_SHADOW_SCENARIOS":         "agent-manager",
			"VROOLI_SWARM_MANAGER_SESSION_ID": "sess-42",
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.CustomEnv) != 2 {
		t.Fatalf("expected 2 custom env entries, got %v", got.CustomEnv)
	}
	if got.CustomEnv["VROOLI_SHADOW_SCENARIOS"] != "agent-manager" ||
		got.CustomEnv["VROOLI_SWARM_MANAGER_SESSION_ID"] != "sess-42" {
		t.Errorf("custom env did not round-trip: %v", got.CustomEnv)
	}

	// Empty/nil custom env must decode as nil (not an empty map) so existing
	// rows and env-free runs behave identically.
	run.CustomEnv = nil
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.CustomEnv != nil {
		t.Errorf("expected nil custom env after clearing, got %v", got.CustomEnv)
	}
}

func TestRunAwaitHandleRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Await Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	deadline := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	registered := time.Now().UTC().Truncate(time.Second)
	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		RunMode:       domain.RunModeSandboxed,
		Status:        domain.RunStatusParked,
		Phase:         domain.RunPhaseExecuting,
		ApprovalState: domain.ApprovalStateNone,
		AwaitHandle: &domain.AwaitHandle{
			Producer:     "test-genie",
			Key:          "run-xyz",
			Deadline:     &deadline,
			RegisteredAt: registered,
		},
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != domain.RunStatusParked {
		t.Errorf("status did not round-trip: %s", got.Status)
	}
	if got.AwaitHandle == nil {
		t.Fatal("await handle did not round-trip (nil)")
	}
	if got.AwaitHandle.Producer != "test-genie" || got.AwaitHandle.Key != "run-xyz" {
		t.Errorf("await handle producer/key did not round-trip: %+v", got.AwaitHandle)
	}
	if got.AwaitHandle.Deadline == nil || !got.AwaitHandle.Deadline.Equal(deadline) {
		t.Errorf("await handle deadline did not round-trip: %v want %v", got.AwaitHandle.Deadline, deadline)
	}

	// Clearing the handle (wake/cancel) must persist as NULL → nil, exactly like
	// a non-parked run, so existing rows and woken runs behave identically.
	run.AwaitHandle = nil
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.AwaitHandle != nil {
		t.Errorf("expected nil await handle after clearing, got %+v", got.AwaitHandle)
	}
}

// TestTouchHeartbeat_StatusGuarded verifies the status-guarded heartbeat update:
// it bumps last_heartbeat for running/starting runs, but is a no-op (no clobber)
// for a parked or terminal run — so a heartbeat racing a park/stop transition
// can never resurrect the run.
func TestTouchHeartbeat_StatusGuarded(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "HB Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	old := time.Now().Add(-time.Hour)
	mk := func(status domain.RunStatus) *domain.Run {
		run := &domain.Run{
			ID:            uuid.New(),
			TaskID:        task.ID,
			RunMode:       domain.RunModeInPlace,
			Status:        status,
			Phase:         domain.RunPhaseExecuting,
			ApprovalState: domain.ApprovalStateNone,
			LastHeartbeat: &old,
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run (%s): %v", status, err)
		}
		return run
	}

	// running → updated, last_heartbeat advances.
	running := mk(domain.RunStatusRunning)
	now := time.Now()
	updated, err := repos.Runs.TouchHeartbeat(ctx, running.ID, now)
	if err != nil {
		t.Fatalf("TouchHeartbeat(running): %v", err)
	}
	if !updated {
		t.Fatal("running run heartbeat should have been updated")
	}
	got, _ := repos.Runs.Get(ctx, running.ID)
	if got.LastHeartbeat == nil || got.LastHeartbeat.Before(now.Add(-2*time.Second)) {
		t.Errorf("running heartbeat not advanced: %v", got.LastHeartbeat)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("running status changed unexpectedly: %s", got.Status)
	}

	// starting → updated.
	starting := mk(domain.RunStatusStarting)
	if updated, err := repos.Runs.TouchHeartbeat(ctx, starting.ID, time.Now()); err != nil || !updated {
		t.Fatalf("TouchHeartbeat(starting): updated=%v err=%v (want true,nil)", updated, err)
	}

	// parked → NO-OP (the critical anti-clobber guarantee). Status stays parked,
	// heartbeat unchanged.
	parked := mk(domain.RunStatusParked)
	updated, err = repos.Runs.TouchHeartbeat(ctx, parked.ID, time.Now())
	if err != nil {
		t.Fatalf("TouchHeartbeat(parked): %v", err)
	}
	if updated {
		t.Fatal("parked run heartbeat must NOT be updated (would clobber the park)")
	}
	got, _ = repos.Runs.Get(ctx, parked.ID)
	if got.Status != domain.RunStatusParked {
		t.Errorf("parked status changed: %s", got.Status)
	}
	if got.LastHeartbeat == nil || !got.LastHeartbeat.Equal(old.UTC().Truncate(time.Nanosecond)) {
		// Heartbeat should be untouched (still ~old). Allow for storage truncation.
		if got.LastHeartbeat != nil && got.LastHeartbeat.After(old.Add(time.Minute)) {
			t.Errorf("parked heartbeat was advanced: %v", got.LastHeartbeat)
		}
	}

	// terminal (complete) → NO-OP.
	complete := mk(domain.RunStatusComplete)
	if updated, err := repos.Runs.TouchHeartbeat(ctx, complete.ID, time.Now()); err != nil || updated {
		t.Fatalf("TouchHeartbeat(complete): updated=%v err=%v (want false,nil)", updated, err)
	}
}

// TestUpdateRunnerStreamState_StatusGuarded pins the park-clobber fix: the
// runner's transcript callbacks (OnAdvance/OnProcessStart/OnSessionID) persist
// streaming columns on every agent output chunk from an in-memory run whose
// Status is the stale "running". During the park turn-end grace the agent keeps
// emitting output; a full-row write would rewrite a just-persisted
// running→parked transition back to running (clobbering the park before
// detectParked reads it). UpdateRunnerStreamState must be status-guarded so a
// parked/terminal run is never resurrected. (Found by the local-inference
// park/resume e2e; the prior detectParked-only guard was structurally too late.)
func TestUpdateRunnerStreamState_StatusGuarded(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Stream Task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	mk := func(status domain.RunStatus) *domain.Run {
		run := &domain.Run{
			ID:            uuid.New(),
			TaskID:        task.ID,
			RunMode:       domain.RunModeSandboxed,
			Status:        status,
			Phase:         domain.RunPhaseExecuting,
			ApprovalState: domain.ApprovalStateNone,
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("Create run (%s): %v", status, err)
		}
		return run
	}

	// running → streaming write applies (cursor persists, status unchanged).
	running := mk(domain.RunStatusRunning)
	running.TranscriptCursor = 42
	running.SessionID = "sess-running"
	updated, err := repos.Runs.UpdateRunnerStreamState(ctx, running)
	if err != nil {
		t.Fatalf("UpdateRunnerStreamState(running): %v", err)
	}
	if !updated {
		t.Fatal("running run streaming state should have been updated")
	}
	got, _ := repos.Runs.Get(ctx, running.ID)
	if got.TranscriptCursor != 42 || got.SessionID != "sess-running" {
		t.Errorf("running streaming state not persisted: cursor=%d session=%q", got.TranscriptCursor, got.SessionID)
	}
	if got.Status != domain.RunStatusRunning {
		t.Errorf("running status changed unexpectedly: %s", got.Status)
	}

	// parked → NO-OP (the critical anti-clobber guarantee). An in-flight
	// transcript callback must not resurrect the park. Status stays parked and
	// the streaming columns are NOT written.
	parked := mk(domain.RunStatusParked)
	parked.TranscriptCursor = 99
	parked.SessionID = "should-not-persist"
	updated, err = repos.Runs.UpdateRunnerStreamState(ctx, parked)
	if err != nil {
		t.Fatalf("UpdateRunnerStreamState(parked): %v", err)
	}
	if updated {
		t.Fatal("parked run streaming state must NOT be updated (would clobber the park)")
	}
	got, _ = repos.Runs.Get(ctx, parked.ID)
	if got.Status != domain.RunStatusParked {
		t.Errorf("parked status was clobbered to %s (the bug)", got.Status)
	}
	if got.TranscriptCursor == 99 || got.SessionID == "should-not-persist" {
		t.Errorf("parked streaming columns were written: cursor=%d session=%q", got.TranscriptCursor, got.SessionID)
	}

	// terminal (failed) → NO-OP.
	failed := mk(domain.RunStatusFailed)
	if updated, err := repos.Runs.UpdateRunnerStreamState(ctx, failed); err != nil || updated {
		t.Fatalf("UpdateRunnerStreamState(failed): updated=%v err=%v (want false,nil)", updated, err)
	}
}

// TestRunLastAwaitFieldsRoundTrip verifies the durable re-fetch SSOT + re-park
// guard bookkeeping (last_await_key/result/resolved_at, last_wake_seq,
// same_key_park_streak) persist and reload through Create/Update/Get.
func TestRunLastAwaitFieldsRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{ID: uuid.New(), Title: "Await Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}

	resolved := time.Now().UTC().Truncate(time.Second)
	run := &domain.Run{
		ID:                  uuid.New(),
		TaskID:              task.ID,
		RunMode:             domain.RunModeSandboxed,
		Status:              domain.RunStatusRunning,
		Phase:               domain.RunPhaseExecuting,
		ApprovalState:       domain.ApprovalStateNone,
		LastAwaitKey:        "git-control-tower:agent-manager/am-park-resume",
		LastAwaitResult:     `{"status":"ready"}`,
		LastAwaitResolvedAt: &resolved,
		LastWakeSeq:         12,
		SameKeyParkStreak:   1,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.LastAwaitKey != run.LastAwaitKey || got.LastAwaitResult != run.LastAwaitResult {
		t.Errorf("await key/result not persisted: key=%q result=%q", got.LastAwaitKey, got.LastAwaitResult)
	}
	if got.LastWakeSeq != 12 || got.SameKeyParkStreak != 1 {
		t.Errorf("guard counters not persisted: wakeSeq=%d streak=%d", got.LastWakeSeq, got.SameKeyParkStreak)
	}
	if got.LastAwaitResolvedAt == nil || !got.LastAwaitResolvedAt.Equal(resolved) {
		t.Errorf("resolved_at not persisted: %v want %v", got.LastAwaitResolvedAt, resolved)
	}

	// Update mutates the fields (next await resolves) — reload reflects it.
	got.SameKeyParkStreak = 0
	got.LastAwaitResult = `{"status":"done"}`
	if err := repos.Runs.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	reloaded, _ := repos.Runs.Get(ctx, run.ID)
	if reloaded.SameKeyParkStreak != 0 || reloaded.LastAwaitResult != `{"status":"done"}` {
		t.Errorf("update did not persist: streak=%d result=%q", reloaded.SameKeyParkStreak, reloaded.LastAwaitResult)
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
		ProfileKey: "filter-profile", RoleRef: "code.default",
	}
	if err := repos.Profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile: %v", err)
	}

	// Create runs with different tags and statuses
	sourceRunID := uuid.New()
	investigationRunID := uuid.New()
	runs := []*domain.Run{
		{ID: uuid.New(), TaskID: task.ID, AgentProfileID: &profile.ID, Tag: "batch-1", Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-1", SourceRunIDs: []uuid.UUID{sourceRunID}},
		{ID: uuid.New(), TaskID: task.ID, Tag: "batch-2", Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone, IdempotencyKey: "filter-key-2", SourceInvestigationRunID: &investigationRunID},
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

	// Filter by source run ID lineage
	investigationRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		InvestigatesRunID: &sourceRunID,
	})
	if err != nil {
		t.Fatalf("List by source run ID: %v", err)
	}
	if len(investigationRuns) != 1 {
		t.Errorf("expected 1 investigation run for source run ID, got %d", len(investigationRuns))
	}

	// Filter by source investigation run ID lineage
	applyRuns, err := repos.Runs.List(ctx, repository.RunListFilter{
		AppliesInvestigationRunID: &investigationRunID,
	})
	if err != nil {
		t.Fatalf("List by source investigation run ID: %v", err)
	}
	if len(applyRuns) != 1 {
		t.Errorf("expected 1 apply run for source investigation run ID, got %d", len(applyRuns))
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

func TestEventRepositoryReadsLegacyTypedPayloadRow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "legacy event task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// This JSON was written by the removed union payload. Existing SQLite rows
	// remain in place and must decode into the current typed event schema.
	if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, ?, 0, ?, ?)`, uuid.New().String(), run.ID.String(), domain.EventTypeToolResult, `{"toolName":"Read","toolOutput":"file contents","toolError":"permission denied"}`); err != nil {
		t.Fatalf("seed legacy event row: %v", err)
	}
	if err := db.migrateRunEventPayloads(ctx); err != nil {
		t.Fatalf("migrate legacy event row: %v", err)
	}
	var persisted string
	if err := db.GetContext(ctx, &persisted, `SELECT data FROM run_events WHERE run_id = ?`, run.ID.String()); err != nil {
		t.Fatalf("read migrated event JSON: %v", err)
	}
	if strings.Contains(persisted, "toolOutput") || strings.Contains(persisted, "toolError") {
		t.Fatalf("legacy event fields remain after migration: %s", persisted)
	}

	events, err := repos.Events.Get(ctx, run.ID, -1, 10)
	if err != nil {
		t.Fatalf("read legacy event row: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	payload, ok := events[0].Data.(*domain.ToolResultEventData)
	if !ok {
		t.Fatalf("payload = %T, want *domain.ToolResultEventData", events[0].Data)
	}
	if payload.ToolName != "Read" || payload.Output != "file contents" || payload.Error != "permission denied" || payload.Success {
		t.Fatalf("migrated payload = %#v", payload)
	}
}

// ============================================================================
// Checkpoint Repository Tests
// ============================================================================
