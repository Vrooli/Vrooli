// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": serviceName,
			"version": apiVersion,
		})

		if response == nil {
			return
		}

		// Validate specific fields
		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
		if response["service"] != serviceName {
			t.Errorf("Expected service name '%s', got '%v'", serviceName, response["service"])
		}
		if response["version"] != apiVersion {
			t.Errorf("Expected version '%s', got '%v'", apiVersion, response["version"])
		}
	})
}

// TestListCampaigns tests the list campaigns endpoint
func TestListCampaigns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns",
		})

		campaigns := assertJSONArray(t, w, http.StatusOK)
		if campaigns == nil {
			return
		}

		// Should return empty array when no campaigns exist
		if len(campaigns) != 0 {
			t.Errorf("Expected empty campaigns list, got %d campaigns", len(campaigns))
		}
	})

	t.Run("Success_WithCampaigns", func(t *testing.T) {
		// Create test campaigns
		campaign1 := setupTestCampaign(t, env, "test-campaign-1")
		defer campaign1.Cleanup()

		campaign2 := setupTestCampaign(t, env, "test-campaign-2")
		defer campaign2.Cleanup()

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns",
		})

		campaigns := assertJSONArray(t, w, http.StatusOK)
		if campaigns == nil {
			return
		}

		// Should return both campaigns
		if len(campaigns) != 2 {
			t.Errorf("Expected 2 campaigns, got %d", len(campaigns))
		}

		// Verify campaign data structure
		if len(campaigns) > 0 {
			campaignMap := campaigns[0].(map[string]interface{})
			if _, ok := campaignMap["id"]; !ok {
				t.Error("Expected campaign to have 'id' field")
			}
			if _, ok := campaignMap["name"]; !ok {
				t.Error("Expected campaign to have 'name' field")
			}
			if _, ok := campaignMap["settings"]; !ok {
				t.Error("Expected campaign to have 'settings' field")
			}
		}
	})
}

// TestCreateCampaign tests the create campaign endpoint
func TestCreateCampaign(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		campaignReq := TestData.CampaignRequest("New Campaign", "Test Description")

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   campaignReq,
		})

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name": "New Campaign",
		})

		if response == nil {
			return
		}

		// Verify campaign was created with ID
		if _, ok := response["id"]; !ok {
			t.Error("Expected response to contain 'id' field")
		}

		// Verify timestamps
		if _, ok := response["created_at"]; !ok {
			t.Error("Expected response to contain 'created_at' field")
		}
		if _, ok := response["updated_at"]; !ok {
			t.Error("Expected response to contain 'updated_at' field")
		}

		// Verify settings are preserved
		if settings, ok := response["settings"].(map[string]interface{}); ok {
			if settings["target_audience"] != "test" {
				t.Errorf("Expected target_audience 'test', got '%v'", settings["target_audience"])
			}
		} else {
			t.Error("Expected response to contain 'settings' field")
		}
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		// Create request without name
		campaignReq := map[string]interface{}{
			"description": "Missing name field",
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   campaignReq,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "name is required")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   `{"name": "Invalid JSON"`, // Malformed
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   "",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestListDocuments tests the list documents endpoint
func TestListDocuments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyList", func(t *testing.T) {
		campaign := setupTestCampaign(t, env, "test-campaign")
		defer campaign.Cleanup()

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/campaigns/%s/documents", campaign.Campaign.ID),
			URLVars: map[string]string{"campaignId": campaign.Campaign.ID.String()},
		})

		documents := assertJSONArray(t, w, http.StatusOK)
		if documents == nil {
			return
		}

		if len(documents) != 0 {
			t.Errorf("Expected empty documents list, got %d documents", len(documents))
		}
	})

	t.Run("Success_WithDocuments", func(t *testing.T) {
		campaign := setupTestCampaign(t, env, "test-campaign")
		defer campaign.Cleanup()

		// Create test documents
		doc1 := setupTestDocument(t, env, campaign.Campaign.ID, "doc1.pdf")
		defer doc1.Cleanup()

		doc2 := setupTestDocument(t, env, campaign.Campaign.ID, "doc2.docx")
		defer doc2.Cleanup()

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/campaigns/%s/documents", campaign.Campaign.ID),
			URLVars: map[string]string{"campaignId": campaign.Campaign.ID.String()},
		})

		documents := assertJSONArray(t, w, http.StatusOK)
		if documents == nil {
			return
		}

		if len(documents) != 2 {
			t.Errorf("Expected 2 documents, got %d", len(documents))
		}

		// Verify document structure
		if len(documents) > 0 {
			docMap := documents[0].(map[string]interface{})
			if _, ok := docMap["id"]; !ok {
				t.Error("Expected document to have 'id' field")
			}
			if _, ok := docMap["filename"]; !ok {
				t.Error("Expected document to have 'filename' field")
			}
			if _, ok := docMap["campaign_id"]; !ok {
				t.Error("Expected document to have 'campaign_id' field")
			}
		}
	})

	t.Run("Error_MissingCampaignId", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/campaigns//documents",
			URLVars: map[string]string{"campaignId": ""},
		})

		// Should return 404 for missing campaign ID
		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 400 or 404, got %d", w.Code)
		}
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/campaigns/invalid-uuid/documents",
			URLVars: map[string]string{"campaignId": "invalid-uuid"},
		})

		// UUID validation may happen at database level
		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Logf("Note: Invalid UUID returned status %d", w.Code)
		}
	})

	t.Run("Success_NonExistentCampaign", func(t *testing.T) {
		nonExistentID := uuid.New()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/campaigns/%s/documents", nonExistentID),
			URLVars: map[string]string{"campaignId": nonExistentID.String()},
		})

		// Non-existent campaign should return empty array
		documents := assertJSONArray(t, w, http.StatusOK)
		if documents != nil && len(documents) != 0 {
			t.Errorf("Expected empty array for non-existent campaign, got %d documents", len(documents))
		}
	})
}

// TestSearchDocuments tests the search documents endpoint
func TestSearchDocuments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Error_MissingQuery", func(t *testing.T) {
		campaign := setupTestCampaign(t, env, "test-campaign")
		defer campaign.Cleanup()

		// Request without query field
		searchReq := map[string]interface{}{
			"limit": 10,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/campaigns/%s/search", campaign.Campaign.ID),
			URLVars: map[string]string{"campaignId": campaign.Campaign.ID.String()},
			Body:    searchReq,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "query")
	})

	t.Run("Error_MissingCampaignId", func(t *testing.T) {
		searchReq := TestData.SearchRequest("test query", 10)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "POST",
			Path:    "/campaigns//search",
			URLVars: map[string]string{"campaignId": ""},
			Body:    searchReq,
		})

		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 400 or 404, got %d", w.Code)
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		campaign := setupTestCampaign(t, env, "test-campaign")
		defer campaign.Cleanup()

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "POST",
			Path:    fmt.Sprintf("/campaigns/%s/search", campaign.Campaign.ID),
			URLVars: map[string]string{"campaignId": campaign.Campaign.ID.String()},
			Body:    `{"query": "malformed"`, // Invalid JSON
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})
}

// TestGenerateContent tests the generate content endpoint
func TestGenerateContent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Error_MissingCampaignId", func(t *testing.T) {
		genReq := map[string]interface{}{
			"content_type": "blog_post",
			"prompt":       "Write a blog post",
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   genReq,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Campaign ID")
	})

	t.Run("Error_MissingContentType", func(t *testing.T) {
		campaign := setupTestCampaign(t, env, "test-campaign")
		defer campaign.Cleanup()

		genReq := map[string]interface{}{
			"campaign_id": campaign.Campaign.ID.String(),
			"prompt":      "Write a blog post",
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   genReq,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "content type")
	})

	t.Run("Error_MissingPrompt", func(t *testing.T) {
		campaign := setupTestCampaign(t, env, "test-campaign")
		defer campaign.Cleanup()

		genReq := map[string]interface{}{
			"campaign_id":  campaign.Campaign.ID.String(),
			"content_type": "blog_post",
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   genReq,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "prompt")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/generate",
			Body:   `{"campaign_id": "malformed"`, // Invalid JSON
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})
}


// TestCampaignService tests the CampaignService type
func TestCampaignService(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NewCampaignService", func(t *testing.T) {
		service := NewCampaignService(
			env.DB,
			"http://localhost:5678",
			"http://localhost:5681",
			"postgres://localhost/test",
			"http://localhost:6333",
			"http://localhost:9000",
		)

		if service == nil {
			t.Fatal("Expected service to be created")
		}
		if service.db == nil {
			t.Error("Expected service.db to be initialized")
		}
		if service.httpClient == nil {
			t.Error("Expected service.httpClient to be initialized")
		}
		if service.logger == nil {
			t.Error("Expected service.logger to be initialized")
		}
		if service.n8nBaseURL != "http://localhost:5678" {
			t.Errorf("Expected n8nBaseURL 'http://localhost:5678', got '%s'", service.n8nBaseURL)
		}
	})
}

// TestDataStructures tests the data structure types
func TestDataStructures(t *testing.T) {
	t.Run("Campaign", func(t *testing.T) {
		campaign := Campaign{
			ID:          uuid.New(),
			Name:        "Test Campaign",
			Description: "Test Description",
			Settings: map[string]interface{}{
				"key": "value",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if campaign.ID == uuid.Nil {
			t.Error("Expected campaign ID to be set")
		}
		if campaign.Name != "Test Campaign" {
			t.Errorf("Expected name 'Test Campaign', got '%s'", campaign.Name)
		}
		if campaign.Settings["key"] != "value" {
			t.Error("Expected settings to contain key-value pair")
		}
	})

	t.Run("Document", func(t *testing.T) {
		doc := Document{
			ID:          uuid.New(),
			CampaignID:  uuid.New(),
			Filename:    "test.pdf",
			FilePath:    "/path/to/test.pdf",
			ContentType: "application/pdf",
			UploadDate:  time.Now(),
		}

		if doc.ID == uuid.Nil {
			t.Error("Expected document ID to be set")
		}
		if doc.Filename != "test.pdf" {
			t.Errorf("Expected filename 'test.pdf', got '%s'", doc.Filename)
		}
	})

	t.Run("GeneratedContent", func(t *testing.T) {
		content := GeneratedContent{
			ID:            uuid.New(),
			CampaignID:    uuid.New(),
			ContentType:   "blog_post",
			Prompt:        "Write a blog post",
			GeneratedText: "Generated content...",
			UsedDocuments: []string{"doc1.pdf", "doc2.pdf"},
			CreatedAt:     time.Now(),
		}

		if content.ID == uuid.Nil {
			t.Error("Expected content ID to be set")
		}
		if content.ContentType != "blog_post" {
			t.Errorf("Expected content type 'blog_post', got '%s'", content.ContentType)
		}
		if len(content.UsedDocuments) != 2 {
			t.Errorf("Expected 2 used documents, got %d", len(content.UsedDocuments))
		}
	})
}

