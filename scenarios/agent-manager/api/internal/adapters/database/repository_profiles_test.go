// Responsibility: retain repository test declarations within their original package.
package database

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
	"testing"
	"time"
)

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
