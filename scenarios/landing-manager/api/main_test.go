package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"landing-manager/handlers"
	"landing-manager/internal/agentmanager"
	"landing-manager/services"
	"landing-manager/util"
)

// fakeAgentRunner is a test double for handlers.AgentRunner.
type fakeAgentRunner struct {
	runID       string
	createErr   error
	getErr      error
	getRun      *domainpb.Run
	createCalls int
	lastReq     agentmanager.RunRequest
}

func (f *fakeAgentRunner) CreateRun(_ context.Context, req agentmanager.RunRequest) (string, error) {
	f.createCalls++
	f.lastReq = req
	if f.createErr != nil {
		return "", f.createErr
	}
	return f.runID, nil
}

func (f *fakeAgentRunner) GetRun(_ context.Context, _ string) (*domainpb.Run, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getRun, nil
}

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

// NOTE: env-var resolution and database-URL construction moved into the shared
// api-core database package during the api-core upgrade; the former local
// helpers (requireEnv, resolveDatabaseURL) no longer exist here, so their unit
// tests were removed along with them.

func TestHandleTemplateList(t *testing.T) {
	t.Run("success path", func(t *testing.T) {
		// Create a temporary templates directory with test data
		tmpDir := t.TempDir()

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistryWithDir(tmpDir)
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(tmpDir)
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
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
	})

	t.Run("error when directory not readable", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		// Use an invalid path to trigger error
		registry := services.NewTemplateRegistryWithDir("/nonexistent/path/that/does/not/exist")
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService("/nonexistent")
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.setupRoutes()

		req := httptest.NewRequest("GET", "/api/v1/templates", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

func TestHandleTemplateShow(t *testing.T) {
	tmpDir := t.TempDir()

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistryWithDir(tmpDir)
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(tmpDir)
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	srv := &Server{
		router:  mux.NewRouter(),
		handler: h,
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

func TestLogStructured(t *testing.T) {
	// Test structured logging (output validation is complex, just ensure no panic)
	util.LogStructured("test_event", map[string]interface{}{
		"key":   "value",
		"count": 42,
	})

	util.LogStructured("test_event_no_fields", nil)
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

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistryWithDir(tmpTemplatesDir)
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(tmpTemplatesDir)
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	srv := &Server{
		router:  mux.NewRouter(),
		handler: h,
	}
	srv.router.HandleFunc("/api/v1/generate", srv.handler.HandleGenerate).Methods("POST")

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
	t.Run("success with generated scenarios", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set generation output directory
		os.Setenv("GEN_OUTPUT_DIR", tmpDir)
		defer os.Unsetenv("GEN_OUTPUT_DIR")

		// Create test generated scenario
		scenarioDir := tmpDir + "/test-scenario"
		os.MkdirAll(scenarioDir+"/.vrooli", 0755)
		serviceJSON := `{"name": "Test Scenario", "slug": "test-scenario"}`
		os.WriteFile(scenarioDir+"/.vrooli/service.json", []byte(serviceJSON), 0644)

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/generated", srv.handler.HandleGeneratedList).Methods("GET")

		req := httptest.NewRequest("GET", "/api/v1/generated", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var scenarios []services.GeneratedScenario
		if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(scenarios))
		}
	})

	t.Run("returns empty list when output directory does not exist", func(t *testing.T) {
		// Set nonexistent output directory
		os.Setenv("GEN_OUTPUT_DIR", "/nonexistent/path/that/does/not/exist")
		defer os.Unsetenv("GEN_OUTPUT_DIR")

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/generated", srv.handler.HandleGeneratedList).Methods("GET")

		req := httptest.NewRequest("GET", "/api/v1/generated", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		// Should return 200 with empty list when directory doesn't exist
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var scenarios []services.GeneratedScenario
		if err := json.Unmarshal(w.Body.Bytes(), &scenarios); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios for nonexistent directory, got %d", len(scenarios))
		}
	})
}

// [REQ:AGENT-TRIGGER] Test agent customization trigger
func TestHandleCustomizeCreatesIssue(t *testing.T) {
	if wd, err := os.Getwd(); err == nil {
		repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(wd)))
		t.Setenv("VROOLI_ROOT", repoRoot)
	}

	t.Run("REQ:AGENT-TRIGGER", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandlerWithHTTPClient(db, registry, generator, personaService, previewService, analyticsService,
			&http.Client{Timeout: 5 * time.Second})
		fake := &fakeAgentRunner{runID: "run-1"}
		h.AgentManager = fake

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/customize", srv.handler.HandleCustomize).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/customize", strings.NewReader(`{"scenario_id":"demo","brief":"make it bold","assets":["logo.svg"],"preview":true}`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d", w.Code)
		}
		if fake.createCalls != 1 {
			t.Fatalf("expected exactly one CreateRun call, got %d", fake.createCalls)
		}
		if fake.lastReq.ScenarioID != "demo" {
			t.Fatalf("expected run scenario demo, got %q", fake.lastReq.ScenarioID)
		}
		if !strings.Contains(fake.lastReq.Prompt, "make it bold") {
			t.Fatalf("expected prompt to carry brief, got %q", fake.lastReq.Prompt)
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp["run_id"] != "run-1" {
			t.Fatalf("expected run_id run-1, got %v", resp["run_id"])
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandlerWithHTTPClient(db, registry, generator, personaService, previewService, analyticsService,
			&http.Client{Timeout: 5 * time.Second})

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/customize", srv.handler.HandleCustomize).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/customize", strings.NewReader(`{invalid json`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("agent-manager unavailable", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandlerWithHTTPClient(db, registry, generator, personaService, previewService, analyticsService,
			&http.Client{Timeout: 5 * time.Second})
		h.AgentManager = nil // no runner configured -> 502

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/customize", srv.handler.HandleCustomize).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/customize", strings.NewReader(`{"scenario_id":"demo","brief":"make it bold"}`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadGateway {
			t.Errorf("Expected status 502, got %d", w.Code)
		}
	})

	t.Run("agent-manager run creation fails", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandlerWithHTTPClient(db, registry, generator, personaService, previewService, analyticsService,
			&http.Client{Timeout: 5 * time.Second})
		h.AgentManager = &fakeAgentRunner{createErr: fmt.Errorf("boom")}

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/customize", srv.handler.HandleCustomize).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/customize", strings.NewReader(`{"scenario_id":"demo","brief":"make it bold"}`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusBadGateway {
			t.Errorf("Expected status 502, got %d", w.Code)
		}
	})

	t.Run("with persona_id included", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandlerWithHTTPClient(db, registry, generator, personaService, previewService, analyticsService,
			&http.Client{Timeout: 5 * time.Second})
		h.AgentManager = &fakeAgentRunner{runID: "run-2"}

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.router.HandleFunc("/api/v1/customize", srv.handler.HandleCustomize).Methods("POST")

		req := httptest.NewRequest("POST", "/api/v1/customize", strings.NewReader(`{"scenario_id":"demo","brief":"make it bold","persona_id":"minimal-design"}`))
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES] Test persona listing endpoint
func TestHandlePersonaList(t *testing.T) {
	t.Run("REQ:TMPL-AGENT-PROFILES", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
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

		var personas []services.Persona
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
		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
		}
		srv.setupRoutes()

		// Test valid persona
		req := httptest.NewRequest("GET", "/api/v1/personas/minimal-design", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var persona services.Persona
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
// NOTE: GetPreviewLinks now uses `vrooli scenario port` command which requires a running scenario.
// This test is skipped in favor of E2E playbook tests in bas/
func TestHandlePreviewLinks(t *testing.T) {
	t.Skip("Requires running scenarios - tested via E2E playbooks")
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

		db := setupTestDB(t)
		defer db.Close()

		registry := services.NewTemplateRegistry()
		generator := services.NewScenarioGenerator(registry)
		personaService := services.NewPersonaService(registry.GetTemplatesDir())
		previewService := services.NewPreviewService()
		analyticsService := services.NewAnalyticsService()

		h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

		srv := &Server{
			router:  mux.NewRouter(),
			handler: h,
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
