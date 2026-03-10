package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/repository"
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

// setupTestHandler creates a handler with SQLite repositories for testing.
func setupTestHandler(t *testing.T) (*Handler, *sql.DB) {
	db := setupTestDB(t)
	eventRepo := repository.NewSQLiteEventRepository(db)
	domainRepo := repository.NewSQLiteDomainRepository(db)
	statsRepo := repository.NewSQLiteStatsRepository(db)
	h := New(eventRepo, domainRepo, statsRepo)
	return h, db
}

// TestNew_CreatesHandler verifies that New creates a valid Handler instance.
// [REQ:LD-QUERY-AGGREGATE] Handler initialization for statistics endpoints.
func TestNew_CreatesHandler(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	if h == nil {
		t.Fatal("Expected handler, got nil")
	}
	if h.Events == nil {
		t.Error("Handler Events repository should not be nil")
	}
	if h.Domains == nil {
		t.Error("Handler Domains repository should not be nil")
	}
	if h.Stats == nil {
		t.Error("Handler Stats repository should not be nil")
	}
}

// TestWriteJSON_SetsContentType verifies JSON response headers.
// [REQ:LD-EVENT-SCHEMA] Proper JSON content-type for API responses.
func TestWriteJSON_SetsContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"message": "test"}

	WriteJSON(rr, http.StatusOK, data)

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", ct)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestWriteError_ReturnsErrorStructure verifies error response format.
// [REQ:LD-EVENT-SCHEMA] Consistent error response structure.
func TestWriteError_ReturnsErrorStructure(t *testing.T) {
	rr := httptest.NewRecorder()

	WriteError(rr, http.StatusBadRequest, "test error message")

	var errResp domain.ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if !errResp.Error {
		t.Error("Expected error field to be true")
	}
	if errResp.Message != "test error message" {
		t.Errorf("Expected message 'test error message', got '%s'", errResp.Message)
	}
}

// TestCreateEvent_Success verifies event creation through handlers.
// [REQ:LD-EVENT-STORAGE] Event persistence through handler layer.
func TestCreateEvent_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"domain": "test-domain", "event_type": "test.created", "payload": {"value": 42}}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var event domain.Event
	if err := json.NewDecoder(rr.Body).Decode(&event); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if event.Domain != "test-domain" {
		t.Errorf("Expected domain 'test-domain', got '%s'", event.Domain)
	}
	if event.EventType != "test.created" {
		t.Errorf("Expected event_type 'test.created', got '%s'", event.EventType)
	}
}

// TestQueryEvents_WithFilters verifies event querying with domain filter.
// [REQ:LD-QUERY-FILTER] Handler-level domain filtering.
func TestQueryEvents_WithFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "domain-a", "event.a")
	h.createTestEvent(t, "domain-b", "event.b")
	h.createTestEvent(t, "domain-a", "event.c")

	// Query with filter
	req := httptest.NewRequest("GET", "/api/v1/events?domain=domain-a", nil)
	rr := httptest.NewRecorder()

	h.QueryEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.EventsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("Expected 2 events for domain-a, got %d", resp.Count)
	}
}

// TestRegisterDomain_Success verifies domain registration through handlers.
// [REQ:LD-DOMAIN-REGISTER] Domain registration through handler layer.
func TestRegisterDomain_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"name": "test-domain", "display_name": "Test Domain", "capabilities": ["events"]}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var d domain.Domain
	if err := json.NewDecoder(rr.Body).Decode(&d); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if d.Name != "test-domain" {
		t.Errorf("Expected name 'test-domain', got '%s'", d.Name)
	}
	if d.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", d.Status)
	}
}

// TestListDomains_ReturnsAll verifies domain listing through handlers.
// [REQ:LD-DOMAIN-DISCOVER] Domain discovery through handler layer.
func TestListDomains_ReturnsAll(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register test domains
	h.registerTestDomain(t, "domain-1", "Domain One")
	h.registerTestDomain(t, "domain-2", "Domain Two")

	req := httptest.NewRequest("GET", "/api/v1/domains", nil)
	rr := httptest.NewRecorder()

	h.ListDomains(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.DomainsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("Expected 2 domains, got %d", resp.Count)
	}
}

// TestGetDomain_WithMuxVars verifies single domain retrieval with router variables.
// [REQ:LD-DOMAIN-DISCOVER] Single domain retrieval through handler layer.
func TestGetDomain_WithMuxVars(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "get-test", "Get Test Domain")

	req := httptest.NewRequest("GET", "/api/v1/domains/get-test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "get-test"})
	rr := httptest.NewRecorder()

	h.GetDomain(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var d domain.Domain
	if err := json.NewDecoder(rr.Body).Decode(&d); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if d.Name != "get-test" {
		t.Errorf("Expected name 'get-test', got '%s'", d.Name)
	}
}

// TestGetTimeline_AggregatesCorrectly verifies timeline statistics.
// [REQ:LD-QUERY-AGGREGATE] Timeline aggregation through handler layer.
func TestGetTimeline_AggregatesCorrectly(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "timeline-test", "event.a")
	h.createTestEvent(t, "timeline-test", "event.b")

	req := httptest.NewRequest("GET", "/api/v1/stats/timeline?days=7", nil)
	rr := httptest.NewRecorder()

	h.GetTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.TimelineResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Days != "7" {
		t.Errorf("Expected days '7', got '%s'", resp.Days)
	}
}

// TestGetSummary_ReturnsTotals verifies summary statistics.
// [REQ:LD-QUERY-AGGREGATE] Summary aggregation through handler layer.
func TestGetSummary_ReturnsTotals(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test data
	h.registerTestDomain(t, "summary-domain", "Summary Test")
	h.createTestEvent(t, "summary-domain", "event.a")
	h.createTestEvent(t, "summary-domain", "event.b")

	req := httptest.NewRequest("GET", "/api/v1/stats/summary", nil)
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.SummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.TotalEvents < 2 {
		t.Errorf("Expected at least 2 events, got %d", resp.TotalEvents)
	}
	if resp.ActiveDomains < 1 {
		t.Errorf("Expected at least 1 active domain, got %d", resp.ActiveDomains)
	}
}

// Helper: create a test event directly via handler
func (h *Handler) createTestEvent(t *testing.T, domainName, eventType string) {
	body := `{"domain": "` + domainName + `", "event_type": "` + eventType + `", "payload": {}}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateEvent(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create test event: %s", rr.Body.String())
	}
}

// Helper: register a test domain directly via handler
func (h *Handler) registerTestDomain(t *testing.T, name, displayName string) {
	body := `{"name": "` + name + `", "display_name": "` + displayName + `"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RegisterDomain(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to register test domain: %s", rr.Body.String())
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
