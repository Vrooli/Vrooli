package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
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

// TestUpdateDomain_Success verifies domain update through handlers.
// [REQ:LD-DOMAIN-REGISTER] Domain update through handler layer.
func TestUpdateDomain_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register a domain first
	h.registerTestDomain(t, "update-test", "Original Name")

	// Update the domain
	body := `{"display_name": "Updated Name", "description": "New description"}`
	req := httptest.NewRequest("PATCH", "/api/v1/domains/update-test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "update-test"})
	rr := httptest.NewRecorder()

	h.UpdateDomain(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var d domain.Domain
	if err := json.NewDecoder(rr.Body).Decode(&d); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if d.DisplayName != "Updated Name" {
		t.Errorf("Expected display_name 'Updated Name', got '%s'", d.DisplayName)
	}
	if d.Description != "New description" {
		t.Errorf("Expected description 'New description', got '%s'", d.Description)
	}
}

// TestUpdateDomain_NotFound verifies 404 response for non-existent domain update.
// [REQ:LD-DOMAIN-REGISTER] Domain update not found error handling.
func TestUpdateDomain_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"display_name": "Updated"}`
	req := httptest.NewRequest("PATCH", "/api/v1/domains/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	rr := httptest.NewRecorder()

	h.UpdateDomain(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

// TestUpdateDomain_InvalidJSON verifies error handling for malformed JSON.
// [REQ:LD-DOMAIN-REGISTER] Domain update error handling.
func TestUpdateDomain_InvalidJSON(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "json-test", "Json Test")

	body := `{invalid json`
	req := httptest.NewRequest("PATCH", "/api/v1/domains/json-test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "json-test"})
	rr := httptest.NewRecorder()

	h.UpdateDomain(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestGetDomainHealth_NoHealthURL verifies health check without health URL.
// [REQ:LD-DOMAIN-HEALTH] Domain health check without configured URL.
func TestGetDomainHealth_NoHealthURL(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register domain without health URL
	h.registerTestDomain(t, "no-health-url", "No Health URL Domain")

	req := httptest.NewRequest("GET", "/api/v1/domains/no-health-url/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "no-health-url"})
	rr := httptest.NewRecorder()

	h.GetDomainHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.HealthCheckResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Domain != "no-health-url" {
		t.Errorf("Expected domain 'no-health-url', got '%s'", resp.Domain)
	}
	if resp.Message != "no health URL configured" {
		t.Errorf("Expected message 'no health URL configured', got '%s'", resp.Message)
	}
}

// TestGetDomainHealth_NotFound verifies 404 response for non-existent domain health check.
// [REQ:LD-DOMAIN-HEALTH] Domain health check not found error handling.
func TestGetDomainHealth_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/domains/nonexistent/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	rr := httptest.NewRecorder()

	h.GetDomainHealth(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

// TestRegisterDomain_MissingDisplayName verifies validation for missing display_name.
// [REQ:LD-DOMAIN-REGISTER] Domain registration validation.
func TestRegisterDomain_MissingDisplayName(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"name": "test-domain"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestRegisterDomain_WithHealthURL verifies domain registration with health URL.
// [REQ:LD-DOMAIN-REGISTER] Domain registration with optional fields.
func TestRegisterDomain_WithHealthURL(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"name": "health-test", "display_name": "Health Test", "health_url": "http://localhost:8080/health"}`
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
	if d.HealthURL != "http://localhost:8080/health" {
		t.Errorf("Expected health_url 'http://localhost:8080/health', got '%s'", d.HealthURL)
	}
}

// TestGetDomainHealth_WithHealthURL_Unhealthy verifies health check with failing health URL.
// [REQ:LD-DOMAIN-HEALTH] Domain health check with failing URL.
func TestGetDomainHealth_WithHealthURL_Unhealthy(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register domain with invalid health URL (will fail to connect)
	body := `{"name": "unhealthy-test", "display_name": "Unhealthy Test", "health_url": "http://localhost:1/invalid"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to register domain: %s", rr.Body.String())
	}

	// Check health - should return unhealthy
	req = httptest.NewRequest("GET", "/api/v1/domains/unhealthy-test/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "unhealthy-test"})
	rr = httptest.NewRecorder()

	h.GetDomainHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.HealthCheckResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", resp.Status)
	}
	if resp.Message == "" {
		t.Error("Expected non-empty error message for unhealthy domain")
	}
}

// TestGetDomainHealth_WithHealthURL_Healthy verifies health check with healthy URL.
// [REQ:LD-DOMAIN-HEALTH] Domain health check with successful URL.
func TestGetDomainHealth_WithHealthURL_Healthy(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create a test server that returns 200
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer healthServer.Close()

	// Register domain with test server health URL
	body := `{"name": "healthy-test", "display_name": "Healthy Test", "health_url": "` + healthServer.URL + `/health"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to register domain: %s", rr.Body.String())
	}

	// Check health - should return healthy
	req = httptest.NewRequest("GET", "/api/v1/domains/healthy-test/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "healthy-test"})
	rr = httptest.NewRecorder()

	h.GetDomainHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.HealthCheckResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", resp.Status)
	}
	if resp.LastCheck == "" {
		t.Error("Expected LastCheck to be set")
	}
}

// TestGetDomainHealth_WithHealthURL_BadStatusCode verifies health check with unhealthy status code.
// [REQ:LD-DOMAIN-HEALTH] Domain health check with bad status code.
func TestGetDomainHealth_WithHealthURL_BadStatusCode(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create a test server that returns 500
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "error"}`))
	}))
	defer healthServer.Close()

	// Register domain with test server health URL
	body := `{"name": "bad-status-test", "display_name": "Bad Status Test", "health_url": "` + healthServer.URL + `/health"}`
	req := httptest.NewRequest("POST", "/api/v1/domains", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.RegisterDomain(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to register domain: %s", rr.Body.String())
	}

	// Check health - should return unhealthy due to status code
	req = httptest.NewRequest("GET", "/api/v1/domains/bad-status-test/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "bad-status-test"})
	rr = httptest.NewRecorder()

	h.GetDomainHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.HealthCheckResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", resp.Status)
	}
	if resp.Message == "" {
		t.Error("Expected non-empty message for bad status code")
	}
}

// TestListDomains_DatabaseError verifies error handling when List returns error.
// [REQ:LD-DOMAIN-DISCOVER] Domain listing handles database errors.
func TestListDomains_DatabaseError(t *testing.T) {
	// Use mock repository with injected error
	mockDomains := &mockDomainRepoWithError{
		listError: fmt.Errorf("database connection lost"),
	}
	h := &Handler{Domains: mockDomains}

	req := httptest.NewRequest("GET", "/api/v1/domains", nil)
	rr := httptest.NewRecorder()

	h.ListDomains(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d: %s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}

	var errResp errors.APIError
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Category != errors.CategoryInternal {
		t.Errorf("Expected category 'internal', got '%s'", errResp.Category)
	}
}

// mockDomainRepoWithError provides controlled error injection for domain repository.
type mockDomainRepoWithError struct {
	listError error
}

func (m *mockDomainRepoWithError) Upsert(ctx context.Context, d *domain.Domain) error {
	return nil
}

func (m *mockDomainRepoWithError) GetByName(ctx context.Context, name string) (*domain.Domain, error) {
	return nil, nil
}

func (m *mockDomainRepoWithError) List(ctx context.Context) ([]domain.Domain, error) {
	return nil, m.listError
}

func (m *mockDomainRepoWithError) UpdateStatus(ctx context.Context, name, status, lastHealthAt string) error {
	return nil
}

func (m *mockDomainRepoWithError) Update(ctx context.Context, name string, updates map[string]interface{}) error {
	return nil
}

// TestGetDomainHealth_NoHealthURL_WithLastHealthAt verifies no health URL response with last health time.
// [REQ:LD-DOMAIN-HEALTH] Domain health check no URL with previous health check.
func TestGetDomainHealth_NoHealthURL_WithLastHealthAt(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register domain without health URL, then set last_health_at directly
	h.registerTestDomain(t, "prev-health-test", "Prev Health Test")

	// Update status to set last_health_at
	h.Domains.UpdateStatus(httptest.NewRequest("GET", "/", nil).Context(), "prev-health-test", "healthy", "2026-03-10T12:00:00Z")

	req := httptest.NewRequest("GET", "/api/v1/domains/prev-health-test/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "prev-health-test"})
	rr := httptest.NewRecorder()

	h.GetDomainHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.HealthCheckResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.LastCheck != "2026-03-10T12:00:00Z" {
		t.Errorf("Expected LastCheck '2026-03-10T12:00:00Z', got '%s'", resp.LastCheck)
	}
}
