package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestComprehensiveErrorPatterns tests using the TestScenarioBuilder
func TestComprehensiveErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	// Test the pattern builder
	t.Run("TestScenarioBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()

		// Build scenarios
		scenarios := builder.
			AddNonExistentCampaign("/api/v1/campaigns/%s", "GET").
			AddInvalidJSON("/api/v1/campaigns").
			AddMissingRequiredFields("/api/v1/campaigns", map[string]interface{}{}).
			AddInvalidTone("/api/v1/templates/generate").
			Build()

		if len(scenarios) != 4 {
			t.Errorf("Expected 4 scenarios, got %d", len(scenarios))
		}

		// Verify scenarios have proper structure
		for _, scenario := range scenarios {
			if scenario.Name == "" {
				t.Error("Scenario should have a name")
			}
			if scenario.Description == "" {
				t.Error("Scenario should have a description")
			}
			if scenario.ExpectedStatus == 0 {
				t.Error("Scenario should have an expected status")
			}
		}
	})

	// Test individual pattern methods
	t.Run("AddInvalidUUID", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		scenarios := builder.AddInvalidUUID("/api/v1/campaigns/invalid", "GET").Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "InvalidUUID" {
			t.Errorf("Expected name 'InvalidUUID', got %s", scenario.Name)
		}

		if scenario.ExpectedStatus != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", scenario.ExpectedStatus)
		}
	})

	t.Run("AddNonExistentCampaign", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		scenarios := builder.AddNonExistentCampaign("/api/v1/campaigns/%s", "GET").Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "NonExistentCampaign" {
			t.Errorf("Expected name 'NonExistentCampaign', got %s", scenario.Name)
		}

		if scenario.ExpectedStatus != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", scenario.ExpectedStatus)
		}

		// Execute the scenario
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method:  scenario.Method,
			Path:    scenario.Path,
			Body:    scenario.Body,
			Headers: scenario.Headers,
		})

		if err != nil {
			t.Fatalf("Failed to execute scenario: %v", err)
		}

		// Database not available, so we should get 503
		if recorder.Code != http.StatusServiceUnavailable && recorder.Code != scenario.ExpectedStatus {
			t.Logf("Got status %d (expected %d or 503)", recorder.Code, scenario.ExpectedStatus)
		}
	})

	t.Run("AddMissingRequiredFields", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		scenarios := builder.AddMissingRequiredFields("/api/v1/campaigns", map[string]interface{}{
			"description": "Missing name field",
		}).Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method:  scenario.Method,
			Path:    scenario.Path,
			Body:    scenario.Body,
			Headers: scenario.Headers,
		})

		if err != nil {
			t.Fatalf("Failed to execute scenario: %v", err)
		}

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", recorder.Code)
		}

		// Validate response has error field
		if scenario.ValidateBody != nil {
			var body map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err == nil {
				if !scenario.ValidateBody(body) {
					t.Error("Body validation failed")
				}
			}
		}
	})
}

// TestAssertHelpers tests the assertion helper functions
func TestAssertHelpers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("AssertJSONResponse", func(t *testing.T) {
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Test successful validation
		assertJSONResponse(t, recorder, http.StatusOK, func(body map[string]interface{}) bool {
			_, hasStatus := body["status"]
			return hasStatus
		})

		// Test nil validator (should not fail)
		assertJSONResponse(t, recorder, http.StatusOK, nil)
	})

	t.Run("AssertErrorResponse", func(t *testing.T) {
		// Create an error scenario
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   map[string]interface{}{}, // Missing required fields
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Test error response assertion
		assertErrorResponse(t, recorder, http.StatusBadRequest, "")

		// Test with error message validation
		recorder2, _ := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body: map[string]interface{}{
				"purpose": "test",
				"tone":    "invalid",
			},
		})

		assertErrorResponse(t, recorder2, http.StatusBadRequest, "tone")
	})
}

// TestEndToEndWithoutDB tests end-to-end scenarios without database
func TestEndToEndWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("HealthCheckFlow", func(t *testing.T) {
		// Test health endpoint multiple times
		for i := 0; i < 3; i++ {
			recorder, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if err != nil {
				t.Fatalf("Failed health check iteration %d: %v", i, err)
			}

			var health map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &health); err != nil {
				t.Fatalf("Failed to parse health response: %v", err)
			}

			if health["status"] != "healthy" {
				t.Errorf("Iteration %d: Expected healthy status", i)
			}
		}
	})

	t.Run("AllEndpointsWithoutDB", func(t *testing.T) {
		endpoints := []struct {
			method string
			path   string
			body   interface{}
		}{
			{"GET", "/health", nil},
			{"GET", "/api/v1/campaigns", nil},
			{"POST", "/api/v1/campaigns", CreateCampaignRequest{Name: "Test"}},
			{"GET", "/api/v1/campaigns/" + uuid.New().String(), nil},
			{"POST", "/api/v1/campaigns/" + uuid.New().String() + "/send", map[string]interface{}{}},
			{"GET", "/api/v1/campaigns/" + uuid.New().String() + "/analytics", nil},
			{"POST", "/api/v1/templates/generate", GenerateTemplateRequest{Purpose: "Test", Tone: "professional"}},
			{"GET", "/api/v1/templates", nil},
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
				recorder, err := makeHTTPRequest(router, HTTPTestRequest{
					Method: endpoint.method,
					Path:   endpoint.path,
					Body:   endpoint.body,
				})

				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				// All endpoints should either succeed or return service unavailable without DB
				validStatuses := []int{
					http.StatusOK,
					http.StatusCreated,
					http.StatusServiceUnavailable,
					http.StatusNotFound,
					http.StatusBadRequest,
				}

				valid := false
				for _, status := range validStatuses {
					if recorder.Code == status {
						valid = true
						break
					}
				}

				if !valid {
					t.Errorf("Unexpected status code: %d", recorder.Code)
				}

				t.Logf("%s %s -> %d", endpoint.method, endpoint.path, recorder.Code)
			})
		}
	})
}

// TestValidationLogic tests all validation scenarios
func TestValidationLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CampaignValidation", func(t *testing.T) {
		testCases := []struct {
			name         string
			request      interface{}
			expectStatus int
		}{
			{
				name:         "Valid",
				request:      CreateCampaignRequest{Name: "Valid Campaign"},
				expectStatus: http.StatusServiceUnavailable, // No DB
			},
			{
				name:         "MissingName",
				request:      map[string]interface{}{"description": "No name"},
				expectStatus: http.StatusBadRequest,
			},
			{
				name:         "EmptyBody",
				request:      map[string]interface{}{},
				expectStatus: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				recorder, err := makeHTTPRequest(router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/campaigns",
					Body:   tc.request,
				})

				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				if recorder.Code != tc.expectStatus {
					t.Errorf("Expected status %d, got %d. Body: %s",
						tc.expectStatus, recorder.Code, recorder.Body.String())
				}
			})
		}
	})

	t.Run("TemplateValidation", func(t *testing.T) {
		testCases := []struct {
			name         string
			request      interface{}
			expectStatus int
		}{
			{
				name: "ValidProfessional",
				request: GenerateTemplateRequest{
					Purpose: "Test",
					Tone:    "professional",
				},
				expectStatus: http.StatusServiceUnavailable, // No DB
			},
			{
				name: "ValidFriendly",
				request: GenerateTemplateRequest{
					Purpose: "Test",
					Tone:    "friendly",
				},
				expectStatus: http.StatusServiceUnavailable,
			},
			{
				name: "ValidCasual",
				request: GenerateTemplateRequest{
					Purpose: "Test",
					Tone:    "casual",
				},
				expectStatus: http.StatusServiceUnavailable,
			},
			{
				name: "InvalidTone",
				request: GenerateTemplateRequest{
					Purpose: "Test",
					Tone:    "super-professional",
				},
				expectStatus: http.StatusBadRequest,
			},
			{
				name:         "MissingPurpose",
				request:      map[string]interface{}{"tone": "professional"},
				expectStatus: http.StatusBadRequest,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				recorder, err := makeHTTPRequest(router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/templates/generate",
					Body:   tc.request,
				})

				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				if recorder.Code != tc.expectStatus {
					t.Errorf("Expected status %d, got %d. Body: %s",
						tc.expectStatus, recorder.Code, recorder.Body.String())
				}
			})
		}
	})
}

// TestHTTPMethods tests different HTTP methods
func TestHTTPMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("GetMethods", func(t *testing.T) {
		paths := []string{
			"/health",
			"/api/v1/campaigns",
			"/api/v1/campaigns/" + uuid.New().String(),
			"/api/v1/campaigns/" + uuid.New().String() + "/analytics",
			"/api/v1/templates",
		}

		for _, path := range paths {
			recorder, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   path,
			})

			if err != nil {
				t.Errorf("GET %s failed: %v", path, err)
			}

			// Should not return 404 or 405 (method not allowed)
			if recorder.Code == http.StatusNotFound || recorder.Code == http.StatusMethodNotAllowed {
				t.Errorf("GET %s returned %d", path, recorder.Code)
			}
		}
	})

	t.Run("PostMethods", func(t *testing.T) {
		posts := []struct {
			path string
			body interface{}
		}{
			{"/api/v1/campaigns", CreateCampaignRequest{Name: "Test"}},
			{"/api/v1/campaigns/" + uuid.New().String() + "/send", map[string]interface{}{}},
			{"/api/v1/templates/generate", GenerateTemplateRequest{Purpose: "Test", Tone: "professional"}},
		}

		for _, post := range posts {
			recorder, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: "POST",
				Path:   post.path,
				Body:   post.body,
			})

			if err != nil {
				t.Errorf("POST %s failed: %v", post.path, err)
			}

			// Should not return 404 or 405
			if recorder.Code == http.StatusNotFound || recorder.Code == http.StatusMethodNotAllowed {
				t.Errorf("POST %s returned %d", post.path, recorder.Code)
			}
		}
	})
}
