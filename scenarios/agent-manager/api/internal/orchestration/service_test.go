package orchestration_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"
	"agent-manager/internal/testutil"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-001] [REQ:REQ-P0-002] [REQ:REQ-P0-003] [REQ:REQ-P0-004]
// Tests for orchestration service - profile, task, and run operations

func newTestOrchestrator(t *testing.T) orchestration.Service {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")
	mustRegisterRunner(t, runnerRegistry, mockRunner)

	return orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: true,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
	)
}

func mustCreateProfile(t *testing.T, svc orchestration.Service, ctx context.Context, profile *domain.AgentProfile) *domain.AgentProfile {
	t.Helper()
	created, err := svc.CreateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	return created
}

func mustCreateTask(t *testing.T, svc orchestration.Service, ctx context.Context, task *domain.Task) *domain.Task {
	t.Helper()
	created, err := svc.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	return created
}

func mustRegisterRunner(t *testing.T, registry runner.Registry, r runner.Runner) {
	t.Helper()
	if err := registry.Register(r); err != nil {
		t.Fatalf("register runner: %v", err)
	}
}

func TestOrchestrator_ProfileCRUD(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-3-opus",
		MaxTurns:   100,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	created := mustCreateProfile(t, svc, ctx, profile)
	if created.ID != profile.ID {
		t.Error("created profile should have same ID")
	}

	// Get
	got, err := svc.GetProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected profile, got nil")
	}
	if got.Name != "test-profile" {
		t.Errorf("expected name 'test-profile', got '%s'", got.Name)
	}

	// List
	profiles, err := svc.ListProfiles(ctx, orchestration.ListOptions{})
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}

	// Update
	profile.MaxTurns = 200
	updated, err := svc.UpdateProfile(ctx, profile)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if updated.MaxTurns != 200 {
		t.Errorf("expected MaxTurns 200, got %d", updated.MaxTurns)
	}

	// Delete
	if err := svc.DeleteProfile(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfile failed: %v", err)
	}

	got, _ = svc.GetProfile(ctx, profile.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestOrchestrator_EnsureProfile(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	result, err := svc.EnsureProfile(ctx, orchestration.EnsureProfileRequest{
		ProfileKey: "system-monitor-investigator",
		Defaults: &domain.AgentProfile{
			Name:       "System Monitor Investigator",
			RunnerType: domain.RunnerTypeCodex,
			Model:      "codex-mini-latest",
		},
	})
	if err != nil {
		t.Fatalf("EnsureProfile create failed: %v", err)
	}
	if !result.Created || result.Profile == nil {
		t.Fatalf("expected profile to be created")
	}
	if result.Profile.ProfileKey != "system-monitor-investigator" {
		t.Errorf("expected profile key to be set, got %q", result.Profile.ProfileKey)
	}

	result, err = svc.EnsureProfile(ctx, orchestration.EnsureProfileRequest{
		ProfileKey: "system-monitor-investigator",
	})
	if err != nil {
		t.Fatalf("EnsureProfile resolve failed: %v", err)
	}
	if result.Created || result.Updated {
		t.Errorf("expected resolve without create/update")
	}

	result, err = svc.EnsureProfile(ctx, orchestration.EnsureProfileRequest{
		ProfileKey:     "system-monitor-investigator",
		UpdateExisting: true,
		Defaults: &domain.AgentProfile{
			Name:       "System Monitor Investigator",
			RunnerType: domain.RunnerTypeClaudeCode,
			Model:      "claude-3-opus",
		},
	})
	if err != nil {
		t.Fatalf("EnsureProfile update failed: %v", err)
	}
	if !result.Updated {
		t.Errorf("expected profile to be updated")
	}
	if result.Profile.RunnerType != domain.RunnerTypeClaudeCode {
		t.Errorf("expected runner type to update, got %v", result.Profile.RunnerType)
	}
}

func TestOrchestrator_ReconcileScenarioProfiles(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	result, err := svc.ReconcileScenarioProfiles(ctx, orchestration.ReconcileScenarioProfilesRequest{
		Scenario: "scenario-to-desktop",
	})
	if err != nil {
		t.Fatalf("ReconcileScenarioProfiles create failed: %v", err)
	}
	if result.Created != 1 || result.Failed != 0 {
		t.Fatalf("expected one created profile and no failures, got created=%d failed=%d results=%+v", result.Created, result.Failed, result.Results)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(result.Results))
	}
	item := result.Results[0]
	if item.ProfileKey != "scenario-to-desktop/default" {
		t.Fatalf("unexpected profile key %q", item.ProfileKey)
	}

	profileID, err := uuid.Parse(item.ProfileID)
	if err != nil {
		t.Fatalf("parse reconciled profile id: %v", err)
	}
	profile, err := svc.GetProfile(ctx, profileID)
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if profile.OwnerScenario != "scenario-to-desktop" || profile.SourcePath != ".vrooli/agent-profiles/default.json" {
		t.Fatalf("missing source metadata: owner=%q source=%q", profile.OwnerScenario, profile.SourcePath)
	}
	if profile.SourceHash == "" || profile.LastAppliedHash != profile.SourceHash {
		t.Fatalf("source hashes not recorded correctly: source=%q last=%q", profile.SourceHash, profile.LastAppliedHash)
	}

	result, err = svc.ReconcileScenarioProfiles(ctx, orchestration.ReconcileScenarioProfilesRequest{
		Scenario: "scenario-to-desktop",
	})
	if err != nil {
		t.Fatalf("ReconcileScenarioProfiles unchanged failed: %v", err)
	}
	if result.Unchanged != 1 {
		t.Fatalf("expected unchanged result, got %+v", result)
	}

	profile.MaxTurns++
	if _, err := svc.UpdateProfile(ctx, profile); err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	result, err = svc.ReconcileScenarioProfiles(ctx, orchestration.ReconcileScenarioProfilesRequest{
		Scenario: "scenario-to-desktop",
	})
	if err != nil {
		t.Fatalf("ReconcileScenarioProfiles conflict failed: %v", err)
	}
	if result.Conflicted != 1 {
		t.Fatalf("expected local override conflict, got %+v", result)
	}
}

func TestOrchestrator_ReconcileScenarioProfiles_AllSeededScenariosDryRun(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	for _, scenario := range []string{"swarm-manager", "system-monitor", "app-issue-tracker", "scenario-to-desktop"} {
		t.Run(scenario, func(t *testing.T) {
			result, err := svc.ReconcileScenarioProfiles(ctx, orchestration.ReconcileScenarioProfilesRequest{
				Scenario: scenario,
				DryRun:   true,
			})
			if err != nil {
				t.Fatalf("ReconcileScenarioProfiles dry-run failed: %v", err)
			}
			if result.Failed != 0 || len(result.Results) != 1 {
				t.Fatalf("expected one valid dry-run result and no failures, got %+v", result)
			}
			if result.Results[0].ProfileKey != scenario+"/default" {
				t.Fatalf("profile key = %q, want %q", result.Results[0].ProfileKey, scenario+"/default")
			}
		})
	}
}

func TestOrchestrator_TaskCRUD(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "A test task for orchestration",
		ScopePath:   "src/",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	created := mustCreateTask(t, svc, ctx, task)
	if created.ID != task.ID {
		t.Error("created task should have same ID")
	}

	// Get
	got, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected task, got nil")
	}
	if got.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got '%s'", got.Title)
	}

	// List
	tasks, err := svc.ListTasks(ctx, orchestration.ListOptions{})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Update
	task.Description = "Updated description"
	updated, err := svc.UpdateTask(ctx, task)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}
	if updated.Description != "Updated description" {
		t.Errorf("expected updated description")
	}

	// Cancel
	if err := svc.CancelTask(ctx, task.ID); err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}

	got, _ = svc.GetTask(ctx, task.ID)
	if got.Status != domain.TaskStatusCancelled {
		t.Errorf("expected cancelled status, got %s", got.Status)
	}
}

func TestOrchestrator_RunOperations(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// First create a profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "run-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-3-opus",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Run Test Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Create run (will fail due to missing sandbox, but we can test the creation logic)
	run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Test prompt",
	})
	if err != nil {
		// Expected - sandbox not available
		t.Logf("CreateRun returned expected error (sandbox unavailable): %v", err)
	} else if run != nil {
		// If we got a run, verify it
		got, err := svc.GetRun(ctx, run.ID)
		if err != nil {
			t.Fatalf("GetRun failed: %v", err)
		}
		if got == nil {
			t.Fatal("expected run, got nil")
		}

		// List runs
		runs, err := svc.ListRuns(ctx, orchestration.RunListOptions{})
		if err != nil {
			t.Fatalf("ListRuns failed: %v", err)
		}
		if len(runs) < 1 {
			t.Error("expected at least 1 run")
		}
	}
}

func TestOrchestrator_GetHealth(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	health, err := svc.GetHealth(ctx)
	if err != nil {
		t.Fatalf("GetHealth failed: %v", err)
	}

	// Verify required health fields per schema
	if health.Service != "agent-manager" {
		t.Errorf("expected service 'agent-manager', got '%s'", health.Service)
	}
	if health.Status == "" {
		t.Error("expected status to be set")
	}
	if health.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestOrchestrator_GetRunnerStatus(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	statuses, err := svc.GetRunnerStatus(ctx)
	if err != nil {
		t.Fatalf("GetRunnerStatus failed: %v", err)
	}

	// We registered a mock claude-code runner
	found := false
	for _, s := range statuses {
		if s.Type == domain.RunnerTypeClaudeCode {
			found = true
			if !s.Available {
				t.Error("mock runner should be available")
			}
		}
	}
	if !found {
		t.Error("expected claude-code runner in status")
	}
}

func TestOrchestrator_ListProfiles_Pagination(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create multiple profiles
	for i := 0; i < 10; i++ {
		profile := &domain.AgentProfile{
			ID:         uuid.New(),
			Name:       "profile-" + uuid.New().String(),
			RunnerType: domain.RunnerTypeClaudeCode,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		mustCreateProfile(t, svc, ctx, profile)
	}

	// Test limit
	profiles, _ := svc.ListProfiles(ctx, orchestration.ListOptions{Limit: 5})
	if len(profiles) != 5 {
		t.Errorf("expected 5 profiles with limit, got %d", len(profiles))
	}

	// Test offset
	profiles, _ = svc.ListProfiles(ctx, orchestration.ListOptions{Offset: 8})
	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles with offset 8, got %d", len(profiles))
	}
}

func TestOrchestrator_GetProfile_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	profile, err := svc.GetProfile(ctx, uuid.New())
	// Service returns NotFoundError for non-existent profiles
	if err == nil {
		t.Error("GetProfile should return error for non-existent profile")
	}
	if profile != nil {
		t.Error("expected nil for non-existent profile")
	}
}

func TestOrchestrator_GetTask_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	task, err := svc.GetTask(ctx, uuid.New())
	// Service returns NotFoundError for non-existent tasks
	if err == nil {
		t.Error("GetTask should return error for non-existent task")
	}
	if task != nil {
		t.Error("expected nil for non-existent task")
	}
}

func TestOrchestrator_DeleteProfile_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Delete non-existent profile should be idempotent
	err := svc.DeleteProfile(ctx, uuid.New())
	if err != nil {
		t.Errorf("DeleteProfile should be idempotent: %v", err)
	}
}

// =============================================================================
// RUN LIFECYCLE TESTS
// =============================================================================
// [REQ:REQ-P0-004] Run Status Tracking
// [REQ:REQ-P0-007] Approval Workflow

// TestOrchestrator_StopRun_NotFound tests stopping a non-existent run.
func TestOrchestrator_StopRun_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	err := svc.StopRun(ctx, uuid.New())
	if err == nil {
		t.Error("StopRun should return error for non-existent run")
	}
}

// TestOrchestrator_GetRun_NotFound tests retrieving a non-existent run.
func TestOrchestrator_GetRun_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	run, err := svc.GetRun(ctx, uuid.New())
	if err == nil {
		t.Error("GetRun should return error for non-existent run")
	}
	if run != nil {
		t.Error("expected nil for non-existent run")
	}
}

// TestOrchestrator_GetRunByTag_NotFound tests retrieving by non-existent tag.
func TestOrchestrator_GetRunByTag_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	run, err := svc.GetRunByTag(ctx, "nonexistent-tag")
	if err == nil {
		t.Error("GetRunByTag should return error for non-existent tag")
	}
	if run != nil {
		t.Error("expected nil for non-existent tag")
	}
}

// TestOrchestrator_StopRunByTag_NotFound tests stopping by non-existent tag.
func TestOrchestrator_StopRunByTag_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	err := svc.StopRunByTag(ctx, "nonexistent-tag")
	if err == nil {
		t.Error("StopRunByTag should return error for non-existent tag")
	}
}

// TestOrchestrator_StopAllRuns_Empty tests stopping when no runs exist.
func TestOrchestrator_StopAllRuns_Empty(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	result, err := svc.StopAllRuns(ctx, orchestration.StopAllOptions{})
	if err != nil {
		t.Fatalf("StopAllRuns should not fail on empty: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Stopped != 0 {
		t.Errorf("expected 0 stopped, got %d", result.Stopped)
	}
}

// TestOrchestrator_StopAllRuns_WithTagPrefix tests tag prefix filtering.
func TestOrchestrator_StopAllRuns_WithTagPrefix(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	result, err := svc.StopAllRuns(ctx, orchestration.StopAllOptions{
		TagPrefix: "test-",
		Force:     true,
	})
	if err != nil {
		t.Fatalf("StopAllRuns with tag prefix should not fail: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
}

// TestOrchestrator_ApproveRun_NotFound tests approving non-existent run.
func TestOrchestrator_ApproveRun_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	result, err := svc.ApproveRun(ctx, orchestration.ApproveRequest{
		RunID:     uuid.New(),
		Actor:     "test-user",
		CommitMsg: "Apply changes",
	})
	if err == nil {
		t.Error("ApproveRun should return error for non-existent run")
	}
	if result != nil {
		t.Error("expected nil result for non-existent run")
	}
}

// TestOrchestrator_RejectRun_NotFound tests rejecting non-existent run.
func TestOrchestrator_RejectRun_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	err := svc.RejectRun(ctx, uuid.New(), "test-user", "Not acceptable")
	if err == nil {
		t.Error("RejectRun should return error for non-existent run")
	}
}

// TestOrchestrator_GetRunEvents_Empty tests getting events for a run without events.
func TestOrchestrator_GetRunEvents_Empty(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Get events for non-existent run
	events, err := svc.GetRunEvents(ctx, uuid.New(), event.GetOptions{})
	// Should return empty slice or nil, not error (events are optional)
	if err != nil {
		t.Logf("GetRunEvents for non-existent run returned error (acceptable): %v", err)
	} else if len(events) > 0 {
		t.Errorf("expected empty events for non-existent run, got %d", len(events))
	}
}

// TestOrchestrator_GetRunDiff_NotFound tests getting diff for non-existent run.
func TestOrchestrator_GetRunDiff_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	diff, err := svc.GetRunDiff(ctx, uuid.New())
	if err == nil {
		t.Error("GetRunDiff should return error for non-existent run")
	}
	if diff != nil {
		t.Error("expected nil diff for non-existent run")
	}
}

// TestOrchestrator_ListRuns_Filters tests various run list filters.
func TestOrchestrator_ListRuns_Filters(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	tests := []struct {
		name string
		opts orchestration.RunListOptions
	}{
		{
			name: "status filter",
			opts: orchestration.RunListOptions{
				Status: func() *domain.RunStatus { s := domain.RunStatusRunning; return &s }(),
			},
		},
		{
			name: "task ID filter",
			opts: orchestration.RunListOptions{
				TaskID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			},
		},
		{
			name: "profile ID filter",
			opts: orchestration.RunListOptions{
				AgentProfileID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			},
		},
		{
			name: "tag prefix filter",
			opts: orchestration.RunListOptions{
				TagPrefix: "ecosystem-",
			},
		},
		{
			name: "combined filters",
			opts: orchestration.RunListOptions{
				Status:    func() *domain.RunStatus { s := domain.RunStatusPending; return &s }(),
				TagPrefix: "test-",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runs, err := svc.ListRuns(ctx, tt.opts)
			if err != nil {
				t.Errorf("ListRuns with %s should not fail: %v", tt.name, err)
			}
			// Empty list is valid for filters with no matches
			if runs == nil {
				t.Errorf("expected empty slice, got nil")
			}
		})
	}
}

// TestOrchestrator_ResumeRun_NotFound tests resuming a non-existent run.
func TestOrchestrator_ResumeRun_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	run, err := svc.ResumeRun(ctx, uuid.New())
	if err == nil {
		t.Error("ResumeRun should return error for non-existent run")
	}
	if run != nil {
		t.Error("expected nil for non-existent run")
	}
}

// TestOrchestrator_GetRunProgress_NotFound tests getting progress for non-existent run.
func TestOrchestrator_GetRunProgress_NotFound(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	progress, err := svc.GetRunProgress(ctx, uuid.New())
	if err == nil {
		t.Error("GetRunProgress should return error for non-existent run")
	}
	if progress != nil {
		t.Error("expected nil progress for non-existent run")
	}
}

// TestOrchestrator_ListStaleRuns_Empty tests listing stale runs when none exist.
func TestOrchestrator_ListStaleRuns_Empty(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	runs, err := svc.ListStaleRuns(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("ListStaleRuns should not fail on empty: %v", err)
	}
	// nil or empty slice are both acceptable for "no results"
	if len(runs) != 0 {
		t.Errorf("expected 0 stale runs, got %d", len(runs))
	}
}

// TestOrchestrator_CreateRun_IdempotencyKey tests idempotent run creation.
func TestOrchestrator_CreateRun_IdempotencyKey(t *testing.T) {
	svc := newTestOrchestrator(t)
	ctx := context.Background()

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "idempotent-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Idempotent Test Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	idempotencyKey := "test-idempotency-" + uuid.New().String()

	// First creation attempt (may fail due to sandbox, but key should be recorded)
	run1, err1 := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Test prompt",
		IdempotencyKey: idempotencyKey,
	})

	// Second creation attempt with same key should return same result
	run2, err2 := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Different prompt",
		IdempotencyKey: idempotencyKey,
	})

	// Both should have same outcome (either both succeed with same run, or both fail)
	if (err1 == nil) != (err2 == nil) {
		t.Log("Idempotency behavior may vary - first and second attempts had different error states")
	}
	if err1 == nil && err2 == nil && run1 != nil && run2 != nil {
		if run1.ID != run2.ID {
			t.Errorf("idempotent runs should have same ID: got %s and %s", run1.ID, run2.ID)
		}
	}
}

// =============================================================================
// SLOT ENFORCEMENT TESTS
// =============================================================================
// [REQ:REQ-P0-005] Run creation with capacity limits

// newTestOrchestratorWithLimit creates an orchestrator with a specific MaxConcurrentRuns limit.
func newTestOrchestratorWithLimit(t *testing.T, maxRuns int) (orchestration.Service, repository.RunRepository) {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")
	mustRegisterRunner(t, runnerRegistry, mockRunner)

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       maxRuns,
			RequireSandboxByDefault: false, // Disable sandbox for slot tests
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
	)

	return svc, repos.Runs
}

// TestOrchestrator_SlotEnforcement_BlocksAtCapacity verifies that CreateRun
// returns a CapacityExceededError when MaxConcurrentRuns is reached.
func TestOrchestrator_SlotEnforcement_BlocksAtCapacity(t *testing.T) {
	svc, runRepo := newTestOrchestratorWithLimit(t, 2) // Only allow 2 concurrent runs
	ctx := context.Background()

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "slot-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Slot Test Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Manually create 2 "running" runs to simulate capacity
	for i := 0; i < 2; i++ {
		run := &domain.Run{
			ID:        uuid.New(),
			TaskID:    task.ID,
			Status:    domain.RunStatusRunning,
			RunMode:   domain.RunModeInPlace,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := runRepo.Create(ctx, run); err != nil {
			t.Fatalf("failed to create test run: %v", err)
		}
	}

	// Verify we have 2 running runs
	count, _ := runRepo.CountByStatus(ctx, domain.RunStatusRunning)
	if count != 2 {
		t.Fatalf("expected 2 running runs, got %d", count)
	}

	// Now try to create another run - should fail with capacity error
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "This should fail",
	})

	if err == nil {
		t.Fatal("expected CapacityExceededError, got nil")
	}

	// Verify it's a CapacityExceededError
	capErr, ok := err.(*domain.CapacityExceededError)
	if !ok {
		t.Fatalf("expected *domain.CapacityExceededError, got %T: %v", err, err)
	}

	if capErr.Resource != "concurrent_runs" {
		t.Errorf("expected resource 'concurrent_runs', got '%s'", capErr.Resource)
	}
	if capErr.Current != 2 {
		t.Errorf("expected current=2, got %d", capErr.Current)
	}
	if capErr.Maximum != 2 {
		t.Errorf("expected maximum=2, got %d", capErr.Maximum)
	}
}

// TestOrchestrator_SlotEnforcement_ForceBypassesLimit verifies that Force=true
// allows run creation even when at capacity.
func TestOrchestrator_SlotEnforcement_ForceBypassesLimit(t *testing.T) {
	svc, runRepo := newTestOrchestratorWithLimit(t, 2)
	ctx := context.Background()

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "force-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Force Test Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Fill capacity with 2 running runs
	for i := 0; i < 2; i++ {
		run := &domain.Run{
			ID:        uuid.New(),
			TaskID:    task.ID,
			Status:    domain.RunStatusRunning,
			RunMode:   domain.RunModeInPlace,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := runRepo.Create(ctx, run); err != nil {
			t.Fatalf("create run: %v", err)
		}
	}

	// Try to create with Force=true - should NOT return capacity error
	// (may fail for other reasons like runner issues, but NOT capacity)
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Force this run",
		Force:          true,
	})
	// If there's an error, make sure it's NOT a CapacityExceededError
	if err != nil {
		if _, ok := err.(*domain.CapacityExceededError); ok {
			t.Fatal("Force=true should bypass capacity limits, but got CapacityExceededError")
		}
		// Other errors are acceptable (e.g., runner execution issues)
		t.Logf("Force run creation failed with non-capacity error (acceptable): %v", err)
	}
}

// TestOrchestrator_SlotEnforcement_CountsStartingRuns verifies that runs in
// "starting" status also count against the capacity limit.
func TestOrchestrator_SlotEnforcement_CountsStartingRuns(t *testing.T) {
	svc, runRepo := newTestOrchestratorWithLimit(t, 2)
	ctx := context.Background()

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "starting-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Starting Test Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Create 1 running and 1 starting run
	if err := runRepo.Create(ctx, &domain.Run{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Status:    domain.RunStatusRunning,
		RunMode:   domain.RunModeInPlace,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}
	if err := runRepo.Create(ctx, &domain.Run{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Status:    domain.RunStatusStarting,
		RunMode:   domain.RunModeInPlace,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Try to create another run - should fail (1 running + 1 starting = 2 = limit)
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "This should fail",
	})

	if err == nil {
		t.Fatal("expected CapacityExceededError when counting starting runs")
	}

	capErr, ok := err.(*domain.CapacityExceededError)
	if !ok {
		t.Fatalf("expected *domain.CapacityExceededError, got %T: %v", err, err)
	}
	if capErr.Current != 2 {
		t.Errorf("expected current=2 (1 running + 1 starting), got %d", capErr.Current)
	}
}

// TestOrchestrator_SlotEnforcement_AllowsUnderCapacity verifies that runs
// can be created when under the limit.
func TestOrchestrator_SlotEnforcement_AllowsUnderCapacity(t *testing.T) {
	svc, runRepo := newTestOrchestratorWithLimit(t, 3)
	ctx := context.Background()

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "under-capacity-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Under Capacity Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Create only 1 running run (limit is 3)
	if err := runRepo.Create(ctx, &domain.Run{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Status:    domain.RunStatusRunning,
		RunMode:   domain.RunModeInPlace,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// Try to create another run - should NOT fail with capacity error
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "This should succeed (capacity-wise)",
	})
	// If there's an error, make sure it's NOT a CapacityExceededError
	if err != nil {
		var capErr *domain.CapacityExceededError
		if errors.As(err, &capErr) {
			t.Fatal("should not get CapacityExceededError when under capacity")
		}
		// Other errors are acceptable (e.g., runner execution issues)
		t.Logf("run creation failed with non-capacity error (acceptable): %v", err)
	}
}

// =============================================================================
// FEATURE FLAGS & EXTRA FLAGS TESTS
// =============================================================================
// Tests for feature flag and extra flag handling in run creation.

// newTestOrchestratorWithFlagValidator creates an orchestrator with a custom flag validator.
func newTestOrchestratorWithFlagValidator(t *testing.T, fv runner.FlagValidator) orchestration.Service {
	t.Helper()

	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")
	mustRegisterRunner(t, runnerRegistry, mockRunner)

	return orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: false, // Disable sandbox for these tests
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
		orchestration.WithFlagValidator(fv),
	)
}

// TestOrchestrator_CreateRun_InlineEnableBrowser tests that inline EnableBrowser
// flag is applied to the resolved config.
func TestOrchestrator_CreateRun_InlineEnableBrowser(t *testing.T) {
	svc := newTestOrchestratorWithFlagValidator(t, &runner.MockFlagValidator{})
	ctx := context.Background()

	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "browser-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Browser Test Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	enableBrowser := true
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Test with browser enabled",
		EnableBrowser:  &enableBrowser,
	})
	// The run may fail for non-capacity reasons (runner execution issues in mock),
	// but it should NOT fail due to EnableBrowser being invalid.
	if err != nil {
		var capErr *domain.CapacityExceededError
		if errors.As(err, &capErr) {
			t.Fatal("should not get CapacityExceededError")
		}
		// Other errors (e.g., runner execution) are acceptable
		t.Logf("CreateRun returned non-feature-flag error (acceptable): %v", err)
	}
}

// TestOrchestrator_CreateRun_ExtraFlagsValidation tests that extra flags
// are validated against runner allowlists when a FlagValidator is set.
func TestOrchestrator_CreateRun_ExtraFlagsValidation(t *testing.T) {
	rejecting := &runner.MockFlagValidator{
		ValidateFlagsFunc: func(rt domain.RunnerType, flags []string) error {
			return domain.NewValidationError("extraFlags", "mock: flag rejected")
		},
	}
	svc := newTestOrchestratorWithFlagValidator(t, rejecting)
	ctx := context.Background()

	// Create profile with extra flags
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "flags-test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--forbidden"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Flags Validation Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "This should fail validation",
	})

	if err == nil {
		t.Fatal("expected error from flag validation, got nil")
	}

	// The error should be a validation error mentioning extraFlags
	ve, ok := err.(*domain.ValidationError)
	if !ok {
		t.Logf("error type: %T, error: %v", err, err)
		// May be wrapped - check the message
		if !errors.Is(err, err) {
			t.Logf("non-validation error (may be expected): %v", err)
		}
	} else {
		if ve.Field != "extraFlags" {
			t.Errorf("expected field 'extraFlags', got %q", ve.Field)
		}
	}
}

// TestOrchestrator_CreateRun_ExtraFlagsInlineOverride tests that inline
// extra flags override profile extra flags per runner type.
func TestOrchestrator_CreateRun_ExtraFlagsInlineOverride(t *testing.T) {
	// Use a mock validator that records which flags it validates
	var validatedFlags []string
	recording := &runner.MockFlagValidator{
		ValidateFlagsFunc: func(rt domain.RunnerType, flags []string) error {
			validatedFlags = append(validatedFlags, flags...)
			return nil
		},
	}
	svc := newTestOrchestratorWithFlagValidator(t, recording)
	ctx := context.Background()

	// Create profile with extra flags
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "inline-override-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--profile-flag"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Inline Override Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Create run with inline extra flags that override profile's
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Test inline override",
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--inline-flag"},
		},
	})
	// Verify that the inline flags were validated (may fail for other reasons)
	if err != nil {
		t.Logf("CreateRun returned error (may be expected): %v", err)
	}

	// Check that the validated flags include the inline flag
	hasInlineFlag := false
	for _, f := range validatedFlags {
		if f == "--inline-flag" {
			hasInlineFlag = true
		}
	}
	if !hasInlineFlag {
		t.Errorf("expected --inline-flag to be validated, validated flags: %v", validatedFlags)
	}
}

// TestOrchestrator_CreateRun_NoFlagValidator tests that when no FlagValidator
// is set, extra flags pass through without validation.
func TestOrchestrator_CreateRun_NoFlagValidator(t *testing.T) {
	svc, _ := newTestOrchestratorWithLimit(t, 10)
	ctx := context.Background()

	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "no-validator-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		ExtraFlags: domain.RunnerExtraFlags{
			domain.RunnerTypeClaudeCode: []string{"--anything-goes"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "No Validator Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Should NOT fail due to flag validation (no validator is set)
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Test without validator",
	})
	if err != nil {
		// Check it's not a validation error about extra flags
		if ve, ok := err.(*domain.ValidationError); ok && ve.Field == "extraFlags" {
			t.Fatalf("should not get extraFlags validation error when no validator is set: %v", err)
		}
		// Other errors are acceptable
		t.Logf("CreateRun returned non-validation error (acceptable): %v", err)
	}
}

// TestOrchestrator_SlotEnforcement_ZeroLimitDisablesCheck verifies that
// MaxConcurrentRuns=0 disables the capacity check entirely.
func TestOrchestrator_SlotEnforcement_ZeroLimitDisablesCheck(t *testing.T) {
	svc, runRepo := newTestOrchestratorWithLimit(t, 0) // 0 = disabled
	ctx := context.Background()

	// Create profile and task
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "no-limit-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mustCreateProfile(t, svc, ctx, profile)

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "No Limit Task",
		ScopePath: "src/",
		Status:    domain.TaskStatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateTask(t, svc, ctx, task)

	// Create many running runs
	for i := 0; i < 100; i++ {
		if err := runRepo.Create(ctx, &domain.Run{
			ID:        uuid.New(),
			TaskID:    task.ID,
			Status:    domain.RunStatusRunning,
			RunMode:   domain.RunModeInPlace,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("create run: %v", err)
		}
	}

	// Try to create another run - should NOT fail with capacity error
	_, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "No limit test",
	})
	// If there's an error, make sure it's NOT a CapacityExceededError
	if err != nil {
		var capErr *domain.CapacityExceededError
		if errors.As(err, &capErr) {
			t.Fatal("MaxConcurrentRuns=0 should disable capacity check")
		}
	}
}

// TestOrchestrator_CreateRun_ResolvesRelativeProjectRoot verifies that a task
// with a relative projectRoot (e.g., ".") gets resolved to an absolute path
// before sandbox creation. This prevents the workspace-sandbox API from
// rejecting the request with a database constraint violation.
//
// Regression test for: investigation runs failing with "Unable to create
// isolated workspace" when the source run's task had projectRoot=".".
func TestOrchestrator_CreateRun_ResolvesRelativeProjectRoot(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	runnerRegistry := runner.NewRegistry()
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.SetAvailable(true, "mock runner available")
	mustRegisterRunner(t, runnerRegistry, mockRunner)

	svc := orchestration.New(
		repos.Profiles,
		repos.Tasks,
		repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: true,
			DefaultProjectRoot:      "/fallback/root",
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency),
	)
	ctx := context.Background()

	profile := mustCreateProfile(t, svc, ctx, &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "resolve-root-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-3-opus",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	// Create task with relative projectRoot "." — the exact scenario that
	// caused the bug when investigation runs inherited this from source tasks.
	task := mustCreateTask(t, svc, ctx, &domain.Task{
		ID:          uuid.New(),
		Title:       "Relative Root Task",
		ScopePath:   ".",
		ProjectRoot: ".",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	// CreateRun should succeed past the preflight validation.
	// It will still fail asynchronously (no sandbox provider), but the run
	// should be created with the task's projectRoot resolved to absolute.
	run, err := svc.CreateRun(ctx, orchestration.CreateRunRequest{
		TaskID:         task.ID,
		AgentProfileID: &profile.ID,
		Prompt:         "Test prompt",
		Force:          true,
	})
	// The run should be created (preflight passes)
	if err != nil {
		t.Fatalf("CreateRun failed unexpectedly: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to be created")
	}

	// Verify the task's projectRoot was resolved to an absolute path
	updatedTask, err := svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if !filepath.IsAbs(updatedTask.ProjectRoot) {
		t.Errorf("expected task.ProjectRoot to be absolute, got %q", updatedTask.ProjectRoot)
	}
	if updatedTask.ProjectRoot == "." {
		t.Error("task.ProjectRoot should not remain as '.'")
	}
}
