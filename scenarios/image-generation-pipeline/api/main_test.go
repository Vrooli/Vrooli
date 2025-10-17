// +build testing

package main

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "image-generation-pipeline",
			"version": "1.0.0",
		})

		if response != nil {
			if _, ok := response["timestamp"]; !ok {
				t.Error("Expected timestamp field in response")
			}
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidMethod("/health", "POST").
			Build()

		// Health handler doesn't enforce method, so this won't fail
		// Just verify it still returns 200
		for _, pattern := range patterns {
			req := pattern.Execute(t, nil)
			w, httpReq, err := makeHTTPRequest(*req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)
			// Health endpoint accepts any method
			assertJSONResponse(t, w, http.StatusOK, nil)
		}
	})
}

// TestCampaignsHandler tests the campaigns endpoint
func TestCampaignsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test brand first
	testBrand := setupTestBrand(t, "Test Brand")
	defer testBrand.Cleanup()

	t.Run("GET_Success", func(t *testing.T) {
		// Create test campaign
		testCampaign := setupTestCampaign(t, "Test Campaign", testBrand.Brand.ID)
		defer testCampaign.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/campaigns",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getCampaigns(w, httpReq)

		campaigns := assertJSONArray(t, w, http.StatusOK)
		if campaigns == nil {
			t.Fatal("Expected campaigns array in response")
		}

		if len(campaigns) == 0 {
			t.Error("Expected at least one campaign in response")
		}
	})

	t.Run("GET_EmptyList", func(t *testing.T) {
		// Clean up all campaigns
		db.Exec("DELETE FROM campaigns")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/campaigns",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getCampaigns(w, httpReq)

		campaigns := assertJSONArray(t, w, http.StatusOK)
		if campaigns == nil {
			t.Fatal("Expected empty campaigns array in response")
		}
	})

	t.Run("POST_Success", func(t *testing.T) {
		campaign := Campaign{
			Name:        "New Test Campaign",
			BrandID:     testBrand.Brand.ID,
			Status:      "active",
			TeamMembers: []string{"user1", "user2"},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/campaigns",
			Body:   campaign,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createCampaign(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name":     campaign.Name,
			"brand_id": campaign.BrandID,
			"status":   campaign.Status,
		})

		if response != nil {
			if _, ok := response["id"]; !ok {
				t.Error("Expected id field in response")
			}
		}
	})

	t.Run("POST_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "createCampaign",
			Handler:     createCampaign,
			BaseURL:     "/api/campaigns",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/campaigns").
			AddEmptyBody("POST", "/api/campaigns").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestBrandsHandler tests the brands endpoint
func TestBrandsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	t.Run("GET_Success", func(t *testing.T) {
		// Create test brand
		testBrand := setupTestBrand(t, "Test Brand for GET")
		defer testBrand.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/brands",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getBrands(w, httpReq)

		brands := assertJSONArray(t, w, http.StatusOK)
		if brands == nil {
			t.Fatal("Expected brands array in response")
		}

		if len(brands) == 0 {
			t.Error("Expected at least one brand in response")
		}
	})

	t.Run("POST_Success", func(t *testing.T) {
		description := "New brand description"
		guidelines := "Brand guidelines"
		logoURL := "https://example.com/logo.png"

		brand := Brand{
			Name:        "New Test Brand",
			Description: &description,
			Guidelines:  &guidelines,
			Colors:      []string{"#FF0000", "#00FF00"},
			Fonts:       []string{"Arial"},
			LogoURL:     &logoURL,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body:   brand,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createBrand(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name": brand.Name,
		})

		if response != nil {
			if _, ok := response["id"]; !ok {
				t.Error("Expected id field in response")
			}
		}
	})

	t.Run("POST_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "createBrand",
			Handler:     createBrand,
			BaseURL:     "/api/brands",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/brands").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGenerateImageHandler tests the image generation endpoint
func TestGenerateImageHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test brand and campaign
	testBrand := setupTestBrand(t, "Test Brand")
	defer testBrand.Cleanup()

	testCampaign := setupTestCampaign(t, "Test Campaign", testBrand.Brand.ID)
	defer testCampaign.Cleanup()

	t.Run("POST_Success", func(t *testing.T) {
		// Create mock n8n server
		mockN8N := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]string{"status": "processing"},
			},
		})
		defer mockN8N.Server.Close()

		genReq := GenerationRequest{
			CampaignID: testCampaign.Campaign.ID,
			Prompt:     "Generate a beautiful landscape",
			Style:      "photographic",
			Dimensions: "1024x1024",
			Metadata:   map[string]interface{}{"test": "data"},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generate",
			Body:   genReq,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateImageHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusAccepted, map[string]interface{}{
			"status":  "processing",
			"message": "Image generation started",
		})

		if response != nil {
			if _, ok := response["id"]; !ok {
				t.Error("Expected id field in response")
			}
		}

		// Give goroutine time to execute
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("POST_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "generateImageHandler",
			Handler:     generateImageHandler,
			BaseURL:     "/api/generate",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/generate").
			AddEmptyBody("POST", "/api/generate").
			AddInvalidMethod("/api/generate", "GET").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("POST_MissingFields", func(t *testing.T) {
		// Test with missing campaign_id
		genReq := GenerationRequest{
			Prompt:     "Generate a beautiful landscape",
			Style:      "photographic",
			Dimensions: "1024x1024",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generate",
			Body:   genReq,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateImageHandler(w, httpReq)

		// Should still create record with empty campaign_id
		assertJSONResponse(t, w, http.StatusAccepted, nil)
	})
}

// TestProcessVoiceBriefHandler tests the voice brief processing endpoint
func TestProcessVoiceBriefHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("POST_Success", func(t *testing.T) {
		// Create mock Whisper server
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]interface{}{"text": "This is the transcribed text"},
			},
		})
		defer mockWhisper.Server.Close()

		// Temporarily override Whisper URL
		originalConfig := loadConfig()
		os.Setenv("WHISPER_BASE_URL", mockWhisper.Server.URL)
		defer os.Setenv("WHISPER_BASE_URL", originalConfig.WhisperBaseURL)

		voiceReq := VoiceBriefRequest{
			AudioData: "base64-encoded-audio-data",
			Format:    "wav",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/voice-brief",
			Body:   voiceReq,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		processVoiceBriefHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "success",
		})

		if response != nil {
			if transcript, ok := response["transcript"]; !ok {
				t.Error("Expected transcript field in response")
			} else if transcript != "This is the transcribed text" {
				t.Errorf("Expected transcript to be 'This is the transcribed text', got '%s'", transcript)
			}
		}
	})

	t.Run("POST_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "processVoiceBriefHandler",
			Handler:     processVoiceBriefHandler,
			BaseURL:     "/api/voice-brief",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/voice-brief").
			AddInvalidMethod("/api/voice-brief", "GET").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("POST_WhisperServiceError", func(t *testing.T) {
		// Create mock Whisper server that returns error
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       map[string]interface{}{"error": "Service unavailable"},
			},
		})
		defer mockWhisper.Server.Close()

		os.Setenv("WHISPER_BASE_URL", mockWhisper.Server.URL)

		voiceReq := VoiceBriefRequest{
			AudioData: "base64-encoded-audio-data",
			Format:    "wav",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/voice-brief",
			Body:   voiceReq,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		processVoiceBriefHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusInternalServerError)
	})
}

// TestGenerationsHandler tests the generations listing endpoint
func TestGenerationsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test brand and campaign
	testBrand := setupTestBrand(t, "Test Brand")
	defer testBrand.Cleanup()

	testCampaign := setupTestCampaign(t, "Test Campaign", testBrand.Brand.ID)
	defer testCampaign.Cleanup()

	t.Run("GET_Success", func(t *testing.T) {
		// Create test image generation
		testGen := setupTestImageGeneration(t, testCampaign.Campaign.ID)
		defer testGen.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/generations",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generationsHandler(w, httpReq)

		generations := assertJSONArray(t, w, http.StatusOK)
		if generations == nil {
			t.Fatal("Expected generations array in response")
		}

		if len(generations) == 0 {
			t.Error("Expected at least one generation in response")
		}
	})

	t.Run("GET_FilterByCampaign", func(t *testing.T) {
		// Create test image generation
		testGen := setupTestImageGeneration(t, testCampaign.Campaign.ID)
		defer testGen.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/generations",
			QueryParams: map[string]string{
				"campaign_id": testCampaign.Campaign.ID,
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generationsHandler(w, httpReq)

		generations := assertJSONArray(t, w, http.StatusOK)
		if generations == nil {
			t.Fatal("Expected generations array in response")
		}
	})

	t.Run("GET_FilterByStatus", func(t *testing.T) {
		// Create test image generation
		testGen := setupTestImageGeneration(t, testCampaign.Campaign.ID)
		defer testGen.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/generations",
			QueryParams: map[string]string{
				"status": "completed",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generationsHandler(w, httpReq)

		generations := assertJSONArray(t, w, http.StatusOK)
		if generations == nil {
			t.Fatal("Expected generations array in response")
		}
	})

	t.Run("GET_InvalidMethod", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "generationsHandler",
			Handler:     generationsHandler,
			BaseURL:     "/api/generations",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidMethod("/api/generations", "POST").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithPostgresURL", func(t *testing.T) {
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("API_PORT", "8080")
		defer os.Unsetenv("POSTGRES_URL")
		defer os.Unsetenv("API_PORT")

		config := loadConfig()

		if config.PostgresURL != "postgres://test:test@localhost:5432/test" {
			t.Errorf("Expected PostgresURL to be set from POSTGRES_URL env var")
		}

		if config.Port != "8080" {
			t.Errorf("Expected Port to be '8080', got '%s'", config.Port)
		}
	})

	t.Run("WithIndividualComponents", func(t *testing.T) {
		os.Unsetenv("POSTGRES_URL")
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("PORT", "9090")

		defer os.Unsetenv("POSTGRES_HOST")
		defer os.Unsetenv("POSTGRES_PORT")
		defer os.Unsetenv("POSTGRES_USER")
		defer os.Unsetenv("POSTGRES_PASSWORD")
		defer os.Unsetenv("POSTGRES_DB")
		defer os.Unsetenv("PORT")

		config := loadConfig()

		expectedURL := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
		if config.PostgresURL != expectedURL {
			t.Errorf("Expected PostgresURL to be '%s', got '%s'", expectedURL, config.PostgresURL)
		}
	})

	t.Run("WithDefaults", func(t *testing.T) {
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test")
		os.Setenv("API_PORT", "8080")
		defer os.Unsetenv("POSTGRES_URL")
		defer os.Unsetenv("API_PORT")

		config := loadConfig()

		if config.N8NBaseURL != "http://localhost:5678" {
			t.Errorf("Expected default N8N URL")
		}

		if config.ComfyUIBaseURL != "http://localhost:8188" {
			t.Errorf("Expected default ComfyUI URL")
		}
	})
}

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	t.Run("WithValue", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		value := getEnv("TEST_VAR", "default")
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}
	})

	t.Run("WithDefault", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		value := getEnv("TEST_VAR", "default")
		if value != "default" {
			t.Errorf("Expected 'default', got '%s'", value)
		}
	})
}

// TestUpdateGenerationStatus tests the status update function
func TestUpdateGenerationStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	// Create test brand and campaign
	testBrand := setupTestBrand(t, "Test Brand")
	defer testBrand.Cleanup()

	testCampaign := setupTestCampaign(t, "Test Campaign", testBrand.Brand.ID)
	defer testCampaign.Cleanup()

	t.Run("UpdateToCompleted", func(t *testing.T) {
		// Create test generation
		testGen := setupTestImageGeneration(t, testCampaign.Campaign.ID)
		defer testGen.Cleanup()

		imageURL := "https://example.com/generated-image.png"
		qualityScore := 0.98

		updateGenerationStatus(testGen.Generation.ID, "completed", &imageURL, &qualityScore)

		// Verify update
		var status, url string
		var score float64
		err := db.QueryRow("SELECT status, image_url, quality_score FROM image_generations WHERE id = $1",
			testGen.Generation.ID).Scan(&status, &url, &score)

		if err != nil {
			t.Fatalf("Failed to query updated generation: %v", err)
		}

		if status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", status)
		}

		if url != imageURL {
			t.Errorf("Expected image_url '%s', got '%s'", imageURL, url)
		}

		if score != qualityScore {
			t.Errorf("Expected quality_score %.2f, got %.2f", qualityScore, score)
		}
	})

	t.Run("UpdateToFailed", func(t *testing.T) {
		// Create test generation
		testGen := setupTestImageGeneration(t, testCampaign.Campaign.ID)
		defer testGen.Cleanup()

		updateGenerationStatus(testGen.Generation.ID, "failed", nil, nil)

		// Verify update
		var status string
		err := db.QueryRow("SELECT status FROM image_generations WHERE id = $1",
			testGen.Generation.ID).Scan(&status)

		if err != nil {
			t.Fatalf("Failed to query updated generation: %v", err)
		}

		if status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", status)
		}
	})
}
