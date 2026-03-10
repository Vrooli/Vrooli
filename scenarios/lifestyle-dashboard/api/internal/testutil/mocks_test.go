package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/repository"
)

// =============================================================================
// MockEventRepository Tests
// =============================================================================

// TestMockEventRepository_Create verifies event creation.
func TestMockEventRepository_Create(t *testing.T) {
	repo := NewMockEventRepository()
	ctx := context.Background()

	event := &domain.Event{
		Domain:    "test-domain",
		EventType: "test.created",
		Payload:   json.RawMessage(`{"key": "value"}`),
	}

	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if event.ID == "" {
		t.Error("Expected ID to be generated")
	}
	if event.CreatedAt == "" {
		t.Error("Expected CreatedAt to be set")
	}
	if event.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
}

// TestMockEventRepository_GetByID verifies event retrieval.
func TestMockEventRepository_GetByID(t *testing.T) {
	repo := NewMockEventRepository()
	ctx := context.Background()

	// Create event
	event := &domain.Event{
		Domain:    "test-domain",
		EventType: "test.event",
	}
	repo.Create(ctx, event)

	// Retrieve it
	retrieved, err := repo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Domain != "test-domain" {
		t.Errorf("Expected domain 'test-domain', got '%s'", retrieved.Domain)
	}
}

// TestMockEventRepository_GetByID_NotFound verifies ErrNotFound.
func TestMockEventRepository_GetByID_NotFound(t *testing.T) {
	repo := NewMockEventRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if !repository.IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestMockEventRepository_List verifies event listing with filter.
func TestMockEventRepository_List(t *testing.T) {
	repo := NewMockEventRepository()
	ctx := context.Background()

	// Create events
	repo.Create(ctx, &domain.Event{Domain: "domain-a", EventType: "type-1"})
	repo.Create(ctx, &domain.Event{Domain: "domain-a", EventType: "type-2"})
	repo.Create(ctx, &domain.Event{Domain: "domain-b", EventType: "type-1"})

	// Test domain filter
	events, err := repo.List(ctx, repository.EventFilter{Domain: "domain-a"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("Expected 2 events for domain-a, got %d", len(events))
	}
}

// TestMockEventRepository_WithCreateError verifies error injection.
func TestMockEventRepository_WithCreateError(t *testing.T) {
	expectedErr := errors.New("database error")
	repo := NewMockEventRepository().WithCreateError(expectedErr)
	ctx := context.Background()

	err := repo.Create(ctx, &domain.Event{Domain: "test"})
	if err != expectedErr {
		t.Errorf("Expected injected error, got %v", err)
	}
}

// TestMockEventRepository_WithEvent verifies pre-population.
func TestMockEventRepository_WithEvent(t *testing.T) {
	event := &domain.Event{
		ID:        "preset-id",
		Domain:    "preset-domain",
		EventType: "preset.event",
	}
	repo := NewMockEventRepository().WithEvent(event)
	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, "preset-id")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.Domain != "preset-domain" {
		t.Errorf("Expected preset domain, got '%s'", retrieved.Domain)
	}
}

// =============================================================================
// MockDomainRepository Tests
// =============================================================================

// TestMockDomainRepository_Upsert verifies domain creation.
func TestMockDomainRepository_Upsert(t *testing.T) {
	repo := NewMockDomainRepository()
	ctx := context.Background()

	d := &domain.Domain{
		Name:        "test-domain",
		DisplayName: "Test Domain",
	}

	err := repo.Upsert(ctx, d)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if d.RegisteredAt == "" {
		t.Error("Expected RegisteredAt to be set")
	}
	if d.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", d.Status)
	}
}

// TestMockDomainRepository_GetByName verifies domain retrieval.
func TestMockDomainRepository_GetByName(t *testing.T) {
	repo := NewMockDomainRepository()
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{Name: "get-test", DisplayName: "Get Test"})

	d, err := repo.GetByName(ctx, "get-test")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if d.DisplayName != "Get Test" {
		t.Errorf("Expected 'Get Test', got '%s'", d.DisplayName)
	}
}

// TestMockDomainRepository_GetByName_NotFound verifies ErrNotFound.
func TestMockDomainRepository_GetByName_NotFound(t *testing.T) {
	repo := NewMockDomainRepository()
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "nonexistent")
	if !repository.IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestMockDomainRepository_List verifies domain listing.
func TestMockDomainRepository_List(t *testing.T) {
	repo := NewMockDomainRepository()
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{Name: "d1"})
	repo.Upsert(ctx, &domain.Domain{Name: "d2"})

	domains, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(domains))
	}
}

// TestMockDomainRepository_UpdateStatus verifies status update.
func TestMockDomainRepository_UpdateStatus(t *testing.T) {
	repo := NewMockDomainRepository()
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{Name: "status-test"})

	err := repo.UpdateStatus(ctx, "status-test", "unhealthy", "2026-03-10T00:00:00Z")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	d, _ := repo.GetByName(ctx, "status-test")
	if d.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", d.Status)
	}
}

// =============================================================================
// MockStatsRepository Tests
// =============================================================================

// TestMockStatsRepository_GetTimeline verifies timeline retrieval.
func TestMockStatsRepository_GetTimeline(t *testing.T) {
	repo := NewMockStatsRepository().WithTimeline([]domain.TimelineEntry{
		{Day: "2026-03-10", Domain: "test", Count: 5},
	})
	ctx := context.Background()

	timeline, err := repo.GetTimeline(ctx, 7)
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}
	if len(timeline) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(timeline))
	}
}

// TestMockStatsRepository_GetSummary verifies summary retrieval.
func TestMockStatsRepository_GetSummary(t *testing.T) {
	repo := NewMockStatsRepository().WithSummary(&domain.SummaryResponse{
		TotalEvents:   100,
		ActiveDomains: 5,
	})
	ctx := context.Background()

	summary, err := repo.GetSummary(ctx)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}
	if summary.TotalEvents != 100 {
		t.Errorf("Expected 100 events, got %d", summary.TotalEvents)
	}
}
