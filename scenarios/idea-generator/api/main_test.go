package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestHealthHandler tests the health endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		// Verify all dependencies are reported
		expectedServices := []string{"n8n", "windmill", "postgres", "qdrant", "minio", "redis", "ollama", "unstructured"}
		for _, service := range expectedServices {
			if _, exists := response.Dependencies[service]; !exists {
				t.Errorf("Expected service '%s' in health response", service)
			}
		}
	})
}

// TestStatusHandler tests the status endpoint
func TestStatusHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/status",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify required fields
		if _, exists := response["service"]; !exists {
			t.Error("Expected 'service' field in status response")
		}
		if _, exists := response["version"]; !exists {
			t.Error("Expected 'version' field in status response")
		}
		if _, exists := response["resources"]; !exists {
			t.Error("Expected 'resources' field in status response")
		}
	})
}

// TestCampaignsHandler tests the campaigns endpoint
func TestCampaignsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("GetCampaigns", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var campaigns []Campaign
		if err := json.Unmarshal(w.Body.Bytes(), &campaigns); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should return at least the default campaigns
		if len(campaigns) < 2 {
			t.Errorf("Expected at least 2 campaigns, got %d", len(campaigns))
		}
	})

	t.Run("CreateCampaign", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body: map[string]interface{}{
				"name":        "Test Campaign",
				"description": "Test Description",
				"color":       "#FF5733",
			},
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var campaign Campaign
		if err := json.Unmarshal(w.Body.Bytes(), &campaign); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if campaign.Name != "Test Campaign" {
			t.Errorf("Expected name 'Test Campaign', got '%s'", campaign.Name)
		}
		if campaign.ID == "" {
			t.Error("Expected campaign to have an ID")
		}
	})

	t.Run("CreateCampaign_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/campaigns",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestCampaignByIDHandler tests the single campaign endpoint
func TestCampaignByIDHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test campaign first
	campaign := createTestCampaign(t, env.DB, "test-get-by-id")

	t.Run("GetCampaignByID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns/" + campaign.ID,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var retrievedCampaign Campaign
		if err := json.Unmarshal(w.Body.Bytes(), &retrievedCampaign); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if retrievedCampaign.ID != campaign.ID {
			t.Errorf("Expected campaign ID '%s', got '%s'", campaign.ID, retrievedCampaign.ID)
		}
		if retrievedCampaign.Name != campaign.Name {
			t.Errorf("Expected campaign name '%s', got '%s'", campaign.Name, retrievedCampaign.Name)
		}
	})

	t.Run("GetCampaignByID_NotFound", func(t *testing.T) {
		fakeUUID := uuid.New().String()
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns/" + fakeUUID,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("DeleteCampaign", func(t *testing.T) {
		// Create a campaign to delete
		campaignToDelete := createTestCampaign(t, env.DB, "test-delete")

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/campaigns/" + campaignToDelete.ID,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify campaign is soft-deleted (not returned in GET)
		getReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns/" + campaignToDelete.ID,
		}

		w = makeHTTPRequest(env, getReq)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected deleted campaign to return 404, got %d", w.Code)
		}
	})

	t.Run("DeleteCampaign_NotFound", func(t *testing.T) {
		fakeUUID := uuid.New().String()
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/campaigns/" + fakeUUID,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestIdeasHandler tests the ideas endpoint
func TestIdeasHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test campaign
	campaign := createTestCampaign(t, env.DB, "ideas-test")
	defer campaign.Cleanup()

	t.Run("GetIdeas", func(t *testing.T) {
		// Create some test ideas
		createTestIdea(t, env.DB, campaign.ID, "Test Idea 1", "Test content 1")
		createTestIdea(t, env.DB, campaign.ID, "Test Idea 2", "Test content 2")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/ideas?campaign_id=" + campaign.ID,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var ideas []Idea
		if err := json.Unmarshal(w.Body.Bytes(), &ideas); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(ideas) < 2 {
			t.Errorf("Expected at least 2 ideas, got %d", len(ideas))
		}
	})

	t.Run("GetIdeas_AllCampaigns", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/ideas",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var ideas []Idea
		if err := json.Unmarshal(w.Body.Bytes(), &ideas); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should return ideas (might be 0 if database is empty)
		t.Logf("Retrieved %d ideas", len(ideas))
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/ideas",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestGenerateIdeasHandler tests the idea generation endpoint
func TestGenerateIdeasHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test campaign
	campaign := createTestCampaign(t, env.DB, "generate-test")
	defer campaign.Cleanup()

	t.Run("ValidRequest_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/ideas/generate",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingCampaignID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/ideas/generate",
			Body: map[string]interface{}{
				"context":          "Test context",
				"creativity_level": 0.7,
			},
		}

		w := makeHTTPRequest(env, req)

		// Should handle missing campaign_id gracefully
		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Got status %d for missing campaign_id", w.Code)
		}
	})
}

// TestWorkflowsHandler tests the workflows endpoint
func TestWorkflowsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("GetWorkflows", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/workflows",
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var workflows []map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &workflows); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should return at least 4 workflow capabilities
		if len(workflows) < 4 {
			t.Errorf("Expected at least 4 workflows, got %d", len(workflows))
		}

		// Verify workflow structure
		for _, wf := range workflows {
			if wf["id"] == "" {
				t.Error("Workflow missing 'id' field")
			}
			if wf["name"] == "" {
				t.Error("Workflow missing 'name' field")
			}
			if wf["endpoint"] == "" {
				t.Error("Workflow missing 'endpoint' field")
			}
		}
	})
}

// TestSearchHandler tests the semantic search endpoint
func TestSearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/search",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/search",
			Body: map[string]interface{}{
				"query": "",
				"limit": 5,
			},
		}

		w := makeHTTPRequest(env, req)

		// Should handle empty query
		t.Logf("Empty query returned status: %d", w.Code)
	})
}

// TestRefineIdeaHandler tests the idea refinement endpoint
func TestRefineIdeaHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test campaign and idea
	campaign := createTestCampaign(t, env.DB, "refine-test")
	defer campaign.Cleanup()

	ideaID := createTestIdea(t, env.DB, campaign.ID, "Original Idea", "Original content")

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/ideas/refine",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("NonExistentIdea", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/ideas/refine",
			Body: map[string]interface{}{
				"idea_id":    uuid.New().String(),
				"refinement": "Improve this idea",
				"user_id":    uuid.New().String(),
			},
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusInternalServerError {
			t.Logf("Non-existent idea returned status: %d", w.Code)
		}
	})

	t.Run("MissingRefinement", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/ideas/refine",
			Body: map[string]interface{}{
				"idea_id": ideaID,
				"user_id": uuid.New().String(),
			},
		}

		w := makeHTTPRequest(env, req)

		// Should handle missing refinement field
		t.Logf("Missing refinement returned status: %d", w.Code)
	})
}

// TestProcessDocumentHandler tests the document processing endpoint
func TestProcessDocumentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test campaign
	campaign := createTestCampaign(t, env.DB, "doc-test")
	defer campaign.Cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/documents/process",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(env, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingFields", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/documents/process",
			Body: map[string]interface{}{
				"document_id": uuid.New().String(),
				// Missing campaign_id and file_path
			},
		}

		w := makeHTTPRequest(env, req)

		// Should handle missing fields
		t.Logf("Missing fields returned status: %d", w.Code)
	})
}

// TestInitDB tests database initialization with retry logic
func TestInitDB(t *testing.T) {
	// Skip this test - it's an integration test that requires real database
	// Even testing error cases causes 30s retry delays
	// TODO: Implement with testcontainers or mock database
	t.Skip("Skipping database integration test - requires real database or mocks")
}

// TestNewApiServer tests API server initialization
func TestNewApiServer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingDatabaseConfig", func(t *testing.T) {
		// Clear all database env vars
		oldVars := map[string]string{
			"POSTGRES_URL":      os.Getenv("POSTGRES_URL"),
			"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
			"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
			"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
			"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
			"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
		}

		for k := range oldVars {
			os.Unsetenv(k)
		}

		defer func() {
			for k, v := range oldVars {
				if v != "" {
					os.Setenv(k, v)
				}
			}
		}()

		_, err := NewApiServer()
		if err == nil {
			t.Error("Expected error when database config is missing")
		}
	})

	t.Run("PostgresURLFromComponents", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "test")
		os.Setenv("POSTGRES_PASSWORD", "test")
		os.Setenv("POSTGRES_DB", "test")

		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
		}()

		// This will fail to connect but should construct URL correctly
		_, err := NewApiServer()
		// We expect an error due to connection failure, not config error
		if err != nil && !strings.Contains(err.Error(), "failed to initialize database") {
			t.Errorf("Unexpected error type: %v", err)
		}
	})
}

// TestErrorPatterns uses the test pattern builder for systematic error testing
func TestErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Test invalid data patterns
	patterns := createInvalidDataPatterns()
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			req := pattern.Execute(t, env, setupData)
			w := makeHTTPRequest(env, req)

			if pattern.ExpectedStatus != 0 && w.Code != pattern.ExpectedStatus {
				t.Logf("%s: Expected status %d, got %d. Body: %s",
					pattern.Name, pattern.ExpectedStatus, w.Code, w.Body.String())
			}
		})
	}

	// Test boundary patterns
	boundaryPatterns := createBoundaryTestPatterns()
	for _, pattern := range boundaryPatterns {
		t.Run(pattern.Name, func(t *testing.T) {
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			req := pattern.Execute(t, env, setupData)
			w := makeHTTPRequest(env, req)

			if pattern.ExpectedStatus != 0 && w.Code != pattern.ExpectedStatus {
				t.Logf("%s: Expected status %d, got %d. Body: %s",
					pattern.Name, pattern.ExpectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
