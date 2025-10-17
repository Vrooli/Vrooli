package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHealthHandler_Comprehensive provides comprehensive health endpoint testing
func TestHealthHandler_Comprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkBody      bool
	}{
		{"GET_Success", "GET", http.StatusOK, true},
		{"POST_Success", "POST", http.StatusOK, true},
		{"PUT_Success", "PUT", http.StatusOK, true},
		{"DELETE_Success", "DELETE", http.StatusOK, true},
		{"OPTIONS_Success", "OPTIONS", http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			healthHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}

				if status, ok := response["status"].(string); !ok || status != "healthy" {
					t.Errorf("Expected status 'healthy', got %v", response["status"])
				}

				if service, ok := response["service"].(string); !ok || service != "travel-map-filler" {
					t.Errorf("Expected service 'travel-map-filler', got %v", response["service"])
				}
			}

			// Verify CORS headers are set
			if w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Error("CORS header not properly set")
			}
		})
	}
}

// TestEnableCORS_AllHeaders verifies all CORS headers are correctly set
func TestEnableCORS_AllHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	enableCORS(w)

	tests := []struct {
		header   string
		expected string
	}{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS"},
		{"Access-Control-Allow-Headers", "Content-Type, Authorization"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			actual := w.Header().Get(tt.header)
			if actual != tt.expected {
				t.Errorf("Expected header %s to be '%s', got '%s'", tt.header, tt.expected, actual)
			}
		})
	}
}

// TestSearchTravelsHandler_QueryParsing tests various query formats
func TestSearchTravelsHandler_QueryParsing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		queryParam     string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "GET_WithQuery",
			method:         "GET",
			queryParam:     "?q=Paris",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET_WithQueryAndLimit",
			method:         "GET",
			queryParam:     "?q=London&limit=5",
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST_WithBody",
			method: "POST",
			body: map[string]interface{}{
				"query":   "Tokyo",
				"limit":   10,
				"user_id": "test_user",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET_NoQuery",
			method:         "GET",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST_EmptyQuery",
			method: "POST",
			body: map[string]interface{}{
				"query": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			if tt.method == "POST" && tt.body != nil {
				bodyJSON, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, "/api/travels/search", bytes.NewBuffer(bodyJSON))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/travels/search"+tt.queryParam, nil)
			}

			w := httptest.NewRecorder()
			searchTravelsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestSearchTravelsHandler_N8NIntegration tests n8n workflow integration
func TestSearchTravelsHandler_N8NIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Save original N8N_URL
	origN8NURL := os.Getenv("N8N_URL")
	defer os.Setenv("N8N_URL", origN8NURL)

	t.Run("N8N_URLFromEnv", func(t *testing.T) {
		os.Setenv("N8N_URL", "http://custom-n8n:5678")

		req := httptest.NewRequest("GET", "/api/travels/search?q=test", nil)
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		// Should return OK even if n8n is unavailable (graceful fallback)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("N8N_DefaultURL", func(t *testing.T) {
		os.Unsetenv("N8N_URL")

		req := httptest.NewRequest("GET", "/api/travels/search?q=test", nil)
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		// Should use default localhost:5678 and handle gracefully
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestAddTravelHandler_ValidationErrors tests input validation
func TestAddTravelHandler_ValidationErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		body           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "InvalidJSON",
			method:         "POST",
			body:           "{invalid json}",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "MethodNotAllowed_GET",
			method:         "GET",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "MethodNotAllowed_DELETE",
			method:         "DELETE",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "MethodNotAllowed_PUT",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "OPTIONS_Success",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/api/add-travel", bytes.NewBufferString(tt.body))
				if tt.contentType != "" {
					req.Header.Set("Content-Type", tt.contentType)
				}
			} else {
				req = httptest.NewRequest(tt.method, "/api/add-travel", nil)
			}

			w := httptest.NewRecorder()
			addTravelHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestDataStructures_EdgeCases tests edge cases in data structures
func TestDataStructures_EdgeCases(t *testing.T) {
	t.Run("Travel_EmptyArrays", func(t *testing.T) {
		travel := &Travel{
			ID:           1,
			UserID:       "test",
			Location:     "Test",
			Photos:       []string{},
			Tags:         []string{},
			CreatedAt:    time.Now(),
		}

		jsonData, err := json.Marshal(travel)
		if err != nil {
			t.Fatalf("Failed to marshal travel with empty arrays: %v", err)
		}

		var unmarshaled Travel
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal travel: %v", err)
		}

		if unmarshaled.Photos == nil {
			t.Error("Expected Photos to be empty slice, got nil")
		}
		if unmarshaled.Tags == nil {
			t.Error("Expected Tags to be empty slice, got nil")
		}
	})

	t.Run("Travel_NilArrays", func(t *testing.T) {
		travel := &Travel{
			ID:        1,
			UserID:    "test",
			Location:  "Test",
			Photos:    nil,
			Tags:      nil,
			CreatedAt: time.Now(),
		}

		jsonData, err := json.Marshal(travel)
		if err != nil {
			t.Fatalf("Failed to marshal travel with nil arrays: %v", err)
		}

		// Verify nil arrays serialize to null
		if !bytes.Contains(jsonData, []byte(`"photos":null`)) {
			t.Error("Expected Photos to serialize to null")
		}
	})

	t.Run("Stats_ZeroValues", func(t *testing.T) {
		stats := &Stats{
			TotalCountries:       0,
			TotalCities:          0,
			TotalContinents:      0,
			TotalDistanceKm:      0.0,
			TotalDaysTraveled:    0,
			WorldCoveragePercent: 0.0,
		}

		jsonData, err := json.Marshal(stats)
		if err != nil {
			t.Fatalf("Failed to marshal stats with zero values: %v", err)
		}

		var unmarshaled Stats
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal stats: %v", err)
		}

		if unmarshaled.TotalCountries != 0 {
			t.Errorf("Expected TotalCountries to be 0, got %d", unmarshaled.TotalCountries)
		}
	})

	t.Run("Achievement_TimeHandling", func(t *testing.T) {
		now := time.Now().UTC()
		achievement := &Achievement{
			Type:        "test",
			Name:        "Test",
			Description: "Description",
			Icon:        "🏆",
			UnlockedAt:  now,
		}

		jsonData, err := json.Marshal(achievement)
		if err != nil {
			t.Fatalf("Failed to marshal achievement: %v", err)
		}

		var unmarshaled Achievement
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal achievement: %v", err)
		}

		// Time should be preserved (within millisecond precision)
		timeDiff := unmarshaled.UnlockedAt.Sub(now)
		if timeDiff > time.Millisecond || timeDiff < -time.Millisecond {
			t.Errorf("Time not preserved correctly: expected %v, got %v", now, unmarshaled.UnlockedAt)
		}
	})

	t.Run("BucketListItem_NegativeValues", func(t *testing.T) {
		item := &BucketListItem{
			ID:             -1,
			Priority:       -5,
			BudgetEstimate: -1000.00,
		}

		jsonData, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("Failed to marshal bucket item with negative values: %v", err)
		}

		var unmarshaled BucketListItem
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal bucket item: %v", err)
		}

		if unmarshaled.ID != -1 {
			t.Errorf("Expected ID to be -1, got %d", unmarshaled.ID)
		}
	})
}

// TestInitDB_EnvValidation tests database initialization environment validation
func TestInitDB_EnvValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Save all original env vars
	originalEnv := map[string]string{
		"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
		"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
		"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
		"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
		"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	tests := []struct {
		name           string
		setEnvVars     map[string]string
		expectError    bool
		errorContains  string
	}{
		{
			name:          "MissingAll",
			setEnvVars:    map[string]string{},
			expectError:   true,
			errorContains: "Missing required database configuration",
		},
		{
			name: "MissingHost",
			setEnvVars: map[string]string{
				"POSTGRES_PORT":     "5432",
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "pass",
				"POSTGRES_DB":       "db",
			},
			expectError:   true,
			errorContains: "Missing required database configuration",
		},
		{
			name: "MissingPort",
			setEnvVars: map[string]string{
				"POSTGRES_HOST":     "localhost",
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "pass",
				"POSTGRES_DB":       "db",
			},
			expectError:   true,
			errorContains: "Missing required database configuration",
		},
		{
			name: "MissingUser",
			setEnvVars: map[string]string{
				"POSTGRES_HOST":     "localhost",
				"POSTGRES_PORT":     "5432",
				"POSTGRES_PASSWORD": "pass",
				"POSTGRES_DB":       "db",
			},
			expectError:   true,
			errorContains: "Missing required database configuration",
		},
		{
			name: "MissingPassword",
			setEnvVars: map[string]string{
				"POSTGRES_HOST": "localhost",
				"POSTGRES_PORT": "5432",
				"POSTGRES_USER": "user",
				"POSTGRES_DB":   "db",
			},
			expectError:   true,
			errorContains: "Missing required database configuration",
		},
		{
			name: "MissingDB",
			setEnvVars: map[string]string{
				"POSTGRES_HOST":     "localhost",
				"POSTGRES_PORT":     "5432",
				"POSTGRES_USER":     "user",
				"POSTGRES_PASSWORD": "pass",
			},
			expectError:   true,
			errorContains: "Missing required database configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for key := range originalEnv {
				os.Unsetenv(key)
			}

			// Set test-specific env vars
			for key, value := range tt.setEnvVars {
				os.Setenv(key, value)
			}

			err := initDB()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil && !strings.Contains(err.Error(), "connection refused") {
					// Connection refused is OK - it means validation passed
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}

			// Close db if it was opened
			if db != nil {
				db.Close()
			}
		})
	}
}

// TestTestHelpers_DataGeneration tests helper data generation functions
func TestTestHelpers_DataGeneration(t *testing.T) {
	t.Run("TravelRequest_Generation", func(t *testing.T) {
		request := TestData.TravelRequest("user123", "Paris, France")

		if request["user_id"] != "user123" {
			t.Errorf("Expected user_id 'user123', got %v", request["user_id"])
		}

		if request["location"] != "Paris, France" {
			t.Errorf("Expected location 'Paris, France', got %v", request["location"])
		}

		// Verify all required fields are present
		requiredFields := []string{"user_id", "location", "lat", "lng", "date", "type", "notes"}
		for _, field := range requiredFields {
			if _, exists := request[field]; !exists {
				t.Errorf("Expected field '%s' to be present in travel request", field)
			}
		}
	})
}

// TestTestPatterns_Builders tests pattern builder functions
func TestTestPatterns_Builders(t *testing.T) {
	t.Run("BuildTestTravel", func(t *testing.T) {
		travel := BuildTestTravel("test_user_123")

		if travel.UserID != "test_user_123" {
			t.Errorf("Expected UserID 'test_user_123', got %s", travel.UserID)
		}

		if travel.ID == 0 {
			t.Error("Expected ID to be set")
		}

		if travel.Location == "" {
			t.Error("Expected Location to be set")
		}
	})

	t.Run("BuildTestStats", func(t *testing.T) {
		stats := BuildTestStats()

		if stats.TotalCountries <= 0 {
			t.Error("Expected TotalCountries to be positive")
		}

		if stats.TotalCities <= 0 {
			t.Error("Expected TotalCities to be positive")
		}
	})

	t.Run("BuildTestAchievement", func(t *testing.T) {
		achievement := BuildTestAchievement("first_trip")

		if achievement.Type != "first_trip" {
			t.Errorf("Expected Type 'first_trip', got %s", achievement.Type)
		}

		if achievement.Name == "" {
			t.Error("Expected Name to be set")
		}
	})

	t.Run("BuildTestBucketItem", func(t *testing.T) {
		item := BuildTestBucketItem()

		if item.Location == "" {
			t.Error("Expected Location to be set")
		}

		if item.Priority == 0 {
			t.Error("Expected Priority to be set")
		}
	})
}

// TestCORSPreflightRequests tests CORS preflight handling
func TestCORSPreflightRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	endpoints := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/health", healthHandler},
		{"/api/travels", travelsHandler},
		{"/api/stats", statsHandler},
		{"/api/achievements", achievementsHandler},
		{"/api/bucket-list", bucketListHandler},
		{"/api/add-travel", addTravelHandler},
		{"/api/travels/search", searchTravelsHandler},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.path, func(t *testing.T) {
			req := httptest.NewRequest("OPTIONS", endpoint.path, nil)
			w := httptest.NewRecorder()

			endpoint.handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for OPTIONS request, got %d", w.Code)
			}

			// Verify CORS headers
			if w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Error("Missing or incorrect Access-Control-Allow-Origin header")
			}
		})
	}
}
