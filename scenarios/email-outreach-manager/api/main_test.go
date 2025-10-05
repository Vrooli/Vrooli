package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Save and clear DB_HOST environment variable for clean test
	originalHost := os.Getenv("DB_HOST")
	os.Unsetenv("DB_HOST")
	defer func() {
		if originalHost != "" {
			os.Setenv("DB_HOST", originalHost)
		}
	}()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		// Ensure db is nil for this test
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, recorder, http.StatusOK, func(body map[string]interface{}) bool {
			status, ok := body["status"].(string)
			return ok && status == "healthy"
		})
	})

	t.Run("WithDatabase", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		if testDB == nil {
			t.Skip("Database not available")
			return
		}
		defer testDB.Cleanup()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, recorder, http.StatusOK, func(body map[string]interface{}) bool {
			dbStatus, ok := body["database"].(string)
			return ok && dbStatus == "healthy"
		})
	})
}

// TestListCampaigns tests the campaign listing endpoint
func TestListCampaigns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("EmptyList", func(t *testing.T) {
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var campaigns []Campaign
		if err := json.Unmarshal(recorder.Body.Bytes(), &campaigns); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(campaigns) != 0 {
			t.Errorf("Expected empty list, got %d campaigns", len(campaigns))
		}
	})

	t.Run("WithCampaigns", func(t *testing.T) {
		// Create test campaigns
		campaign1 := createTestCampaign(t, testDB.DB, "Campaign 1")
		campaign2 := createTestCampaign(t, testDB.DB, "Campaign 2")

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var campaigns []Campaign
		if err := json.Unmarshal(recorder.Body.Bytes(), &campaigns); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(campaigns) != 2 {
			t.Errorf("Expected 2 campaigns, got %d", len(campaigns))
		}

		// Verify campaign IDs are present
		foundCampaign1 := false
		foundCampaign2 := false
		for _, c := range campaigns {
			if c.ID == campaign1.ID {
				foundCampaign1 = true
			}
			if c.ID == campaign2.ID {
				foundCampaign2 = true
			}
		}

		if !foundCampaign1 || !foundCampaign2 {
			t.Error("Not all created campaigns were returned")
		}
	})

	t.Run("DatabaseUnavailable", func(t *testing.T) {
		// Temporarily disable database
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})
}

// TestCreateCampaign tests campaign creation endpoint
func TestCreateCampaign(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		createReq := CreateCampaignRequest{
			Name:        "Test Campaign",
			Description: "This is a test campaign",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var campaign Campaign
		if err := json.Unmarshal(recorder.Body.Bytes(), &campaign); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response
		if campaign.Name != createReq.Name {
			t.Errorf("Expected name %s, got %s", createReq.Name, campaign.Name)
		}

		if campaign.Status != "draft" {
			t.Errorf("Expected status 'draft', got %s", campaign.Status)
		}

		if campaign.ID == "" {
			t.Error("Campaign ID should not be empty")
		}

		// Verify campaign was created in database
		if !assertCampaignExists(t, testDB.DB, campaign.ID) {
			t.Error("Campaign was not created in database")
		}
	})

	t.Run("WithTemplateID", func(t *testing.T) {
		template := createTestTemplate(t, testDB.DB, "Test Template", "professional")

		createReq := CreateCampaignRequest{
			Name:        "Campaign with Template",
			Description: "Campaign linked to template",
			TemplateID:  template.ID,
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var campaign Campaign
		if err := json.Unmarshal(recorder.Body.Bytes(), &campaign); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if campaign.TemplateID == nil || *campaign.TemplateID != template.ID {
			t.Errorf("Expected template_id %s, got %v", template.ID, campaign.TemplateID)
		}
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		createReq := map[string]interface{}{
			"description": "Missing name field",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/campaigns", bytes.NewBufferString(`{"invalid": "json"`))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", recorder.Code)
		}
	})

	t.Run("DatabaseUnavailable", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		createReq := CreateCampaignRequest{
			Name: "Test Campaign",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})
}

// TestGetCampaign tests getting a single campaign by ID
func TestGetCampaign(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		campaign := createTestCampaign(t, testDB.DB, "Test Campaign")

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + campaign.ID,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var returnedCampaign Campaign
		if err := json.Unmarshal(recorder.Body.Bytes(), &returnedCampaign); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if returnedCampaign.ID != campaign.ID {
			t.Errorf("Expected ID %s, got %s", campaign.ID, returnedCampaign.ID)
		}

		if returnedCampaign.Name != campaign.Name {
			t.Errorf("Expected name %s, got %s", campaign.Name, returnedCampaign.Name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + nonExistentID,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusNotFound, "not found")
	})

	t.Run("DatabaseUnavailable", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + uuid.New().String(),
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})
}

// TestSendCampaign tests the campaign sending endpoint
func TestSendCampaign(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		campaign := createTestCampaign(t, testDB.DB, "Test Campaign")

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns/" + campaign.ID + "/send",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response has required fields
		if _, ok := response["send_job_id"]; !ok {
			t.Error("Response missing send_job_id")
		}

		if _, ok := response["status"]; !ok {
			t.Error("Response missing status")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns/" + nonExistentID + "/send",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusNotFound, "")
	})
}

// TestGetCampaignAnalytics tests the campaign analytics endpoint
func TestGetCampaignAnalytics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		campaign := createTestCampaign(t, testDB.DB, "Test Campaign")

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + campaign.ID + "/analytics",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response structure
		if campaignID, ok := response["campaign_id"].(string); !ok || campaignID != campaign.ID {
			t.Errorf("Expected campaign_id %s, got %v", campaign.ID, response["campaign_id"])
		}

		if _, ok := response["metrics"]; !ok {
			t.Error("Response missing metrics field")
		}

		if _, ok := response["recipient_breakdown"]; !ok {
			t.Error("Response missing recipient_breakdown field")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + nonExistentID + "/analytics",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusNotFound, "not found")
	})
}

// TestGenerateTemplate tests the template generation endpoint
func TestGenerateTemplate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("Success_Professional", func(t *testing.T) {
		genReq := GenerateTemplateRequest{
			Purpose:   "Product launch announcement",
			Tone:      "professional",
			Documents: []string{},
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var template Template
		if err := json.Unmarshal(recorder.Body.Bytes(), &template); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate template fields
		if template.ID == "" {
			t.Error("Template ID should not be empty")
		}

		if template.StyleCategory != "professional" {
			t.Errorf("Expected tone 'professional', got %s", template.StyleCategory)
		}

		if template.Subject == "" {
			t.Error("Template subject should not be empty")
		}

		// Verify template was created in database
		if !assertTemplateExists(t, testDB.DB, template.ID) {
			t.Error("Template was not created in database")
		}
	})

	t.Run("Success_Friendly", func(t *testing.T) {
		genReq := GenerateTemplateRequest{
			Purpose:   "Newsletter update",
			Tone:      "friendly",
			Documents: []string{},
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", recorder.Code)
		}

		var template Template
		if err := json.Unmarshal(recorder.Body.Bytes(), &template); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if template.StyleCategory != "friendly" {
			t.Errorf("Expected tone 'friendly', got %s", template.StyleCategory)
		}
	})

	t.Run("Success_DefaultTone", func(t *testing.T) {
		genReq := GenerateTemplateRequest{
			Purpose: "Event invitation",
			// Tone not specified, should default to professional
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", recorder.Code)
		}

		var template Template
		if err := json.Unmarshal(recorder.Body.Bytes(), &template); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if template.StyleCategory != "professional" {
			t.Errorf("Expected default tone 'professional', got %s", template.StyleCategory)
		}
	})

	t.Run("InvalidTone", func(t *testing.T) {
		genReq := GenerateTemplateRequest{
			Purpose: "Test email",
			Tone:    "invalid-tone",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "tone")
	})

	t.Run("MissingPurpose", func(t *testing.T) {
		genReq := map[string]interface{}{
			"tone": "professional",
			// Missing "purpose" field
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusBadRequest, "")
	})
}

// TestListTemplates tests the template listing endpoint
func TestListTemplates(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("EmptyList", func(t *testing.T) {
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var templates []Template
		if err := json.Unmarshal(recorder.Body.Bytes(), &templates); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(templates) != 0 {
			t.Errorf("Expected empty list, got %d templates", len(templates))
		}
	})

	t.Run("WithTemplates", func(t *testing.T) {
		template1 := createTestTemplate(t, testDB.DB, "Template 1", "professional")
		template2 := createTestTemplate(t, testDB.DB, "Template 2", "friendly")

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var templates []Template
		if err := json.Unmarshal(recorder.Body.Bytes(), &templates); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(templates) != 2 {
			t.Errorf("Expected 2 templates, got %d", len(templates))
		}

		// Verify template IDs are present
		foundTemplate1 := false
		foundTemplate2 := false
		for _, tmpl := range templates {
			if tmpl.ID == template1.ID {
				foundTemplate1 = true
			}
			if tmpl.ID == template2.ID {
				foundTemplate2 = true
			}
		}

		if !foundTemplate1 || !foundTemplate2 {
			t.Error("Not all created templates were returned")
		}
	})
}

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("TemplateGeneration", func(t *testing.T) {
		genReq := GenerateTemplateRequest{
			Purpose: "Performance test email",
			Tone:    "professional",
		}

		start := time.Now()
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", recorder.Code)
		}

		// PRD specifies template generation should complete within 30 seconds
		maxDuration := 30 * time.Second
		if duration > maxDuration {
			t.Errorf("Template generation took %v, expected < %v", duration, maxDuration)
		}

		t.Logf("Template generation completed in %v", duration)
	})

	t.Run("CampaignCreation", func(t *testing.T) {
		createReq := CreateCampaignRequest{
			Name:        "Performance Test Campaign",
			Description: "Testing campaign creation performance",
		}

		start := time.Now()
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", recorder.Code)
		}

		// Campaign creation should be fast (< 5 seconds per PRD)
		maxDuration := 5 * time.Second
		if duration > maxDuration {
			t.Errorf("Campaign creation took %v, expected < %v", duration, maxDuration)
		}

		t.Logf("Campaign creation completed in %v", duration)
	})
}
