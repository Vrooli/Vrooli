// +build testing

package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestMain(m *testing.M) {
	// Set environment variable to bypass lifecycle check
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "15999")

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) bool {
			// Validate required fields
			if _, ok := data["status"]; !ok {
				t.Error("Missing 'status' field")
				return false
			}
			if _, ok := data["components"]; !ok {
				t.Error("Missing 'components' field")
				return false
			}
			if _, ok := data["active_issues"]; !ok {
				t.Error("Missing 'active_issues' field")
				return false
			}
			if _, ok := data["last_check"]; !ok {
				t.Error("Missing 'last_check' field")
				return false
			}

			// Validate components is an array
			components, ok := data["components"].([]interface{})
			if !ok {
				t.Error("Components is not an array")
				return false
			}

			// Validate each component has required fields
			for i, comp := range components {
				if !validateComponentHealth(t, comp) {
					t.Errorf("Component %d failed validation", i)
					return false
				}
			}

			return true
		})
	})

	t.Run("RootHealthEndpoint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) bool {
			return data["status"] != nil
		})
	})

	t.Run("ComponentHealthStatus", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)
		data := parseJSONResponse(t, w)

		components, ok := data["components"].([]interface{})
		if !ok || len(components) == 0 {
			t.Fatal("Expected at least one component in health check")
		}

		// Verify first component structure
		comp := components[0].(map[string]interface{})
		if comp["component"] == nil {
			t.Error("Component missing 'component' field")
		}
		if comp["status"] == nil {
			t.Error("Component missing 'status' field")
		}
	})
}

func TestListIssuesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)

		data := parseJSONResponse(t, w)
		issues := data["issues"]
		if issues != nil {
			issuesList := issues.([]interface{})
			if len(issuesList) != 0 {
				t.Errorf("Expected empty issues list, got %d issues", len(issuesList))
			}
		}
	})

	t.Run("WithIssues", func(t *testing.T) {
		// Create test issues
		issue1 := NewTestIssue("test-issue-1", "cli").
			WithSeverity("critical").
			WithStatus("active").
			Build()
		createTestIssue(t, env, issue1)

		issue2 := NewTestIssue("test-issue-2", "orchestrator").
			WithSeverity("medium").
			WithStatus("resolved").
			Build()
		createTestIssue(t, env, issue2)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)

		data := parseJSONResponse(t, w)
		issues := data["issues"].([]interface{})
		if len(issues) != 2 {
			t.Errorf("Expected 2 issues, got %d", len(issues))
		}
	})

	t.Run("FilterByComponent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues?component=cli",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)

		data := parseJSONResponse(t, w)
		issues := data["issues"].([]interface{})

		for _, issue := range issues {
			issueMap := issue.(map[string]interface{})
			if issueMap["component"] != "cli" {
				t.Errorf("Expected component 'cli', got '%v'", issueMap["component"])
			}
		}
	})

	t.Run("FilterBySeverity", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues?severity=critical",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)

		data := parseJSONResponse(t, w)
		issues := data["issues"].([]interface{})

		for _, issue := range issues {
			issueMap := issue.(map[string]interface{})
			if issueMap["severity"] != "critical" {
				t.Errorf("Expected severity 'critical', got '%v'", issueMap["severity"])
			}
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues?status=resolved",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)
	})

	t.Run("MultipleFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues?component=cli&severity=critical&status=active",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateIssuesResponse)
	})

	t.Run("InvalidFilter", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues?component=nonexistent",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) bool {
			issues := data["issues"]
			if issues == nil {
				return true // Empty list
			}
			issuesList, ok := issues.([]interface{})
			return ok && len(issuesList) == 0 // Should return empty list
		})
	})
}

func TestGetWorkaroundsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("IssueWithWorkarounds", func(t *testing.T) {
		workaround := createTestWorkaround("test-issue-3")
		issue := NewTestIssue("test-issue-3", "cli").
			WithWorkaround(workaround).
			Build()
		createTestIssue(t, env, issue)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues/test-issue-3/workarounds",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateWorkaroundsResponse)

		data := parseJSONResponse(t, w)
		workarounds := data["workarounds"].([]interface{})
		if len(workarounds) != 1 {
			t.Errorf("Expected 1 workaround, got %d", len(workarounds))
		}
	})

	t.Run("IssueWithoutWorkarounds", func(t *testing.T) {
		issue := NewTestIssue("test-issue-4", "orchestrator").Build()
		createTestIssue(t, env, issue)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues/test-issue-4/workarounds",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateWorkaroundsResponse)

		// Should return common workarounds (empty list in test env)
		data := parseJSONResponse(t, w)
		workarounds := data["workarounds"].([]interface{})
		if len(workarounds) != 0 {
			t.Logf("Got %d common workarounds", len(workarounds))
		}
	})

	t.Run("NonExistentIssue", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/issues/nonexistent-issue/workarounds",
		}

		w := makeHTTPRequest(req, router)
		// Should return common workarounds even if issue doesn't exist
		assertJSONResponse(t, w, http.StatusOK, validateWorkaroundsResponse)
	})
}

func TestAnalyzeIssueHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("ValidIssue", func(t *testing.T) {
		issue := NewTestIssue("test-issue-5", "cli").
			WithSeverity("high").
			WithDescription("Test error description").
			Build()
		createTestIssue(t, env, issue)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/test-issue-5/analyze",
		}

		w := makeHTTPRequest(req, router)
		assertJSONResponse(t, w, http.StatusOK, validateAnalysisResponse)

		// Verify fix attempt was recorded
		updatedIssue := readIssueFile(t, env, "test-issue-5")
		if len(updatedIssue.FixAttempts) != 1 {
			t.Errorf("Expected 1 fix attempt, got %d", len(updatedIssue.FixAttempts))
		}
	})

	t.Run("NonExistentIssue", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/nonexistent-issue/analyze",
		}

		w := makeHTTPRequest(req, router)
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("AnalysisContent", func(t *testing.T) {
		issue := NewTestIssue("test-issue-6", "orchestrator").
			WithDescription("Detailed error description").
			Build()
		createTestIssue(t, env, issue)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues/test-issue-6/analyze",
		}

		w := makeHTTPRequest(req, router)
		data := parseJSONResponse(t, w)

		// Verify analysis fields exist and have content
		analysis, ok := data["analysis"].(string)
		if !ok || analysis == "" {
			t.Error("Expected non-empty analysis string")
		}

		suggestedFix, ok := data["suggested_fix"].(string)
		if !ok || suggestedFix == "" {
			t.Error("Expected non-empty suggested_fix string")
		}

		confidence, ok := data["confidence"].(float64)
		if !ok || confidence <= 0 {
			t.Error("Expected positive confidence value")
		}
	})
}

func TestReportIssueHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("ValidIssue", func(t *testing.T) {
		issueData := map[string]interface{}{
			"component":   "cli",
			"severity":    "critical",
			"description": "Test issue from handler",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}

		w := makeHTTPRequest(req, router)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		data := parseJSONResponse(t, w)

		// Verify issue was created with correct fields
		if data["id"] == nil || data["id"].(string) == "" {
			t.Error("Expected auto-generated ID")
		}
		if data["component"] != "cli" {
			t.Errorf("Expected component 'cli', got '%v'", data["component"])
		}
		if data["severity"] != "critical" {
			t.Errorf("Expected severity 'critical', got '%v'", data["severity"])
		}
		if data["status"] != "active" {
			t.Errorf("Expected status 'active', got '%v'", data["status"])
		}
	})

	t.Run("WithID", func(t *testing.T) {
		issueData := map[string]interface{}{
			"id":          "custom-id-123",
			"component":   "orchestrator",
			"severity":    "medium",
			"description": "Issue with custom ID",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}

		w := makeHTTPRequest(req, router)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		data := parseJSONResponse(t, w)
		if data["id"] != "custom-id-123" {
			t.Errorf("Expected custom ID, got '%v'", data["id"])
		}

		// Verify file was created
		issueFile := filepath.Join(env.DataDir, "issues", "custom-id-123.json")
		if _, err := os.Stat(issueFile); os.IsNotExist(err) {
			t.Error("Issue file was not created")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		issueData := map[string]interface{}{
			"component":   "setup",
			"description": "Minimal issue",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   createJSONBody(t, issueData),
		}

		w := makeHTTPRequest(req, router)
		data := parseJSONResponse(t, w)

		// Check default values were set
		if data["status"] != "active" {
			t.Errorf("Expected default status 'active', got '%v'", data["status"])
		}
		if data["severity"] != "medium" {
			t.Errorf("Expected default severity 'medium', got '%v'", data["severity"])
		}
		if data["occurrence_count"].(float64) != 1 {
			t.Errorf("Expected default occurrence_count 1, got '%v'", data["occurrence_count"])
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   `{"invalid": json}`,
		}

		w := makeHTTPRequest(req, router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/issues",
			Body:   "",
		}

		w := makeHTTPRequest(req, router)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

func TestCheckAllComponents(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("AllComponentsChecked", func(t *testing.T) {
		results := checkAllComponents()

		if len(results) != len(components) {
			t.Errorf("Expected %d component health results, got %d", len(components), len(results))
		}

		for _, result := range results {
			if result.Component == "" {
				t.Error("Component ID should not be empty")
			}
			if result.Status == "" {
				t.Error("Status should not be empty")
			}
			if result.LastCheck.IsZero() {
				t.Error("LastCheck should be set")
			}
		}
	})

	t.Run("HealthStatesSaved", func(t *testing.T) {
		// Clear health log
		healthFile := filepath.Join(env.DataDir, "health", "latest.jsonl")
		os.Remove(healthFile)

		checkAllComponents()

		// Verify health states were saved
		if _, err := os.Stat(healthFile); os.IsNotExist(err) {
			t.Error("Health state file was not created")
		}
	})
}

func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		name     string
		health   []ComponentHealth
		expected string
	}{
		{
			name: "AllHealthy",
			health: []ComponentHealth{
				{Component: "cli", Status: "healthy"},
				{Component: "orchestrator", Status: "healthy"},
			},
			expected: "healthy",
		},
		{
			name: "CriticalDown",
			health: []ComponentHealth{
				{Component: "cli", Status: "unhealthy"},
				{Component: "orchestrator", Status: "healthy"},
			},
			expected: "critical", // First component is critical by default
		},
		{
			name: "NonCriticalDown",
			health: []ComponentHealth{
				{Component: "cli", Status: "healthy"},
				{Component: "test-component", Status: "unhealthy"},
			},
			expected: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineOverallStatus(tt.health)
			if result != tt.expected {
				t.Errorf("Expected status '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CORSHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)

		// Check CORS headers
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", origin)
		}

		methods := w.Header().Get("Access-Control-Allow-Methods")
		if methods == "" {
			t.Error("Access-Control-Allow-Methods header not set")
		}
	})

	t.Run("OPTIONSRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/health",
		}

		w := makeHTTPRequest(req, router)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

func TestLoadComponents(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidComponentFile", func(t *testing.T) {
		originalComponents := components
		loadComponents()

		if len(components) == 0 {
			t.Error("Expected components to be loaded")
		}

		// Verify component structure
		for _, comp := range components {
			if comp.ID == "" {
				t.Error("Component ID should not be empty")
			}
			if comp.Name == "" {
				t.Error("Component Name should not be empty")
			}
			if comp.HealthCheck == "" {
				t.Error("Component HealthCheck should not be empty")
			}
		}

		components = originalComponents
	})

	t.Run("MissingComponentFile", func(t *testing.T) {
		// Temporarily rename components file
		componentsFile := filepath.Join(env.DataDir, "components.json")
		backupFile := componentsFile + ".bak"
		os.Rename(componentsFile, backupFile)
		defer os.Rename(backupFile, componentsFile)

		originalComponents := components
		components = nil

		loadComponents()

		// Should handle missing file gracefully
		if components == nil {
			components = originalComponents
		}
	})
}

// Helper function to set up test router
func setupTestRouter() http.Handler {
	router := mux.NewRouter()

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/issues", listIssuesHandler).Methods("GET")
	api.HandleFunc("/issues", reportIssueHandler).Methods("POST")
	api.HandleFunc("/issues/{id}/workarounds", getWorkaroundsHandler).Methods("GET")
	api.HandleFunc("/issues/{id}/analyze", analyzeIssueHandler).Methods("POST")

	router.HandleFunc("/health", healthHandler).Methods("GET")

	return corsMiddleware(router)
}
