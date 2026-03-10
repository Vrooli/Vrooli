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
