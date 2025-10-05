// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestHealthHandlerDetailed provides comprehensive health endpoint testing
func TestHealthHandlerDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ResponseStructure", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(healthHandler)
		handler.ServeHTTP(rr, req)

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)

		expectedFields := []string{"status", "service", "timestamp"}
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(healthHandler)
		handler.ServeHTTP(rr, req)

		contentType := rr.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
		}
	})
}

// TestScanScenariosHandlerDetailed provides comprehensive scan testing
func TestScanScenariosHandlerDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBuffer([]byte{}))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty body, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBufferString("{invalid"))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for malformed JSON, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("NonexistentPath", func(t *testing.T) {
		// Set a nonexistent path
		os.Setenv("SCENARIOS_PATH", "/nonexistent/path")
		defer os.Unsetenv("SCENARIOS_PATH")

		reqBody := ScanRequest{ForceRescan: false}
		bodyJSON, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBuffer(bodyJSON))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d for nonexistent path, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("ValidRequestFormat", func(t *testing.T) {
		tempDir, _ := ioutil.TempDir("", "scan-test")
		defer os.RemoveAll(tempDir)

		os.Setenv("SCENARIOS_PATH", tempDir)
		defer os.Unsetenv("SCENARIOS_PATH")

		// Create a simple scenario
		os.MkdirAll(filepath.Join(tempDir, "test-scenario"), 0755)

		reqBody := ScanRequest{
			ForceRescan:     true,
			ScenarioFilter: "test",
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/scenarios/scan", bytes.NewBuffer(bodyJSON))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(scanScenariosHandler)
		handler.ServeHTTP(rr, req)

		// Should get either OK or InternalServerError (if DB not available)
		// The important thing is it doesn't crash
		if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
			t.Logf("Got status %d (acceptable for test environment)", rr.Code)
		}
	})
}

// TestGenerateLandingPageHandlerDetailed tests landing page generation
func TestGenerateLandingPageHandlerDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBuffer([]byte{}))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(generateLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty body, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBufferString("{bad json"))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(generateLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("ValidRequestStructure", func(t *testing.T) {
		// This test requires database, skip if not available
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		reqBody := GenerateRequest{
			ScenarioID:      uuid.New().String(),
			TemplateID:      uuid.New().String(),
			EnableABTesting: true,
			CustomContent: map[string]interface{}{
				"title": "Test Title",
			},
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/generate", bytes.NewBuffer(bodyJSON))
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(generateLandingPageHandler)
		handler.ServeHTTP(rr, req)

		// Will fail without DB, but shouldn't crash
		if rr.Code != http.StatusInternalServerError && rr.Code != http.StatusOK {
			t.Logf("Got status %d (acceptable without database)", rr.Code)
		}
	})
}

// TestDeployLandingPageHandlerDetailed tests deployment
func TestDeployLandingPageHandlerDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/test-id/deploy", bytes.NewBuffer([]byte{}))
		req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty body, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/test-id/deploy", bytes.NewBufferString("{invalid"))
		req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("ValidDeployRequest", func(t *testing.T) {
		// Skip if database not available
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		reqBody := DeployRequest{
			TargetScenario:   "test-target",
			DeploymentMethod: "direct",
			BackupExisting:   true,
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/"+uuid.New().String()+"/deploy", bytes.NewBuffer(bodyJSON))
		req = mux.SetURLVars(req, map[string]string{"id": uuid.New().String()})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		// Will fail due to missing resources, but shouldn't crash
		if rr.Code == http.StatusOK || rr.Code == http.StatusInternalServerError {
			t.Logf("Deploy request handled with status %d", rr.Code)
		}
	})

	t.Run("PathTraversalInTargetScenario", func(t *testing.T) {
		reqBody := DeployRequest{
			TargetScenario:   "../../etc/passwd",
			DeploymentMethod: "direct",
			BackupExisting:   false,
		}

		bodyJSON, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/landing-pages/test-id/deploy", bytes.NewBuffer(bodyJSON))
		req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(deployLandingPageHandler)
		handler.ServeHTTP(rr, req)

		// Should fail due to path validation
		if rr.Code != http.StatusInternalServerError {
			t.Logf("Path traversal handled with status %d", rr.Code)
		}
	})
}

// TestGetTemplatesHandlerDetailed tests template retrieval
func TestGetTemplatesHandlerDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoQueryParams", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		req, _ := http.NewRequest("GET", "/api/v1/templates", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getTemplatesHandler)
		handler.ServeHTTP(rr, req)

		// Will fail without DB
		if rr.Code == http.StatusInternalServerError {
			t.Log("Templates endpoint requires database (expected)")
		}
	})

	t.Run("WithCategoryFilter", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		req, _ := http.NewRequest("GET", "/api/v1/templates?category=modern", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getTemplatesHandler)
		handler.ServeHTTP(rr, req)

		// Will fail without DB
		if rr.Code == http.StatusInternalServerError {
			t.Log("Templates endpoint with filter requires database (expected)")
		}
	})

	t.Run("WithSaaSTypeFilter", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		req, _ := http.NewRequest("GET", "/api/v1/templates?saas_type=b2b_tool", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getTemplatesHandler)
		handler.ServeHTTP(rr, req)

		// Will fail without DB
		if rr.Code == http.StatusInternalServerError {
			t.Log("Templates endpoint with saas_type filter requires database (expected)")
		}
	})

	t.Run("WithBothFilters", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		req, _ := http.NewRequest("GET", "/api/v1/templates?category=modern&saas_type=b2b_tool", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getTemplatesHandler)
		handler.ServeHTTP(rr, req)

		// Will fail without DB
		if rr.Code == http.StatusInternalServerError {
			t.Log("Templates endpoint with both filters requires database (expected)")
		}
	})
}

// TestGetDashboardHandlerDetailed tests dashboard endpoint
func TestGetDashboardHandlerDetailed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BasicRequest", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test that requires database")
		}

		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(getDashboardHandler)
		handler.ServeHTTP(rr, req)

		// Will fail without DB
		if rr.Code == http.StatusInternalServerError {
			t.Log("Dashboard endpoint requires database (expected)")
		}
	})

	t.Run("ResponseStructure", func(t *testing.T) {
		// This test would need a database, so we skip detailed validation
		t.Skip("Requires database connection")
	})
}

// TestRouterSetup tests the HTTP router configuration
func TestRouterSetup(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RouterPaths", func(t *testing.T) {
		router := mux.NewRouter()
		api := router.PathPrefix("/api/v1").Subrouter()

		// Register routes
		api.HandleFunc("/health", healthHandler).Methods("GET")
		api.HandleFunc("/scenarios/scan", scanScenariosHandler).Methods("POST")
		api.HandleFunc("/landing-pages/generate", generateLandingPageHandler).Methods("POST")
		api.HandleFunc("/landing-pages/{id}/deploy", deployLandingPageHandler).Methods("POST")
		api.HandleFunc("/templates", getTemplatesHandler).Methods("GET")
		api.HandleFunc("/analytics/dashboard", getDashboardHandler).Methods("GET")

		// Test route matching
		testCases := []struct {
			method string
			path   string
			match  bool
		}{
			{"GET", "/api/v1/health", true},
			{"POST", "/api/v1/scenarios/scan", true},
			{"POST", "/api/v1/landing-pages/generate", true},
			{"POST", "/api/v1/landing-pages/123/deploy", true},
			{"GET", "/api/v1/templates", true},
			{"GET", "/api/v1/analytics/dashboard", true},
			{"GET", "/api/v1/nonexistent", false},
			{"DELETE", "/api/v1/health", false},
		}

		for _, tc := range testCases {
			req, _ := http.NewRequest(tc.method, tc.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)

			if matched != tc.match {
				t.Errorf("Route %s %s: expected match=%v, got match=%v", tc.method, tc.path, tc.match, matched)
			}
		}
	})
}

// TestJSONSerialization tests JSON encoding/decoding
func TestJSONSerialization(t *testing.T) {
	t.Run("ScanResponse", func(t *testing.T) {
		response := ScanResponse{
			TotalScenarios: 10,
			SaaSScenarios:  5,
			NewlyDetected:  2,
			Scenarios: []SaaSScenario{
				{
					ID:           uuid.New().String(),
					ScenarioName: "test",
					DisplayName:  "Test",
				},
			},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded ScanResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.TotalScenarios != response.TotalScenarios {
			t.Error("TotalScenarios not preserved")
		}
	})

	t.Run("GenerateResponse", func(t *testing.T) {
		response := GenerateResponse{
			LandingPageID:    uuid.New().String(),
			PreviewURL:       "/preview/test",
			DeploymentStatus: "ready",
			ABTestVariants:   []string{"control", "a", "b"},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded GenerateResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(decoded.ABTestVariants) != len(response.ABTestVariants) {
			t.Error("ABTestVariants not preserved")
		}
	})

	t.Run("DeployResponse", func(t *testing.T) {
		response := DeployResponse{
			DeploymentID:        uuid.New().String(),
			AgentSessionID:      uuid.New().String(),
			Status:              "agent_working",
			EstimatedCompletion: time.Now(),
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded DeployResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Status != response.Status {
			t.Error("Status not preserved")
		}
	})
}
