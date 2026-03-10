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

// TestRegisterDomain_MissingName verifies validation for missing name field.
// [REQ:LD-DOMAIN-REGISTER] Domain registration validation.
func TestRegisterDomain_MissingName(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"display_name": "Test Domain"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestRegisterDomain_InvalidJSON verifies error handling for malformed JSON.
// [REQ:LD-DOMAIN-REGISTER] Domain registration error handling.
func TestRegisterDomain_InvalidJSON(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{invalid json`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
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

// TestListDomains_EmptyResult verifies listing with no registered domains.
// [REQ:LD-DOMAIN-DISCOVER] Domain discovery with empty result.
func TestListDomains_EmptyResult(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

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
	if resp.Count != 0 {
		t.Errorf("Expected 0 domains, got %d", resp.Count)
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

// TestGetDomain_NotFound verifies 404 response for non-existent domain.
// [REQ:LD-DOMAIN-DISCOVER] Domain not found error handling.
func TestGetDomain_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/domains/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	rr := httptest.NewRecorder()

	h.GetDomain(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}
