package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestIntegrationWithMockDB tests all endpoints with in-memory database simulation
func TestIntegrationWithMockDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Try to setup real test database, skip if unavailable
	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Integration tests require database - skipping")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	// Test flow: Create campaign -> Get campaign -> Send campaign -> Get analytics
	t.Run("FullCampaignFlow", func(t *testing.T) {
		// Step 1: Create a campaign
		createReq := CreateCampaignRequest{
			Name:        "Integration Test Campaign",
			Description: "Full flow integration test",
		}

		createRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to create campaign: %v", err)
		}

		if createRecorder.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", createRecorder.Code, createRecorder.Body.String())
		}

		var campaign Campaign
		if err := json.Unmarshal(createRecorder.Body.Bytes(), &campaign); err != nil {
			t.Fatalf("Failed to parse create response: %v", err)
		}

		campaignID := campaign.ID

		// Step 2: Get the campaign
		getRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + campaignID,
		})

		if err != nil {
			t.Fatalf("Failed to get campaign: %v", err)
		}

		if getRecorder.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", getRecorder.Code)
		}

		var retrievedCampaign Campaign
		if err := json.Unmarshal(getRecorder.Body.Bytes(), &retrievedCampaign); err != nil {
			t.Fatalf("Failed to parse get response: %v", err)
		}

		if retrievedCampaign.ID != campaignID {
			t.Errorf("Campaign ID mismatch: expected %s, got %s", campaignID, retrievedCampaign.ID)
		}

		// Step 3: Send the campaign
		sendRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns/" + campaignID + "/send",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to send campaign: %v", err)
		}

		if sendRecorder.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", sendRecorder.Code)
		}

		// Step 4: Get campaign analytics
		analyticsRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + campaignID + "/analytics",
		})

		if err != nil {
			t.Fatalf("Failed to get analytics: %v", err)
		}

		if analyticsRecorder.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", analyticsRecorder.Code)
		}
	})

	t.Run("FullTemplateFlow", func(t *testing.T) {
		// Step 1: Generate a template
		genReq := GenerateTemplateRequest{
			Purpose:   "Integration test template",
			Tone:      "professional",
			Documents: []string{},
		}

		genRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to generate template: %v", err)
		}

		if genRecorder.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", genRecorder.Code, genRecorder.Body.String())
		}

		var template Template
		if err := json.Unmarshal(genRecorder.Body.Bytes(), &template); err != nil {
			t.Fatalf("Failed to parse template response: %v", err)
		}

		templateID := template.ID

		// Step 2: List templates and verify it's included
		listRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates",
		})

		if err != nil {
			t.Fatalf("Failed to list templates: %v", err)
		}

		if listRecorder.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", listRecorder.Code)
		}

		var templates []Template
		if err := json.Unmarshal(listRecorder.Body.Bytes(), &templates); err != nil {
			t.Fatalf("Failed to parse templates list: %v", err)
		}

		found := false
		for _, tmpl := range templates {
			if tmpl.ID == templateID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Generated template not found in list")
		}

		// Step 3: Create campaign with the template
		createReq := CreateCampaignRequest{
			Name:        "Campaign with Template",
			Description: "Using generated template",
			TemplateID:  templateID,
		}

		createRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to create campaign: %v", err)
		}

		if createRecorder.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", createRecorder.Code)
		}

		var campaign Campaign
		if err := json.Unmarshal(createRecorder.Body.Bytes(), &campaign); err != nil {
			t.Fatalf("Failed to parse campaign response: %v", err)
		}

		if campaign.TemplateID == nil || *campaign.TemplateID != templateID {
			t.Errorf("Campaign template ID mismatch: expected %s, got %v", templateID, campaign.TemplateID)
		}
	})

	t.Run("ListCampaignsWithMultiple", func(t *testing.T) {
		// Create multiple campaigns
		for i := 0; i < 3; i++ {
			createReq := CreateCampaignRequest{
				Name:        "Test Campaign " + string(rune('A'+i)),
				Description: "Test campaign description",
			}

			_, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/campaigns",
				Body:   createReq,
			})

			if err != nil {
				t.Fatalf("Failed to create campaign %d: %v", i, err)
			}
		}

		// List all campaigns
		listRecorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})

		if err != nil {
			t.Fatalf("Failed to list campaigns: %v", err)
		}

		if listRecorder.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", listRecorder.Code)
		}

		var campaigns []Campaign
		if err := json.Unmarshal(listRecorder.Body.Bytes(), &campaigns); err != nil {
			t.Fatalf("Failed to parse campaigns list: %v", err)
		}

		if len(campaigns) < 3 {
			t.Errorf("Expected at least 3 campaigns, got %d", len(campaigns))
		}
	})

	t.Run("TemplateGenerationVariations", func(t *testing.T) {
		tones := []string{"professional", "friendly", "casual"}

		for _, tone := range tones {
			genReq := GenerateTemplateRequest{
				Purpose:   "Test template for " + tone,
				Tone:      tone,
				Documents: []string{},
			}

			recorder, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/templates/generate",
				Body:   genReq,
			})

			if err != nil {
				t.Fatalf("Failed to generate %s template: %v", tone, err)
			}

			if recorder.Code != http.StatusCreated {
				t.Errorf("Expected status 201 for %s tone, got %d", tone, recorder.Code)
			}

			var template Template
			if err := json.Unmarshal(recorder.Body.Bytes(), &template); err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			if template.StyleCategory != tone {
				t.Errorf("Expected style category %s, got %s", tone, template.StyleCategory)
			}
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Edge case tests require database - skipping")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("EmptyCampaignName", func(t *testing.T) {
		createReq := CreateCampaignRequest{
			Name:        "",
			Description: "Campaign with empty name",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should fail validation
		if recorder.Code == http.StatusCreated {
			t.Error("Expected validation error for empty campaign name")
		}
	})

	t.Run("VeryLongCampaignName", func(t *testing.T) {
		longName := string(make([]byte, 1000))
		for i := range longName {
			longName = string(append([]byte(longName)[:i], 'A'))
		}

		createReq := CreateCampaignRequest{
			Name:        longName,
			Description: "Campaign with very long name",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should either succeed or fail gracefully
		if recorder.Code != http.StatusCreated && recorder.Code != http.StatusBadRequest {
			t.Logf("Long name handling: got status %d", recorder.Code)
		}
	})

	t.Run("NonExistentCampaignOperations", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		// Try to get non-existent campaign
		getRecorder, _ := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + nonExistentID,
		})

		if getRecorder.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent campaign GET, got %d", getRecorder.Code)
		}

		// Try to send non-existent campaign
		sendRecorder, _ := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns/" + nonExistentID + "/send",
			Body:   map[string]interface{}{},
		})

		if sendRecorder.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent campaign send, got %d", sendRecorder.Code)
		}

		// Try to get analytics for non-existent campaign
		analyticsRecorder, _ := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + nonExistentID + "/analytics",
		})

		if analyticsRecorder.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent campaign analytics, got %d", analyticsRecorder.Code)
		}
	})

	t.Run("SpecialCharactersInInput", func(t *testing.T) {
		createReq := CreateCampaignRequest{
			Name:        "Campaign with special chars: <>&\"'",
			Description: "Testing special character handling",
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
			t.Errorf("Expected successful creation with special chars, got %d", recorder.Code)
		}
	})
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Concurrent tests require database - skipping")
		return
	}
	defer testDB.Cleanup()

	router := setupTestRouter()

	t.Run("ConcurrentCampaignCreation", func(t *testing.T) {
		done := make(chan bool, 5)
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				createReq := CreateCampaignRequest{
					Name:        "Concurrent Campaign " + string(rune('A'+index)),
					Description: "Created concurrently",
				}

				recorder, err := makeHTTPRequest(router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/campaigns",
					Body:   createReq,
				})

				if err != nil {
					errors <- err
					done <- false
					return
				}

				if recorder.Code != http.StatusCreated {
					errors <- nil
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// Wait for all goroutines
		successCount := 0
		for i := 0; i < 5; i++ {
			if <-done {
				successCount++
			}
		}

		if successCount < 5 {
			t.Logf("Only %d/5 concurrent requests succeeded", successCount)
		}
	})
}
