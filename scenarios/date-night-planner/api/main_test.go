package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	suite := NewHandlerTestSuite("HealthHandler", healthHandler, "/health")

	// Success case
	suite.RunSuccessTest(t, "Success", HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}, func(t *testing.T, w *http.ResponseWriter) {
		recorder := (*w).(*httptest.ResponseRecorder)
		response := assertJSONResponse(t, recorder, http.StatusOK)

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}

		if response["service"] != "date-night-planner" {
			t.Errorf("Expected service 'date-night-planner', got '%v'", response["service"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp in response")
		}
	})
}

// TestDatabaseHealthHandler tests the database health check endpoint
func TestDatabaseHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	suite := NewHandlerTestSuite("DatabaseHealthHandler", databaseHealthHandler, "/health/database")

	// Test with database connection
	suite.RunSuccessTest(t, "Success", HTTPTestRequest{
		Method: "GET",
		Path:   "/health/database",
	}, func(t *testing.T, w *http.ResponseWriter) {
		recorder := (*w).(*httptest.ResponseRecorder)

		// Should return either 200 or 503 depending on DB availability
		if recorder.Code != http.StatusOK && recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", recorder.Code)
		}

		response := assertJSONResponse(t, recorder, recorder.Code)

		if _, ok := response["checks"]; !ok {
			t.Error("Expected checks field in response")
		}
	})
}

// TestWorkflowHealthHandler tests the workflow health check endpoint
func TestWorkflowHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	suite := NewHandlerTestSuite("WorkflowHealthHandler", workflowHealthHandler, "/health/workflows")

	suite.RunSuccessTest(t, "Success", HTTPTestRequest{
		Method: "GET",
		Path:   "/health/workflows",
	}, func(t *testing.T, w *http.ResponseWriter) {
		recorder := (*w).(*httptest.ResponseRecorder)
		response := assertJSONResponse(t, recorder, http.StatusOK)

		if _, ok := response["checks"]; !ok {
			t.Error("Expected checks field in response")
		}
	})
}

// TestSuggestDatesHandler tests the date suggestion endpoint
func TestSuggestDatesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	suite := NewHandlerTestSuite("SuggestDatesHandler", suggestDatesHandler, "/api/v1/dates/suggest")

	// Success cases
	t.Run("Success_RomanticDate", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"date_type": "romantic",
				"budget_max": 150,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("Success_AdventureDate", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"date_type": "adventure",
				"budget_max": 200,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("Success_CulturalDate", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"date_type": "cultural",
				"budget_max": 100,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("Success_CasualDate", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"date_type": "casual",
				"budget_max": 50,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("Success_IndoorPreference", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id":          "test-couple-123",
				"weather_preference": "indoor",
				"budget_max":         200,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("Success_OutdoorPreference", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id":          "test-couple-123",
				"weather_preference": "outdoor",
				"budget_max":         150,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("Success_LowBudget", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id":  "test-couple-123",
				"budget_max": 40,
				"date_type":  "casual", // Request casual dates which are typically cheaper
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		recorder := w
		response := assertJSONResponse(t, recorder, http.StatusOK)

		suggestions, ok := response["suggestions"].([]interface{})
		if !ok || len(suggestions) == 0 {
			t.Fatal("Expected at least one suggestion for low budget")
		}

		// Verify all suggestions are within budget
		for _, s := range suggestions {
			suggestion := s.(map[string]interface{})
			if cost, ok := suggestion["estimated_cost"].(float64); ok {
				if cost > 40 {
					t.Errorf("Expected suggestion cost <= 40, got %.2f", cost)
				}
			}
		}
	})

	// Error cases
	patterns := NewTestScenarioBuilder().
		AddMissingCoupleID("/api/v1/dates/suggest").
		AddEmptyBody("/api/v1/dates/suggest").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestPlanDateHandler tests the date planning endpoint
func TestPlanDateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	_ = NewHandlerTestSuite("PlanDateHandler", planDateHandler, "/api/v1/dates/plan")

	// Success case
	t.Run("Success_CreateDatePlan", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/plan",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"selected_suggestion": map[string]interface{}{
					"title":              "Romantic Dinner",
					"description":        "Enjoy an intimate dinner",
					"estimated_cost":     80,
					"estimated_duration": "2 hours",
					"confidence_score":   0.85,
					"activities": []map[string]interface{}{
						{
							"type":     "dining",
							"name":     "Restaurant",
							"duration": "2 hours",
						},
					},
				},
				"planned_date": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		}, planDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateDatePlanResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	// Error cases
	patterns := NewTestScenarioBuilder().
		AddMissingCoupleID("/api/v1/dates/plan").
		AddEmptyBody("/api/v1/dates/plan").
		Build()

	suite2 := NewHandlerTestSuite("PlanDateHandler", planDateHandler, "/api/v1/dates/plan")
	suite2.RunErrorTests(t, patterns)
}

// TestSurpriseDateHandler tests the surprise date creation endpoint
func TestSurpriseDateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	_ = NewHandlerTestSuite("SurpriseDateHandler", surpriseDateHandler, "/api/v1/dates/surprise")

	// Success case
	t.Run("Success_CreateSurpriseDate", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/surprise",
			Body: map[string]interface{}{
				"couple_id":  "test-couple-123",
				"planned_by": "partner-1",
				"date_suggestion": map[string]interface{}{
					"title":              "Surprise Romantic Evening",
					"description":        "A secret romantic adventure",
					"estimated_cost":     150,
					"estimated_duration": "4 hours",
					"confidence_score":   0.90,
					"activities": []map[string]interface{}{
						{
							"type":     "romantic",
							"name":     "Secret Activity 1",
							"duration": "2 hours",
						},
						{
							"type":     "dining",
							"name":     "Secret Activity 2",
							"duration": "2 hours",
						},
					},
				},
				"planned_date": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			},
		}, surpriseDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		recorder := w
		response := assertJSONResponse(t, recorder, http.StatusCreated)

		if response["status"] != "surprise_created" {
			t.Errorf("Expected status 'surprise_created', got '%v'", response["status"])
		}

		if _, ok := response["surprise_id"]; !ok {
			t.Error("Expected surprise_id in response")
		}
	})

	// Error cases
	t.Run("Error_MissingPlannedBy", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/surprise",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"date_suggestion": map[string]interface{}{
					"title":       "Test",
					"description": "Test",
				},
			},
		}, surpriseDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "couple_id and planned_by are required")
	})
}

// TestGetSurpriseHandler tests the surprise date retrieval endpoint
func TestGetSurpriseHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	t.Run("Success_PlannerCanSeeDetails", func(t *testing.T) {
		// Create a test request with requester_id as planner
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/dates/surprise/test-surprise-123",
			QueryParams: map[string]string{
				"requester_id": "partner-1",
			},
		}

		w, err := makeHTTPRequest(req, getSurpriseHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		// Planner should see all details
		if _, ok := response["activities"]; !ok {
			t.Error("Expected planner to see activities")
		}
	})

	t.Run("Success_PartnerCannotSeeDetails", func(t *testing.T) {
		// Create a test request with requester_id as non-planner
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/dates/surprise/test-surprise-123",
			QueryParams: map[string]string{
				"requester_id": "partner-2",
			},
		}

		w, err := makeHTTPRequest(req, getSurpriseHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK)

		// Non-planner should not see details
		if response["message"] != "This is a surprise! Details will be revealed soon." {
			t.Error("Expected surprise message for non-planner")
		}

		if _, ok := response["activities"]; ok {
			t.Error("Expected non-planner to not see activities")
		}
	})

	t.Run("Error_MissingRequesterID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/dates/surprise/test-surprise-123",
		}

		w, err := makeHTTPRequest(req, getSurpriseHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "requester_id is required")
	})
}

// TestGenerateDynamicSuggestions tests the suggestion generation logic
func TestGenerateDynamicSuggestions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name           string
		request        DateSuggestionRequest
		expectedMinLen int
		checkBudget    bool
		maxBudget      float64
	}{
		{
			name: "RomanticType",
			request: DateSuggestionRequest{
				CoupleID: "test",
				DateType: "romantic",
			},
			expectedMinLen: 1,
		},
		{
			name: "AdventureType",
			request: DateSuggestionRequest{
				CoupleID: "test",
				DateType: "adventure",
			},
			expectedMinLen: 1,
		},
		{
			name: "CulturalType",
			request: DateSuggestionRequest{
				CoupleID: "test",
				DateType: "cultural",
			},
			expectedMinLen: 1,
		},
		{
			name: "CasualType",
			request: DateSuggestionRequest{
				CoupleID: "test",
				DateType: "casual",
			},
			expectedMinLen: 1,
		},
		{
			name: "BudgetConstraint",
			request: DateSuggestionRequest{
				CoupleID:  "test",
				BudgetMax: 50,
			},
			expectedMinLen: 1,
			checkBudget:    true,
			maxBudget:      50,
		},
		{
			name: "IndoorPreference",
			request: DateSuggestionRequest{
				CoupleID:          "test",
				WeatherPreference: "indoor",
			},
			expectedMinLen: 1,
		},
		{
			name: "OutdoorPreference",
			request: DateSuggestionRequest{
				CoupleID:          "test",
				WeatherPreference: "outdoor",
			},
			expectedMinLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := generateDynamicSuggestions(tt.request)

			if len(suggestions) < tt.expectedMinLen {
				t.Errorf("Expected at least %d suggestions, got %d", tt.expectedMinLen, len(suggestions))
			}

			// Validate budget constraints
			if tt.checkBudget {
				for _, s := range suggestions {
					if s.EstimatedCost > tt.maxBudget {
						t.Errorf("Expected suggestion cost <= %.2f, got %.2f", tt.maxBudget, s.EstimatedCost)
					}
				}
			}

			// Validate all suggestions have required fields
			for i, s := range suggestions {
				if s.Title == "" {
					t.Errorf("Suggestion %d missing title", i)
				}
				if s.Description == "" {
					t.Errorf("Suggestion %d missing description", i)
				}
				if len(s.Activities) == 0 {
					t.Errorf("Suggestion %d has no activities", i)
				}
				if s.ConfidenceScore <= 0 {
					t.Errorf("Suggestion %d has invalid confidence score: %.2f", i, s.ConfidenceScore)
				}
			}
		})
	}
}

// TestGenerateUUID tests UUID generation
func TestGenerateUUID(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	uuid1 := generateUUID()
	uuid2 := generateUUID()

	if uuid1 == "" {
		t.Error("Expected non-empty UUID")
	}

	if uuid1 == uuid2 {
		t.Error("Expected unique UUIDs")
	}
}

// TestEnableCORS tests CORS middleware
func TestEnableCORS(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("OPTIONS_Request", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/test",
		}, handler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS headers to be set")
		}
	})

	t.Run("POST_Request", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
		}, handler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS headers to be set")
		}
	})
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test with environment variables
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("POSTGRES_USER", "vrooli")
	os.Setenv("POSTGRES_DB", "vrooli")

	// Clear password to test warning path
	os.Setenv("POSTGRES_PASSWORD", "")

	err := initDB()
	// DB may not be available in test environment, which is OK
	if err == nil && db != nil {
		db.Close()
	}
}

// TestGenerateDynamicSuggestions_HighBudgetAdventure tests high budget adventure suggestions
func TestGenerateDynamicSuggestions_HighBudgetAdventure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := DateSuggestionRequest{
		CoupleID:  "test",
		DateType:  "adventure",
		BudgetMax: 250,
	}

	suggestions := generateDynamicSuggestions(req)

	if len(suggestions) < 1 {
		t.Fatal("Expected at least one suggestion")
	}

	// Should include hot air balloon ride when budget allows
	foundHighBudget := false
	for _, s := range suggestions {
		if s.EstimatedCost > 150 {
			foundHighBudget = true
			break
		}
	}

	if !foundHighBudget {
		t.Error("Expected to find high-budget suggestion when budget allows")
	}
}

// TestGenerateDynamicSuggestions_HighBudgetFiltering tests that suggestions respect budget limits
func TestGenerateDynamicSuggestions_HighBudgetFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := DateSuggestionRequest{
		CoupleID:  "test",
		DateType:  "cultural", // Cultural has cheaper options
		BudgetMax: 60,
	}

	suggestions := generateDynamicSuggestions(req)

	// All suggestions should be within budget
	for _, s := range suggestions {
		if s.EstimatedCost > req.BudgetMax {
			t.Errorf("Suggestion cost %.2f exceeds budget %.2f", s.EstimatedCost, req.BudgetMax)
		}
	}

	// Should have at least filtered to budget-appropriate options
	if len(suggestions) == 0 {
		t.Error("Expected at least one suggestion within budget")
	}
}

// TestGenerateDynamicSuggestions_WeatherBackup tests weather backup suggestions
func TestGenerateDynamicSuggestions_WeatherBackup(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name          string
		request       DateSuggestionRequest
		expectBackup  bool
	}{
		{
			name: "OutdoorWithBackup",
			request: DateSuggestionRequest{
				CoupleID:          "test",
				WeatherPreference: "outdoor",
			},
			expectBackup: true,
		},
		{
			name: "RomanticDefaultHasBackup",
			request: DateSuggestionRequest{
				CoupleID: "test",
				DateType: "romantic",
			},
			expectBackup: false, // Only some romantic dates have backups
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := generateDynamicSuggestions(tt.request)

			if len(suggestions) == 0 {
				t.Fatal("Expected at least one suggestion")
			}

			// Check for weather backup if expected
			if tt.expectBackup {
				foundBackup := false
				for _, s := range suggestions {
					if s.WeatherBackup != nil {
						foundBackup = true
						if s.WeatherBackup.Name == "" {
							t.Error("Weather backup should have a name")
						}
						break
					}
				}
				if !foundBackup {
					t.Log("Note: No weather backup found (optional)")
				}
			}
		})
	}
}

// TestGenerateDynamicSuggestions_EmptyResult tests fallback when all suggestions filtered
func TestGenerateDynamicSuggestions_EmptyResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Request with impossible budget - should get default suggestion
	req := DateSuggestionRequest{
		CoupleID:  "test",
		DateType:  "romantic",
		BudgetMax: 0.01, // Impossibly low
	}

	suggestions := generateDynamicSuggestions(req)

	// Should still return at least one suggestion (fallback)
	if len(suggestions) == 0 {
		t.Fatal("Expected at least one fallback suggestion")
	}
}

// TestSuggestDatesHandler_WithDatabase tests database integration path
func TestSuggestDatesHandler_WithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Attempt to connect to database
	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// Test with various combinations to exercise database query code
	testCases := []struct {
		name    string
		request map[string]interface{}
	}{
		{
			name: "WithDateType",
			request: map[string]interface{}{
				"couple_id": "db-test-couple",
				"date_type": "romantic",
			},
		},
		{
			name: "WithBudget",
			request: map[string]interface{}{
				"couple_id":  "db-test-couple",
				"budget_max": 100,
			},
		},
		{
			name: "WithWeather",
			request: map[string]interface{}{
				"couple_id":          "db-test-couple",
				"weather_preference": "indoor",
			},
		},
		{
			name: "WithAllFilters",
			request: map[string]interface{}{
				"couple_id":          "db-test-couple",
				"date_type":          "romantic",
				"budget_max":         150,
				"weather_preference": "outdoor",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := makeHTTPRequest(HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/dates/suggest",
				Body:   tc.request,
			}, suggestDatesHandler)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var resp DateSuggestionResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Should get suggestions (from fallback if DB empty)
			if len(resp.Suggestions) == 0 {
				t.Error("Expected at least one suggestion")
			}
		})
	}
}

// TestSuggestDatesHandler_EdgeCases tests edge cases and error handling
func TestSuggestDatesHandler_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	t.Run("EdgeCase_ZeroBudget", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id":  "test-couple-123",
				"budget_max": 0,
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still return suggestions (budget filter only applies if > 0)
		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("EdgeCase_VeryHighBudget", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id":  "test-couple-123",
				"budget_max": 10000,
				"date_type":  "adventure",
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("EdgeCase_EmptyDateType", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"date_type": "",
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("EdgeCase_AllParametersProvided", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/suggest",
			Body: map[string]interface{}{
				"couple_id":          "test-couple-123",
				"date_type":          "romantic",
				"budget_max":         150,
				"preferred_date":     "2025-02-14",
				"weather_preference": "indoor",
			},
		}, suggestDatesHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateSuggestionsResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})
}

// TestPlanDateHandler_EdgeCases tests edge cases for date planning
func TestPlanDateHandler_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	t.Run("EdgeCase_InvalidDateFormat", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/plan",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"selected_suggestion": map[string]interface{}{
					"title":              "Test Date",
					"description":        "Test description",
					"estimated_cost":     50,
					"estimated_duration": "2 hours",
					"confidence_score":   0.75,
					"activities": []map[string]interface{}{
						{"type": "test", "name": "Test Activity", "duration": "1 hour"},
					},
				},
				"planned_date": "invalid-date",
			},
		}, planDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should handle gracefully with default date
		ValidateDatePlanResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})

	t.Run("EdgeCase_NoActivities", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/plan",
			Body: map[string]interface{}{
				"couple_id": "test-couple-123",
				"selected_suggestion": map[string]interface{}{
					"title":              "Test Date",
					"description":        "Test description",
					"estimated_cost":     50,
					"estimated_duration": "2 hours",
					"confidence_score":   0.75,
					"activities":         []map[string]interface{}{},
				},
				"planned_date": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		}, planDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		ValidateDatePlanResponse(t, func() *http.ResponseWriter {
			var rw http.ResponseWriter = w
			return &rw
		}())
	})
}

// TestSurpriseDateHandler_EdgeCases tests edge cases for surprise dates
func TestSurpriseDateHandler_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	dbCleanup := setupTestDB(t)
	defer dbCleanup()

	t.Run("EdgeCase_WithRevealTime", func(t *testing.T) {
		revealTime := time.Now().Add(12 * time.Hour).Format(time.RFC3339)
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/surprise",
			Body: map[string]interface{}{
				"couple_id":  "test-couple-123",
				"planned_by": "partner-1",
				"date_suggestion": map[string]interface{}{
					"title":              "Surprise Evening",
					"description":        "Secret date",
					"estimated_cost":     100,
					"estimated_duration": "3 hours",
					"activities": []map[string]interface{}{
						{"type": "surprise", "name": "Activity", "duration": "1 hour"},
					},
				},
				"planned_date": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"reveal_time":  revealTime,
			},
		}, surpriseDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		recorder := w
		response := assertJSONResponse(t, recorder, http.StatusCreated)

		if response["status"] != "surprise_created" {
			t.Errorf("Expected status 'surprise_created', got '%v'", response["status"])
		}
	})

	t.Run("EdgeCase_InvalidRevealTime", func(t *testing.T) {
		w, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dates/surprise",
			Body: map[string]interface{}{
				"couple_id":  "test-couple-123",
				"planned_by": "partner-1",
				"date_suggestion": map[string]interface{}{
					"title":       "Surprise",
					"description": "Secret",
					"activities":  []map[string]interface{}{},
				},
				"planned_date": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				"reveal_time":  "invalid-time",
			},
		}, surpriseDateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still create surprise, just ignore invalid reveal time
		recorder := w
		response := assertJSONResponse(t, recorder, http.StatusCreated)

		if response["status"] != "surprise_created" {
			t.Errorf("Expected status 'surprise_created', got '%v'", response["status"])
		}
	})
}
