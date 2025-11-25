package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestHealthEndpoint(t *testing.T) {
	// Set required environment variables for test
	os.Setenv("API_PORT", "15000")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USER", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("POSTGRES_DB", "test")

	// Note: This test requires a database connection
	// In a real scenario, you'd use a test database or mock
	srv, err := NewServer()
	if err != nil {
		t.Skip("Skipping test - database not available:", err)
	}
	defer srv.db.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestRequireEnv(t *testing.T) {
	// Test valid environment variable
	t.Run("valid environment variable", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := requireEnv("TEST_VAR")
		if result != "test_value" {
			t.Errorf("Expected test_value, got %s", result)
		}
	})

	// Test whitespace trimming
	t.Run("whitespace trimming", func(t *testing.T) {
		os.Setenv("TEST_VAR_WS", "  trimmed  ")
		defer os.Unsetenv("TEST_VAR_WS")

		result := requireEnv("TEST_VAR_WS")
		if result != "trimmed" {
			t.Errorf("Expected trimmed, got '%s'", result)
		}
	})
}

func TestHandleTemplateList(t *testing.T) {
	// Create a temporary templates directory with test data
	tmpDir := t.TempDir()

	// Create test server with mock template service
	srv := &Server{
		router:          mux.NewRouter(),
		templateService: &TemplateService{templatesDir: tmpDir},
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestHandleTemplateShow(t *testing.T) {
	tmpDir := t.TempDir()

	srv := &Server{
		router:          mux.NewRouter(),
		templateService: &TemplateService{templatesDir: tmpDir},
	}
	srv.setupRoutes()

	t.Run("non-existing template returns error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates/non-existing", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestResolveDatabaseURL(t *testing.T) {
	t.Run("explicit DATABASE_URL", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://explicit:5432/db")
		defer os.Unsetenv("DATABASE_URL")

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if url != "postgres://explicit:5432/db" {
			t.Errorf("Expected explicit URL, got %s", url)
		}
	})

	t.Run("constructed from components", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("POSTGRES_HOST", "testhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
		}()

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := "postgres://testuser:testpass@testhost:5432/testdb?sslmode=disable"
		if url != expected {
			t.Errorf("Expected %s, got %s", expected, url)
		}
	})
}

func TestLogStructured(t *testing.T) {
	// Test structured logging (output validation is complex, just ensure no panic)
	logStructured("test_event", map[string]interface{}{
		"key":   "value",
		"count": 42,
	})

	logStructured("test_event_no_fields", nil)
}

func TestHandleGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	tmpTemplatesDir := t.TempDir()

	// Set generation output directory
	os.Setenv("GEN_OUTPUT_DIR", tmpDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	// Create a minimal test template
	tmplPath := tmpTemplatesDir + "/test-template.json"
	tmplContent := `{
		"id": "test-template",
		"name": "Test Template",
		"description": "Test",
		"version": "1.0.0",
		"payload_path": "test-payload"
	}`
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create payload directory with minimal structure
	payloadDir := tmpTemplatesDir + "/test-payload"
	os.MkdirAll(payloadDir+"/.vrooli", 0755)
	os.WriteFile(payloadDir+"/.vrooli/service.json", []byte(`{"name":"template"}`), 0644)
	os.WriteFile(payloadDir+"/test.txt", []byte("test"), 0644)

	srv := &Server{
		router:          mux.NewRouter(),
		templateService: &TemplateService{templatesDir: tmpTemplatesDir},
	}
	srv.router.HandleFunc("/api/v1/generate", srv.handleGenerate).Methods("POST")

	t.Run("dry run mode", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/generate",
			strings.NewReader(`{"template_id":"test-template","name":"Test","slug":"test-dry","options":{"dry_run":true}}`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["status"] != "dry_run" {
			t.Errorf("Expected status dry_run, got %v", resp["status"])
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/generate", strings.NewReader(`{invalid json`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestHandleGeneratedList(t *testing.T) {
	tmpDir := t.TempDir()

	// Set generation output directory
	os.Setenv("GEN_OUTPUT_DIR", tmpDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	// Create test generated scenario
	scenarioDir := tmpDir + "/test-scenario"
	os.MkdirAll(scenarioDir+"/.vrooli", 0755)
	serviceJSON := `{"name": "Test Scenario", "slug": "test-scenario"}`
	os.WriteFile(scenarioDir+"/.vrooli/service.json", []byte(serviceJSON), 0644)

	srv := &Server{
		router:          mux.NewRouter(),
		templateService: &TemplateService{},
	}
	srv.router.HandleFunc("/api/v1/generated", srv.handleGeneratedList).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/generated", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var scenarios []GeneratedScenario
	if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(scenarios))
	}
}

// [REQ:AGENT-TRIGGER] Test agent customization trigger
func TestHandleCustomizeCreatesIssue(t *testing.T) {
	t.Run("REQ:AGENT-TRIGGER", func(t *testing.T) {
		issueCalled := false
		investigateCalled := false

		mockTracker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if strings.HasSuffix(r.URL.Path, "/issues") {
				issueCalled = true
				_ = json.Unmarshal(body, &map[string]interface{}{})
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true, "data": {"issue_id": "ISS-123"}}`))
				return
			}
			if strings.HasSuffix(r.URL.Path, "/investigate") {
				investigateCalled = true
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true, "data": {"run_id": "run-1", "status": "active", "agent_id": "unified-resolver"}}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer mockTracker.Close()

		os.Setenv("APP_ISSUE_TRACKER_API_BASE", mockTracker.URL)
		defer os.Unsetenv("APP_ISSUE_TRACKER_API_BASE")

		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
			httpClient:      &http.Client{Timeout: 5 * time.Second},
		}
		srv.router.HandleFunc("/api/v1/customize", srv.handleCustomize).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/customize", strings.NewReader(`{"scenario_id":"demo","brief":"make it bold","assets":["logo.svg"],"preview":true}`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", w.Code)
		}
		if !issueCalled {
			t.Fatal("expected issue tracker create to be called")
		}
		if !investigateCalled {
			t.Fatal("expected investigation to be triggered")
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["issue_id"] != "ISS-123" {
			t.Fatalf("expected issue_id ISS-123, got %v", resp["issue_id"])
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES] Test persona listing endpoint
func TestHandlePersonaList(t *testing.T) {
	t.Run("REQ:TMPL-AGENT-PROFILES", func(t *testing.T) {
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}
		srv.setupRoutes()

		req := httptest.NewRequest("GET", "/api/v1/personas", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		var personas []Persona
		if err := json.Unmarshal(w.Body.Bytes(), &personas); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have at least one persona
		if len(personas) == 0 {
			t.Error("Expected at least one persona")
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES] Test persona show endpoint
func TestHandlePersonaShow(t *testing.T) {
	t.Run("REQ:TMPL-AGENT-PROFILES", func(t *testing.T) {
		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}
		srv.setupRoutes()

		// Test valid persona
		req := httptest.NewRequest("GET", "/api/v1/personas/minimal-design", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var persona Persona
		if err := json.Unmarshal(w.Body.Bytes(), &persona); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if persona.ID != "minimal-design" {
			t.Errorf("Expected persona ID minimal-design, got %s", persona.ID)
		}

		// Test invalid persona
		req = httptest.NewRequest("GET", "/api/v1/personas/nonexistent", nil)
		w = httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for nonexistent persona, got %d", w.Code)
		}
	})
}

// [REQ:TMPL-PREVIEW-LINKS] Test preview links endpoint
func TestHandlePreviewLinks(t *testing.T) {
	t.Run("REQ:TMPL-PREVIEW-LINKS", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set GEN_OUTPUT_DIR to tmpDir so GetPreviewLinks uses it
		os.Setenv("GEN_OUTPUT_DIR", tmpDir)
		defer os.Unsetenv("GEN_OUTPUT_DIR")

		// Create mock generated scenario structure
		scenarioDir := tmpDir + "/test-scenario"
		os.MkdirAll(scenarioDir+"/.vrooli", 0755)

		// Create service.json with UI_PORT
		serviceJSON := `{
			"name": "test-scenario",
			"description": "Test",
			"version": "1.0.0",
			"ports": {
				"UI_PORT": 12345
			}
		}`
		os.WriteFile(scenarioDir+"/.vrooli/service.json", []byte(serviceJSON), 0644)

		srv := &Server{
			router:          mux.NewRouter(),
			templateService: NewTemplateService(),
		}
		srv.setupRoutes()

		req := httptest.NewRequest("GET", "/api/v1/preview/test-scenario", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var preview map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &preview); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if preview["scenario_id"] != "test-scenario" {
			t.Errorf("Expected scenario ID test-scenario, got %v", preview["scenario_id"])
		}

		if baseURL, ok := preview["base_url"].(string); ok {
			if !strings.Contains(baseURL, "12345") {
				t.Errorf("Expected base_url to contain port 12345, got %s", baseURL)
			}
		} else {
			t.Error("Expected base_url in preview response")
		}
	})
}
