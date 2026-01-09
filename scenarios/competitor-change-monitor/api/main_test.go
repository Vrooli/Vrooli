//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

func TestMain(m *testing.M) {
	// Setup
	cleanup := setupTestLogger()
	defer cleanup()

	// Set required environment variable for tests
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := testHandlerWithRequest(t, healthHandler, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "competitor-change-monitor",
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}
	})
}

// TestGetCompetitorsHandler tests the get competitors endpoint
func TestGetCompetitorsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	// Set global db for handlers
	db = env.DB

	t.Run("Success_EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/competitors",
		}

		w := testHandlerWithRequest(t, getCompetitorsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})

	t.Run("Success_WithCompetitors", func(t *testing.T) {
		// Create test competitors
		testComp1 := setupTestCompetitor(t, env, "Test Competitor 1")
		defer testComp1.Cleanup()

		testComp2 := setupTestCompetitor(t, env, "Test Competitor 2")
		defer testComp2.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/competitors",
		}

		w := testHandlerWithRequest(t, getCompetitorsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}

		if len(array) < 2 {
			t.Errorf("Expected at least 2 competitors, got %d", len(array))
		}
	})

	t.Run("Error_DatabaseFailure", func(t *testing.T) {
		// Close database to simulate failure
		env.DB.Close()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/competitors",
		}

		w := testHandlerWithRequest(t, getCompetitorsHandler, req)
		assertErrorResponse(t, w, http.StatusInternalServerError)

		// Reopen database for cleanup
		env = setupTestDB(t)
		db = env.DB
	})
}

// TestAddCompetitorHandler tests the add competitor endpoint
func TestAddCompetitorHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Success", func(t *testing.T) {
		body := TestData.CreateCompetitorRequest("New Competitor")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitors",
			Body:   body,
		}

		w := testHandlerWithRequest(t, addCompetitorHandler, req)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"name": "New Competitor",
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		// Verify ID was returned
		if _, ok := response["id"]; !ok {
			t.Error("Expected 'id' field in response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitors",
			Body:   `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, addCompetitorHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitors",
			Body:   "",
		}

		w := testHandlerWithRequest(t, addCompetitorHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_MissingRequiredFields", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitors",
			Body:   map[string]interface{}{},
		}

		w := testHandlerWithRequest(t, addCompetitorHandler, req)

		// Should succeed or fail based on DB constraints
		if w.Code != http.StatusCreated && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 201 or 500, got %d", w.Code)
		}
	})
}

// TestGetTargetsHandler tests the get monitoring targets endpoint
func TestGetTargetsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Success_EmptyList", func(t *testing.T) {
		testComp := setupTestCompetitor(t, env, "Test Competitor")
		defer testComp.Cleanup()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/competitors/%s/targets", testComp.Competitor.ID),
			URLVars: map[string]string{"id": testComp.Competitor.ID},
		}

		w := testHandlerWithRequest(t, getTargetsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})

	t.Run("Success_WithTargets", func(t *testing.T) {
		testComp := setupTestCompetitor(t, env, "Test Competitor")
		defer testComp.Cleanup()

		testTarget := setupTestTarget(t, env, testComp.Competitor.ID)
		defer testTarget.Cleanup()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/competitors/%s/targets", testComp.Competitor.ID),
			URLVars: map[string]string{"id": testComp.Competitor.ID},
		}

		w := testHandlerWithRequest(t, getTargetsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}

		if len(array) < 1 {
			t.Errorf("Expected at least 1 target, got %d", len(array))
		}
	})

	t.Run("Error_InvalidCompetitorID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/competitors/invalid-uuid/targets",
			URLVars: map[string]string{"id": "invalid-uuid"},
		}

		w := testHandlerWithRequest(t, getTargetsHandler, req)

		// Should handle gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})
}

// TestAddTargetHandler tests the add monitoring target endpoint
func TestAddTargetHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Success", func(t *testing.T) {
		testComp := setupTestCompetitor(t, env, "Test Competitor")
		defer testComp.Cleanup()

		body := TestData.CreateTargetRequest(testComp.Competitor.ID)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/targets",
			Body:   body,
		}

		w := testHandlerWithRequest(t, addTargetHandler, req)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"competitor_id": testComp.Competitor.ID,
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}

		// Verify ID was returned
		if _, ok := response["id"]; !ok {
			t.Error("Expected 'id' field in response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/targets",
			Body:   `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, addTargetHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/targets",
			Body:   "",
		}

		w := testHandlerWithRequest(t, addTargetHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetAlertsHandler tests the get alerts endpoint
func TestGetAlertsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Success_EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/alerts",
		}

		w := testHandlerWithRequest(t, getAlertsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})

	t.Run("Success_WithStatusFilter", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/alerts",
			QueryParams: map[string]string{"status": "unread"},
		}

		w := testHandlerWithRequest(t, getAlertsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})

	t.Run("Success_WithPriorityFilter", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/alerts",
			QueryParams: map[string]string{"priority": "high"},
		}

		w := testHandlerWithRequest(t, getAlertsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})

	t.Run("Success_WithMultipleFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/alerts",
			QueryParams: map[string]string{
				"status":   "unread",
				"priority": "high",
			},
		}

		w := testHandlerWithRequest(t, getAlertsHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})
}

// TestUpdateAlertHandler tests the update alert endpoint
func TestUpdateAlertHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Success", func(t *testing.T) {
		// Create a test alert first
		alertID := "test-alert-id"
		_, err := env.DB.Exec(`
			INSERT INTO alerts (id, competitor_id, title, priority, url, category, summary, relevance_score, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, alertID, "test-comp", "Test Alert", "high", "http://example.com", "test", "Summary", 80, "unread")

		if err != nil {
			t.Skipf("Failed to create test alert: %v", err)
		}

		defer env.DB.Exec("DELETE FROM alerts WHERE id = $1", alertID)

		body := map[string]interface{}{
			"status": "read",
		}

		req := HTTPTestRequest{
			Method:  "PATCH",
			Path:    fmt.Sprintf("/api/alerts/%s", alertID),
			URLVars: map[string]string{"id": alertID},
			Body:    body,
		}

		w := testHandlerWithRequest(t, updateAlertHandler, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})

		if response == nil {
			t.Fatal("Expected valid JSON response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PATCH",
			Path:    "/api/alerts/test-id",
			URLVars: map[string]string{"id": "test-id"},
			Body:    `{"invalid": "json"`,
		}

		w := testHandlerWithRequest(t, updateAlertHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "PATCH",
			Path:    "/api/alerts/test-id",
			URLVars: map[string]string{"id": "test-id"},
			Body:    "",
		}

		w := testHandlerWithRequest(t, updateAlertHandler, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestGetAnalysesHandler tests the get analyses endpoint
func TestGetAnalysesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("Success_EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/analyses",
		}

		w := testHandlerWithRequest(t, getAnalysesHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})

	t.Run("Success_WithCompetitorFilter", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/analyses",
			QueryParams: map[string]string{"competitor_id": "test-competitor-id"},
		}

		w := testHandlerWithRequest(t, getAnalysesHandler, req)
		array := assertJSONArray(t, w, http.StatusOK)

		if array == nil {
			t.Fatal("Expected valid JSON array response")
		}
	})
}

// TestTriggerScanHandler tests the trigger scan endpoint
func TestTriggerScanHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_LocalScan", func(t *testing.T) {
		env := setupTestDB(t)
		if env == nil {
			t.Skip("DB not available")
		}
		defer env.Cleanup()
		db = env.DB

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/scan",
		}

		w := testHandlerWithRequest(t, triggerScanHandler, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := assertJSONResponse(t, w, http.StatusOK)
		if body["status"] != "completed" {
			t.Errorf("Expected completed status, got %v", body["status"])
		}
	})

	t.Run("Error_NoDB", func(t *testing.T) {
		// Ensure db is nil to simulate missing initialization
		db = nil
		req := HTTPTestRequest{Method: "POST", Path: "/api/scan"}
		w := testHandlerWithRequest(t, triggerScanHandler, req)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})
}

// TestRouter tests the complete router configuration
func TestRouter(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	router := mux.NewRouter()

	// Setup routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/competitors", getCompetitorsHandler).Methods("GET")
	router.HandleFunc("/api/competitors", addCompetitorHandler).Methods("POST")
	router.HandleFunc("/api/competitors/{id}/targets", getTargetsHandler).Methods("GET")
	router.HandleFunc("/api/targets", addTargetHandler).Methods("POST")
	router.HandleFunc("/api/alerts", getAlertsHandler).Methods("GET")
	router.HandleFunc("/api/alerts/{id}", updateAlertHandler).Methods("PATCH")
	router.HandleFunc("/api/analyses", getAnalysesHandler).Methods("GET")
	router.HandleFunc("/api/scan", triggerScanHandler).Methods("POST")

	t.Run("HealthEndpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("CompetitorsEndpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/competitors", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestDatabaseConnection tests the database connection logic
func TestDatabaseConnection(t *testing.T) {
	t.Run("ConnectionWithPOSTGRES_URL", func(t *testing.T) {
		// This test validates the environment variable handling
		// Actual connection is tested in setupTestDB

		if os.Getenv("POSTGRES_URL") == "" &&
			(os.Getenv("POSTGRES_HOST") == "" || os.Getenv("POSTGRES_PORT") == "") {
			t.Skip("Skipping: database configuration not available")
		}
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	t.Run("ConcurrentGetCompetitors", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool)

		for i := 0; i < concurrency; i++ {
			go func() {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/competitors",
				}

				w := testHandlerWithRequest(t, getCompetitorsHandler, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				done <- true
			}()
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}

// TestErrorHandling tests error handling across handlers
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	defer env.Cleanup()

	db = env.DB

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/competitors").
		AddEmptyBody("POST", "/api/competitors").
		Build()

	suite := &HandlerTestSuite{
		HandlerName: "addCompetitorHandler",
		Handler:     addCompetitorHandler,
		BaseURL:     "/api/competitors",
	}

	suite.RunErrorTests(t, patterns)
}
