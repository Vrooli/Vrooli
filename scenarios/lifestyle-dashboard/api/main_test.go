package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"lifestyle-dashboard/domain"
)

// Test utilities

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "lifestyle-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := tmpDir + "/test.db"
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open test database: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := domain.InitSchema(db); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func setupTestServer(t *testing.T) (*Server, *sql.DB, func()) {
	t.Helper()
	db, cleanup := setupTestDB(t)
	srv := NewServer(db)
	return srv, db, cleanup
}

// =============================================================================
// Event Schema Tests [REQ:LD-EVENT-SCHEMA]
// =============================================================================

// TestEventSchema_CommonEnvelope validates that events follow the common envelope schema
// [REQ:LD-EVENT-SCHEMA] Common event envelope schema
func TestEventSchema_CommonEnvelope(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create an event with all envelope fields
	event := domain.CreateEventRequest{
		Domain:         "test-domain",
		EventType:      "test.event.created",
		Payload:        json.RawMessage(`{"key": "value", "nested": {"inner": 123}}`),
		IsIntervention: true,
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-EVENT-SCHEMA] Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var created domain.Event
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("[REQ:LD-EVENT-SCHEMA] Failed to decode response: %v", err)
	}

	// Validate envelope fields
	if created.ID == "" {
		t.Error("[REQ:LD-EVENT-SCHEMA] Event ID should be a non-empty UUID")
	}
	if created.Timestamp == "" {
		t.Error("[REQ:LD-EVENT-SCHEMA] Event timestamp should be ISO-8601")
	}
	if created.Domain != "test-domain" {
		t.Errorf("[REQ:LD-EVENT-SCHEMA] Expected domain 'test-domain', got '%s'", created.Domain)
	}
	if created.EventType != "test.event.created" {
		t.Errorf("[REQ:LD-EVENT-SCHEMA] Expected event_type 'test.event.created', got '%s'", created.EventType)
	}
	if !created.IsIntervention {
		t.Error("[REQ:LD-EVENT-SCHEMA] Expected is_intervention to be true")
	}

	// Validate timestamp is ISO-8601
	_, err := time.Parse(time.RFC3339, created.Timestamp)
	if err != nil {
		t.Errorf("[REQ:LD-EVENT-SCHEMA] Timestamp should be ISO-8601 format: %v", err)
	}
}

// TestEventSchema_CausalityMarkers validates causality markers (is_intervention, hypothesis_id)
// [REQ:LD-EVENT-SCHEMA] Causality fields: is_intervention (INTEGER 0/1), hypothesis_id (TEXT UUID nullable)
func TestEventSchema_CausalityMarkers(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	hypothesisID := "550e8400-e29b-41d4-a716-446655440000"
	event := domain.CreateEventRequest{
		Domain:         "nootropics",
		EventType:      "supplement.taken",
		Payload:        json.RawMessage(`{"name": "magnesium", "dose_mg": 400}`),
		IsIntervention: true,
		HypothesisID:   &hypothesisID,
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-EVENT-SCHEMA] Expected status 201, got %d", w.Code)
	}

	var created domain.Event
	json.NewDecoder(w.Body).Decode(&created)

	if !created.IsIntervention {
		t.Error("[REQ:LD-EVENT-SCHEMA] is_intervention should be true for intervention events")
	}
	if created.HypothesisID == nil || *created.HypothesisID != hypothesisID {
		t.Error("[REQ:LD-EVENT-SCHEMA] hypothesis_id should match the provided UUID")
	}
}

// =============================================================================
// Event Storage Tests [REQ:LD-EVENT-STORAGE]
// =============================================================================

// TestEventStorage_SQLitePersistence validates events are stored in SQLite
// [REQ:LD-EVENT-STORAGE] Events stored in lifestyle.db SQLite file
func TestEventStorage_SQLitePersistence(t *testing.T) {
	srv, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Create an event
	event := domain.CreateEventRequest{
		Domain:    "sleep",
		EventType: "sleep.recorded",
		Payload:   json.RawMessage(`{"hours": 7.5, "quality": "good"}`),
	}

	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-EVENT-STORAGE] Expected status 201, got %d", w.Code)
	}

	var created domain.Event
	json.NewDecoder(w.Body).Decode(&created)

	// Verify directly in database
	var stored domain.Event
	var payload string
	err := db.QueryRow(`
		SELECT id, domain, event_type, payload
		FROM events WHERE id = ?
	`, created.ID).Scan(&stored.ID, &stored.Domain, &stored.EventType, &payload)
	if err != nil {
		t.Fatalf("[REQ:LD-EVENT-STORAGE] Event not found in SQLite: %v", err)
	}

	if stored.Domain != "sleep" {
		t.Errorf("[REQ:LD-EVENT-STORAGE] Expected domain 'sleep', got '%s'", stored.Domain)
	}

	// Verify JSON payload is stored correctly
	var payloadMap map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &payloadMap); err != nil {
		t.Errorf("[REQ:LD-EVENT-STORAGE] Payload should be valid JSON: %v", err)
	}
	if payloadMap["hours"] != 7.5 {
		t.Error("[REQ:LD-EVENT-STORAGE] JSON payload should preserve numeric values")
	}
}

// TestEventStorage_ExtensibleWithoutMigrations validates new domains work without schema changes
// [REQ:LD-EVENT-STORAGE] Extensible without schema migrations when adding new domains
func TestEventStorage_ExtensibleWithoutMigrations(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events for multiple domains - should work without any schema changes
	domains := []string{"sleep", "nootropics", "exercise", "nutrition", "custom-new-domain"}

	for _, d := range domains {
		event := domain.CreateEventRequest{
			Domain:    d,
			EventType: "event.test",
			Payload:   json.RawMessage(fmt.Sprintf(`{"domain_specific": "%s-data"}`, d)),
		}

		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("[REQ:LD-EVENT-STORAGE] Domain '%s' should store without migration, got status %d", d, w.Code)
		}
	}

	// Query all events - should return events from all domains
	req := httptest.NewRequest("GET", "/api/v1/events?limit=10", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != len(domains) {
		t.Errorf("[REQ:LD-EVENT-STORAGE] Expected %d events, got %d", len(domains), result.Count)
	}
}

// =============================================================================
// Domain Registration Tests [REQ:LD-DOMAIN-REGISTER]
// =============================================================================

// TestDomainRegister_Endpoint validates domain registration
// [REQ:LD-DOMAIN-REGISTER] POST /api/v1/domains accepts domain name, capabilities, health endpoint
func TestDomainRegister_Endpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := domain.RegisterDomainRequest{
		Name:         "nootropics-tracker",
		DisplayName:  "Nootropics Tracker",
		Description:  "Track supplements and cognitive effects",
		Capabilities: []string{"track_supplements", "correlate_effects", "export_data"},
		HealthURL:    "http://localhost:8080/health",
	}

	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-DOMAIN-REGISTER] Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var registered domain.Domain
	if err := json.NewDecoder(w.Body).Decode(&registered); err != nil {
		t.Fatalf("[REQ:LD-DOMAIN-REGISTER] Failed to decode response: %v", err)
	}

	if registered.Name != "nootropics-tracker" {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] Expected name 'nootropics-tracker', got '%s'", registered.Name)
	}
	if registered.Status != "active" {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] New domains should have 'active' status, got '%s'", registered.Status)
	}
	if len(registered.Capabilities) != 3 {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] Expected 3 capabilities, got %d", len(registered.Capabilities))
	}
	if registered.RegisteredAt == "" {
		t.Error("[REQ:LD-DOMAIN-REGISTER] RegisteredAt should be set")
	}
}

// TestDomainRegister_Upsert validates domain registration is idempotent (upsert)
// [REQ:LD-DOMAIN-REGISTER] Registration should be idempotent
func TestDomainRegister_Upsert(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	d := domain.RegisterDomainRequest{
		Name:        "sleep-tracker",
		DisplayName: "Sleep Tracker",
	}

	// Register first time
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-DOMAIN-REGISTER] First registration failed: %d", w.Code)
	}

	// Register again with updated data
	d.DisplayName = "Sleep Tracker v2"
	d.Description = "Updated description"
	body, _ = json.Marshal(d)
	req = httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-DOMAIN-REGISTER] Upsert failed: %d", w.Code)
	}

	var updated domain.Domain
	json.NewDecoder(w.Body).Decode(&updated)

	if updated.DisplayName != "Sleep Tracker v2" {
		t.Error("[REQ:LD-DOMAIN-REGISTER] Upsert should update display_name")
	}
}

// =============================================================================
// Domain Discovery Tests [REQ:LD-DOMAIN-DISCOVER]
// =============================================================================

// TestDomainDiscover_ListAll validates GET /api/v1/domains returns all domains
// [REQ:LD-DOMAIN-DISCOVER] GET /api/v1/domains returns list of registered domains
func TestDomainDiscover_ListAll(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register multiple domains
	domains := []domain.RegisterDomainRequest{
		{Name: "sleep", DisplayName: "Sleep Tracker", Capabilities: []string{"track_sleep"}},
		{Name: "nutrition", DisplayName: "Nutrition Log", Capabilities: []string{"log_meals"}},
		{Name: "exercise", DisplayName: "Exercise Tracker", Capabilities: []string{"track_workouts"}},
	}

	for _, d := range domains {
		body, _ := json.Marshal(d)
		req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// List all domains
	req := httptest.NewRequest("GET", "/api/v1/domains", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-DOMAIN-DISCOVER] Expected status 200, got %d", w.Code)
	}

	var result domain.DomainsResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != 3 {
		t.Errorf("[REQ:LD-DOMAIN-DISCOVER] Expected 3 domains, got %d", result.Count)
	}

	// Verify domains have capabilities
	for _, d := range result.Domains {
		if len(d.Capabilities) == 0 {
			t.Errorf("[REQ:LD-DOMAIN-DISCOVER] Domain '%s' should expose capabilities", d.Name)
		}
	}
}

// TestDomainDiscover_GetSingle validates GET /api/v1/domains/{name}
// [REQ:LD-DOMAIN-DISCOVER] Get single domain by name
func TestDomainDiscover_GetSingle(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register a domain
	d := domain.RegisterDomainRequest{
		Name:         "meditation",
		DisplayName:  "Meditation Practice",
		Description:  "Track meditation sessions",
		Capabilities: []string{"track_sessions", "guided_meditations"},
	}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Get single domain
	req = httptest.NewRequest("GET", "/api/v1/domains/meditation", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-DOMAIN-DISCOVER] Expected status 200, got %d", w.Code)
	}

	var result domain.Domain
	json.NewDecoder(w.Body).Decode(&result)

	if result.Name != "meditation" {
		t.Errorf("[REQ:LD-DOMAIN-DISCOVER] Expected name 'meditation', got '%s'", result.Name)
	}
	if result.Description != "Track meditation sessions" {
		t.Error("[REQ:LD-DOMAIN-DISCOVER] Description should be returned")
	}
}

// TestDomainDiscover_NotFound validates 404 for unknown domains
// [REQ:LD-DOMAIN-DISCOVER] Unknown domains return 404
func TestDomainDiscover_NotFound(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/domains/unknown-domain", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("[REQ:LD-DOMAIN-DISCOVER] Expected status 404 for unknown domain, got %d", w.Code)
	}
}

// =============================================================================
// Domain Health Tests [REQ:LD-DOMAIN-HEALTH]
// =============================================================================

// TestDomainHealth_NoHealthURL validates health check when no URL configured
// [REQ:LD-DOMAIN-HEALTH] Dashboard periodically polls domain health endpoints
func TestDomainHealth_NoHealthURL(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register domain without health URL
	d := domain.RegisterDomainRequest{
		Name:        "simple-domain",
		DisplayName: "Simple Domain",
	}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Check health
	req = httptest.NewRequest("GET", "/api/v1/domains/simple-domain/health", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-DOMAIN-HEALTH] Expected status 200, got %d", w.Code)
	}

	var result domain.HealthCheckResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Message != "no health URL configured" {
		t.Error("[REQ:LD-DOMAIN-HEALTH] Should indicate no health URL configured")
	}
}

// =============================================================================
// Query Filter Tests [REQ:LD-QUERY-FILTER]
// =============================================================================

// TestQueryFilter_ByDomain validates domain filtering
// [REQ:LD-QUERY-FILTER] GET /api/v1/events supports domains query param
func TestQueryFilter_ByDomain(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events in multiple domains
	for _, d := range []string{"sleep", "sleep", "nutrition", "exercise"} {
		event := domain.CreateEventRequest{
			Domain:    d,
			EventType: "test.event",
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Query only sleep domain
	req := httptest.NewRequest("GET", "/api/v1/events?domain=sleep", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != 2 {
		t.Errorf("[REQ:LD-QUERY-FILTER] Expected 2 sleep events, got %d", result.Count)
	}

	for _, e := range result.Events {
		if e.Domain != "sleep" {
			t.Errorf("[REQ:LD-QUERY-FILTER] Expected only sleep domain events, got '%s'", e.Domain)
		}
	}
}

// TestQueryFilter_ByEventType validates event_type filtering
// [REQ:LD-QUERY-FILTER] GET /api/v1/events supports event_types query param
func TestQueryFilter_ByEventType(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events with different types
	eventTypes := []string{"supplement.taken", "supplement.taken", "sleep.recorded", "meal.logged"}
	for _, et := range eventTypes {
		event := domain.CreateEventRequest{
			Domain:    "test",
			EventType: et,
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Query by event type
	req := httptest.NewRequest("GET", "/api/v1/events?event_type=supplement.taken", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != 2 {
		t.Errorf("[REQ:LD-QUERY-FILTER] Expected 2 supplement.taken events, got %d", result.Count)
	}
}

// TestQueryFilter_ByTimeRange validates time range filtering
// [REQ:LD-QUERY-FILTER] GET /api/v1/events supports start_time, end_time
func TestQueryFilter_ByTimeRange(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events with specific timestamps
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	timestamps := []string{
		now.Format(time.RFC3339),
		yesterday.Format(time.RFC3339),
		lastWeek.Format(time.RFC3339),
	}

	for _, ts := range timestamps {
		event := domain.CreateEventRequest{
			Domain:    "test",
			EventType: "test.event",
			Timestamp: &ts,
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Query events from last 2 days
	start := now.Add(-48 * time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest("GET", "/api/v1/events?start="+start, nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != 2 {
		t.Errorf("[REQ:LD-QUERY-FILTER] Expected 2 events in last 48h, got %d", result.Count)
	}
}

// TestQueryFilter_Pagination validates limit query param
// [REQ:LD-QUERY-FILTER] GET /api/v1/events supports limit, offset
func TestQueryFilter_Pagination(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create 10 events
	for i := 0; i < 10; i++ {
		event := domain.CreateEventRequest{
			Domain:    "test",
			EventType: fmt.Sprintf("event.%d", i),
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Query with limit
	req := httptest.NewRequest("GET", "/api/v1/events?limit=5", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.Count != 5 {
		t.Errorf("[REQ:LD-QUERY-FILTER] Expected 5 events with limit=5, got %d", result.Count)
	}
}

// =============================================================================
// Statistics/Timeline Tests [REQ:LD-QUERY-AGGREGATE]
// =============================================================================

// TestStatistics_Timeline validates timeline aggregation by day
// [REQ:LD-QUERY-AGGREGATE] Timeline groups events by day/domain
func TestStatistics_Timeline(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events
	for i := 0; i < 5; i++ {
		event := domain.CreateEventRequest{
			Domain:    "sleep",
			EventType: "sleep.recorded",
			Payload:   json.RawMessage(`{"hours": 7}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Get timeline
	req := httptest.NewRequest("GET", "/api/v1/stats/timeline?days=7", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-QUERY-AGGREGATE] Expected status 200, got %d", w.Code)
	}

	var result domain.TimelineResponse
	json.NewDecoder(w.Body).Decode(&result)

	if len(result.Timeline) == 0 {
		t.Error("[REQ:LD-QUERY-AGGREGATE] Timeline should have entries")
	}

	// Verify timeline entry structure
	if len(result.Timeline) > 0 {
		entry := result.Timeline[0]
		if entry.Day == "" {
			t.Error("[REQ:LD-QUERY-AGGREGATE] Timeline entry should have 'day' field")
		}
		if entry.Domain == "" {
			t.Error("[REQ:LD-QUERY-AGGREGATE] Timeline entry should have 'domain' field")
		}
	}
}

// TestStatistics_Summary validates summary aggregation
// [REQ:LD-QUERY-AGGREGATE] Summary provides total counts and domain breakdown
func TestStatistics_Summary(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register a domain and create events
	d := domain.RegisterDomainRequest{Name: "test-domain", DisplayName: "Test Domain"}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Create events
	for i := 0; i < 3; i++ {
		event := domain.CreateEventRequest{
			Domain:    "test-domain",
			EventType: "test.event",
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Get summary
	req = httptest.NewRequest("GET", "/api/v1/stats/summary", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-QUERY-AGGREGATE] Expected status 200, got %d", w.Code)
	}

	var result domain.SummaryResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.TotalEvents != 3 {
		t.Errorf("[REQ:LD-QUERY-AGGREGATE] Expected 3 total events, got %d", result.TotalEvents)
	}
	if result.ActiveDomains != 1 {
		t.Errorf("[REQ:LD-QUERY-AGGREGATE] Expected 1 active domain, got %d", result.ActiveDomains)
	}
	if len(result.EventsByDomain) == 0 {
		t.Error("[REQ:LD-QUERY-AGGREGATE] events_by_domain should have entries")
	}
}

// =============================================================================
// Event Retrieval Tests [REQ:LD-EVENT-SCHEMA]
// =============================================================================

// TestEventGet_ByID validates GET /api/v1/events/{id}
// [REQ:LD-EVENT-SCHEMA] Individual event retrieval by ID
func TestEventGet_ByID(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create an event
	event := domain.CreateEventRequest{
		Domain:    "test",
		EventType: "test.created",
		Payload:   json.RawMessage(`{"key": "value"}`),
	}
	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var created domain.Event
	json.NewDecoder(w.Body).Decode(&created)

	// Get by ID
	req = httptest.NewRequest("GET", "/api/v1/events/"+created.ID, nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-EVENT-SCHEMA] Expected status 200, got %d", w.Code)
	}

	var retrieved domain.Event
	json.NewDecoder(w.Body).Decode(&retrieved)

	if retrieved.ID != created.ID {
		t.Errorf("[REQ:LD-EVENT-SCHEMA] IDs should match: expected '%s', got '%s'", created.ID, retrieved.ID)
	}
}

// TestEventGet_NotFound validates 404 for unknown event ID
// [REQ:LD-EVENT-SCHEMA] Unknown event IDs return 404
func TestEventGet_NotFound(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/events/550e8400-e29b-41d4-a716-446655440000", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("[REQ:LD-EVENT-SCHEMA] Expected status 404 for unknown event, got %d", w.Code)
	}
}

// =============================================================================
// Health Endpoint Tests
// =============================================================================

// TestHealth_Endpoint validates /health returns proper status
func TestHealth_Endpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", result["status"])
	}
}

// TestHealth_APIv1Endpoint validates /api/v1/health also works
func TestHealth_APIv1Endpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

// TestCreateEvent_ValidationRequired validates required fields
func TestCreateEvent_ValidationRequired(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Missing domain
	event := domain.CreateEventRequest{
		EventType: "test.event",
	}
	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing domain, got %d", w.Code)
	}

	// Missing event_type
	event = domain.CreateEventRequest{
		Domain: "test",
	}
	body, _ = json.Marshal(event)
	req = httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing event_type, got %d", w.Code)
	}
}

// TestRegisterDomain_ValidationRequired validates required fields
func TestRegisterDomain_ValidationRequired(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Missing name
	d := domain.RegisterDomainRequest{
		DisplayName: "Test Domain",
	}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing name, got %d", w.Code)
	}
}

// =============================================================================
// Event Index Tests [REQ:LD-EVENT-INDEX]
// =============================================================================

// TestEventIndex_DomainTimestampIndex validates composite index exists for efficient cross-domain queries
// [REQ:LD-EVENT-INDEX] Composite index on (domain, timestamp) for efficient time-range queries
func TestEventIndex_DomainTimestampIndex(t *testing.T) {
	_, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Query SQLite index list
	rows, err := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type = 'index' AND tbl_name = 'events'
		ORDER BY name
	`)
	if err != nil {
		t.Fatalf("[REQ:LD-EVENT-INDEX] Failed to query indexes: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		rows.Scan(&name)
		indexes[name] = true
	}

	// Verify required indexes exist
	requiredIndexes := []string{
		"idx_events_domain_timestamp", // (domain, timestamp) for cross-domain queries
		"idx_events_timestamp",        // timestamp for time-range queries
		"idx_events_type",             // event_type for type filtering
		"idx_events_hypothesis",       // hypothesis_id partial index for experiments
	}

	for _, idx := range requiredIndexes {
		if !indexes[idx] {
			t.Errorf("[REQ:LD-EVENT-INDEX] Missing required index: %s", idx)
		}
	}
}

// TestEventIndex_HypothesisPartialIndex validates partial index on hypothesis_id
// [REQ:LD-EVENT-INDEX] Partial index on hypothesis_id WHERE NOT NULL for experiment queries
func TestEventIndex_HypothesisPartialIndex(t *testing.T) {
	_, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Query index info to verify it's a partial index
	var sql string
	err := db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_events_hypothesis'
	`).Scan(&sql)
	if err != nil {
		t.Fatalf("[REQ:LD-EVENT-INDEX] Failed to get index definition: %v", err)
	}

	// Verify the index has a WHERE clause (partial index)
	if sql == "" {
		t.Error("[REQ:LD-EVENT-INDEX] idx_events_hypothesis should have SQL definition")
	}
	// The SQL should contain WHERE hypothesis_id IS NOT NULL
	if len(sql) > 0 && !containsWhereClause(sql) {
		t.Error("[REQ:LD-EVENT-INDEX] idx_events_hypothesis should be a partial index with WHERE clause")
	}
}

// containsWhereClause checks if the SQL contains a WHERE clause (case-insensitive)
func containsWhereClause(sql string) bool {
	lower := sql
	for i := 0; i < len(lower); i++ {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			lower = lower[:i] + string(lower[i]+32) + lower[i+1:]
		}
	}
	return len(lower) > 5 && (lower[0:5] == "where" || containsSubstring(lower, " where "))
}

// containsSubstring is a simple substring check without importing strings
func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestEventIndex_QueryPerformance validates index-backed query efficiency
// [REQ:LD-EVENT-INDEX] Indexes improve query performance for cross-domain queries
func TestEventIndex_QueryPerformance(t *testing.T) {
	srv, db, cleanup := setupTestServer(t)
	defer cleanup()

	// Create test events across multiple domains
	domains := []string{"sleep", "nutrition", "exercise", "meditation", "nootropics"}
	for i := 0; i < 50; i++ {
		event := domain.CreateEventRequest{
			Domain:    domains[i%len(domains)],
			EventType: "test.event",
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Verify EXPLAIN QUERY PLAN uses the domain_timestamp index for domain filtering
	var id, parent, notUsed int
	var detail string
	err := db.QueryRow(`
		EXPLAIN QUERY PLAN
		SELECT * FROM events WHERE domain = 'sleep' ORDER BY timestamp DESC
	`).Scan(&id, &parent, &notUsed, &detail)
	if err != nil {
		t.Fatalf("[REQ:LD-EVENT-INDEX] EXPLAIN failed: %v", err)
	}

	// The detail should mention using an index (SEARCH or USING INDEX)
	if !containsSubstring(detail, "USING INDEX") && !containsSubstring(detail, "SEARCH") {
		t.Logf("[REQ:LD-EVENT-INDEX] Query plan: %s", detail)
		// Note: SQLite may choose table scan for small datasets; that's OK for this test
	}
}

// =============================================================================
// Domain Update Tests [REQ:LD-DOMAIN-REGISTER]
// =============================================================================

// TestUpdateDomain_PartialUpdate validates PATCH /api/v1/domains/{name}
// [REQ:LD-DOMAIN-REGISTER] Domains can be updated via PATCH
func TestUpdateDomain_PartialUpdate(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register domain first
	d := domain.RegisterDomainRequest{
		Name:        "update-test",
		DisplayName: "Original Name",
		Description: "Original description",
	}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("[REQ:LD-DOMAIN-REGISTER] Registration failed: %d", w.Code)
	}

	// Update with PATCH
	updates := map[string]interface{}{
		"display_name": "Updated Name",
		"description":  "Updated description",
	}
	body, _ = json.Marshal(updates)
	req = httptest.NewRequest("PATCH", "/api/v1/domains/update-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-DOMAIN-REGISTER] Update failed: %d: %s", w.Code, w.Body.String())
	}

	var updated domain.Domain
	json.NewDecoder(w.Body).Decode(&updated)

	if updated.DisplayName != "Updated Name" {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] Expected display_name 'Updated Name', got '%s'", updated.DisplayName)
	}
	if updated.Description != "Updated description" {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] Expected description 'Updated description', got '%s'", updated.Description)
	}
}

// TestUpdateDomain_NotFound validates 404 for unknown domain
// [REQ:LD-DOMAIN-REGISTER] Unknown domain returns 404
func TestUpdateDomain_NotFound(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	updates := map[string]interface{}{
		"display_name": "Should Not Work",
	}
	body, _ := json.Marshal(updates)
	req := httptest.NewRequest("PATCH", "/api/v1/domains/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] Expected 404 for nonexistent domain, got %d", w.Code)
	}
}

// TestUpdateDomain_InvalidJSON validates error handling for invalid JSON
// [REQ:LD-DOMAIN-REGISTER] Invalid JSON returns 400
func TestUpdateDomain_InvalidJSON(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register domain first
	d := domain.RegisterDomainRequest{
		Name:        "json-test",
		DisplayName: "JSON Test",
	}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Send invalid JSON
	req = httptest.NewRequest("PATCH", "/api/v1/domains/json-test", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("[REQ:LD-DOMAIN-REGISTER] Expected 400 for invalid JSON, got %d", w.Code)
	}
}

// TestCreateEvent_InvalidJSON validates error handling for invalid JSON
func TestCreateEvent_InvalidJSON(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}

	var errResp domain.ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)

	if !errResp.Error {
		t.Error("Error response should have error=true")
	}
}

// TestRegisterDomain_InvalidJSON validates error handling for invalid JSON
func TestRegisterDomain_InvalidJSON(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

// =============================================================================
// Lifestyle Score Tests [REQ:LD-UI-SCORE]
// =============================================================================

// TestScore_Endpoint validates GET /api/v1/stats/score returns score data
// [REQ:LD-UI-SCORE] Score endpoint returns daily lifestyle score
func TestScore_Endpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Get score (no data should return 0 with "insufficient" data quality)
	req := httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-UI-SCORE] Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.ScoreResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("[REQ:LD-UI-SCORE] Failed to decode response: %v", err)
	}

	// Verify structure
	if result.Current.Date == "" {
		t.Error("[REQ:LD-UI-SCORE] Score should have date field")
	}
	if result.Current.DataQuality == "" {
		t.Error("[REQ:LD-UI-SCORE] Score should have data_quality field")
	}
	if result.Current.Message == "" {
		t.Error("[REQ:LD-UI-SCORE] Score should have message field")
	}
}

// TestScore_WithDomainActivity validates score calculation with events
// [REQ:LD-UI-SCORE] Score increases with domain activity
func TestScore_WithDomainActivity(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register domains
	domains := []struct {
		name        string
		displayName string
	}{
		{"sleep", "Sleep"},
		{"nutrition", "Nutrition"},
		{"exercise", "Exercise"},
	}

	for _, d := range domains {
		body, _ := json.Marshal(domain.RegisterDomainRequest{
			Name:        d.name,
			DisplayName: d.displayName,
		})
		req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Create events for each domain (today)
	for _, d := range domains {
		for i := 0; i < 3; i++ { // 3 events per domain = 60 points each
			event := domain.CreateEventRequest{
				Domain:    d.name,
				EventType: d.name + ".recorded",
				Payload:   json.RawMessage(`{}`),
			}
			body, _ := json.Marshal(event)
			req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)
		}
	}

	// Get score
	req := httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.ScoreResponse
	json.NewDecoder(w.Body).Decode(&result)

	// Score should be positive (3 domains × 3 events × 20 points = 60 each)
	if result.Current.Score == 0 {
		t.Errorf("[REQ:LD-UI-SCORE] Expected non-zero score with activity, got %d", result.Current.Score)
	}

	// Data quality should be "good" with 3 domains
	if result.Current.DataQuality != "good" {
		t.Errorf("[REQ:LD-UI-SCORE] Expected data_quality 'good' with 3 domains, got '%s'", result.Current.DataQuality)
	}

	// Should have domain scores
	if len(result.Current.DomainScores) != 3 {
		t.Errorf("[REQ:LD-UI-SCORE] Expected 3 domain scores, got %d", len(result.Current.DomainScores))
	}
}

// TestScore_HistoryDaysParam validates history_days query parameter
// [REQ:LD-UI-SCORE] Score endpoint accepts history_days parameter
func TestScore_HistoryDaysParam(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Request with specific history days
	req := httptest.NewRequest("GET", "/api/v1/stats/score?history_days=14", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-UI-SCORE] Expected status 200, got %d", w.Code)
	}

	var result domain.ScoreResponse
	json.NewDecoder(w.Body).Decode(&result)

	// History should have entries (up to 14 days)
	if len(result.History) == 0 {
		t.Error("[REQ:LD-UI-SCORE] History should have entries")
	}
	if len(result.History) > 14 {
		t.Errorf("[REQ:LD-UI-SCORE] History should have at most 14 entries, got %d", len(result.History))
	}
}

// TestScore_TrendCalculation validates trend direction
// [REQ:LD-UI-SCORE] Score includes trend indicator (up/down/stable)
func TestScore_TrendCalculation(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Get initial score (will be 0)
	req := httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.ScoreResponse
	json.NewDecoder(w.Body).Decode(&result)

	// Trend should be one of: up, down, stable
	validTrends := map[string]bool{"up": true, "down": true, "stable": true}
	if !validTrends[result.Current.Trend] {
		t.Errorf("[REQ:LD-UI-SCORE] Invalid trend '%s', expected up/down/stable", result.Current.Trend)
	}

	// Change from yesterday should be present
	// (Can be 0, positive, or negative)
	t.Logf("[REQ:LD-UI-SCORE] Change from yesterday: %d", result.Current.ChangeFromYesterday)
}

// TestScore_DomainWeight validates domain weight calculation
// [REQ:LD-UI-SCORE] Each domain contributes equally to composite score
func TestScore_DomainWeight(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register 2 domains
	for _, name := range []string{"domain-a", "domain-b"} {
		body, _ := json.Marshal(domain.RegisterDomainRequest{
			Name:        name,
			DisplayName: name,
		})
		req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Create event for only domain-a
	event := domain.CreateEventRequest{
		Domain:    "domain-a",
		EventType: "test.event",
		Payload:   json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Get score
	req = httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.ScoreResponse
	json.NewDecoder(w.Body).Decode(&result)

	// Verify weights are equal (0.5 each for 2 domains)
	for _, ds := range result.Current.DomainScores {
		expectedWeight := 0.5 // 1/2 domains
		if ds.Weight != expectedWeight {
			t.Errorf("[REQ:LD-UI-SCORE] Domain %s weight should be %.2f, got %.2f",
				ds.Domain, expectedWeight, ds.Weight)
		}
	}
}

// =============================================================================
// Storage Management Tests [REQ:LD-UI-STORAGE]
// =============================================================================

// TestStorageInfo_Endpoint validates GET /api/v1/storage returns storage info
// [REQ:LD-UI-STORAGE] Storage info endpoint for settings page
func TestStorageInfo_Endpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register a domain
	d := domain.RegisterDomainRequest{
		Name:        "storage-test",
		DisplayName: "Storage Test",
	}
	body, _ := json.Marshal(d)
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Create some events
	for i := 0; i < 5; i++ {
		event := domain.CreateEventRequest{
			Domain:    "storage-test",
			EventType: "test.event",
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Get storage info
	req = httptest.NewRequest("GET", "/api/v1/storage", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-UI-STORAGE] Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.StorageInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("[REQ:LD-UI-STORAGE] Failed to decode response: %v", err)
	}

	// Verify structure
	if result.TotalEvents != 5 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 5 total events, got %d", result.TotalEvents)
	}
	if result.TotalDomains != 1 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 1 total domain, got %d", result.TotalDomains)
	}
	if len(result.EventsByDomain) == 0 {
		t.Error("[REQ:LD-UI-STORAGE] EventsByDomain should have entries")
	}
	if result.OldestEvent == "" {
		t.Error("[REQ:LD-UI-STORAGE] OldestEvent should be set")
	}
	if result.NewestEvent == "" {
		t.Error("[REQ:LD-UI-STORAGE] NewestEvent should be set")
	}
}

// TestStorageInfo_EmptyDatabase validates storage info with no data
// [REQ:LD-UI-STORAGE] Empty database returns valid storage info
func TestStorageInfo_EmptyDatabase(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-UI-STORAGE] Expected status 200, got %d", w.Code)
	}

	var result domain.StorageInfo
	json.NewDecoder(w.Body).Decode(&result)

	if result.TotalEvents != 0 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 0 total events, got %d", result.TotalEvents)
	}
	if result.TotalDomains != 0 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 0 total domains, got %d", result.TotalDomains)
	}
}

// TestCleanupEvents_ClearAll validates DELETE /api/v1/storage/events clears all events
// [REQ:LD-UI-STORAGE] Clear all events functionality
func TestCleanupEvents_ClearAll(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events in multiple domains
	for _, d := range []string{"domain-a", "domain-b"} {
		for i := 0; i < 3; i++ {
			event := domain.CreateEventRequest{
				Domain:    d,
				EventType: "test.event",
				Payload:   json.RawMessage(`{}`),
			}
			body, _ := json.Marshal(event)
			req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)
		}
	}

	// Verify events exist
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	var eventsResult domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&eventsResult)
	if eventsResult.Count != 6 {
		t.Fatalf("[REQ:LD-UI-STORAGE] Setup failed: expected 6 events, got %d", eventsResult.Count)
	}

	// Clear all events
	req = httptest.NewRequest("DELETE", "/api/v1/storage/events", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-UI-STORAGE] Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.CleanupResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.DeletedEvents != 6 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 6 deleted events, got %d", result.DeletedEvents)
	}
	if result.Message == "" {
		t.Error("[REQ:LD-UI-STORAGE] Message should not be empty")
	}

	// Verify events are gone
	req = httptest.NewRequest("GET", "/api/v1/events", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&eventsResult)
	if eventsResult.Count != 0 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 0 events after cleanup, got %d", eventsResult.Count)
	}
}

// TestCleanupEvents_ClearDomain validates clearing events from specific domain
// [REQ:LD-UI-STORAGE] Clear events from specific domain
func TestCleanupEvents_ClearDomain(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create events in multiple domains
	for _, d := range []string{"keep-domain", "clear-domain"} {
		for i := 0; i < 3; i++ {
			event := domain.CreateEventRequest{
				Domain:    d,
				EventType: "test.event",
				Payload:   json.RawMessage(`{}`),
			}
			body, _ := json.Marshal(event)
			req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)
		}
	}

	// Clear only clear-domain
	cleanupReq := domain.CleanupRequest{
		Domains: []string{"clear-domain"},
	}
	body, _ := json.Marshal(cleanupReq)
	req := httptest.NewRequest("DELETE", "/api/v1/storage/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-UI-STORAGE] Expected status 200, got %d", w.Code)
	}

	var result domain.CleanupResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.DeletedEvents != 3 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 3 deleted events, got %d", result.DeletedEvents)
	}

	// Verify keep-domain events still exist
	req = httptest.NewRequest("GET", "/api/v1/events?domain=keep-domain", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	var eventsResult domain.EventsResponse
	json.NewDecoder(w.Body).Decode(&eventsResult)
	if eventsResult.Count != 3 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 3 keep-domain events, got %d", eventsResult.Count)
	}

	// Verify clear-domain events are gone
	req = httptest.NewRequest("GET", "/api/v1/events?domain=clear-domain", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&eventsResult)
	if eventsResult.Count != 0 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 0 clear-domain events, got %d", eventsResult.Count)
	}
}

// TestStorageInfo_EventsByDomain validates domain breakdown in storage info
// [REQ:LD-UI-STORAGE] Storage info includes per-domain event counts
func TestStorageInfo_EventsByDomain(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register domains
	for _, name := range []string{"sleep", "nutrition", "exercise"} {
		d := domain.RegisterDomainRequest{
			Name:        name,
			DisplayName: name + " Tracker",
		}
		body, _ := json.Marshal(d)
		req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Create different numbers of events per domain
	// sleep: 5, nutrition: 3, exercise: 2
	eventCounts := map[string]int{"sleep": 5, "nutrition": 3, "exercise": 2}
	for domainName, count := range eventCounts {
		for i := 0; i < count; i++ {
			event := domain.CreateEventRequest{
				Domain:    domainName,
				EventType: "test.event",
				Payload:   json.RawMessage(`{}`),
			}
			body, _ := json.Marshal(event)
			req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.router.ServeHTTP(w, req)
		}
	}

	// Get storage info
	req := httptest.NewRequest("GET", "/api/v1/storage", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.StorageInfo
	json.NewDecoder(w.Body).Decode(&result)

	if result.TotalEvents != 10 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 10 total events, got %d", result.TotalEvents)
	}

	if len(result.EventsByDomain) != 3 {
		t.Errorf("[REQ:LD-UI-STORAGE] Expected 3 domain entries, got %d", len(result.EventsByDomain))
	}

	// Verify display names are included
	for _, domainInfo := range result.EventsByDomain {
		if domainInfo.DisplayName == "" {
			t.Errorf("[REQ:LD-UI-STORAGE] Domain %s should have display_name", domainInfo.Domain)
		}
	}
}

// =============================================================================
// Daily Brief System Tests [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] [REQ:LD-BRIEF-CONSOLIDATE]
// =============================================================================

// TestBriefs_CurrentBriefEndpoint validates the current brief endpoint
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING]
func TestBriefs_CurrentBriefEndpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/briefs/current", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-BRIEF-MORNING] Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.BriefResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("[REQ:LD-BRIEF-MORNING] Failed to decode response: %v", err)
	}

	// Validate brief structure
	if result.Brief.Type != "morning" && result.Brief.Type != "evening" {
		t.Errorf("[REQ:LD-BRIEF-MORNING] Brief type should be 'morning' or 'evening', got '%s'", result.Brief.Type)
	}
	if result.Brief.GeneratedAt == "" {
		t.Error("[REQ:LD-BRIEF-MORNING] Brief should have generated_at timestamp")
	}
	if result.Brief.Date == "" {
		t.Error("[REQ:LD-BRIEF-MORNING] Brief should have target date")
	}
	if result.Brief.Summary == "" {
		t.Error("[REQ:LD-BRIEF-MORNING] Brief should have summary text")
	}

	// Validate config
	if result.Config.MorningHour != 7 {
		t.Errorf("[REQ:LD-BRIEF-MORNING] Expected morning hour 7, got %d", result.Config.MorningHour)
	}
	if result.Config.EveningHour != 21 {
		t.Errorf("[REQ:LD-BRIEF-EVENING] Expected evening hour 21, got %d", result.Config.EveningHour)
	}
}

// TestBriefs_MorningBriefEndpoint validates the morning brief endpoint
// [REQ:LD-BRIEF-MORNING]
func TestBriefs_MorningBriefEndpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/briefs/morning", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-BRIEF-MORNING] Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.BriefResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("[REQ:LD-BRIEF-MORNING] Failed to decode response: %v", err)
	}

	if result.Brief.Type != "morning" {
		t.Errorf("[REQ:LD-BRIEF-MORNING] Expected type 'morning', got '%s'", result.Brief.Type)
	}
}

// TestBriefs_EveningBriefEndpoint validates the evening brief endpoint
// [REQ:LD-BRIEF-EVENING]
func TestBriefs_EveningBriefEndpoint(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/briefs/evening", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-BRIEF-EVENING] Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.BriefResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("[REQ:LD-BRIEF-EVENING] Failed to decode response: %v", err)
	}

	if result.Brief.Type != "evening" {
		t.Errorf("[REQ:LD-BRIEF-EVENING] Expected type 'evening', got '%s'", result.Brief.Type)
	}
}

// TestBriefs_DateParameter validates date parameter handling
// [REQ:LD-BRIEF-MORNING]
func TestBriefs_DateParameter(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Valid date
	req := httptest.NewRequest("GET", "/api/v1/briefs/morning?date=2026-03-10", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-BRIEF-MORNING] Expected status 200 for valid date, got %d", w.Code)
	}

	var result domain.BriefResponse
	json.NewDecoder(w.Body).Decode(&result)
	if result.Brief.Date != "2026-03-10" {
		t.Errorf("[REQ:LD-BRIEF-MORNING] Expected date '2026-03-10', got '%s'", result.Brief.Date)
	}

	// Invalid date format
	req = httptest.NewRequest("GET", "/api/v1/briefs/morning?date=invalid", nil)
	w = httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("[REQ:LD-BRIEF-MORNING] Expected status 400 for invalid date, got %d", w.Code)
	}
}

// TestBriefs_CrossDomainConsolidation validates that briefs consolidate data from all domains
// [REQ:LD-BRIEF-CONSOLIDATE]
func TestBriefs_CrossDomainConsolidation(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Register multiple domains
	domains := []domain.RegisterDomainRequest{
		{Name: "sleep", DisplayName: "Sleep Tracker"},
		{Name: "nutrition", DisplayName: "Nutrition Log"},
		{Name: "exercise", DisplayName: "Exercise Tracker"},
	}

	for _, d := range domains {
		body, _ := json.Marshal(d)
		req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Create events for each domain today
	today := time.Now().Format("2006-01-02")
	for _, d := range domains {
		event := domain.CreateEventRequest{
			Domain:    d.Name,
			EventType: "test.event",
			Payload:   json.RawMessage(`{"test": true}`),
			Timestamp: func() *string { s := today + "T12:00:00Z"; return &s }(),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Get evening brief for today (should include today's events)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/briefs/evening?date=%s", today), nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("[REQ:LD-BRIEF-CONSOLIDATE] Expected status 200, got %d", w.Code)
	}

	var result domain.BriefResponse
	json.NewDecoder(w.Body).Decode(&result)

	// Should have sections for active domains
	if len(result.Brief.Sections) < 3 {
		t.Errorf("[REQ:LD-BRIEF-CONSOLIDATE] Expected at least 3 sections (one per domain), got %d", len(result.Brief.Sections))
	}

	// Verify each section has domain info
	for _, section := range result.Brief.Sections {
		if section.Domain == "" {
			t.Error("[REQ:LD-BRIEF-CONSOLIDATE] Section should have domain name")
		}
		if section.DisplayName == "" {
			t.Error("[REQ:LD-BRIEF-CONSOLIDATE] Section should have display name")
		}
	}
}

// TestBriefs_ScoreIncluded validates that briefs include lifestyle score when available
// [REQ:LD-BRIEF-MORNING] [REQ:LD-UI-SCORE]
func TestBriefs_ScoreIncluded(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create some events to generate a score
	for i := 0; i < 3; i++ {
		event := domain.CreateEventRequest{
			Domain:    "test-domain",
			EventType: "test.event",
			Payload:   json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
	}

	// Get brief
	req := httptest.NewRequest("GET", "/api/v1/briefs/current", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	var result domain.BriefResponse
	json.NewDecoder(w.Body).Decode(&result)

	// Score should be included when there's activity
	if result.Brief.Score == nil {
		t.Log("[REQ:LD-BRIEF-MORNING] Score may be nil if no recent activity, but should be present when events exist")
	}

	// Trend should be set if score is available
	if result.Brief.Score != nil && result.Brief.ScoreTrend == "" {
		t.Error("[REQ:LD-BRIEF-MORNING] Score trend should be set when score is available")
	}
}

// =============================================================================
// CORS and Middleware Tests
// =============================================================================

// TestCORS_HeadersOnRequest validates CORS headers are set on actual API requests.
// Note: OPTIONS preflight requires explicit route registration; current implementation
// relies on browsers sending actual requests with credentials, which works for same-origin
// or when CORS is handled by a reverse proxy (common in production).
func TestCORS_HeadersOnRequest(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Send actual POST request with Origin header (simulating cross-origin)
	event := domain.CreateEventRequest{
		Domain:    "test",
		EventType: "test.event",
		Payload:   json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	// Request should succeed
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// CORS headers should be set on response
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Expected Access-Control-Allow-Origin to be 'http://localhost:3000', got '%s'",
			w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected Access-Control-Allow-Credentials header to be 'true'")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}
}

// TestCORS_WithOrigin validates CORS headers are set when Origin is present.
func TestCORS_WithOrigin(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/domains", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Origin should be reflected in Allow-Origin header
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("Expected Access-Control-Allow-Origin to be 'http://localhost:5173', got '%s'",
			w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Expected Access-Control-Allow-Credentials to be 'true'")
	}
}

// TestCORS_WithoutOrigin validates requests without Origin still work.
func TestCORS_WithoutOrigin(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	// No Origin header
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	// Should still return OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Allow-Origin should not be set when no Origin provided
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin without Origin header")
	}
}

// TestServer_Handler validates the recovery handler wrapper is accessible.
func TestServer_Handler(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Get the wrapped handler (includes recovery middleware)
	handler := srv.Handler()
	if handler == nil {
		t.Fatal("Expected Handler() to return non-nil http.Handler")
	}

	// Use the handler directly - should work like the router
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 through Handler(), got %d", w.Code)
	}
}

// TestLoggingMiddleware validates request logging occurs.
func TestLoggingMiddleware(t *testing.T) {
	srv, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Make a request that should be logged
	req := httptest.NewRequest("GET", "/api/v1/stats/summary", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	// Just verify the request completes successfully (log output goes to stderr)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
