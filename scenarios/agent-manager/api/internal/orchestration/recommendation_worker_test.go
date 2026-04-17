package orchestration

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/recommendation"
	"agent-manager/internal/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/testutil"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockAllowlistProvider implements AllowlistProvider for testing.
type mockAllowlistProvider struct {
	rules []domain.InvestigationTagRule
}

func (m *mockAllowlistProvider) GetAllowlist(ctx context.Context) []domain.InvestigationTagRule {
	if m.rules == nil {
		return domain.DefaultInvestigationTagAllowlist()
	}
	return m.rules
}

// mockBroadcaster implements EventBroadcaster for testing.
type mockBroadcaster struct {
	mu         sync.Mutex
	broadcasts []*domain.Run
}

func (m *mockBroadcaster) BroadcastEvent(event *domain.RunEvent) {
	// No-op for testing
}

func (m *mockBroadcaster) BroadcastRunStatus(run *domain.Run) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcasts = append(m.broadcasts, run)
}

func (m *mockBroadcaster) BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string) {
	// No-op for testing
}

func (m *mockBroadcaster) getBroadcasts() []*domain.Run {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*domain.Run{}, m.broadcasts...)
}

// setupWorkerRepos creates SQLite-backed repos for recommendation worker tests.
// Returns the run repository, event store, and a task ID for seeding runs.
type workerTestDeps struct {
	runRepo    repository.RunRepository
	eventStore event.Store
	taskID     uuid.UUID
	db         *database.DB
}

func setupWorkerRepos(t *testing.T) workerTestDeps {
	t.Helper()
	db, dbCleanup := testutil.SetupTestDB(t)
	t.Cleanup(dbCleanup)

	repos, eventStore, cleanup := testutil.SetupTestReposWithDB(t, db)
	t.Cleanup(cleanup)

	ctx := context.Background()

	// Create a task that runs can reference
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		Title:       "Test Task",
		Description: "Test task for recommendation worker tests",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	return workerTestDeps{
		runRepo:    repos.Runs,
		eventStore: eventStore,
		taskID:     taskID,
		db:         db,
	}
}

func TestRecommendationWorker_StartStop(t *testing.T) {
	deps := setupWorkerRepos(t)
	runRepo, eventStore := deps.runRepo, deps.eventStore
	extractor := recommendation.NewMockExtractor()

	worker := NewRecommendationWorker(runRepo, eventStore, extractor)

	// Should not be running initially
	if worker.IsRunning() {
		t.Error("worker should not be running before Start()")
	}

	// Start the worker
	ctx := context.Background()
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Should be running now
	if !worker.IsRunning() {
		t.Error("worker should be running after Start()")
	}

	// Double start should fail
	if err := worker.Start(ctx); err == nil {
		t.Error("double Start() should return error")
	}

	// Stop the worker
	if err := worker.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Should not be running after stop
	if worker.IsRunning() {
		t.Error("worker should not be running after Stop()")
	}

	// Double stop should be no-op
	if err := worker.Stop(); err != nil {
		t.Errorf("double Stop() should not error: %v", err)
	}
}

func TestRecommendationWorker_AllowlistFiltering(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		rules     []domain.InvestigationTagRule
		expectRun bool
	}{
		{
			name: "matches default allowlist - investigation",
			tag:  "investigation",
			rules: []domain.InvestigationTagRule{
				{Pattern: "investigation", IsRegex: false, CaseSensitive: false},
			},
			expectRun: true,
		},
		{
			name: "matches default allowlist - prefix-investigation",
			tag:  "scenario-investigation",
			rules: []domain.InvestigationTagRule{
				{Pattern: "*-investigation", IsRegex: false, CaseSensitive: false},
			},
			expectRun: true,
		},
		{
			name: "does not match - apply suffix",
			tag:  "investigation-apply",
			rules: []domain.InvestigationTagRule{
				{Pattern: "investigation", IsRegex: false, CaseSensitive: false},
			},
			expectRun: false, // The tag doesn't match exactly "investigation"
		},
		{
			name: "does not match - different tag",
			tag:  "some-other-tag",
			rules: []domain.InvestigationTagRule{
				{Pattern: "investigation", IsRegex: false, CaseSensitive: false},
			},
			expectRun: false,
		},
		{
			name: "matches regex pattern",
			tag:  "my-custom-investigation-v2",
			rules: []domain.InvestigationTagRule{
				{Pattern: ".*investigation.*", IsRegex: true, CaseSensitive: false},
			},
			expectRun: true,
		},
		{
			name: "case insensitive match",
			tag:  "INVESTIGATION",
			rules: []domain.InvestigationTagRule{
				{Pattern: "investigation", IsRegex: false, CaseSensitive: false},
			},
			expectRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := setupWorkerRepos(t)
			runRepo, eventStore := deps.runRepo, deps.eventStore
			extractor := recommendation.NewMockExtractor()
			allowlist := &mockAllowlistProvider{rules: tt.rules}

			worker := NewRecommendationWorker(
				runRepo,
				eventStore,
				extractor,
				WithRecommendationWorkerAllowlist(allowlist),
			)

			// Create a test run
			run := &domain.Run{
				ID:     uuid.New(),
				Tag:    tt.tag,
				Status: domain.RunStatusComplete,
			}

			ctx := context.Background()
			eligible := worker.isEligibleForExtraction(ctx, run)

			if eligible != tt.expectRun {
				t.Errorf("isEligibleForExtraction() = %v, want %v", eligible, tt.expectRun)
			}
		})
	}
}

func TestRecommendationWorker_ProcessQueue(t *testing.T) {
	ctx := context.Background()
	deps := setupWorkerRepos(t)
	runRepo, eventStore, taskID := deps.runRepo, deps.eventStore, deps.taskID

	// Create a mock extractor that returns success
	extractor := &recommendation.MockExtractor{
		ExtractFunc: func(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error) {
			return &domain.ExtractionResult{
				Success: true,
				Categories: []domain.RecommendationCategory{
					{
						ID:   "cat-1",
						Name: "Test Category",
						Recommendations: []domain.Recommendation{
							{ID: "rec-1", Text: "Test recommendation", Selected: true},
						},
					},
				},
			}, nil
		},
		IsAvailableFunc: func(ctx context.Context) bool {
			return true
		},
	}

	broadcaster := &mockBroadcaster{}

	worker := NewRecommendationWorker(
		runRepo,
		eventStore,
		extractor,
		WithRecommendationWorkerBroadcaster(broadcaster),
		WithRecommendationWorkerConfig(RecommendationWorkerConfig{
			Interval:      100 * time.Millisecond,
			MaxRetries:    3,
			RetryBackoff:  10 * time.Millisecond,
			MaxConcurrent: 5,
		}),
	)

	// Create a complete investigation run with pending recommendation status
	now := time.Now()
	run := &domain.Run{
		ID:                     uuid.New(),
		TaskID:                 taskID,
		Tag:                    "agent-manager-investigation",
		Status:                 domain.RunStatusComplete,
		RecommendationStatus:   domain.RecommendationStatusPending,
		RecommendationQueuedAt: &now,
		Summary: &domain.RunSummary{
			Description: "Test investigation output with recommendations.",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := runRepo.Create(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}

	// Process the queue
	stats := worker.processQueue(ctx)

	if stats.RunsProcessed != 1 {
		t.Errorf("RunsProcessed = %d, want 1", stats.RunsProcessed)
	}
	if stats.ExtractionsSuccess != 1 {
		t.Errorf("ExtractionsSuccess = %d, want 1", stats.ExtractionsSuccess)
	}
	if len(stats.Errors) > 0 {
		t.Errorf("unexpected errors: %v", stats.Errors)
	}

	// Verify the run was updated
	updatedRun, err := runRepo.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Failed to get updated run: %v", err)
	}
	if updatedRun.RecommendationStatus != domain.RecommendationStatusComplete {
		t.Errorf("RecommendationStatus = %s, want %s",
			updatedRun.RecommendationStatus, domain.RecommendationStatusComplete)
	}
	if updatedRun.RecommendationResult == nil {
		t.Error("RecommendationResult should not be nil")
	}

	// Verify broadcast was sent
	broadcasts := broadcaster.getBroadcasts()
	if len(broadcasts) == 0 {
		t.Error("expected at least one broadcast")
	}
}

func TestRecommendationWorker_RetryOnFailure(t *testing.T) {
	ctx := context.Background()
	deps := setupWorkerRepos(t)
	runRepo, eventStore, taskID := deps.runRepo, deps.eventStore, deps.taskID

	// Create a mock extractor that fails
	callCount := 0
	extractor := &recommendation.MockExtractor{
		ExtractFunc: func(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error) {
			callCount++
			return &domain.ExtractionResult{
				Success: false,
				Error:   "mock extraction failure",
			}, nil
		},
		IsAvailableFunc: func(ctx context.Context) bool {
			return true
		},
	}

	worker := NewRecommendationWorker(
		runRepo,
		eventStore,
		extractor,
		WithRecommendationWorkerConfig(RecommendationWorkerConfig{
			Interval:      100 * time.Millisecond,
			MaxRetries:    3,
			RetryBackoff:  1 * time.Millisecond, // Very short for testing
			MaxConcurrent: 5,
		}),
	)

	// Create a complete investigation run
	now := time.Now()
	run := &domain.Run{
		ID:                     uuid.New(),
		TaskID:                 taskID,
		Tag:                    "agent-manager-investigation",
		Status:                 domain.RunStatusComplete,
		RecommendationStatus:   domain.RecommendationStatusPending,
		RecommendationQueuedAt: &now,
		Summary: &domain.RunSummary{
			Description: "Test investigation output.",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := runRepo.Create(ctx, run); err != nil {
		t.Fatalf("Failed to create run: %v", err)
	}

	// Process the queue - first attempt
	stats := worker.processQueue(ctx)
	if stats.ExtractionsRetried != 1 {
		t.Errorf("First attempt: ExtractionsRetried = %d, want 1", stats.ExtractionsRetried)
	}

	// Check the run was updated for retry
	updatedRun, _ := runRepo.Get(ctx, run.ID)
	if updatedRun.RecommendationAttempts != 1 {
		t.Errorf("RecommendationAttempts = %d, want 1", updatedRun.RecommendationAttempts)
	}
	if updatedRun.RecommendationStatus != domain.RecommendationStatusPending {
		t.Errorf("RecommendationStatus = %s, want %s",
			updatedRun.RecommendationStatus, domain.RecommendationStatusPending)
	}

	// Wait for backoff and process again
	time.Sleep(5 * time.Millisecond)
	worker.processQueue(ctx) // attempt 2
	time.Sleep(5 * time.Millisecond)
	stats = worker.processQueue(ctx) // attempt 3 - should mark as failed

	// Check final state
	finalRun, _ := runRepo.Get(ctx, run.ID)
	if finalRun.RecommendationStatus != domain.RecommendationStatusFailed {
		t.Errorf("Final RecommendationStatus = %s, want %s",
			finalRun.RecommendationStatus, domain.RecommendationStatusFailed)
	}
	if finalRun.RecommendationAttempts != 3 {
		t.Errorf("Final RecommendationAttempts = %d, want 3", finalRun.RecommendationAttempts)
	}
}

func TestRecommendationWorker_SkipsIneligibleRuns(t *testing.T) {
	ctx := context.Background()
	deps := setupWorkerRepos(t)
	runRepo, eventStore, taskID := deps.runRepo, deps.eventStore, deps.taskID

	extractor := &recommendation.MockExtractor{
		ExtractFunc: func(ctx context.Context, req domain.ExtractionRequest) (*domain.ExtractionResult, error) {
			return &domain.ExtractionResult{
				Success:    true,
				Categories: []domain.RecommendationCategory{},
			}, nil
		},
		IsAvailableFunc: func(ctx context.Context) bool {
			return true
		},
	}

	// Use a restrictive allowlist
	allowlist := &mockAllowlistProvider{
		rules: []domain.InvestigationTagRule{
			{Pattern: "special-investigation", IsRegex: false, CaseSensitive: false},
		},
	}

	worker := NewRecommendationWorker(
		runRepo,
		eventStore,
		extractor,
		WithRecommendationWorkerAllowlist(allowlist),
		WithRecommendationWorkerConfig(RecommendationWorkerConfig{
			Interval:      100 * time.Millisecond,
			MaxRetries:    3,
			RetryBackoff:  1 * time.Millisecond,
			MaxConcurrent: 10,
		}),
	)

	now := time.Now()

	// Create a run that matches the allowlist
	matchingRun := &domain.Run{
		ID:                     uuid.New(),
		TaskID:                 taskID,
		Tag:                    "special-investigation",
		Status:                 domain.RunStatusComplete,
		RecommendationStatus:   domain.RecommendationStatusPending,
		RecommendationQueuedAt: &now,
		Summary:                &domain.RunSummary{Description: "Test"},
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	_ = runRepo.Create(ctx, matchingRun)

	// Create a run that doesn't match the allowlist
	nonMatchingRun := &domain.Run{
		ID:                     uuid.New(),
		TaskID:                 taskID,
		Tag:                    "agent-manager-investigation", // Doesn't match "special-investigation"
		Status:                 domain.RunStatusComplete,
		RecommendationStatus:   domain.RecommendationStatusPending,
		RecommendationQueuedAt: &now,
		Summary:                &domain.RunSummary{Description: "Test"},
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	_ = runRepo.Create(ctx, nonMatchingRun)

	// Process the queue
	stats := worker.processQueue(ctx)

	// Only the matching run should be processed
	if stats.RunsProcessed != 1 {
		t.Errorf("RunsProcessed = %d, want 1 (only matching run)", stats.RunsProcessed)
	}

	// Verify matching run was updated
	updated, _ := runRepo.Get(ctx, matchingRun.ID)
	if updated.RecommendationStatus != domain.RecommendationStatusComplete {
		t.Errorf("Matching run status = %s, want complete", updated.RecommendationStatus)
	}

	// Verify non-matching run was not updated
	notUpdated, _ := runRepo.Get(ctx, nonMatchingRun.ID)
	if notUpdated.RecommendationStatus != domain.RecommendationStatusPending {
		t.Errorf("Non-matching run status = %s, want pending (unchanged)", notUpdated.RecommendationStatus)
	}
}

func TestSettingsAllowlistProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("returns settings allowlist", func(t *testing.T) {
		repos, _, cleanup := testutil.SetupTestRepos(t)
		t.Cleanup(cleanup)

		settingsRepo := repos.InvestigationSettings

		// Update settings with custom allowlist
		settings := domain.DefaultInvestigationSettings()
		settings.InvestigationTagAllowlist = []domain.InvestigationTagRule{
			{Pattern: "custom-*", IsRegex: false, CaseSensitive: false},
		}
		if err := settingsRepo.Update(ctx, settings); err != nil {
			t.Fatalf("Update settings failed: %v", err)
		}

		provider := NewSettingsAllowlistProvider(settingsRepo)
		rules := provider.GetAllowlist(ctx)

		if len(rules) != 1 {
			t.Errorf("len(rules) = %d, want 1", len(rules))
		}
		if len(rules) > 0 && rules[0].Pattern != "custom-*" {
			t.Errorf("rules[0].Pattern = %s, want custom-*", rules[0].Pattern)
		}
	})

	t.Run("returns defaults when repo is nil", func(t *testing.T) {
		provider := NewSettingsAllowlistProvider(nil)
		rules := provider.GetAllowlist(ctx)

		defaults := domain.DefaultInvestigationTagAllowlist()
		if len(rules) != len(defaults) {
			t.Errorf("len(rules) = %d, want %d (defaults)", len(rules), len(defaults))
		}
	})
}

func TestGetAllowlist_FallsBackToDefaults(t *testing.T) {
	ctx := context.Background()
	deps := setupWorkerRepos(t)
	runRepo, eventStore := deps.runRepo, deps.eventStore
	extractor := recommendation.NewMockExtractor()

	// Worker without allowlist provider
	worker := NewRecommendationWorker(runRepo, eventStore, extractor)

	rules := worker.getAllowlist(ctx)
	defaults := domain.DefaultInvestigationTagAllowlist()

	if len(rules) != len(defaults) {
		t.Errorf("len(rules) = %d, want %d (defaults)", len(rules), len(defaults))
	}
}

func TestRecommendationWorker_RecoverStaleExtractions(t *testing.T) {
	ctx := context.Background()
	deps := setupWorkerRepos(t)
	runRepo, eventStore, taskID := deps.runRepo, deps.eventStore, deps.taskID
	extractor := recommendation.NewMockExtractor()
	broadcaster := &mockBroadcaster{}

	// Use a very short stale timeout for testing
	worker := NewRecommendationWorker(
		runRepo,
		eventStore,
		extractor,
		WithRecommendationWorkerBroadcaster(broadcaster),
		WithRecommendationWorkerConfig(RecommendationWorkerConfig{
			Interval:          100 * time.Millisecond,
			MaxRetries:        3,
			RetryBackoff:      1 * time.Millisecond,
			MaxConcurrent:     5,
			StaleTimeout:      10 * time.Millisecond, // Very short for testing
			ExtractionTimeout: 1 * time.Minute,
		}),
	)

	// Create a run stuck in "extracting" status
	staleRun := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               taskID,
		Tag:                  "agent-manager-investigation",
		Status:               domain.RunStatusComplete,
		RecommendationStatus: domain.RecommendationStatusExtracting, // Stuck in extracting
		Summary:              &domain.RunSummary{Description: "Test"},
	}
	_ = runRepo.Create(ctx, staleRun)

	// Backdate updated_at via raw SQL (Create/Update auto-set time.Now())
	staleTime := time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339Nano)
	_, _ = deps.db.Exec("UPDATE runs SET updated_at = ? WHERE id = ?", staleTime, staleRun.ID)

	// Run recovery
	recovered := worker.recoverStaleExtractions(ctx)

	if recovered != 1 {
		t.Errorf("recovered = %d, want 1", recovered)
	}

	// Verify run was reset to pending
	updatedRun, _ := runRepo.Get(ctx, staleRun.ID)
	if updatedRun.RecommendationStatus != domain.RecommendationStatusPending {
		t.Errorf("RecommendationStatus = %s, want pending", updatedRun.RecommendationStatus)
	}
	if updatedRun.RecommendationError == "" {
		t.Error("expected RecommendationError to be set with recovery message")
	}

	// Verify broadcast was sent
	broadcasts := broadcaster.getBroadcasts()
	if len(broadcasts) == 0 {
		t.Error("expected broadcast for recovered run")
	}
}

func TestRecommendationWorker_DoesNotRecoverRecentExtractions(t *testing.T) {
	ctx := context.Background()
	deps := setupWorkerRepos(t)
	runRepo, eventStore, taskID := deps.runRepo, deps.eventStore, deps.taskID
	extractor := recommendation.NewMockExtractor()

	worker := NewRecommendationWorker(
		runRepo,
		eventStore,
		extractor,
		WithRecommendationWorkerConfig(RecommendationWorkerConfig{
			Interval:          100 * time.Millisecond,
			MaxRetries:        3,
			RetryBackoff:      1 * time.Millisecond,
			MaxConcurrent:     5,
			StaleTimeout:      5 * time.Minute, // 5 minute stale timeout
			ExtractionTimeout: 1 * time.Minute,
		}),
	)

	// Create a run in "extracting" status with a RECENT updated_at
	now := time.Now()
	activeRun := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               taskID,
		Tag:                  "agent-manager-investigation",
		Status:               domain.RunStatusComplete,
		RecommendationStatus: domain.RecommendationStatusExtracting, // Currently extracting
		Summary:              &domain.RunSummary{Description: "Test"},
		CreatedAt:            now,
		UpdatedAt:            now, // Just updated = not stale
	}
	_ = runRepo.Create(ctx, activeRun)

	// Run recovery - should NOT recover this run
	recovered := worker.recoverStaleExtractions(ctx)

	if recovered != 0 {
		t.Errorf("recovered = %d, want 0 (run is not stale)", recovered)
	}

	// Verify run is still extracting
	stillActiveRun, _ := runRepo.Get(ctx, activeRun.ID)
	if stillActiveRun.RecommendationStatus != domain.RecommendationStatusExtracting {
		t.Errorf("RecommendationStatus = %s, want extracting (should not be recovered)",
			stillActiveRun.RecommendationStatus)
	}
}
