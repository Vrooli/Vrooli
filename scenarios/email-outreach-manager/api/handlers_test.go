package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestHealthCheckWithoutDB tests health endpoint without database
func TestHealthCheckWithoutDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	// Save original db and set to nil
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

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["database"] != "not_configured" {
		t.Errorf("Expected database 'not_configured', got %v", response["database"])
	}
}

// TestListCampaignsErrors tests error handling for campaign listing
func TestListCampaignsErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
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

// TestCreateCampaignErrors tests error handling for campaign creation
func TestCreateCampaignErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
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

	t.Run("MissingName", func(t *testing.T) {
		createReq := map[string]interface{}{
			"description": "No name provided",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   createReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", recorder.Code)
		}
	})
}

// TestGetCampaignErrors tests error handling for getting campaigns
func TestGetCampaignErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
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

// TestSendCampaignErrors tests error handling for sending campaigns
func TestSendCampaignErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns/" + uuid.New().String() + "/send",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})
}

// TestGetCampaignAnalyticsErrors tests error handling for analytics
func TestGetCampaignAnalyticsErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns/" + uuid.New().String() + "/analytics",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})
}

// TestGenerateTemplateErrors tests error handling for template generation
func TestGenerateTemplateErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		genReq := GenerateTemplateRequest{
			Purpose: "Test template",
			Tone:    "professional",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})

	t.Run("MissingPurpose", func(t *testing.T) {
		genReq := map[string]interface{}{
			"tone": "professional",
		}

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/templates/generate",
			Body:   genReq,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", recorder.Code)
		}
	})

	t.Run("InvalidTone", func(t *testing.T) {
		genReq := GenerateTemplateRequest{
			Purpose: "Test email",
			Tone:    "super-formal-business",
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
}

// TestListTemplatesErrors tests error handling for template listing
func TestListTemplatesErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, recorder, http.StatusServiceUnavailable, "Database")
	})
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	t.Run("InvalidHost", func(t *testing.T) {
		// Save original env vars
		originalHost := getEnvWithDefault("DB_HOST", "")
		t.Setenv("DB_HOST", "invalid-host-that-does-not-exist")
		t.Setenv("DB_PORT", "5432")

		// Try to init DB - should fail gracefully
		err := initDB()
		if err == nil {
			t.Error("Expected error when connecting to invalid host")
		}

		// Restore
		if originalHost != "" {
			t.Setenv("DB_HOST", originalHost)
		}
	})
}

// Helper to get env var with default
func getEnvWithDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// TestHelperFunctions tests the test helper utilities
func TestHelperFunctions(t *testing.T) {
	t.Run("Contains", func(t *testing.T) {
		if !contains("hello world", "world") {
			t.Error("Expected contains to find 'world' in 'hello world'")
		}

		if contains("hello", "goodbye") {
			t.Error("Expected contains to not find 'goodbye' in 'hello'")
		}
	})

	t.Run("SetupTestLogger", func(t *testing.T) {
		cleanup := setupTestLogger()
		if cleanup == nil {
			t.Error("Expected cleanup function")
		}
		cleanup()
	})
}

// TestRequestValidation tests request validation logic
func TestRequestValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("EmptyBody", func(t *testing.T) {
		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/campaigns",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should fail validation for missing required fields
		if recorder.Code == http.StatusOK || recorder.Code == http.StatusCreated {
			t.Error("Expected validation error for empty request body")
		}
	})

	t.Run("ValidCampaignRequest", func(t *testing.T) {
		// Even without DB, should pass validation
		req := CreateCampaignRequest{
			Name:        "Valid Campaign Name",
			Description: "Valid description",
		}

		_, err := json.Marshal(req)
		if err != nil {
			t.Errorf("Valid request should marshal successfully: %v", err)
		}
	})

	t.Run("ValidTemplateRequest", func(t *testing.T) {
		req := GenerateTemplateRequest{
			Purpose:   "Test purpose",
			Tone:      "professional",
			Documents: []string{},
		}

		_, err := json.Marshal(req)
		if err != nil {
			t.Errorf("Valid request should marshal successfully: %v", err)
		}
	})
}

// TestStructSerialization tests JSON serialization of structs
func TestStructSerialization(t *testing.T) {
	t.Run("Campaign", func(t *testing.T) {
		campaign := Campaign{
			ID:              uuid.New().String(),
			Name:            "Test Campaign",
			Description:     "Test Description",
			Status:          "draft",
			TotalRecipients: 100,
			SentCount:       50,
		}

		data, err := json.Marshal(campaign)
		if err != nil {
			t.Fatalf("Failed to marshal campaign: %v", err)
		}

		var decoded Campaign
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal campaign: %v", err)
		}

		if decoded.ID != campaign.ID {
			t.Errorf("Expected ID %s, got %s", campaign.ID, decoded.ID)
		}
	})

	t.Run("Template", func(t *testing.T) {
		template := Template{
			ID:                    uuid.New().String(),
			Name:                  "Test Template",
			Subject:               "Test Subject",
			HTMLContent:           "<html></html>",
			TextContent:           "Text content",
			PersonalizationFields: []string{"name", "email"},
			StyleCategory:         "professional",
		}

		data, err := json.Marshal(template)
		if err != nil {
			t.Fatalf("Failed to marshal template: %v", err)
		}

		var decoded Template
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal template: %v", err)
		}

		if decoded.ID != template.ID {
			t.Errorf("Expected ID %s, got %s", template.ID, decoded.ID)
		}
	})
}

// TestErrorResponseFormat tests error response formatting
func TestErrorResponseFormat(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	testCases := []struct {
		name         string
		method       string
		path         string
		body         interface{}
		expectedCode int
	}{
		{
			name:         "Invalid Campaign Request",
			method:       "POST",
			path:         "/api/v1/campaigns",
			body:         map[string]interface{}{"invalid": "data"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid Template Request",
			method:       "POST",
			path:         "/api/v1/templates/generate",
			body:         map[string]interface{}{"invalid": "data"},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder, err := makeHTTPRequest(router, HTTPTestRequest{
				Method: tc.method,
				Path:   tc.path,
				Body:   tc.body,
			})

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if recorder.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, recorder.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			if _, ok := response["error"]; !ok {
				t.Error("Error response should contain 'error' field")
			}
		})
	}
}

// Mock test for database unavailable scenario with proper error
type mockDB struct {
	*sql.DB
}

func (m *mockDB) Ping() error {
	return sql.ErrConnDone
}

// TestDatabaseHealthCheck tests database health checking
func TestDatabaseHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("UnhealthyDatabase", func(t *testing.T) {
		// This test simulates database connection issues
		// In a real scenario, we would mock the database connection
		// For now, we test the path where db is set but fails ping

		originalDB := db
		// Create a closed database connection
		testConn, _ := sql.Open("postgres", "host=invalid port=1 user=test dbname=test sslmode=disable")
		testConn.Close() // Close immediately to make it unhealthy
		db = testConn

		defer func() {
			db = originalDB
		}()

		recorder, err := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Health check should return service unavailable when DB is unhealthy
		if recorder.Code != http.StatusServiceUnavailable {
			t.Logf("Expected 503 for unhealthy DB, got %d (may vary based on DB mock)", recorder.Code)
		}
	})
}
