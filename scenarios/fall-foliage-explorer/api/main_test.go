package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// [REQ:REQ-P0-001] Health endpoint returns meaningful scenario status.
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}, healthHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode response: %v. Body: %s", err, rr.Body.String())
		}

		status, ok := response["status"].(string)
		if !ok || status == "" {
			t.Fatalf("Expected health status string, got %v", response)
		}
	})

	t.Run("WithDatabase", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}, healthHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to decode response: %v. Body: %s", err, rr.Body.String())
		}

		status, ok := response["status"].(string)
		if !ok || status == "" {
			t.Fatalf("Expected health status string, got %v", response)
		}
	})
}

// [REQ:REQ-P0-003] Regions endpoint returns foliage regions and fallback metadata.
func TestRegionsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithDatabase", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		// Create test region
		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/regions",
		}, regionsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		payload, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected payload map, got %T", response.Data)
		}

		regions, ok := payload["regions"].([]interface{})
		if !ok {
			t.Fatal("Expected regions collection in response")
		}

		if len(regions) == 0 {
			t.Error("Expected at least one region")
		}

		meta, ok := payload["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected meta information in regions response")
		}

		if fallback, _ := meta["using_fallback"].(bool); fallback {
			t.Error("Did not expect fallback dataset when database is connected")
		}
	})

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/regions",
		}, regionsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		if response.Status != "success" {
			t.Errorf("Expected success status with fallback dataset, got %s", response.Status)
		}

		payload, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected payload map, got %T", response.Data)
		}

		meta, ok := payload["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected meta information for fallback dataset")
		}

		if fallback, _ := meta["using_fallback"].(bool); !fallback {
			t.Error("Expected fallback metadata when database unavailable")
		}
	})
}

// TestFoliageHandler tests the foliage data endpoint
// [REQ:REQ-P0-003] Foliage endpoint validates input and returns current status.
func TestFoliageHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		setupTestFoliageData(t, testDB.DB, regionID)

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/foliage",
			QueryParams: map[string]string{"region_id": "1"},
		}, foliageHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("MissingRegionID", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/foliage",
		}, foliageHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "region_id")
	})

	t.Run("InvalidRegionID", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/foliage",
			QueryParams: map[string]string{"region_id": "invalid"},
		}, foliageHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("NoObservations", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/foliage",
			QueryParams: map[string]string{"region_id": "99999"},
		}, foliageHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return default data, not error
		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/foliage",
			QueryParams: map[string]string{"region_id": "1"},
		}, foliageHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return mock data
		assertJSONResponse(t, rr, http.StatusOK, true)
	})
}

// [REQ:REQ-P0-005][REQ:REQ-P2-001] Prediction endpoint supports AI and fallback peak generation.
func TestPredictHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MethodNotAllowed", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/predict",
		}, predictHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusMethodNotAllowed, "")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/predict", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		predictHandler(rr, req)

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("MissingRegionID", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/predict",
			Body:   map[string]interface{}{},
		}, predictHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "region_id")
	})

	t.Run("NonExistentRegion", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/predict",
			Body:   map[string]interface{}{"region_id": 99999},
		}, predictHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusNotFound, "")
	})

	t.Run("FallbackMode", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		// Set invalid Ollama URL to trigger fallback
		restoreEnv := setTestEnv(t, "OLLAMA_URL", "http://invalid:9999")
		defer restoreEnv()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/predict",
			Body:   map[string]interface{}{"region_id": float64(regionID)},
		}, predictHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should succeed with fallback
		assertJSONResponse(t, rr, http.StatusOK, true)

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		method, ok := data["method"].(string)
		if !ok || method != "typical_week_fallback" {
			t.Errorf("Expected fallback method, got %v", method)
		}
	})
}

// [REQ:REQ-P0-004] Weather endpoint returns stored weather data for a region and date.
func TestWeatherHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/weather",
			QueryParams: map[string]string{"region_id": "1"},
		}, weatherHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("MissingRegionID", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/weather",
		}, weatherHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "region_id")
	})

	t.Run("InvalidRegionID", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/weather",
			QueryParams: map[string]string{"region_id": "invalid"},
		}, weatherHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("WithDate", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/weather",
			QueryParams: map[string]string{
				"region_id": "1",
				"date":      "2025-10-01",
			},
		}, weatherHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/weather",
			QueryParams: map[string]string{"region_id": "1"},
		}, weatherHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return mock data
		assertJSONResponse(t, rr, http.StatusOK, true)
	})
}

// [REQ:REQ-P1-001][REQ:REQ-P1-004] Reports endpoint accepts reports and photo metadata.
func TestReportsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GET_Success", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		reportID := setupTestUserReport(t, testDB.DB, regionID)
		defer testDB.DB.Exec("DELETE FROM user_reports WHERE id = $1", reportID)

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/reports",
			QueryParams: map[string]string{"region_id": "1"},
		}, reportsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("GET_MissingRegionID", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/reports",
		}, reportsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "region_id")
	})

	t.Run("POST_Success", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body: map[string]interface{}{
				"region_id":      regionID,
				"foliage_status": "peak",
				"description":    "Test report",
			},
		}, reportsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("POST_MissingFields", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body:   map[string]interface{}{},
		}, reportsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/reports",
		}, reportsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusMethodNotAllowed, "")
	})
}

// TestTripsHandler tests trip planning endpoint
// [REQ:REQ-P1-003] Trips endpoint saves and lists multi-region trip plans.
func TestTripsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GET_Success", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/trips",
		}, tripsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("POST_Success", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body: map[string]interface{}{
				"name":       "Test Trip",
				"start_date": "2025-10-01",
				"end_date":   "2025-10-10",
				"regions":    []int{1, 2, 3},
				"notes":      "Beautiful autumn colors",
			},
		}, tripsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)

		// Cleanup
		var tripID int
		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)
		if data, ok := response.Data.(map[string]interface{}); ok {
			if id, ok := data["id"].(float64); ok {
				tripID = int(id)
				defer testDB.DB.Exec("DELETE FROM trip_plans WHERE id = $1", tripID)
			}
		}
	})

	t.Run("POST_MissingFields", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body:   map[string]interface{}{"name": "Incomplete Trip"},
		}, tripsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/trips",
		}, tripsHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusMethodNotAllowed, "")
	})
}

// TestEnableCORS tests CORS middleware
// [REQ:REQ-P0-007] API lifecycle surface exposes browser-compatible CORS behavior.
func TestEnableCORS(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handler := enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("CORS_Headers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:22116")
		rr := httptest.NewRecorder()
		handler(rr, req)

		if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:22116" {
			t.Error("CORS origin header not set correctly")
		}

		if rr.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("CORS methods header not set")
		}
	})

	t.Run("OPTIONS_Request", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:22116")
		rr := httptest.NewRecorder()
		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200 for OPTIONS, got %d", rr.Code)
		}
	})
}

// TestGetEnv tests environment variable retrieval
func TestGetEnv(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		value := getEnv("TEST_VAR", "default")
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}
	})

	t.Run("DefaultValue", func(t *testing.T) {
		value := getEnv("NONEXISTENT_VAR", "default")
		if value != "default" {
			t.Errorf("Expected 'default', got '%s'", value)
		}
	})
}

// TestGetIntValue tests integer pointer helper
func TestGetIntValue(t *testing.T) {
	t.Run("NilPointer", func(t *testing.T) {
		var val *int
		result := getIntValue(val)
		if result != 0 {
			t.Errorf("Expected 0 for nil pointer, got %d", result)
		}
	})

	t.Run("ValidPointer", func(t *testing.T) {
		val := 42
		result := getIntValue(&val)
		if result != 42 {
			t.Errorf("Expected 42, got %d", result)
		}
	})
}

// TestInitDB tests database initialization
// [REQ:REQ-P0-002] Database initialization uses lifecycle-provided resource configuration.
func TestInitDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// This will skip if DB not available
		if _, ok := os.LookupEnv("POSTGRES_HOST"); !ok {
			t.Skip("Skipping DB test: POSTGRES_HOST not set")
		}

		err := initDB()
		if err != nil {
			t.Skipf("Database not available: %v", err)
		}

		if db == nil {
			t.Fatal("Expected db to be initialized")
		}

		if err := db.Ping(); err != nil {
			t.Errorf("Database ping failed: %v", err)
		}
	})
}

// [REQ:REQ-P0-005][REQ:REQ-P2-001] Ollama prediction parsing clamps and validates model output.
func TestGenerateFoliagePrediction(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithMockOllama", func(t *testing.T) {
		restoreGateway := mockResourceOllamaGateway(t, `{"response":"{\"predicted_date\":\"2025-10-15\",\"confidence\":0.85}"}`, 0)
		defer restoreGateway()

		elevation := 500
		typicalWeek := 41
		region := Region{
			ID:              1,
			Name:            "Test Region",
			State:           "VT",
			Latitude:        44.5,
			Longitude:       -72.7,
			ElevationMeters: &elevation,
			TypicalPeakWeek: &typicalWeek,
		}

		date, confidence, err := generateFoliagePrediction(region)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}

		if date == "" {
			t.Error("Expected non-empty date")
		}

		if confidence < 0 || confidence > 1 {
			t.Errorf("Expected confidence between 0 and 1, got %f", confidence)
		}

		// Validate date format
		_, parseErr := time.Parse("2006-01-02", date)
		if parseErr != nil {
			t.Errorf("Invalid date format: %v", parseErr)
		}
	})

	t.Run("OllamaUnavailable", func(t *testing.T) {
		restoreGateway := mockResourceOllamaGateway(t, `{"error":"unavailable"}`, 1)
		defer restoreGateway()

		elevation := 500
		typicalWeek := 41
		region := Region{
			ID:              1,
			Name:            "Test Region",
			State:           "VT",
			Latitude:        44.5,
			Longitude:       -72.7,
			ElevationMeters: &elevation,
			TypicalPeakWeek: &typicalWeek,
		}

		_, _, err := generateFoliagePrediction(region)
		if err == nil {
			t.Error("Expected error when Ollama unavailable")
		}
	})
}
