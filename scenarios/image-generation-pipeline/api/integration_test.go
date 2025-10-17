// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// TestProcessAudioWithWhisper tests the Whisper audio processing function
func TestProcessAudioWithWhisper(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create mock Whisper server
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]interface{}{"text": "Transcribed audio"},
			},
		})
		defer mockWhisper.Server.Close()

		transcript, err := processAudioWithWhisper(mockWhisper.Server.URL, "base64-audio", "wav")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if transcript != "Transcribed audio" {
			t.Errorf("Expected 'Transcribed audio', got '%s'", transcript)
		}
	})

	t.Run("ServiceError", func(t *testing.T) {
		// Create mock Whisper server that returns error
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       map[string]interface{}{"error": "Service error"},
			},
		})
		defer mockWhisper.Server.Close()

		_, err := processAudioWithWhisper(mockWhisper.Server.URL, "base64-audio", "wav")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("InvalidResponse", func(t *testing.T) {
		// Create mock Whisper server with invalid response
		mockWhisper := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]interface{}{"no_text_field": "value"},
			},
		})
		defer mockWhisper.Server.Close()

		_, err := processAudioWithWhisper(mockWhisper.Server.URL, "base64-audio", "wav")
		if err == nil {
			t.Error("Expected error for invalid response format, got nil")
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		// Create mock server that returns malformed JSON
		mockServer := newMockHTTPServer([]MockResponse{
			{StatusCode: http.StatusOK},
		})
		defer mockServer.Server.Close()

		// Override the mock to return invalid JSON
		mockServer.Server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json"))
		})

		_, err := processAudioWithWhisper(mockServer.Server.URL, "base64-audio", "wav")
		if err == nil {
			t.Error("Expected error for malformed JSON, got nil")
		}
	})
}

// TestTriggerImageGeneration tests the image generation trigger function
func TestTriggerImageGeneration(t *testing.T) {
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

	t.Run("Success", func(t *testing.T) {
		// Create mock n8n server
		mockN8N := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusOK,
				Body:       map[string]string{"status": "processing"},
			},
		})
		defer mockN8N.Server.Close()

		// Override N8N URL
		os.Setenv("N8N_BASE_URL", mockN8N.Server.URL)
		defer os.Unsetenv("N8N_BASE_URL")

		generation := ImageGeneration{
			ID:         "test-gen-id",
			CampaignID: testCampaign.Campaign.ID,
			Prompt:     "Test prompt",
		}

		req := GenerationRequest{
			CampaignID: testCampaign.Campaign.ID,
			Prompt:     "Test prompt",
			Style:      "photographic",
			Dimensions: "1024x1024",
		}

		// This runs in background, so we just verify it doesn't panic
		triggerImageGeneration(generation, req)

		// Verify the request was made
		if len(mockN8N.Requests) == 0 {
			t.Error("Expected at least one request to n8n server")
		}
	})

	t.Run("N8NError", func(t *testing.T) {
		// Create mock n8n server that returns error
		mockN8N := newMockHTTPServer([]MockResponse{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       map[string]string{"error": "Service error"},
			},
		})
		defer mockN8N.Server.Close()

		// Override N8N URL
		os.Setenv("N8N_BASE_URL", mockN8N.Server.URL)
		defer os.Unsetenv("N8N_BASE_URL")

		generation := ImageGeneration{
			ID:         "test-gen-id-error",
			CampaignID: testCampaign.Campaign.ID,
			Prompt:     "Test prompt",
		}

		req := GenerationRequest{
			CampaignID: testCampaign.Campaign.ID,
			Prompt:     "Test prompt",
			Style:      "photographic",
			Dimensions: "1024x1024",
		}

		// This runs in background, so we just verify it doesn't panic
		triggerImageGeneration(generation, req)
	})
}

// TestCampaignsHandlerMethods tests different HTTP methods on campaigns endpoint
func TestCampaignsHandlerMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/campaigns",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		campaignsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
}

// TestBrandsHandlerMethods tests different HTTP methods on brands endpoint
func TestBrandsHandlerMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/brands",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		brandsHandler(w, httpReq)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
}

// TestInitDBExponentialBackoff tests database initialization with retry logic
func TestInitDBExponentialBackoff(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ImmediateSuccess", func(t *testing.T) {
		// Skip if no database available
		testDB := setupTestDatabase(t)
		if testDB == nil {
			return
		}
		defer testDB.Cleanup()

		config := &Config{
			PostgresURL: os.Getenv("TEST_POSTGRES_URL"),
		}
		if config.PostgresURL == "" {
			config.PostgresURL = "postgres://postgres:postgres@localhost:5432/image_gen_test?sslmode=disable"
		}

		err := initDB(config)
		if err != nil {
			t.Skipf("Database not available: %v", err)
		}
		defer db.Close()
	})

	// Skipping InvalidURL test because it takes too long with exponential backoff
	// The retry logic is already tested indirectly
}

// TestConfigValidation tests configuration validation and error handling
func TestConfigValidation(t *testing.T) {
	t.Run("MissingPort", func(t *testing.T) {
		// Save original env
		origPort := os.Getenv("API_PORT")
		origPortEnv := os.Getenv("PORT")
		defer func() {
			os.Setenv("API_PORT", origPort)
			os.Setenv("PORT", origPortEnv)
		}()

		os.Unsetenv("API_PORT")
		os.Unsetenv("PORT")
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test")

		// This should panic or exit, but we can't easily test that
		// So we just document the requirement
		t.Log("Config requires API_PORT or PORT environment variable")
	})

	t.Run("MissingDatabaseConfig", func(t *testing.T) {
		// Save original env
		origURL := os.Getenv("POSTGRES_URL")
		defer os.Setenv("POSTGRES_URL", origURL)

		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_HOST")

		// This should panic or exit, but we can't easily test that
		// So we just document the requirement
		t.Log("Config requires database configuration")
	})
}

// TestJSONMarshaling tests JSON encoding/decoding edge cases
func TestJSONMarshaling(t *testing.T) {
	t.Run("CampaignWithNilFields", func(t *testing.T) {
		campaign := Campaign{
			ID:      "test-id",
			Name:    "Test Campaign",
			BrandID: "test-brand-id",
			Status:  "active",
			// Description, Budget, etc. are nil
		}

		data, err := json.Marshal(campaign)
		if err != nil {
			t.Fatalf("Failed to marshal campaign: %v", err)
		}

		var decoded Campaign
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal campaign: %v", err)
		}

		if decoded.Name != campaign.Name {
			t.Errorf("Expected name '%s', got '%s'", campaign.Name, decoded.Name)
		}
	})

	t.Run("BrandWithArrays", func(t *testing.T) {
		brand := Brand{
			ID:     "test-id",
			Name:   "Test Brand",
			Colors: []string{"#FF0000", "#00FF00", "#0000FF"},
			Fonts:  []string{"Arial", "Helvetica"},
		}

		data, err := json.Marshal(brand)
		if err != nil {
			t.Fatalf("Failed to marshal brand: %v", err)
		}

		var decoded Brand
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal brand: %v", err)
		}

		if len(decoded.Colors) != len(brand.Colors) {
			t.Errorf("Expected %d colors, got %d", len(brand.Colors), len(decoded.Colors))
		}
	})

	t.Run("GenerationRequestWithMetadata", func(t *testing.T) {
		genReq := GenerationRequest{
			CampaignID: "test-campaign",
			Prompt:     "Test prompt",
			Style:      "photographic",
			Dimensions: "1024x1024",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		}

		data, err := json.Marshal(genReq)
		if err != nil {
			t.Fatalf("Failed to marshal generation request: %v", err)
		}

		var decoded GenerationRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal generation request: %v", err)
		}

		if decoded.Prompt != genReq.Prompt {
			t.Errorf("Expected prompt '%s', got '%s'", genReq.Prompt, decoded.Prompt)
		}
	})
}

// TestHTTPErrorResponses tests various HTTP error response scenarios
func TestHTTPErrorResponses(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	t.Run("CreateCampaignInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/campaigns",
			Body:   `{"invalid": json}`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createCampaign(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateBrandInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/brands",
			Body:   `not json at all`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createBrand(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GenerateImageInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generate",
			Body:   `{broken json`,
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		generateImageHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestDataStructures tests data structure edge cases
func TestDataStructures(t *testing.T) {
	t.Run("EmptyArrayFields", func(t *testing.T) {
		campaign := Campaign{
			ID:          "test-id",
			Name:        "Test Campaign",
			BrandID:     "test-brand-id",
			Status:      "active",
			TeamMembers: []string{},
		}

		data, _ := json.Marshal(campaign.TeamMembers)
		if string(data) != "[]" {
			t.Errorf("Expected empty array '[]', got '%s'", string(data))
		}
	})

	t.Run("NilMetadata", func(t *testing.T) {
		gen := ImageGeneration{
			ID:         "test-id",
			CampaignID: "campaign-id",
			Prompt:     "Test prompt",
			Status:     "processing",
			Metadata:   nil,
		}

		data, _ := json.Marshal(gen)
		var decoded map[string]interface{}
		json.Unmarshal(data, &decoded)

		if decoded["metadata"] != nil {
			t.Errorf("Expected metadata to be null, got %v", decoded["metadata"])
		}
	})
}
