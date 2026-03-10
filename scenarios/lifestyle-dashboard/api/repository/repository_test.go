package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"lifestyle-dashboard/domain"
)

// setupTestDB creates an in-memory SQLite database with schema for testing.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := domain.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	return db
}

// =============================================================================
// EventRepository Tests
// =============================================================================

// TestEventRepository_Create verifies event creation and ID generation.
// [REQ:LD-EVENT-STORAGE] Repository creates events with generated IDs.
func TestEventRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	event := &domain.Event{
		Domain:    "test-domain",
		EventType: "test.created",
		Payload:   json.RawMessage(`{"value": 42}`),
	}

	err := repo.Create(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	if event.ID == "" {
		t.Error("Expected event ID to be generated")
	}
	if event.CreatedAt == "" {
		t.Error("Expected CreatedAt to be set")
	}
	if event.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
}

// TestEventRepository_GetByID verifies event retrieval by ID.
// [REQ:LD-EVENT-STORAGE] Repository retrieves events by ID with full payload.
func TestEventRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	// Create an event
	event := &domain.Event{
		Domain:    "get-test",
		EventType: "event.retrieved",
		Payload:   json.RawMessage(`{"key": "value"}`),
	}
	repo.Create(ctx, event)

	// Retrieve it
	retrieved, err := repo.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("Failed to get event: %v", err)
	}

	if retrieved.Domain != "get-test" {
		t.Errorf("Expected domain 'get-test', got '%s'", retrieved.Domain)
	}
	if retrieved.EventType != "event.retrieved" {
		t.Errorf("Expected event_type 'event.retrieved', got '%s'", retrieved.EventType)
	}
}

// TestEventRepository_GetByID_NotFound verifies ErrNotFound for missing events.
// [REQ:LD-EVENT-STORAGE] Repository returns ErrNotFound for missing events.
func TestEventRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestEventRepository_List verifies event listing with filters.
// [REQ:LD-QUERY-FILTER] Repository supports domain and type filtering.
func TestEventRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	// Create test events
	repo.Create(ctx, &domain.Event{Domain: "domain-a", EventType: "type-1"})
	repo.Create(ctx, &domain.Event{Domain: "domain-a", EventType: "type-2"})
	repo.Create(ctx, &domain.Event{Domain: "domain-b", EventType: "type-1"})

	// Test domain filter
	events, err := repo.List(ctx, EventFilter{Domain: "domain-a"})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("Expected 2 events for domain-a, got %d", len(events))
	}

	// Test event_type filter
	events, err = repo.List(ctx, EventFilter{EventType: "type-1"})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("Expected 2 events for type-1, got %d", len(events))
	}

	// Test combined filter
	events, err = repo.List(ctx, EventFilter{Domain: "domain-a", EventType: "type-1"})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("Expected 1 event for domain-a+type-1, got %d", len(events))
	}
}

// TestEventRepository_List_Limit verifies limit parameter.
// [REQ:LD-QUERY-FILTER] Repository respects limit parameter.
func TestEventRepository_List_Limit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	// Create 5 events
	for i := 0; i < 5; i++ {
		repo.Create(ctx, &domain.Event{Domain: "limit-test", EventType: "event"})
	}

	// Verify limit works
	events, err := repo.List(ctx, EventFilter{Limit: 3})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("Expected 3 events with limit=3, got %d", len(events))
	}
}

// =============================================================================
// DomainRepository Tests
// =============================================================================

// TestDomainRepository_Upsert verifies domain creation.
// [REQ:LD-DOMAIN-REGISTER] Repository creates domains with timestamps.
func TestDomainRepository_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	d := &domain.Domain{
		Name:         "test-domain",
		DisplayName:  "Test Domain",
		Description:  "A test domain",
		Capabilities: []string{"events", "metrics"},
	}

	err := repo.Upsert(ctx, d)
	if err != nil {
		t.Fatalf("Failed to upsert domain: %v", err)
	}

	if d.RegisteredAt == "" {
		t.Error("Expected RegisteredAt to be set")
	}
	if d.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be set")
	}
	if d.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", d.Status)
	}
}

// TestDomainRepository_Upsert_Update verifies domain update on conflict.
// [REQ:LD-DOMAIN-REGISTER] Repository updates existing domains on conflict.
func TestDomainRepository_Upsert_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	// Create domain
	d := &domain.Domain{
		Name:        "update-test",
		DisplayName: "Original Name",
	}
	repo.Upsert(ctx, d)

	// Update it
	d2 := &domain.Domain{
		Name:        "update-test",
		DisplayName: "Updated Name",
	}
	repo.Upsert(ctx, d2)

	// Verify update
	retrieved, _ := repo.GetByName(ctx, "update-test")
	if retrieved.DisplayName != "Updated Name" {
		t.Errorf("Expected updated name, got '%s'", retrieved.DisplayName)
	}
}

// TestDomainRepository_GetByName verifies domain retrieval by name.
// [REQ:LD-DOMAIN-DISCOVER] Repository retrieves domains by name.
func TestDomainRepository_GetByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{
		Name:        "get-test",
		DisplayName: "Get Test",
	})

	d, err := repo.GetByName(ctx, "get-test")
	if err != nil {
		t.Fatalf("Failed to get domain: %v", err)
	}
	if d.DisplayName != "Get Test" {
		t.Errorf("Expected 'Get Test', got '%s'", d.DisplayName)
	}
}

// TestDomainRepository_GetByName_NotFound verifies ErrNotFound for missing domains.
// [REQ:LD-DOMAIN-DISCOVER] Repository returns ErrNotFound for missing domains.
func TestDomainRepository_GetByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "nonexistent")
	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestDomainRepository_List verifies domain listing.
// [REQ:LD-DOMAIN-DISCOVER] Repository lists all domains.
func TestDomainRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{Name: "domain-1", DisplayName: "D1"})
	repo.Upsert(ctx, &domain.Domain{Name: "domain-2", DisplayName: "D2"})
	repo.Upsert(ctx, &domain.Domain{Name: "domain-3", DisplayName: "D3"})

	domains, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list domains: %v", err)
	}
	if len(domains) != 3 {
		t.Errorf("Expected 3 domains, got %d", len(domains))
	}
}

// TestDomainRepository_UpdateStatus verifies status update.
// [REQ:LD-DOMAIN-HEALTH] Repository updates domain status.
func TestDomainRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{Name: "status-test", DisplayName: "Status"})

	err := repo.UpdateStatus(ctx, "status-test", "unhealthy", "2026-03-10T00:00:00Z")
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	d, _ := repo.GetByName(ctx, "status-test")
	if d.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", d.Status)
	}
}

// TestDomainRepository_Update verifies partial update.
// [REQ:LD-DOMAIN-REGISTER] Repository supports partial updates.
func TestDomainRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{
		Name:        "partial-update",
		DisplayName: "Original",
		Description: "Original Desc",
	})

	err := repo.Update(ctx, "partial-update", map[string]interface{}{
		"display_name": "Updated Display",
	})
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	d, _ := repo.GetByName(ctx, "partial-update")
	if d.DisplayName != "Updated Display" {
		t.Errorf("Expected 'Updated Display', got '%s'", d.DisplayName)
	}
	// Description should be unchanged
	if d.Description != "Original Desc" {
		t.Errorf("Description should be unchanged, got '%s'", d.Description)
	}
}

// =============================================================================
// StatsRepository Tests
// =============================================================================

// TestStatsRepository_GetTimeline verifies timeline aggregation.
// [REQ:LD-QUERY-AGGREGATE] Repository aggregates events by day and domain.
func TestStatsRepository_GetTimeline(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Create events
	eventRepo.Create(ctx, &domain.Event{Domain: "timeline-domain", EventType: "event.a"})
	eventRepo.Create(ctx, &domain.Event{Domain: "timeline-domain", EventType: "event.b"})
	eventRepo.Create(ctx, &domain.Event{Domain: "other-domain", EventType: "event.c"})

	timeline, err := statsRepo.GetTimeline(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	if len(timeline) == 0 {
		t.Error("Expected timeline entries")
	}
}

// TestStatsRepository_GetSummary verifies summary statistics.
// [REQ:LD-QUERY-AGGREGATE] Repository provides accurate summary stats.
func TestStatsRepository_GetSummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	domainRepo := NewSQLiteDomainRepository(db)
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Create domains and events
	domainRepo.Upsert(ctx, &domain.Domain{Name: "summary-dom", DisplayName: "Summary"})
	eventRepo.Create(ctx, &domain.Event{Domain: "summary-dom", EventType: "event.1"})
	eventRepo.Create(ctx, &domain.Event{Domain: "summary-dom", EventType: "event.2"})
	eventRepo.Create(ctx, &domain.Event{Domain: "summary-dom", EventType: "event.3"})

	summary, err := statsRepo.GetSummary(ctx)
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	if summary.TotalEvents != 3 {
		t.Errorf("Expected 3 total events, got %d", summary.TotalEvents)
	}
	if summary.ActiveDomains != 1 {
		t.Errorf("Expected 1 active domain, got %d", summary.ActiveDomains)
	}
}

// =============================================================================
// Error Type Tests
// =============================================================================

// TestIsNotFound verifies error type checking.
func TestIsNotFound(t *testing.T) {
	err := ErrNotFound{Entity: "test", ID: "123"}
	if !IsNotFound(err) {
		t.Error("Expected IsNotFound to return true")
	}

	otherErr := sql.ErrNoRows
	if IsNotFound(otherErr) {
		t.Error("Expected IsNotFound to return false for other errors")
	}
}
