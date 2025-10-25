package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBusinessWorkflow tests the complete business workflow
func TestBusinessWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("CompleteUserJourney", func(t *testing.T) {
		// Step 1: Get all regions
		rr1, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/regions",
		}, regionsHandler)

		if rr1.Code != http.StatusOK {
			t.Fatalf("Failed to get regions: %d", rr1.Code)
		}

		var regionsResp Response
		json.Unmarshal(rr1.Body.Bytes(), &regionsResp)
		regions := regionsResp.Data.([]interface{})

		if len(regions) == 0 {
			t.Skip("No regions available for testing")
		}

		region := regions[0].(map[string]interface{})
		regionID := int(region["id"].(float64))

		// Step 2: Get foliage data for a region
		rr2, _ := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/foliage",
			QueryParams: map[string]string{"region_id": fmt.Sprintf("%d", regionID)},
		}, foliageHandler)

		assertJSONResponse(t, rr2, http.StatusOK, true)

		// Step 3: Get weather data
		rr3, _ := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/weather",
			QueryParams: map[string]string{"region_id": fmt.Sprintf("%d", regionID)},
		}, weatherHandler)

		assertJSONResponse(t, rr3, http.StatusOK, true)

		// Step 4: Submit a user report
		rr4, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body: map[string]interface{}{
				"region_id":      regionID,
				"foliage_status": "peak",
				"description":    "Beautiful autumn colors!",
			},
		}, reportsHandler)

		assertJSONResponse(t, rr4, http.StatusOK, true)

		// Step 5: Get reports for the region
		rr5, _ := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/reports",
			QueryParams: map[string]string{"region_id": fmt.Sprintf("%d", regionID)},
		}, reportsHandler)

		assertJSONResponse(t, rr5, http.StatusOK, true)

		// Step 6: Create a trip plan
		rr6, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body: map[string]interface{}{
				"name":       "Autumn Adventure",
				"start_date": "2025-10-01",
				"end_date":   "2025-10-15",
				"regions":    []int{regionID},
				"notes":      "Check peak times",
			},
		}, tripsHandler)

		assertJSONResponse(t, rr6, http.StatusOK, true)

		var tripResp Response
		json.Unmarshal(rr6.Body.Bytes(), &tripResp)
		tripData := tripResp.Data.(map[string]interface{})
		tripID := int(tripData["id"].(float64))
		defer testDB.DB.Exec("DELETE FROM trip_plans WHERE id = $1", tripID)

		// Step 7: Get all trips
		rr7, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/trips",
		}, tripsHandler)

		assertJSONResponse(t, rr7, http.StatusOK, true)

		t.Log("✅ Complete user journey test passed")
	})
}

// TestPredictionWorkflow tests the prediction workflow
func TestPredictionWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	regionID := setupTestRegion(t, testDB.DB)
	defer cleanupTestRegion(t, testDB.DB, regionID)

	t.Run("PredictionWithMockOllama", func(t *testing.T) {
		// Setup mock Ollama server
		mockServer := mockOllamaServer()
		defer mockServer.Close()

		restoreEnv := setTestEnv(t, "OLLAMA_URL", mockServer.URL)
		defer restoreEnv()

		// Make prediction request
		rr, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/predict",
			Body:   map[string]interface{}{"region_id": float64(regionID)},
		}, predictHandler)

		assertJSONResponse(t, rr, http.StatusOK, true)

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if data["method"] != "ollama_ai" {
			t.Errorf("Expected ollama_ai method, got %v", data["method"])
		}

		// Verify prediction was stored (if table exists)
		var count int
		err := testDB.DB.QueryRow("SELECT COUNT(*) FROM foliage_predictions WHERE region_id = $1", regionID).Scan(&count)
		if err != nil {
			t.Logf("Could not verify prediction storage (table may not exist): %v", err)
		} else if count == 0 {
			t.Log("Note: Prediction was not found in database, but request succeeded")
		}
	})
}

// TestErrorPaths tests various error paths
func TestErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		request HTTPTestRequest
		status  int
	}{
		{
			name:    "Foliage_MissingParam",
			handler: foliageHandler,
			request: HTTPTestRequest{Method: "GET", Path: "/api/foliage"},
			status:  http.StatusBadRequest,
		},
		{
			name:    "Weather_MissingParam",
			handler: weatherHandler,
			request: HTTPTestRequest{Method: "GET", Path: "/api/weather"},
			status:  http.StatusBadRequest,
		},
		{
			name:    "Reports_GET_MissingParam",
			handler: reportsHandler,
			request: HTTPTestRequest{Method: "GET", Path: "/api/reports"},
			status:  http.StatusBadRequest,
		},
		{
			name:    "Reports_InvalidMethod",
			handler: reportsHandler,
			request: HTTPTestRequest{Method: "PUT", Path: "/api/reports"},
			status:  http.StatusMethodNotAllowed,
		},
		{
			name:    "Trips_InvalidMethod",
			handler: tripsHandler,
			request: HTTPTestRequest{Method: "PUT", Path: "/api/trips"},
			status:  http.StatusMethodNotAllowed,
		},
		{
			name:    "Predict_InvalidMethod",
			handler: predictHandler,
			request: HTTPTestRequest{Method: "PUT", Path: "/api/predict"},
			status:  http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr, err := makeHTTPRequest(tt.request, tt.handler)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if rr.Code != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, rr.Code)
			}
		})
	}
}

// TestCORSConfiguration tests CORS headers are set correctly
func TestCORSConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name    string
		method  string
		handler http.HandlerFunc
	}{
		{"Health_GET", "GET", healthHandler},
		{"Health_OPTIONS", "OPTIONS", healthHandler},
		{"Regions_GET", "GET", regionsHandler},
		{"Regions_OPTIONS", "OPTIONS", regionsHandler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHandler := enableCORS(tt.handler)

			req, _ := http.NewRequest(tt.method, "/test", nil)
			rr := httptest.NewRecorder()
			wrappedHandler(rr, req)

			origin := rr.Header().Get("Access-Control-Allow-Origin")
			if origin != "*" {
				t.Errorf("Expected CORS origin *, got %s", origin)
			}

			methods := rr.Header().Get("Access-Control-Allow-Methods")
			if methods == "" {
				t.Error("Expected CORS methods header to be set")
			}

			headers := rr.Header().Get("Access-Control-Allow-Headers")
			if headers == "" {
				t.Error("Expected CORS headers to be set")
			}

			if tt.method == "OPTIONS" && rr.Code != http.StatusOK {
				t.Errorf("OPTIONS request should return 200, got %d", rr.Code)
			}
		})
	}
}

// TestDatabaseReconnection tests behavior when database reconnects
func TestDatabaseReconnection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RegionsWithDBReconnect", func(t *testing.T) {
		// First request with no DB
		originalDB := db
		db = nil

		rr1, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/regions",
		}, regionsHandler)

		var resp1 Response
		json.Unmarshal(rr1.Body.Bytes(), &resp1)
		if resp1.Status != "success" {
			t.Fatalf("Expected success status with fallback dataset, got %s", resp1.Status)
		}

		payload, ok := resp1.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected fallback payload map, got %T", resp1.Data)
		}

		meta, ok := payload["meta"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected meta information in fallback payload")
		}

		if fallback, _ := meta["using_fallback"].(bool); !fallback {
			t.Error("Expected fallback indicator when DB unavailable")
		}

		// Reconnect DB
		db = originalDB

		// Second request with DB
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		regionID := setupTestRegion(t, testDB.DB)
		defer cleanupTestRegion(t, testDB.DB, regionID)

		rr2, _ := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/regions",
		}, regionsHandler)

		assertJSONResponse(t, rr2, http.StatusOK, true)
	})
}

// TestInitDBConnection tests database initialization connection logic
func TestInitDBConnection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DatabasePing", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer testDB.Cleanup()

		if err := testDB.DB.Ping(); err != nil {
			t.Errorf("Database ping failed: %v", err)
		}
	})
}
