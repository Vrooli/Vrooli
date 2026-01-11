package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/recommendation"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

func TestExtractRecommendations_ReturnsCachedResult(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run with cached recommendation result
	cachedResult := &domain.ExtractionResult{
		Success: true,
		Categories: []domain.RecommendationCategory{
			{ID: "cat-1", Name: "Cached", Recommendations: []domain.Recommendation{
				{ID: "rec-1", Text: "Cached recommendation", Selected: true},
			}},
		},
		ExtractedFrom: "summary",
	}

	now := time.Now()
	run := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               task.ID,
		Tag:                  "agent-manager-investigation",
		Status:               domain.RunStatusComplete,
		RecommendationStatus: domain.RecommendationStatusComplete,
		RecommendationResult: cachedResult,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_ = runRepo.Create(ctx, run)

	// Extract recommendations - should return cached result
	result, err := orch.ExtractRecommendations(ctx, run.ID)
	if err != nil {
		t.Fatalf("ExtractRecommendations failed: %v", err)
	}

	if !result.Success {
		t.Error("expected successful result from cache")
	}
	if len(result.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(result.Categories))
	}
	if result.Categories[0].Name != "Cached" {
		t.Errorf("expected category name 'Cached', got '%s'", result.Categories[0].Name)
	}
}

func TestExtractRecommendations_QueuesPendingExtraction(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()
	broadcaster := &mockBroadcaster{}

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
		WithBroadcaster(broadcaster),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run with no recommendation status (needs extraction)
	now := time.Now()
	run := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               task.ID,
		Tag:                  "agent-manager-investigation",
		Status:               domain.RunStatusComplete,
		RecommendationStatus: domain.RecommendationStatusNone, // Not yet extracted
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_ = runRepo.Create(ctx, run)

	// Extract recommendations - should queue and return pending
	result, err := orch.ExtractRecommendations(ctx, run.ID)
	if err != nil {
		t.Fatalf("ExtractRecommendations failed: %v", err)
	}

	// Should return pending status
	if result.Success {
		t.Error("expected unsuccessful result (pending)")
	}
	if result.ExtractedFrom != "pending" {
		t.Errorf("expected ExtractedFrom='pending', got '%s'", result.ExtractedFrom)
	}

	// Verify run was queued
	updatedRun, _ := runRepo.Get(ctx, run.ID)
	if updatedRun.RecommendationStatus != domain.RecommendationStatusPending {
		t.Errorf("expected status pending, got '%s'", updatedRun.RecommendationStatus)
	}
	if updatedRun.RecommendationQueuedAt == nil {
		t.Error("expected RecommendationQueuedAt to be set")
	}

	// Verify broadcast was sent
	broadcasts := broadcaster.getBroadcasts()
	if len(broadcasts) == 0 {
		t.Error("expected broadcast to be sent")
	}
}

func TestExtractRecommendations_ReturnsPendingForInProgressExtraction(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run with extracting status
	now := time.Now()
	run := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               task.ID,
		Tag:                  "agent-manager-investigation",
		Status:               domain.RunStatusComplete,
		RecommendationStatus: domain.RecommendationStatusExtracting,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_ = runRepo.Create(ctx, run)

	// Extract recommendations - should return pending
	result, err := orch.ExtractRecommendations(ctx, run.ID)
	if err != nil {
		t.Fatalf("ExtractRecommendations failed: %v", err)
	}

	if result.Success {
		t.Error("expected unsuccessful result (in progress)")
	}
	if result.ExtractedFrom != "pending" {
		t.Errorf("expected ExtractedFrom='pending', got '%s'", result.ExtractedFrom)
	}
}

func TestExtractRecommendations_ReturnsFailedWithRawText(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run with failed status
	now := time.Now()
	run := &domain.Run{
		ID:                   uuid.New(),
		TaskID:               task.ID,
		Tag:                  "agent-manager-investigation",
		Status:               domain.RunStatusComplete,
		RecommendationStatus: domain.RecommendationStatusFailed,
		RecommendationError:  "extraction failed after 3 retries",
		Summary:              &domain.RunSummary{Description: "Test summary text"},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_ = runRepo.Create(ctx, run)

	// Extract recommendations - should return failed with raw text
	result, err := orch.ExtractRecommendations(ctx, run.ID)
	if err != nil {
		t.Fatalf("ExtractRecommendations failed: %v", err)
	}

	if result.Success {
		t.Error("expected unsuccessful result (failed)")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
	if result.RawText == "" {
		t.Error("expected raw text for fallback")
	}
	if result.ExtractedFrom != "summary" {
		t.Errorf("expected ExtractedFrom='summary', got '%s'", result.ExtractedFrom)
	}
}

func TestRegenerateRecommendations_ResetsState(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()
	broadcaster := &mockBroadcaster{}

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
		WithBroadcaster(broadcaster),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run with existing recommendation result
	cachedResult := &domain.ExtractionResult{
		Success:    true,
		Categories: []domain.RecommendationCategory{},
	}

	now := time.Now()
	run := &domain.Run{
		ID:                     uuid.New(),
		TaskID:                 task.ID,
		Tag:                    "agent-manager-investigation",
		Status:                 domain.RunStatusComplete,
		RecommendationStatus:   domain.RecommendationStatusComplete,
		RecommendationResult:   cachedResult,
		RecommendationAttempts: 1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	_ = runRepo.Create(ctx, run)

	// Regenerate recommendations
	err := orch.RegenerateRecommendations(ctx, run.ID)
	if err != nil {
		t.Fatalf("RegenerateRecommendations failed: %v", err)
	}

	// Verify state was reset
	updatedRun, _ := runRepo.Get(ctx, run.ID)
	if updatedRun.RecommendationStatus != domain.RecommendationStatusPending {
		t.Errorf("expected status pending, got '%s'", updatedRun.RecommendationStatus)
	}
	if updatedRun.RecommendationResult != nil {
		t.Error("expected result to be cleared")
	}
	if updatedRun.RecommendationAttempts != 0 {
		t.Errorf("expected attempts=0, got %d", updatedRun.RecommendationAttempts)
	}
	if updatedRun.RecommendationQueuedAt == nil {
		t.Error("expected RecommendationQueuedAt to be set")
	}

	// Verify broadcast was sent
	broadcasts := broadcaster.getBroadcasts()
	if len(broadcasts) == 0 {
		t.Error("expected broadcast to be sent")
	}
}

func TestExtractRecommendations_RejectsNonInvestigationRun(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run with a non-investigation tag
	now := time.Now()
	run := &domain.Run{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Tag:       "some-other-task",
		Status:    domain.RunStatusComplete,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = runRepo.Create(ctx, run)

	// Try to extract recommendations - should fail
	_, err := orch.ExtractRecommendations(ctx, run.ID)
	if err == nil {
		t.Error("expected error for non-investigation run")
	}
}

func TestExtractRecommendations_RejectsIncompleteRun(t *testing.T) {
	ctx := context.Background()

	// Setup
	runRepo := repository.NewMemoryRunRepository()
	taskRepo := repository.NewMemoryTaskRepository()
	profileRepo := repository.NewMemoryProfileRepository()
	eventStore := event.NewMemoryStore()
	settingsRepo := repository.NewMemoryInvestigationSettingsRepository()

	extractor := recommendation.NewMockExtractor()

	orch := New(
		profileRepo,
		taskRepo,
		runRepo,
		WithEvents(eventStore),
		WithInvestigationSettings(settingsRepo),
		WithRecommendationExtractor(extractor),
	)

	// Create a task
	task := &domain.Task{ID: uuid.New(), Title: "Test"}
	_ = taskRepo.Create(ctx, task)

	// Create a run that is still running
	now := time.Now()
	run := &domain.Run{
		ID:        uuid.New(),
		TaskID:    task.ID,
		Tag:       "agent-manager-investigation",
		Status:    domain.RunStatusRunning, // Not complete
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = runRepo.Create(ctx, run)

	// Try to extract recommendations - should fail
	_, err := orch.ExtractRecommendations(ctx, run.ID)
	if err == nil {
		t.Error("expected error for incomplete run")
	}
}
