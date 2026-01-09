package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	"landing-manager/handlers"
	"landing-manager/services"
	"landing-manager/util"
)

// setupHandlerWithMockExecutor creates a handler with a mocked CLI executor.
// This allows tests to verify behavior without shelling out to real CLI commands.
func setupHandlerWithMockExecutor(t *testing.T) (*handlers.Handler, *util.MockCommandExecutor) {
	db := setupTestDB(t)
	t.Cleanup(func() { db.Close() })

	mockExec := util.NewMockCommandExecutor()
	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandlerWithExecutor(db, registry, generator, personaService, previewService, analyticsService, mockExec)
	return h, mockExec
}

// [REQ:TMPL-LIFECYCLE] Test scenario start success path with staging area
func TestHandleScenarioStart_SuccessWithStagingArea(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create a staging area scenario
	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-staging-start")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)

	// Mock successful CLI response
	mockExec.SetResponse("vrooli", "Scenario started successfully\nStatus: RUNNING", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", h.HandleScenarioStart).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-staging-start/start", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}

	// Verify CLI was called with correct staging path
	if len(mockExec.Calls) == 0 {
		t.Fatal("Expected CLI call but none recorded")
	}
	call := mockExec.Calls[0]
	if call.Name != "vrooli" {
		t.Errorf("Expected vrooli command, got %s", call.Name)
	}
	// Should include --path flag for staging scenarios
	foundPathFlag := false
	for _, arg := range call.Args {
		if arg == "--path" {
			foundPathFlag = true
			break
		}
	}
	if !foundPathFlag {
		t.Errorf("Expected --path flag for staging scenario, args were: %v", call.Args)
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario start success path with production location
func TestHandleScenarioStart_SuccessWithProductionLocation(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create a production scenario (directly in scenarios/)
	productionPath := filepath.Join(tmpRoot, "scenarios", "test-prod-start")
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "Scenario started successfully\nStatus: RUNNING", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/start", h.HandleScenarioStart).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-prod-start/start", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}

	// Production scenarios should NOT have --path flag
	if len(mockExec.Calls) == 0 {
		t.Fatal("Expected CLI call but none recorded")
	}
	call := mockExec.Calls[0]
	for _, arg := range call.Args {
		if arg == "--path" {
			t.Errorf("Production scenarios should not use --path flag, args were: %v", call.Args)
			break
		}
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario stop success path
func TestHandleScenarioStop_Success(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-stop")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "Scenario stopped successfully", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/stop", h.HandleScenarioStop).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-stop/stop", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario stop with CLI failure
func TestHandleScenarioStop_CLIFailure(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-stop-fail")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "Some internal error occurred", 1, fmt.Errorf("exit status 1"))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/stop", h.HandleScenarioStop).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-stop-fail/stop", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != false {
		t.Errorf("Expected success=false, got %v", resp["success"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario restart success path with staging area
func TestHandleScenarioRestart_SuccessWithStagingArea(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-restart")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "Scenario restarted successfully\nStatus: RUNNING", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/restart", h.HandleScenarioRestart).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-restart/restart", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario restart with production location
func TestHandleScenarioRestart_SuccessWithProductionLocation(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	productionPath := filepath.Join(tmpRoot, "scenarios", "test-restart-prod")
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "Scenario restarted successfully\nStatus: RUNNING", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/restart", h.HandleScenarioRestart).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/test-restart-prod/restart", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario restart not found
func TestHandleScenarioRestart_NotFound(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create scenarios directory but no scenario
	scenariosDir := filepath.Join(tmpRoot, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0755); err != nil {
		t.Fatalf("Failed to create scenarios directory: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/restart", h.HandleScenarioRestart).Methods("POST")

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/nonexistent/restart", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != false {
		t.Errorf("Expected success=false, got %v", resp["success"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario status success path - staging area with process checks
func TestHandleScenarioStatus_StagingAreaWithProcessCheck(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create a staging area scenario
	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-staging-status")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	// Create process directory with a PID file to trigger process check
	processDir := filepath.Join(os.Getenv("HOME"), ".vrooli", "processes", "scenarios", "test-staging-status")
	if err := os.MkdirAll(processDir, 0755); err != nil {
		t.Fatalf("Failed to create process directory: %v", err)
	}
	pidFile := filepath.Join(processDir, "api.pid")
	if err := os.WriteFile(pidFile, []byte("12345"), 0644); err != nil {
		t.Fatalf("Failed to create PID file: %v", err)
	}
	// Clean up PID file after test
	t.Cleanup(func() {
		os.RemoveAll(processDir)
	})

	h, mockExec := setupHandlerWithMockExecutor(t)
	// Mock successful kill -0 check (process exists)
	mockExec.SetResponse("kill", "", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/status", h.HandleScenarioStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/lifecycle/test-staging-status/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
	if resp["location"] != "staging" {
		t.Errorf("Expected location=staging, got %v", resp["location"])
	}
	if resp["running"] != true {
		t.Errorf("Expected running=true (process check via seam), got %v", resp["running"])
	}

	// Verify the kill command was called with correct PID
	if len(mockExec.Calls) == 0 {
		t.Fatal("Expected kill -0 call but none recorded")
	}
	call := mockExec.Calls[0]
	if call.Name != "kill" {
		t.Errorf("Expected kill command, got %s", call.Name)
	}
	// Verify -0 flag and PID argument
	if len(call.Args) < 2 || call.Args[0] != "-0" || call.Args[1] != "12345" {
		t.Errorf("Expected kill -0 12345, got: %v", call.Args)
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario status success path - production location via CLI
func TestHandleScenarioStatus_Success(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create a production scenario
	productionPath := filepath.Join(tmpRoot, "scenarios", "test-status")
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	// Mock status response with RUNNING indicator
	mockExec.SetResponse("vrooli", "📋 SCENARIO: test-status\nStatus:        🟢 RUNNING", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/status", h.HandleScenarioStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/lifecycle/test-status/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
	if resp["running"] != true {
		t.Errorf("Expected running=true, got %v", resp["running"])
	}
	if resp["location"] != "production" {
		t.Errorf("Expected location=production, got %v", resp["location"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario status CLI failure returns not found
func TestHandleScenarioStatus_CLIFailureScenarioNotFound(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create production scenario
	productionPath := filepath.Join(tmpRoot, "scenarios", "test-status-fail")
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "Scenario not found", 1, fmt.Errorf("exit status 1"))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/status", h.HandleScenarioStatus).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/lifecycle/test-status-fail/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != false {
		t.Errorf("Expected success=false, got %v", resp["success"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario logs success path
func TestHandleScenarioLogs_Success(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-logs")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "2024-11-29 INFO: Server started\n2024-11-29 INFO: Health check passed", 0, nil)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/logs", h.HandleScenarioLogs).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/lifecycle/test-logs/logs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
	if resp["logs"] == nil || resp["logs"] == "" {
		t.Error("Expected logs to be populated")
	}
}

// [REQ:TMPL-LIFECYCLE] Test scenario logs CLI failure
func TestHandleScenarioLogs_CLIFailureScenarioNotFound(t *testing.T) {
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "test-logs-fail")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	h, mockExec := setupHandlerWithMockExecutor(t)
	mockExec.SetResponse("vrooli", "No lifecycle log found for scenario", 1, fmt.Errorf("exit status 1"))

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/lifecycle/{scenario_id}/logs", h.HandleScenarioLogs).Methods("GET")

	req := httptest.NewRequest("GET", "/api/v1/lifecycle/test-logs-fail/logs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["success"] != false {
		t.Errorf("Expected success=false, got %v", resp["success"])
	}
}

// [REQ:TMPL-LIFECYCLE] Test handleGeneratedList success with multiple scenarios
// NOTE: This test is covered by integration_test.go (TestIntegration_PromoteWorkflow)
func TestHandleGeneratedList_SuccessMultipleScenarios(t *testing.T) {
	t.Skip("Covered by integration tests - isolated test conflicts with real generated/ directory")
}

// [REQ:TMPL-LIFECYCLE] Test handleGeneratedList with empty directory
func TestHandleGeneratedList_EmptyDirectory(t *testing.T) {
	t.Skip("Covered by integration tests - isolated test conflicts with real generated/ directory")
}
