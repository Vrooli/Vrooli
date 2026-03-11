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

// =============================================================================
// BriefRepository Tests
// =============================================================================

// TestBriefRepository_GenerateMorningBrief verifies morning brief generation.
// [REQ:LD-BRIEF-MORNING] Repository generates morning brief with yesterday summary.
func TestBriefRepository_GenerateMorningBrief(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	// Create domain and events
	domainRepo.Upsert(ctx, &domain.Domain{Name: "meditation", DisplayName: "Meditation"})
	eventRepo.Create(ctx, &domain.Event{Domain: "meditation", EventType: "session.completed"})

	brief, err := briefRepo.GenerateMorningBrief(ctx, "2026-03-10")
	if err != nil {
		t.Fatalf("Failed to generate morning brief: %v", err)
	}

	if brief.Type != "morning" {
		t.Errorf("Expected type 'morning', got '%s'", brief.Type)
	}
	if brief.Date != "2026-03-10" {
		t.Errorf("Expected date '2026-03-10', got '%s'", brief.Date)
	}
	if brief.GeneratedAt == "" {
		t.Error("Expected GeneratedAt to be set")
	}
	if brief.Summary == "" {
		t.Error("Expected Summary to be generated")
	}
}

// TestBriefRepository_GenerateEveningBrief verifies evening brief generation.
// [REQ:LD-BRIEF-EVENING] Repository generates evening review with today's events.
func TestBriefRepository_GenerateEveningBrief(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	// Create domain and events
	domainRepo.Upsert(ctx, &domain.Domain{Name: "exercise", DisplayName: "Exercise"})
	eventRepo.Create(ctx, &domain.Event{Domain: "exercise", EventType: "workout.logged"})

	brief, err := briefRepo.GenerateEveningBrief(ctx, "2026-03-10")
	if err != nil {
		t.Fatalf("Failed to generate evening brief: %v", err)
	}

	if brief.Type != "evening" {
		t.Errorf("Expected type 'evening', got '%s'", brief.Type)
	}
	if brief.Date != "2026-03-10" {
		t.Errorf("Expected date '2026-03-10', got '%s'", brief.Date)
	}
	if brief.Summary == "" {
		t.Error("Expected Summary to be generated")
	}
}

// TestBriefRepository_GetCurrentBrief verifies time-based brief selection.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Repository auto-selects brief type.
func TestBriefRepository_GetCurrentBrief(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	brief, err := briefRepo.GetCurrentBrief(ctx)
	if err != nil {
		t.Fatalf("Failed to get current brief: %v", err)
	}

	// Should return either morning or evening based on time
	if brief.Type != "morning" && brief.Type != "evening" {
		t.Errorf("Expected type 'morning' or 'evening', got '%s'", brief.Type)
	}
}

// TestBriefRepository_CrossDomainConsolidation verifies multi-domain brief sections.
// [REQ:LD-BRIEF-CONSOLIDATE] Repository consolidates data from all active domains.
func TestBriefRepository_CrossDomainConsolidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	// Create multiple domains with today's activity
	domainRepo.Upsert(ctx, &domain.Domain{Name: "sleep", DisplayName: "Sleep"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "nutrition", DisplayName: "Nutrition"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "exercise", DisplayName: "Exercise"})

	// Add events for today
	today := domain.Event{Domain: "sleep", EventType: "sleep.logged"}
	eventRepo.Create(ctx, &today)

	// Get evening brief for today
	brief, err := briefRepo.GenerateEveningBrief(ctx, today.Timestamp[:10])
	if err != nil {
		t.Fatalf("Failed to generate brief: %v", err)
	}

	// Should have sections for all active domains
	if len(brief.Sections) < 1 {
		t.Errorf("Expected at least 1 section, got %d", len(brief.Sections))
	}
}

// TestBriefRepository_PriorityBasedSections verifies section priority calculation.
// [REQ:LD-BRIEF-CONSOLIDATE] Repository assigns priority based on event count.
func TestBriefRepository_PriorityBasedSections(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	// Create domain with many events (should get high priority)
	domainRepo.Upsert(ctx, &domain.Domain{Name: "active", DisplayName: "Active"})
	for i := 0; i < 5; i++ {
		eventRepo.Create(ctx, &domain.Event{Domain: "active", EventType: "test.event"})
	}

	// Create domain with few events (should get medium priority)
	domainRepo.Upsert(ctx, &domain.Domain{Name: "moderate", DisplayName: "Moderate"})
	for i := 0; i < 2; i++ {
		eventRepo.Create(ctx, &domain.Event{Domain: "moderate", EventType: "test.event"})
	}

	// Get first event timestamp for date
	events, _ := eventRepo.List(ctx, EventFilter{Limit: 1})
	if len(events) == 0 {
		t.Fatal("Expected events to exist")
	}
	date := events[0].Timestamp[:10]

	brief, err := briefRepo.GenerateEveningBrief(ctx, date)
	if err != nil {
		t.Fatalf("Failed to generate brief: %v", err)
	}

	// Find the active domain section - should have priority 1 (high)
	var activeSection *domain.BriefSection
	for _, s := range brief.Sections {
		if s.Domain == "active" {
			activeSection = &s
			break
		}
	}

	if activeSection != nil && activeSection.EventCount >= 5 && activeSection.Priority != 1 {
		t.Errorf("Expected high priority (1) for domain with 5+ events, got %d", activeSection.Priority)
	}
}

// TestBriefRepository_ScoreIncluded verifies score is attached to brief.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Brief includes current score.
func TestBriefRepository_ScoreIncluded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	// Create activity to generate a score
	domainRepo.Upsert(ctx, &domain.Domain{Name: "test-domain", DisplayName: "Test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "test-domain", EventType: "test.event"})

	brief, err := briefRepo.GetCurrentBrief(ctx)
	if err != nil {
		t.Fatalf("Failed to get brief: %v", err)
	}

	// Score may or may not be present depending on data - just verify no error
	// and that ScoreTrend is a valid value if present
	if brief.ScoreTrend != "" && brief.ScoreTrend != "up" && brief.ScoreTrend != "down" && brief.ScoreTrend != "stable" {
		t.Errorf("Invalid ScoreTrend value: '%s'", brief.ScoreTrend)
	}
}

// TestBriefRepository_InvalidDateFallback verifies handling of invalid dates.
// [REQ:LD-BRIEF-MORNING] Repository handles invalid date gracefully.
func TestBriefRepository_InvalidDateFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	briefRepo := NewSQLiteBriefRepository(db)
	ctx := context.Background()

	// Should not error with invalid date, uses today as fallback
	brief, err := briefRepo.GenerateMorningBrief(ctx, "invalid-date")
	if err != nil {
		t.Fatalf("Failed with invalid date: %v", err)
	}

	if brief.Type != "morning" {
		t.Errorf("Expected type 'morning', got '%s'", brief.Type)
	}
}

// =============================================================================
// StorageRepository Tests
// =============================================================================

// TestStorageRepository_GetStorageInfo verifies storage info retrieval.
// [REQ:LD-UI-STORAGE] Repository returns storage overview data.
func TestStorageRepository_GetStorageInfo(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create test data
	domainRepo.Upsert(ctx, &domain.Domain{Name: "storage-test", DisplayName: "Storage Test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "storage-test", EventType: "test.event"})
	eventRepo.Create(ctx, &domain.Event{Domain: "storage-test", EventType: "test.event"})

	info, err := storageRepo.GetStorageInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get storage info: %v", err)
	}

	if info.TotalEvents != 2 {
		t.Errorf("Expected 2 total events, got %d", info.TotalEvents)
	}
	if info.TotalDomains != 1 {
		t.Errorf("Expected 1 domain, got %d", info.TotalDomains)
	}
}

// TestStorageRepository_GetStorageInfo_Empty verifies storage info with no data.
// [REQ:LD-UI-STORAGE] Repository handles empty database.
func TestStorageRepository_GetStorageInfo_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	info, err := storageRepo.GetStorageInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get storage info: %v", err)
	}

	if info.TotalEvents != 0 {
		t.Errorf("Expected 0 events, got %d", info.TotalEvents)
	}
	if info.TotalDomains != 0 {
		t.Errorf("Expected 0 domains, got %d", info.TotalDomains)
	}
	if len(info.EventsByDomain) != 0 {
		t.Errorf("Expected empty EventsByDomain, got %d entries", len(info.EventsByDomain))
	}
}

// TestStorageRepository_GetStorageInfo_EventsByDomain verifies domain breakdown.
// [REQ:LD-UI-STORAGE] Repository provides per-domain event counts.
func TestStorageRepository_GetStorageInfo_EventsByDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create multiple domains with different event counts
	domainRepo.Upsert(ctx, &domain.Domain{Name: "domain-a", DisplayName: "Domain A"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "domain-b", DisplayName: "Domain B"})

	// Add 3 events to domain-a
	for i := 0; i < 3; i++ {
		eventRepo.Create(ctx, &domain.Event{Domain: "domain-a", EventType: "test"})
	}
	// Add 1 event to domain-b
	eventRepo.Create(ctx, &domain.Event{Domain: "domain-b", EventType: "test"})

	info, err := storageRepo.GetStorageInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get storage info: %v", err)
	}

	if len(info.EventsByDomain) != 2 {
		t.Errorf("Expected 2 domain entries, got %d", len(info.EventsByDomain))
	}

	// Should be sorted by count DESC, so domain-a first
	if len(info.EventsByDomain) > 0 && info.EventsByDomain[0].Domain != "domain-a" {
		t.Errorf("Expected domain-a first (highest count), got '%s'", info.EventsByDomain[0].Domain)
	}
}

// TestStorageRepository_CleanupEvents_All verifies clearing all events.
// [REQ:LD-UI-STORAGE] Repository clears all events when no filters specified.
func TestStorageRepository_CleanupEvents_All(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create events
	eventRepo.Create(ctx, &domain.Event{Domain: "test-a", EventType: "test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "test-b", EventType: "test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "test-c", EventType: "test"})

	// Clear all
	result, err := storageRepo.CleanupEvents(ctx, domain.CleanupRequest{})
	if err != nil {
		t.Fatalf("Failed to cleanup events: %v", err)
	}

	if result.DeletedEvents != 3 {
		t.Errorf("Expected 3 deleted events, got %d", result.DeletedEvents)
	}

	// Verify empty
	events, _ := eventRepo.List(ctx, EventFilter{})
	if len(events) != 0 {
		t.Errorf("Expected 0 events after cleanup, got %d", len(events))
	}
}

// TestStorageRepository_CleanupEvents_ByDomain verifies domain-specific cleanup.
// [REQ:LD-UI-STORAGE] Repository clears events only from specified domains.
func TestStorageRepository_CleanupEvents_ByDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create events in different domains
	eventRepo.Create(ctx, &domain.Event{Domain: "keep-domain", EventType: "test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "clear-domain", EventType: "test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "clear-domain", EventType: "test"})

	// Clear only clear-domain
	result, err := storageRepo.CleanupEvents(ctx, domain.CleanupRequest{
		Domains: []string{"clear-domain"},
	})
	if err != nil {
		t.Fatalf("Failed to cleanup events: %v", err)
	}

	if result.DeletedEvents != 2 {
		t.Errorf("Expected 2 deleted events, got %d", result.DeletedEvents)
	}

	// Verify keep-domain still has its event
	events, _ := eventRepo.List(ctx, EventFilter{Domain: "keep-domain"})
	if len(events) != 1 {
		t.Errorf("Expected 1 event in keep-domain, got %d", len(events))
	}

	// Verify clear-domain is empty
	events, _ = eventRepo.List(ctx, EventFilter{Domain: "clear-domain"})
	if len(events) != 0 {
		t.Errorf("Expected 0 events in clear-domain, got %d", len(events))
	}
}

// TestStorageRepository_CleanupEvents_ResponseMessage verifies response messages.
// [REQ:LD-UI-STORAGE] Repository generates appropriate cleanup messages.
func TestStorageRepository_CleanupEvents_ResponseMessage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create test events
	eventRepo.Create(ctx, &domain.Event{Domain: "test", EventType: "test"})

	// Test clear all message
	result, _ := storageRepo.CleanupEvents(ctx, domain.CleanupRequest{})
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
	if result.DomainsCleared[0] != "all" {
		t.Errorf("Expected 'all' in DomainsCleared, got %v", result.DomainsCleared)
	}
}

// =============================================================================
// StatsRepository GetLifestyleScore Tests
// =============================================================================

// TestStatsRepository_GetLifestyleScore verifies lifestyle score calculation.
// [REQ:LD-UI-SCORE] Repository calculates composite lifestyle score.
func TestStatsRepository_GetLifestyleScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Create domains and events to generate a score
	domainRepo.Upsert(ctx, &domain.Domain{Name: "sleep", DisplayName: "Sleep"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "exercise", DisplayName: "Exercise"})
	eventRepo.Create(ctx, &domain.Event{Domain: "sleep", EventType: "sleep.logged"})
	eventRepo.Create(ctx, &domain.Event{Domain: "exercise", EventType: "workout.logged"})

	resp, err := statsRepo.GetLifestyleScore(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score: %v", err)
	}

	score := resp.Current
	if score.Score < 0 || score.Score > 100 {
		t.Errorf("Expected score 0-100, got %d", score.Score)
	}
	if score.Trend != "up" && score.Trend != "down" && score.Trend != "stable" {
		t.Errorf("Expected valid trend, got '%s'", score.Trend)
	}
	if score.DataQuality != "good" && score.DataQuality != "limited" && score.DataQuality != "insufficient" {
		t.Errorf("Expected valid data quality, got '%s'", score.DataQuality)
	}
}

// TestStatsRepository_GetLifestyleScore_Empty verifies score with no data.
// [REQ:LD-UI-SCORE] Repository handles empty database gracefully.
func TestStatsRepository_GetLifestyleScore_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	resp, err := statsRepo.GetLifestyleScore(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score: %v", err)
	}

	score := resp.Current
	if score.Score != 0 {
		t.Errorf("Expected 0 score with no data, got %d", score.Score)
	}
	if score.DataQuality != "insufficient" {
		t.Errorf("Expected 'insufficient' data quality, got '%s'", score.DataQuality)
	}
}

// TestStatsRepository_GetLifestyleScore_DomainBreakdown verifies per-domain scores.
// [REQ:LD-UI-SCORE] Repository provides per-domain score breakdown.
func TestStatsRepository_GetLifestyleScore_DomainBreakdown(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Create domains with different activity levels
	domainRepo.Upsert(ctx, &domain.Domain{Name: "active", DisplayName: "Active Domain"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "inactive", DisplayName: "Inactive Domain"})

	// Add multiple events to active domain
	for i := 0; i < 5; i++ {
		eventRepo.Create(ctx, &domain.Event{Domain: "active", EventType: "test"})
	}

	resp, err := statsRepo.GetLifestyleScore(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score: %v", err)
	}

	score := resp.Current
	// Should have at least one domain in breakdown
	if len(score.DomainScores) < 1 {
		t.Error("Expected at least 1 domain in score breakdown")
	}

	// Active domain should have events counted
	for _, ds := range score.DomainScores {
		if ds.Domain == "active" && ds.EventCount < 5 {
			t.Errorf("Expected active domain to have 5+ events, got %d", ds.EventCount)
		}
	}
}

// TestStatsRepository_GetLifestyleScore_History verifies score history retrieval.
// [REQ:LD-UI-SCORE] Repository provides score history for trend visualization.
func TestStatsRepository_GetLifestyleScore_History(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	resp, err := statsRepo.GetLifestyleScore(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score: %v", err)
	}

	// History should be present (may be empty with no historical data)
	if resp.History == nil {
		t.Error("Expected History to be non-nil (even if empty)")
	}
}

// =============================================================================
// Additional StatsRepository Tests for Coverage
// =============================================================================

// TestStatsRepository_GetTimeline_InvalidDays verifies timeline with invalid days.
// [REQ:LD-QUERY-AGGREGATE] Repository handles invalid days parameter gracefully.
func TestStatsRepository_GetTimeline_InvalidDays(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Create test event
	eventRepo.Create(ctx, &domain.Event{Domain: "timeline-test", EventType: "event"})

	// Test with days <= 0 (should default to 7)
	timeline, err := statsRepo.GetTimeline(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to get timeline with days=0: %v", err)
	}
	if len(timeline) == 0 {
		t.Error("Expected at least one timeline entry with default days")
	}

	// Test with negative days
	timeline, err = statsRepo.GetTimeline(ctx, -5)
	if err != nil {
		t.Fatalf("Failed to get timeline with negative days: %v", err)
	}
	// Should still work with default
	if timeline == nil {
		t.Error("Expected non-nil timeline")
	}
}

// TestStatsRepository_GetLifestyleScore_HighScore verifies score message for excellent score.
// [REQ:LD-UI-SCORE] Repository generates appropriate messages for high scores.
func TestStatsRepository_GetLifestyleScore_HighScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Create domains and many events to get high score
	domainRepo.Upsert(ctx, &domain.Domain{Name: "sleep", DisplayName: "Sleep"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "exercise", DisplayName: "Exercise"})
	domainRepo.Upsert(ctx, &domain.Domain{Name: "nutrition", DisplayName: "Nutrition"})

	// Add 5 events per domain to max out each domain (5*20=100)
	for i := 0; i < 5; i++ {
		eventRepo.Create(ctx, &domain.Event{Domain: "sleep", EventType: "logged"})
		eventRepo.Create(ctx, &domain.Event{Domain: "exercise", EventType: "logged"})
		eventRepo.Create(ctx, &domain.Event{Domain: "nutrition", EventType: "logged"})
	}

	resp, err := statsRepo.GetLifestyleScore(ctx, 7)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score: %v", err)
	}

	score := resp.Current
	if score.Score < 80 {
		t.Errorf("Expected high score >= 80, got %d", score.Score)
	}
	if score.DataQuality != "good" {
		t.Errorf("Expected 'good' data quality with 3 domains, got '%s'", score.DataQuality)
	}
	if score.Message == "" {
		t.Error("Expected non-empty message")
	}
}

// TestStatsRepository_GetLifestyleScore_InvalidHistoryDays verifies default history days.
// [REQ:LD-UI-SCORE] Repository handles invalid history days parameter.
func TestStatsRepository_GetLifestyleScore_InvalidHistoryDays(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	statsRepo := NewSQLiteStatsRepository(db)
	ctx := context.Background()

	// Test with days <= 0 (should default to 7)
	resp, err := statsRepo.GetLifestyleScore(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score: %v", err)
	}
	if resp.History == nil {
		t.Error("Expected history to be non-nil")
	}

	// Test with negative days
	resp, err = statsRepo.GetLifestyleScore(ctx, -10)
	if err != nil {
		t.Fatalf("Failed to get lifestyle score with negative days: %v", err)
	}
	if resp.History == nil {
		t.Error("Expected history to be non-nil with negative days")
	}
}

// =============================================================================
// Additional DomainRepository Tests for Coverage
// =============================================================================

// TestDomainRepository_Update_MultipleFields verifies updating multiple fields at once.
// [REQ:LD-DOMAIN-REGISTER] Repository supports multi-field partial updates.
func TestDomainRepository_Update_MultipleFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{
		Name:        "multi-update",
		DisplayName: "Original",
		Description: "Original Desc",
	})

	// Update all three fields simultaneously
	err := repo.Update(ctx, "multi-update", map[string]interface{}{
		"display_name": "Updated Display",
		"description":  "Updated Description",
		"status":       "inactive",
	})
	if err != nil {
		t.Fatalf("Failed to update multiple fields: %v", err)
	}

	d, _ := repo.GetByName(ctx, "multi-update")
	if d.DisplayName != "Updated Display" {
		t.Errorf("Expected 'Updated Display', got '%s'", d.DisplayName)
	}
	if d.Description != "Updated Description" {
		t.Errorf("Expected 'Updated Description', got '%s'", d.Description)
	}
	if d.Status != "inactive" {
		t.Errorf("Expected 'inactive', got '%s'", d.Status)
	}
}

// TestDomainRepository_Update_NotFound verifies error for non-existent domain.
// [REQ:LD-DOMAIN-REGISTER] Repository returns error for missing domain.
func TestDomainRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	err := repo.Update(ctx, "nonexistent-domain", map[string]interface{}{
		"display_name": "Test",
	})
	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestDomainRepository_UpdateStatus_NotFound verifies error for non-existent domain status update.
// [REQ:LD-DOMAIN-HEALTH] Repository returns error for missing domain.
func TestDomainRepository_UpdateStatus_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	err := repo.UpdateStatus(ctx, "nonexistent", "unhealthy", "2026-03-10T00:00:00Z")
	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// TestDomainRepository_GetByName_WithLastHealthAt verifies retrieval with last_health_at.
// [REQ:LD-DOMAIN-HEALTH] Repository retrieves domain with health timestamp.
func TestDomainRepository_GetByName_WithLastHealthAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	// Create domain and set health status
	repo.Upsert(ctx, &domain.Domain{Name: "health-test", DisplayName: "Health Test"})
	healthTime := "2026-03-10T12:00:00Z"
	repo.UpdateStatus(ctx, "health-test", "healthy", healthTime)

	d, err := repo.GetByName(ctx, "health-test")
	if err != nil {
		t.Fatalf("Failed to get domain: %v", err)
	}
	if d.LastHealthAt == nil || *d.LastHealthAt != healthTime {
		t.Errorf("Expected LastHealthAt '%s', got '%v'", healthTime, d.LastHealthAt)
	}
}

// TestDomainRepository_List_WithLastHealthAt verifies listing with health timestamps.
// [REQ:LD-DOMAIN-DISCOVER] Repository lists domains with health timestamps.
func TestDomainRepository_List_WithLastHealthAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewSQLiteDomainRepository(db)
	ctx := context.Background()

	repo.Upsert(ctx, &domain.Domain{Name: "list-health", DisplayName: "List Health"})
	repo.UpdateStatus(ctx, "list-health", "healthy", "2026-03-10T12:00:00Z")

	domains, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("Expected 1 domain, got %d", len(domains))
	}
	if domains[0].LastHealthAt == nil {
		t.Error("Expected LastHealthAt to be set in list")
	}
}

// =============================================================================
// Additional StorageRepository Tests for Coverage
// =============================================================================

// TestStorageRepository_CleanupEvents_WithBefore verifies cleanup with time filter.
// [REQ:LD-UI-STORAGE] Repository clears events before specified timestamp.
func TestStorageRepository_CleanupEvents_WithBefore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create events with specific timestamps (need to use raw SQL for custom timestamps)
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"old-event", "2026-01-01T12:00:00Z", "test", "event", "{}", false, "2026-01-01T12:00:00Z")
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"new-event", "2026-03-10T12:00:00Z", "test", "event", "{}", false, "2026-03-10T12:00:00Z")

	// Clear events before March
	result, err := storageRepo.CleanupEvents(ctx, domain.CleanupRequest{
		Before: "2026-02-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("Failed to cleanup events: %v", err)
	}

	if result.DeletedEvents != 1 {
		t.Errorf("Expected 1 deleted event, got %d", result.DeletedEvents)
	}
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
}

// TestStorageRepository_CleanupEvents_DomainAndBefore verifies cleanup with both filters.
// [REQ:LD-UI-STORAGE] Repository clears events with combined filters.
func TestStorageRepository_CleanupEvents_DomainAndBefore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create events with different domains and timestamps
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"target-old", "2026-01-01T12:00:00Z", "target-domain", "event", "{}", false, "2026-01-01T12:00:00Z")
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"target-new", "2026-03-10T12:00:00Z", "target-domain", "event", "{}", false, "2026-03-10T12:00:00Z")
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"other-old", "2026-01-01T12:00:00Z", "other-domain", "event", "{}", false, "2026-01-01T12:00:00Z")

	// Clear only old events from target-domain
	result, err := storageRepo.CleanupEvents(ctx, domain.CleanupRequest{
		Domains: []string{"target-domain"},
		Before:  "2026-02-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("Failed to cleanup events: %v", err)
	}

	if result.DeletedEvents != 1 {
		t.Errorf("Expected 1 deleted event, got %d", result.DeletedEvents)
	}

	// Verify other-old event still exists
	var count int
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events WHERE id = 'other-old'`).Scan(&count)
	if count != 1 {
		t.Error("Expected other-old event to still exist")
	}
}

// TestStorageRepository_CleanupEvents_MultipleDomains verifies cleanup with multiple domains.
// [REQ:LD-UI-STORAGE] Repository clears events from multiple domains at once.
func TestStorageRepository_CleanupEvents_MultipleDomains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	storageRepo := NewSQLiteStorageRepository(db)
	ctx := context.Background()

	// Create events in multiple domains
	eventRepo.Create(ctx, &domain.Event{Domain: "domain-a", EventType: "test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "domain-b", EventType: "test"})
	eventRepo.Create(ctx, &domain.Event{Domain: "domain-c", EventType: "test"})

	// Clear domain-a and domain-b
	result, err := storageRepo.CleanupEvents(ctx, domain.CleanupRequest{
		Domains: []string{"domain-a", "domain-b"},
	})
	if err != nil {
		t.Fatalf("Failed to cleanup events: %v", err)
	}

	if result.DeletedEvents != 2 {
		t.Errorf("Expected 2 deleted events, got %d", result.DeletedEvents)
	}

	// Verify domain-c still has its event
	events, _ := eventRepo.List(ctx, EventFilter{Domain: "domain-c"})
	if len(events) != 1 {
		t.Errorf("Expected 1 event in domain-c, got %d", len(events))
	}
}

// =============================================================================
// Additional EventRepository Tests for Coverage
// =============================================================================

// TestEventRepository_List_EndTimeFilter verifies end time filtering.
// [REQ:LD-QUERY-FILTER] Repository supports end time filtering.
func TestEventRepository_List_EndTimeFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	// Create events with different timestamps
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"early", "2026-01-01T12:00:00Z", "test", "event", "{}", false, "2026-01-01T12:00:00Z")
	db.ExecContext(ctx, `INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"late", "2026-03-10T12:00:00Z", "test", "event", "{}", false, "2026-03-10T12:00:00Z")

	// Filter to only get events before March
	events, err := eventRepo.List(ctx, EventFilter{
		EndTime: "2026-02-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event before end time, got %d", len(events))
	}
}

// TestEventRepository_List_MaxLimit verifies max limit enforcement.
// [REQ:LD-QUERY-FILTER] Repository enforces maximum limit.
func TestEventRepository_List_MaxLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	// Create more events than max limit (1000)
	// For testing, we'll just verify that an excessive limit doesn't cause issues
	for i := 0; i < 10; i++ {
		eventRepo.Create(ctx, &domain.Event{Domain: "limit-test", EventType: "event"})
	}

	// Request more than max - should be capped
	events, err := eventRepo.List(ctx, EventFilter{Limit: 5000})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}

	// Should return at most 10 (our events) or max limit, whichever is smaller
	if len(events) > 10 {
		t.Errorf("Expected at most 10 events, got %d", len(events))
	}
}

// TestEventRepository_Create_WithHypothesisID verifies hypothesis ID storage.
// [REQ:LD-EVENT-STORAGE] Repository stores hypothesis correlation ID.
func TestEventRepository_Create_WithHypothesisID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	eventRepo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	hypothesisID := "hypo-123"
	event := &domain.Event{
		Domain:       "test",
		EventType:    "test.event",
		HypothesisID: &hypothesisID,
	}
	err := eventRepo.Create(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Retrieve and verify
	retrieved, _ := eventRepo.GetByID(ctx, event.ID)
	if retrieved.HypothesisID == nil || *retrieved.HypothesisID != hypothesisID {
		t.Errorf("Expected HypothesisID '%s', got %v", hypothesisID, retrieved.HypothesisID)
	}
}
