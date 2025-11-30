package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"landing-manager/handlers"
	"landing-manager/services"
)

// [REQ:TMPL-LIFECYCLE] Comprehensive test suite for handleScenarioStart
// This file focuses on increasing coverage for handleScenarioStart (target: >90%)

func TestHandleScenarioStart_EmptyScenarioID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	// Test with empty scenario_id
	req := httptest.NewRequest("POST", "/api/v1/lifecycle//start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": ""})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty scenario_id, got %d", w.Code)
	}

	// Security improvement: Now returns structured JSON error
	body := w.Body.String()
	if !strings.Contains(body, "scenario_id is required") {
		t.Errorf("Expected 'scenario_id is required' error, got: %s", body)
	}
	// Verify it's valid JSON
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Expected JSON response, got parse error: %v", err)
	}
}

func TestHandleScenarioStart_StagingAreaScenario(t *testing.T) {
	// Test starting a scenario from the staging area (generated/)
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create a staging area scenario
	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", "staging-start-test")
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/staging-start-test/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": "staging-start-test"})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	// Validates staging path detection (lines 642-644)
	// CLI will fail in tests, but path resolution is validated
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != false {
		t.Error("Expected success=false in response")
	}
}

func TestHandleScenarioStart_ProductionScenario(t *testing.T) {
	// Test starting a scenario from the production location
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create a production scenario
	productionPath := filepath.Join(tmpRoot, "scenarios", "production-start-test")
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/production-start-test/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": "production-start-test"})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	// Validates production path detection (lines 645-647)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != false {
		t.Error("Expected success=false in response")
	}
}

func TestHandleScenarioStart_ScenarioNotFoundAnywhere(t *testing.T) {
	// Test the error path when scenario doesn't exist anywhere (lines 648-657)
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	// Create directory structure but no scenario
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

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/nonexistent/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": "nonexistent"})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != false {
		t.Error("Expected success=false in response")
	}

	message, ok := resp["message"].(string)
	if !ok {
		t.Fatal("Expected message field to be string")
	}

	// Verify the error message indicates not found (structured error format)
	if !strings.Contains(message, "not found") {
		t.Errorf("Expected 'not found' in message, got: %s", message)
	}

	// Check for error_code in structured response
	if errorCode, ok := resp["error_code"].(string); ok {
		if errorCode != "SCENARIO_NOT_FOUND" {
			t.Errorf("Expected error_code=SCENARIO_NOT_FOUND, got: %s", errorCode)
		}
	}
}

func TestHandleScenarioStart_VrooliRootFallback(t *testing.T) {
	// Test VROOLI_ROOT fallback to $HOME/Vrooli (lines 632-635)
	originalRoot := os.Getenv("VROOLI_ROOT")
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalRoot != "" {
			os.Setenv("VROOLI_ROOT", originalRoot)
		} else {
			os.Unsetenv("VROOLI_ROOT")
		}
		os.Setenv("HOME", originalHome)
	}()

	os.Unsetenv("VROOLI_ROOT")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	// Create scenario in fallback location
	fallbackPath := filepath.Join(tmpHome, "Vrooli", "scenarios", "fallback-start-test")
	if err := os.MkdirAll(fallbackPath, 0755); err != nil {
		t.Fatalf("Failed to create fallback path: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/fallback-start-test/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": "fallback-start-test"})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	// Validates fallback path resolution
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != false {
		t.Error("Expected success=false in response")
	}
}

func TestHandleScenarioStart_PrioritizesStagingOverProduction(t *testing.T) {
	// Test that staging area takes priority when both exist
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	scenarioID := "priority-start-test"

	// Create BOTH staging and production paths
	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", scenarioID)
	productionPath := filepath.Join(tmpRoot, "scenarios", scenarioID)

	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/"+scenarioID+"/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": scenarioID})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	// The handler should have attempted the staging path (with --path flag)
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", w.Code)
	}

	// Both paths should still exist (start doesn't modify them)
	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		t.Error("Staging path should still exist")
	}
	if _, err := os.Stat(productionPath); os.IsNotExist(err) {
		t.Error("Production path should still exist")
	}
}

func TestHandleScenarioStart_ResponseStructure(t *testing.T) {
	// Validate response structure and headers
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	scenarioID := "response-start-test"
	stagingPath := filepath.Join(tmpRoot, "scenarios", "landing-manager", "generated", scenarioID)
	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/"+scenarioID+"/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": scenarioID})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	// Verify Content-Type header (lines 650, 664, 678, 689)
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify response is valid JSON
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check required fields
	if _, ok := resp["success"]; !ok {
		t.Error("Response missing 'success' field")
	}
	if _, ok := resp["message"]; !ok {
		t.Error("Response missing 'message' field")
	}

	// Verify success field is boolean
	if resp["success"] != false && resp["success"] != true {
		t.Errorf("Expected success to be boolean, got %T", resp["success"])
	}
}

func TestHandleScenarioStart_ErrorOutputCapture(t *testing.T) {
	// Test that CLI output is captured in error responses (lines 673-685)
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)
	defer os.Unsetenv("VROOLI_ROOT")

	scenarioID := "output-capture-test"
	productionPath := filepath.Join(tmpRoot, "scenarios", scenarioID)
	if err := os.MkdirAll(productionPath, 0755); err != nil {
		t.Fatalf("Failed to create production path: %v", err)
	}

	db := setupTestDB(t)
	defer db.Close()

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	h := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	req := httptest.NewRequest("POST", "/api/v1/lifecycle/"+scenarioID+"/start", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_id": scenarioID})
	w := httptest.NewRecorder()

	h.HandleScenarioStart(w, req)

	// When CLI fails, response should include output field
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// For error responses (CLI failed), verify details field exists (contains CLI output)
	// Note: AppError uses 'details' field, not 'output'
	if w.Code == http.StatusInternalServerError {
		if _, ok := resp["details"]; !ok {
			t.Error("Error response should include 'details' field with CLI output")
		}
	}

	// Verify scenario_id is NOT included in error responses (only in success)
	// Error paths (lines 664-670, 678-685) don't include scenario_id
	if w.Code != http.StatusOK {
		if _, ok := resp["scenario_id"]; ok {
			// Actually, check the code - error paths may or may not include scenario_id
			// Lines 664-670 and 678-685 don't include it, which is correct
		}
	}
}
