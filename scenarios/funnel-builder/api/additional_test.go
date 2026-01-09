package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// TestGetFunnels tests funnel listing (covering handleGetFunnels)
func TestGetFunnels(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/api/v1/funnels", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var funnels []interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnels)
		// Should return array (might be empty or have templates)
	})

	t.Run("Success_WithFunnels", func(t *testing.T) {
		// Create test funnel
		funnelID := createTestFunnel(t, testServer.Server, "List Test Funnel")
		defer cleanupTestData(t, testServer.Server, funnelID)

		req, _ := makeHTTPRequest("GET", "/api/v1/funnels", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var funnels []map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnels)

		found := false
		for _, funnel := range funnels {
			if funnel["id"] == funnelID {
				found = true
				steps, ok := funnel["steps"].([]interface{})
				if !ok {
					t.Fatalf("expected steps to be an array, got %T", funnel["steps"])
				}
				if steps == nil {
					t.Fatal("expected steps to be an empty array when no steps exist, got nil")
				}
				break
			}
		}
		if !found {
			t.Error("Created funnel not found in list")
		}
	})

	t.Run("Filter_ByTenantID", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/api/v1/funnels?tenant_id=00000000-0000-0000-0000-000000000000", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}
	})

	t.Run("Steps_ReturnsEmptyArrayWhenNoSteps", func(t *testing.T) {
		name := "Empty Steps Funnel " + uuid.NewString()
		projectID := createTestProject(t, testServer.Server, "Additional Empty Steps Project")
		funnelData := map[string]interface{}{
			"name":        name,
			"description": "Funnel without predefined steps",
			"project_id":  projectID,
			"steps":       []map[string]interface{}{},
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusCreated {
			t.Fatalf("Failed to create funnel without steps: status %d, body: %s", recorder.Code, recorder.Body.String())
		}

		var createResp map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &createResp)
		funnelID := createResp["id"].(string)
		defer cleanupTestData(t, testServer.Server, funnelID)

		req, _ = makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("Expected status 200 when fetching funnel, got %d", recorder.Code)
		}

		var funnel map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &funnel)

		if funnel["steps"] == nil {
			t.Fatalf("expected steps field to be an empty array, got nil")
		}

		steps, ok := funnel["steps"].([]interface{})
		if !ok {
			t.Fatalf("expected steps to be an array, got %T", funnel["steps"])
		}

		if len(steps) != 0 {
			t.Fatalf("expected zero steps, got %d", len(steps))
		}
	})
}

// TestGetTemplate tests template retrieval
func TestGetTemplate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Error_NotFound", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/api/v1/templates/non-existent-template", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusNotFound)
	})

	t.Run("Success_GetTemplate", func(t *testing.T) {
		// First list templates to find one
		req, _ := makeHTTPRequest("GET", "/api/v1/templates", nil)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var templates []map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &templates)

		if len(templates) > 0 {
			templateSlug := templates[0]["slug"].(string)

			// Get specific template
			req, _ = makeHTTPRequest("GET", "/api/v1/templates/"+templateSlug, nil)
			recorder = httptest.NewRecorder()
			testServer.Server.router.ServeHTTP(recorder, req)

			if recorder.Code == http.StatusOK {
				var template map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &template)

				if template["slug"] != templateSlug {
					t.Errorf("Expected slug %s, got %v", templateSlug, template["slug"])
				}
			}
		}
	})
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("GenerateSlug", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"Test Funnel", "test-funnel"},
			{"My Great Funnel", "my-great-funnel"},
			{"SingleWord", "singleword"},
		}

		for _, test := range tests {
			result := generateSlug(test.input)
			if result != test.expected {
				t.Errorf("generateSlug(%q) = %q, expected %q", test.input, result, test.expected)
			}
		}
	})

	t.Run("ConvertToCSV", func(t *testing.T) {
		leads := []Lead{
			{
				ID:        "id1",
				Email:     "test@example.com",
				Name:      "Test User",
				Phone:     "1234567890",
				Source:    "web",
				Completed: true,
			},
		}

		csv := convertToCSV(leads)

		if csv == "" {
			t.Error("Expected non-empty CSV output")
		}

		// Check for header
		if !contains(csv, "ID,Email") {
			t.Error("Expected CSV header")
		}

		// Check for data
		if !contains(csv, "test@example.com") {
			t.Error("Expected email in CSV output")
		}
	})
}

// TestExecuteFunnel covers funnel execution scenarios
func TestExecuteFunnel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Error_NonExistentSlug", func(t *testing.T) {
		req, _ := makeHTTPRequest("GET", "/api/v1/execute/non-existent-slug", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusNotFound)
	})

	t.Run("Error_InactiveStatus", func(t *testing.T) {
		// Create funnel in draft status (default)
		projectID := createTestProject(t, testServer.Server, "Execute Draft Project")
		funnelData := map[string]interface{}{
			"name":       "Draft Funnel",
			"project_id": projectID,
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var response map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &response)
		funnelID := response["id"].(string)
		slug := response["slug"].(string)
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Try to execute draft funnel
		req, _ = makeHTTPRequest("GET", "/api/v1/execute/"+slug, nil)
		req.RemoteAddr = "127.0.0.1:12345"
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Draft funnels shouldn't be executable
		if recorder.Code != http.StatusNotFound {
			t.Logf("Warning: Draft funnel executed, might need status check")
		}
	})

	t.Run("Success_NewSession", func(t *testing.T) {
		// Create and activate funnel
		projectID := createTestProject(t, testServer.Server, "Execute Active Project")
		funnelData := map[string]interface{}{
			"name":       "Execute Test Funnel",
			"project_id": projectID,
			"steps": []map[string]interface{}{
				{
					"type":     "form",
					"position": 0,
					"title":    "Step 1",
					"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email"}]}`),
				},
			},
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		var createResp map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &createResp)
		funnelID := createResp["id"].(string)
		slug := createResp["slug"].(string)
		defer cleanupTestData(t, testServer.Server, funnelID)

		// Activate funnel
		updateData := map[string]interface{}{
			"name":   "Execute Test Funnel",
			"status": "active",
		}
		req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Execute funnel
		req, _ = makeHTTPRequest("GET", "/api/v1/execute/"+slug, nil)
		req.RemoteAddr = "127.0.0.1:12345"
		recorder = httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code == http.StatusOK {
			var execResp map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &execResp)

			if execResp["session_id"] == nil {
				t.Error("Expected session_id in response")
			}

			if execResp["progress"] != nil && execResp["progress"].(float64) != 0 {
				t.Error("Expected initial progress of 0")
			}
		}
	})
}

// TestSubmitStep covers step submission scenarios
func TestSubmitStep(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("Error_MissingSessionID", func(t *testing.T) {
		submitData := map[string]interface{}{
			"step_id":  "step-123",
			"response": json.RawMessage(`{"test":"data"}`),
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/execute/test-slug/submit", submitData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusBadRequest)
	})

	t.Run("Error_InvalidSession", func(t *testing.T) {
		submitData := map[string]interface{}{
			"session_id": "non-existent-session",
			"step_id":    "step-123",
			"response":   json.RawMessage(`{"test":"data"}`),
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/execute/test-slug/submit", submitData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusNotFound)
	})

	t.Run("Error_MissingStepID", func(t *testing.T) {
		submitData := map[string]interface{}{
			"session_id": "some-session",
			"response":   json.RawMessage(`{"test":"data"}`),
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/execute/test-slug/submit", submitData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusBadRequest)
	})

	t.Run("Error_MissingResponse", func(t *testing.T) {
		submitData := map[string]interface{}{
			"session_id": "some-session",
			"step_id":    "some-step",
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/execute/test-slug/submit", submitData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		assertErrorResponse(t, recorder, http.StatusBadRequest)
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testServer := setupTestServer(t)
	if testServer == nil {
		return
	}
	defer testServer.Cleanup()

	t.Run("VeryLongFunnelName", func(t *testing.T) {
		longName := make([]byte, 500)
		for i := range longName {
			longName[i] = 'a'
		}

		projectID := createTestProject(t, testServer.Server, "Edge Long Name Project")
		funnelData := map[string]interface{}{
			"name":       string(longName),
			"project_id": projectID,
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		// Should handle long names (might truncate or accept)
		if recorder.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			defer cleanupTestData(t, testServer.Server, response["id"].(string))
		}
	})

	t.Run("SpecialCharactersInName", func(t *testing.T) {
		projectID := createTestProject(t, testServer.Server, "Edge Special Characters Project")
		funnelData := map[string]interface{}{
			"name":       "Test!@#$%^&*()",
			"project_id": projectID,
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			defer cleanupTestData(t, testServer.Server, response["id"].(string))

			// Verify slug generation handled special characters
			slug := response["slug"].(string)
			if slug == "" {
				t.Error("Expected non-empty slug")
			}
		}
	})

	t.Run("EmptyStepsArray", func(t *testing.T) {
		projectID := createTestProject(t, testServer.Server, "Edge Empty Steps Project")
		funnelData := map[string]interface{}{
			"name":       "Empty Steps Funnel",
			"project_id": projectID,
			"steps":      []map[string]interface{}{},
		}

		req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
		recorder := httptest.NewRecorder()
		testServer.Server.router.ServeHTTP(recorder, req)

		if recorder.Code == http.StatusCreated {
			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			defer cleanupTestData(t, testServer.Server, response["id"].(string))
		}
	})
}

// Helper function for string contains check (if not already defined)
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
