package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"lifestyle-dashboard/domain"
)

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

// TestCreateEvent_MissingDomain verifies validation for missing domain field.
// [REQ:LD-EVENT-STORAGE] Event validation before persistence.
func TestCreateEvent_MissingDomain(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"event_type": "test.created", "payload": {}}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestCreateEvent_MissingEventType verifies validation for missing event_type field.
// [REQ:LD-EVENT-STORAGE] Event validation before persistence.
func TestCreateEvent_MissingEventType(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"domain": "test-domain", "payload": {}}`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestCreateEvent_InvalidJSON verifies error handling for malformed JSON.
// [REQ:LD-EVENT-STORAGE] Event request parsing error handling.
func TestCreateEvent_InvalidJSON(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{invalid json`
	req := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.CreateEvent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestQueryEvents_NoFilters verifies listing all events without filters.
// [REQ:LD-EVENT-STORAGE] Event retrieval without filters.
func TestQueryEvents_NoFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "domain-a", "event.a")
	h.createTestEvent(t, "domain-b", "event.b")

	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	rr := httptest.NewRecorder()

	h.QueryEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.EventsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count < 2 {
		t.Errorf("Expected at least 2 events, got %d", resp.Count)
	}
}

// TestGetEvent_Success verifies single event retrieval.
// [REQ:LD-EVENT-STORAGE] Single event retrieval through handler layer.
func TestGetEvent_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create a test event
	body := `{"domain": "test-domain", "event_type": "test.event", "payload": {}}`
	createReq := httptest.NewRequest("POST", "/api/v1/events", bytes.NewBufferString(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	h.CreateEvent(createRR, createReq)

	var created domain.Event
	json.NewDecoder(createRR.Body).Decode(&created)

	// Retrieve the event by ID
	req := httptest.NewRequest("GET", "/api/v1/events/"+created.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	rr := httptest.NewRecorder()

	h.GetEvent(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var event domain.Event
	if err := json.NewDecoder(rr.Body).Decode(&event); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if event.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, event.ID)
	}
	if event.Domain != "test-domain" {
		t.Errorf("Expected domain 'test-domain', got '%s'", event.Domain)
	}
}

// TestGetEvent_NotFound verifies 404 response for non-existent event.
// [REQ:LD-EVENT-STORAGE] Event not found error handling.
func TestGetEvent_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/events/nonexistent-id", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent-id"})
	rr := httptest.NewRecorder()

	h.GetEvent(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

// TestCreateEvent_WithTimestamp verifies event creation with custom timestamp.
// [REQ:LD-EVENT-STORAGE] Event creation with custom timestamp.
func TestCreateEvent_WithTimestamp(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	customTimestamp := "2026-01-15T10:00:00Z"
	body := `{"domain": "test-domain", "event_type": "test.event", "payload": {}, "timestamp": "` + customTimestamp + `"}`
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
	if event.Timestamp != customTimestamp {
		t.Errorf("Expected timestamp '%s', got '%s'", customTimestamp, event.Timestamp)
	}
}

// TestCreateEvent_WithInterventionFlag verifies event creation with intervention flag.
// [REQ:LD-EVENT-STORAGE] Event creation with causality markers.
func TestCreateEvent_WithInterventionFlag(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"domain": "test-domain", "event_type": "intervention.applied", "payload": {}, "is_intervention": true}`
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
	if !event.IsIntervention {
		t.Error("Expected is_intervention to be true")
	}
}

// TestCreateEvent_WithHypothesisID verifies event creation with hypothesis ID.
// [REQ:LD-EVENT-STORAGE] Event creation with hypothesis correlation.
func TestCreateEvent_WithHypothesisID(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	hypothesisID := "hypo-test-123"
	body := `{"domain": "test-domain", "event_type": "test.event", "payload": {}, "hypothesis_id": "` + hypothesisID + `"}`
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
	if event.HypothesisID == nil || *event.HypothesisID != hypothesisID {
		t.Errorf("Expected hypothesis_id '%s', got '%v'", hypothesisID, event.HypothesisID)
	}
}
