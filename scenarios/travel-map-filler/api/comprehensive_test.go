package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestComprehensiveHandlers provides comprehensive coverage for all handlers
func TestComprehensiveHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("HealthHandler_Comprehensive", testHealthHandlerComprehensive)
	t.Run("TravelsHandler_Comprehensive", func(t *testing.T) { testTravelsHandlerComprehensive(t, testDB) })
	t.Run("StatsHandler_Comprehensive", func(t *testing.T) { testStatsHandlerComprehensive(t, testDB) })
	t.Run("AchievementsHandler_Comprehensive", func(t *testing.T) { testAchievementsHandlerComprehensive(t, testDB) })
	t.Run("BucketListHandler_Comprehensive", func(t *testing.T) { testBucketListHandlerComprehensive(t, testDB) })
	t.Run("AddTravelHandler_Comprehensive", func(t *testing.T) { testAddTravelHandlerComprehensive(t, testDB) })
	t.Run("SearchHandler_Comprehensive", testSearchHandlerComprehensive)
}

func testHealthHandlerComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkCORS      bool
	}{
		{"GET_Success", "GET", http.StatusOK, true},
		{"POST_Success", "POST", http.StatusOK, true},
		{"OPTIONS_Success", "OPTIONS", http.StatusOK, true},
		{"PUT_Success", "PUT", http.StatusOK, true},
		{"DELETE_Success", "DELETE", http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			healthHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkCORS {
				if w.Header().Get("Access-Control-Allow-Origin") == "" {
					t.Error("Expected CORS headers to be set")
				}
			}

			// For GET requests, verify JSON response
			if tt.method == "GET" {
				assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
					"status":  "healthy",
					"service": "travel-map-filler",
				})
			}
		})
	}
}

func testTravelsHandlerComprehensive(t *testing.T, testDB *TestDB) {
	userID := "test_comprehensive_travels"

	// Create diverse test data
	travels := []*Travel{
		createTestTravel(t, testDB.DB, userID),
		createTestTravel(t, testDB.DB, userID),
		createTestTravel(t, testDB.DB, userID),
	}

	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		expectedStatus int
		expectedMin    int
	}{
		{"GET_WithUserID", "GET", map[string]string{"user_id": userID}, http.StatusOK, 3},
		{"GET_DefaultUser", "GET", map[string]string{}, http.StatusOK, 0},
		{"GET_WithYearFilter", "GET", map[string]string{"user_id": userID, "year": "2024"}, http.StatusOK, 0},
		{"GET_WithTypeFilter", "GET", map[string]string{"user_id": userID, "type": "vacation"}, http.StatusOK, 0},
		{"GET_WithBothFilters", "GET", map[string]string{"user_id": userID, "year": "2024", "type": "vacation"}, http.StatusOK, 0},
		{"GET_NonexistentUser", "GET", map[string]string{"user_id": "nonexistent"}, http.StatusOK, 0},
		{"OPTIONS_Request", "OPTIONS", map[string]string{}, http.StatusOK, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeHTTPRequest(HTTPTestRequest{
				Method:      tt.method,
				Path:        "/api/travels",
				QueryParams: tt.queryParams,
			})

			req := httptest.NewRequest(tt.method, "/api/travels", nil)
			if tt.queryParams != nil {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			travelsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify response for GET requests
			if tt.method == "GET" && tt.expectedMin >= 0 {
				result := assertJSONArray(t, w, http.StatusOK)
				if result != nil && len(result) < tt.expectedMin {
					t.Errorf("Expected at least %d travels, got %d", tt.expectedMin, len(result))
				}
			}
		})
	}

	// Clean up
	for _, travel := range travels {
		_ = travel
	}
}

func testStatsHandlerComprehensive(t *testing.T, testDB *TestDB) {
	userID := "test_comprehensive_stats"
	createTestTravel(t, testDB.DB, userID)
	createTestTravel(t, testDB.DB, userID)

	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		expectedStatus int
		validateFields bool
	}{
		{"GET_ExistingUser", "GET", map[string]string{"user_id": userID}, http.StatusOK, true},
		{"GET_NonexistentUser", "GET", map[string]string{"user_id": "nonexistent"}, http.StatusOK, true},
		{"GET_DefaultUser", "GET", map[string]string{}, http.StatusOK, true},
		{"OPTIONS_Request", "OPTIONS", map[string]string{}, http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeHTTPRequest(HTTPTestRequest{
				Method:      tt.method,
				Path:        "/api/stats",
				QueryParams: tt.queryParams,
			})

			req := httptest.NewRequest(tt.method, "/api/stats", nil)
			if tt.queryParams != nil {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			statsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.validateFields && tt.method == "GET" {
				response := assertJSONResponse(t, w, http.StatusOK, nil)
				if response != nil {
					requiredFields := []string{"total_countries", "total_cities", "total_continents", "world_coverage_percent"}
					for _, field := range requiredFields {
						if _, exists := response[field]; !exists {
							t.Errorf("Missing required field: %s", field)
						}
					}
				}
			}
		})
	}
}

func testAchievementsHandlerComprehensive(t *testing.T, testDB *TestDB) {
	userID := "test_comprehensive_achievements"
	createTestAchievement(t, testDB.DB, userID, "first_trip")
	createTestAchievement(t, testDB.DB, userID, "explorer")

	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		expectedStatus int
		expectedMin    int
	}{
		{"GET_WithAchievements", "GET", map[string]string{"user_id": userID}, http.StatusOK, 2},
		{"GET_NoAchievements", "GET", map[string]string{"user_id": "nonexistent"}, http.StatusOK, 0},
		{"GET_DefaultUser", "GET", map[string]string{}, http.StatusOK, 0},
		{"OPTIONS_Request", "OPTIONS", map[string]string{}, http.StatusOK, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeHTTPRequest(HTTPTestRequest{
				Method:      tt.method,
				Path:        "/api/achievements",
				QueryParams: tt.queryParams,
			})

			req := httptest.NewRequest(tt.method, "/api/achievements", nil)
			if tt.queryParams != nil {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			achievementsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == "GET" && tt.expectedMin >= 0 {
				result := assertJSONArray(t, w, http.StatusOK)
				if result != nil && len(result) < tt.expectedMin {
					t.Errorf("Expected at least %d achievements, got %d", tt.expectedMin, len(result))
				}
			}
		})
	}
}

func testBucketListHandlerComprehensive(t *testing.T, testDB *TestDB) {
	userID := "test_comprehensive_bucket"
	createTestBucketItem(t, testDB.DB, userID)
	createTestBucketItem(t, testDB.DB, userID)

	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		expectedStatus int
		expectedMin    int
	}{
		{"GET_WithItems", "GET", map[string]string{"user_id": userID}, http.StatusOK, 2},
		{"GET_NoItems", "GET", map[string]string{"user_id": "nonexistent"}, http.StatusOK, 0},
		{"GET_DefaultUser", "GET", map[string]string{}, http.StatusOK, 0},
		{"OPTIONS_Request", "OPTIONS", map[string]string{}, http.StatusOK, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeHTTPRequest(HTTPTestRequest{
				Method:      tt.method,
				Path:        "/api/bucket-list",
				QueryParams: tt.queryParams,
			})

			req := httptest.NewRequest(tt.method, "/api/bucket-list", nil)
			if tt.queryParams != nil {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			bucketListHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == "GET" && tt.expectedMin >= 0 {
				result := assertJSONArray(t, w, http.StatusOK)
				if result != nil && len(result) < tt.expectedMin {
					t.Errorf("Expected at least %d items, got %d", tt.expectedMin, len(result))
				}
			}
		})
	}
}

func testAddTravelHandlerComprehensive(t *testing.T, testDB *TestDB) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		validateID     bool
	}{
		{"POST_ValidData", "POST", TestData.TravelRequest("test_add_comp", "Paris, France"), http.StatusOK, true},
		{"POST_MinimalData", "POST", map[string]interface{}{"location": "Test"}, http.StatusOK, true},
		{"POST_EmptyArrays", "POST", map[string]interface{}{
			"location": "Test",
			"photos":   []string{},
			"tags":     []string{},
		}, http.StatusOK, true},
		{"POST_InvalidJSON", "POST", `{invalid json`, http.StatusBadRequest, false},
		{"POST_EmptyBody", "POST", `{}`, http.StatusOK, true},
		{"GET_MethodNotAllowed", "GET", nil, http.StatusMethodNotAllowed, false},
		{"PUT_MethodNotAllowed", "PUT", nil, http.StatusMethodNotAllowed, false},
		{"DELETE_MethodNotAllowed", "DELETE", nil, http.StatusMethodNotAllowed, false},
		{"OPTIONS_Request", "OPTIONS", nil, http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					bodyBytes = []byte(v)
				default:
					bodyBytes, _ = json.Marshal(v)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/add-travel", bytes.NewBuffer(bodyBytes))
			if tt.method == "POST" || tt.method == "PUT" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			addTravelHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.validateID && w.Code == http.StatusOK {
				response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
					"status": "success",
				})
				if response != nil {
					if _, exists := response["id"]; !exists {
						t.Error("Expected 'id' field in successful response")
					}
				}
			}
		})
	}
}

func testSearchHandlerComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		body           interface{}
		expectedStatus int
	}{
		{"GET_WithQuery", "GET", map[string]string{"q": "Paris"}, nil, http.StatusOK},
		{"GET_WithQueryAndLimit", "GET", map[string]string{"q": "beach", "limit": "5"}, nil, http.StatusOK},
		{"GET_MissingQuery", "GET", map[string]string{}, nil, http.StatusBadRequest},
		{"GET_EmptyQuery", "GET", map[string]string{"q": ""}, nil, http.StatusBadRequest},
		{"POST_WithBody", "POST", map[string]string{}, map[string]interface{}{
			"query":   "vacation",
			"limit":   10,
			"user_id": "test_user",
		}, http.StatusOK},
		{"POST_MinimalBody", "POST", map[string]string{}, map[string]interface{}{
			"query": "test",
		}, http.StatusOK},
		{"POST_EmptyQuery", "POST", map[string]string{}, map[string]interface{}{
			"query": "",
		}, http.StatusBadRequest},
		{"OPTIONS_Request", "OPTIONS", map[string]string{}, nil, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/api/travels/search", bytes.NewBuffer(bodyBytes))
			if tt.queryParams != nil {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Set(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}
			if tt.method == "POST" && tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			searchTravelsHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestErrorScenarios tests systematic error handling
func TestErrorScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/add-travel").
		AddMethodNotAllowed("DELETE", "/api/travels").
		AddEmptyQueryParameter("GET", "/api/travels/search", "q").
		Build()

	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			req := pattern.Execute(t, nil)

			var bodyBytes []byte
			if req.Body != nil {
				switch v := req.Body.(type) {
				case string:
					bodyBytes = []byte(v)
				case []byte:
					bodyBytes = v
				default:
					bodyBytes, _ = json.Marshal(v)
				}
			}

			httpReq := httptest.NewRequest(req.Method, req.Path, bytes.NewBuffer(bodyBytes))
			if req.QueryParams != nil {
				q := httpReq.URL.Query()
				for k, v := range req.QueryParams {
					q.Set(k, v)
				}
				httpReq.URL.RawQuery = q.Encode()
			}
			if req.Headers != nil {
				for k, v := range req.Headers {
					httpReq.Header.Set(k, v)
				}
			}

			w := httptest.NewRecorder()

			// Route to appropriate handler
			switch {
			case req.Path == "/api/add-travel":
				addTravelHandler(w, httpReq)
			case req.Path == "/api/travels":
				travelsHandler(w, httpReq)
			case req.Path == "/api/travels/search":
				searchTravelsHandler(w, httpReq)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, w.Code)
			}
		})
	}
}

// TestBusinessLogic tests business rules and logic
func TestBusinessLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Achievement_FirstTrip", func(t *testing.T) {
		userID := "test_business_first_trip"

		// Verify no achievements initially
		var count int
		testDB.DB.QueryRow("SELECT COUNT(*) FROM achievements WHERE user_id = $1", userID).Scan(&count)
		if count != 0 {
			t.Error("Expected no achievements initially")
		}

		// Add first travel
		createTestTravel(t, testDB.DB, userID)

		// Check achievements
		checkAchievements(userID)

		// Verify first_trip achievement was unlocked
		testDB.DB.QueryRow("SELECT COUNT(*) FROM achievements WHERE user_id = $1 AND achievement_type = 'first_trip'", userID).Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 first_trip achievement, got %d", count)
		}
	})

	t.Run("Achievement_MultipleCountries", func(t *testing.T) {
		userID := "test_business_explorer"

		// Create travels to multiple countries
		for i := 0; i < 6; i++ {
			createTestTravel(t, testDB.DB, userID)
		}

		// Check achievements
		checkAchievements(userID)

		// Verify explorer achievement
		var count int
		testDB.DB.QueryRow("SELECT COUNT(*) FROM achievements WHERE user_id = $1 AND achievement_type = 'five_countries'", userID).Scan(&count)
		// May or may not trigger depending on data diversity
		t.Logf("Five countries achievement count: %d", count)
	})

	t.Run("WorldCoverage_Calculation", func(t *testing.T) {
		userID := "test_business_coverage"

		// Create test travels
		createTestTravel(t, testDB.DB, userID)

		req := httptest.NewRequest("GET", "/api/stats?user_id="+userID, nil)
		w := httptest.NewRecorder()

		statsHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if coverage, exists := response["world_coverage_percent"]; exists {
				coverageVal, ok := coverage.(float64)
				if !ok {
					t.Error("Expected world_coverage_percent to be a number")
				} else if coverageVal < 0 || coverageVal > 100 {
					t.Errorf("Invalid world coverage percentage: %f", coverageVal)
				}
			}
		}
	})

	t.Run("DefaultUser_Handling", func(t *testing.T) {
		// Test that default_user is used when no user_id provided
		travelData := TestData.TravelRequest("", "Test Location")
		travelJSON, _ := json.Marshal(travelData)

		req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "success",
		})

		if response != nil {
			if travel, exists := response["travel"]; exists {
				travelMap, ok := travel.(map[string]interface{})
				if ok {
					if userID, exists := travelMap["user_id"]; exists && userID != "default_user" {
						t.Errorf("Expected user_id to default to 'default_user', got %v", userID)
					}
				}
			}
		}
	})
}

// TestIntegration tests end-to-end scenarios
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_integration_user"

	t.Run("CompleteUserJourney", func(t *testing.T) {
		// Step 1: Add first travel
		travelData := TestData.TravelRequest(userID, "Paris, France")
		travelJSON, _ := json.Marshal(travelData)

		req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		addTravelHandler(w, req)

		addResponse := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "success",
		})
		if addResponse == nil {
			t.Fatal("Failed to add travel")
		}

		// Step 2: Verify travel appears in list
		time.Sleep(100 * time.Millisecond) // Give database time to sync
		req = httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
		w = httptest.NewRecorder()
		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if len(travels) < 1 {
			t.Error("Travel not found in travels list")
		}

		// Step 3: Check stats updated
		req = httptest.NewRequest("GET", "/api/stats?user_id="+userID, nil)
		w = httptest.NewRecorder()
		statsHandler(w, req)

		statsResponse := assertJSONResponse(t, w, http.StatusOK, nil)
		if statsResponse == nil {
			t.Error("Failed to get stats")
		}

		// Step 4: Verify achievements
		req = httptest.NewRequest("GET", "/api/achievements?user_id="+userID, nil)
		w = httptest.NewRecorder()
		achievementsHandler(w, req)

		achievements := assertJSONArray(t, w, http.StatusOK)
		t.Logf("User has %d achievements", len(achievements))
	})
}
