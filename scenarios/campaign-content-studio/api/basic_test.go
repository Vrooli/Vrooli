// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestBasicStructures tests basic data structures without database
func TestBasicStructures(t *testing.T) {
	t.Run("Campaign_Structure", func(t *testing.T) {
		campaign := Campaign{
			ID:          uuid.New(),
			Name:        "Test Campaign",
			Description: "Test Description",
			Settings: map[string]interface{}{
				"target_audience": "developers",
				"tone":           "professional",
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
		if campaign.Settings["target_audience"] != "developers" {
			t.Error("Expected settings to contain target_audience")
		}
	})

	t.Run("Document_Structure", func(t *testing.T) {
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
		if doc.ContentType != "application/pdf" {
			t.Errorf("Expected content type 'application/pdf', got '%s'", doc.ContentType)
		}
	})

	t.Run("GeneratedContent_Structure", func(t *testing.T) {
		content := GeneratedContent{
			ID:            uuid.New(),
			CampaignID:    uuid.New(),
			ContentType:   "blog_post",
			Prompt:        "Write a blog post about AI",
			GeneratedText: "AI is transforming the world...",
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

// TestConstants validates all constants
func TestConstants(t *testing.T) {
	t.Run("API_Constants", func(t *testing.T) {
		if apiVersion == "" {
			t.Error("apiVersion should not be empty")
		}
		if serviceName == "" {
			t.Error("serviceName should not be empty")
		}
		if serviceName != "campaign-content-studio" {
			t.Errorf("Expected service name 'campaign-content-studio', got '%s'", serviceName)
		}
		t.Logf("API Version: %s", apiVersion)
		t.Logf("Service Name: %s", serviceName)
	})

	t.Run("Timeout_Constants", func(t *testing.T) {
		if httpTimeout == 0 {
			t.Error("httpTimeout should not be zero")
		}
		if discoveryDelay == 0 {
			t.Error("discoveryDelay should not be zero")
		}
		if httpTimeout < 1*time.Second {
			t.Errorf("httpTimeout seems too short: %v", httpTimeout)
		}
		t.Logf("HTTP Timeout: %v", httpTimeout)
		t.Logf("Discovery Delay: %v", discoveryDelay)
	})

	t.Run("Database_Constants", func(t *testing.T) {
		if maxDBConnections == 0 {
			t.Error("maxDBConnections should not be zero")
		}
		if maxIdleConnections == 0 {
			t.Error("maxIdleConnections should not be zero")
		}
		if connMaxLifetime == 0 {
			t.Error("connMaxLifetime should not be zero")
		}
		if maxIdleConnections > maxDBConnections {
			t.Errorf("maxIdleConnections (%d) should not exceed maxDBConnections (%d)",
				maxIdleConnections, maxDBConnections)
		}
		t.Logf("Max DB Connections: %d", maxDBConnections)
		t.Logf("Max Idle Connections: %d", maxIdleConnections)
		t.Logf("Connection Max Lifetime: %v", connMaxLifetime)
	})
}

// TestLogger validates the Logger implementation
func TestLogger(t *testing.T) {
	t.Run("NewLogger_Creation", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatal("NewLogger should return a non-nil logger")
		}
		if logger.Logger == nil {
			t.Error("Logger.Logger should be initialized")
		}
	})

	t.Run("Logger_Methods", func(t *testing.T) {
		logger := NewLogger()

		// These should not panic
		logger.Info("Test info message")
		logger.Warn("Test warning", fmt.Errorf("test error"))
		logger.Error("Test error", fmt.Errorf("test error"))
	})
}

// TestHTTPError validates the HTTPError function
func TestHTTPError(t *testing.T) {
	t.Run("Error_Response_Structure", func(t *testing.T) {
		w := httptest.NewRecorder()
		testErr := fmt.Errorf("test error")

		HTTPError(w, "Test error message", http.StatusBadRequest, testErr)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("Error_Status_Codes", func(t *testing.T) {
		testCases := []struct {
			statusCode int
			message    string
		}{
			{http.StatusBadRequest, "Bad request"},
			{http.StatusNotFound, "Not found"},
			{http.StatusInternalServerError, "Internal error"},
			{http.StatusUnauthorized, "Unauthorized"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Status_%d", tc.statusCode), func(t *testing.T) {
				w := httptest.NewRecorder()
				HTTPError(w, tc.message, tc.statusCode, nil)

				if w.Code != tc.statusCode {
					t.Errorf("Expected status %d, got %d", tc.statusCode, w.Code)
				}
			})
		}
	})
}

// TestHealth_Standalone validates the health endpoint without database
func TestHealth_Standalone(t *testing.T) {
	t.Run("Health_Response", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		Health(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Verify response contains expected fields
		body := w.Body.String()
		if body == "" {
			t.Error("Expected non-empty response body")
		}
	})
}

// TestNewCampaignService validates service initialization
func TestNewCampaignService(t *testing.T) {
	t.Run("Service_Initialization", func(t *testing.T) {
		// Create a nil DB for testing (we won't use it)
		service := NewCampaignService(
			nil,
			"http://localhost:5678",
			"http://localhost:5681",
			"postgres://localhost/test",
			"http://localhost:6333",
			"http://localhost:9000",
		)

		if service == nil {
			t.Fatal("NewCampaignService should return a non-nil service")
		}
		if service.httpClient == nil {
			t.Error("Service httpClient should be initialized")
		}
		if service.logger == nil {
			t.Error("Service logger should be initialized")
		}
		if service.n8nBaseURL != "http://localhost:5678" {
			t.Errorf("Expected n8nBaseURL 'http://localhost:5678', got '%s'", service.n8nBaseURL)
		}
		if service.httpClient.Timeout != httpTimeout {
			t.Errorf("Expected HTTP timeout %v, got %v", httpTimeout, service.httpClient.Timeout)
		}
	})

	t.Run("Service_URLs", func(t *testing.T) {
		testURLs := map[string]string{
			"n8n":      "http://n8n:5678",
			"windmill": "http://windmill:5681",
			"postgres": "postgres://db:5432/campaign",
			"qdrant":   "http://qdrant:6333",
			"minio":    "http://minio:9000",
		}

		service := NewCampaignService(
			nil,
			testURLs["n8n"],
			testURLs["windmill"],
			testURLs["postgres"],
			testURLs["qdrant"],
			testURLs["minio"],
		)

		if service.n8nBaseURL != testURLs["n8n"] {
			t.Errorf("n8nBaseURL mismatch: expected '%s', got '%s'",
				testURLs["n8n"], service.n8nBaseURL)
		}
		if service.windmillURL != testURLs["windmill"] {
			t.Errorf("windmillURL mismatch: expected '%s', got '%s'",
				testURLs["windmill"], service.windmillURL)
		}
		if service.qdrantURL != testURLs["qdrant"] {
			t.Errorf("qdrantURL mismatch: expected '%s', got '%s'",
				testURLs["qdrant"], service.qdrantURL)
		}
	})
}

// TestTestHelpers validates test helper functions
func TestTestHelpers(t *testing.T) {
	t.Run("TestDataGenerator", func(t *testing.T) {
		gen := &TestDataGenerator{}

		campaignReq := gen.CampaignRequest("Test Campaign", "Test Description")
		if campaignReq["name"] != "Test Campaign" {
			t.Errorf("Expected name 'Test Campaign', got '%v'", campaignReq["name"])
		}

		contentReq := gen.GenerateContentRequest("campaign-id", "blog_post", "test prompt")
		if contentReq["content_type"] != "blog_post" {
			t.Errorf("Expected content_type 'blog_post', got '%v'", contentReq["content_type"])
		}

		searchReq := gen.SearchRequest("test query", 10)
		if searchReq["query"] != "test query" {
			t.Errorf("Expected query 'test query', got '%v'", searchReq["query"])
		}
		if searchReq["limit"] != 10 {
			t.Errorf("Expected limit 10, got '%v'", searchReq["limit"])
		}
	})
}
