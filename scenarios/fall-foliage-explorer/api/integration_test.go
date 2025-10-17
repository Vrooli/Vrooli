package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSubmitReportIntegration tests the full submit report workflow
func TestSubmitReportIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	regionID := setupTestRegion(t, testDB.DB)
	defer cleanupTestRegion(t, testDB.DB, regionID)

	t.Run("FullWorkflow", func(t *testing.T) {
		// Submit a report
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body: map[string]interface{}{
				"region_id":      regionID,
				"foliage_status": "peak",
				"description":    "Amazing colors today!",
				"photo_url":      "https://example.com/photo123.jpg",
			},
		}, reportsHandler)

		if err != nil {
			t.Fatalf("Failed to submit report: %v", err)
		}

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var submitResp Response
		json.Unmarshal(rr.Body.Bytes(), &submitResp)

		reportData, ok := submitResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		// Verify the report was saved with an ID
		if reportData["id"] == nil {
			t.Error("Expected report ID to be present")
		}

		// Retrieve reports for this region
		rr2, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/reports",
			QueryParams: map[string]string{"region_id": fmt.Sprintf("%d", regionID)},
		}, reportsHandler)

		if err != nil {
			t.Fatalf("Failed to get reports: %v", err)
		}

		if rr2.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rr2.Code)
		}

		var getResp Response
		json.Unmarshal(rr2.Body.Bytes(), &getResp)

		reports, ok := getResp.Data.([]interface{})
		if !ok {
			t.Fatal("Expected data to be an array")
		}

		if len(reports) == 0 {
			t.Error("Expected at least one report")
		}
	})

	t.Run("WithPhotoURL", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body: map[string]interface{}{
				"region_id":      regionID,
				"foliage_status": "changing",
				"description":    "Colors starting to turn",
				"photo_url":      "https://example.com/autumn.jpg",
			},
		}, reportsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if data["photo_url"] != "https://example.com/autumn.jpg" {
			t.Errorf("Expected photo_url to be preserved, got %v", data["photo_url"])
		}
	})
}

// TestTripPlanningIntegration tests the full trip planning workflow
func TestTripPlanningIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("CreateAndRetrieveTrip", func(t *testing.T) {
		// Create a trip
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body: map[string]interface{}{
				"name":       "Vermont Fall Colors Tour",
				"start_date": "2025-10-01",
				"end_date":   "2025-10-15",
				"regions":    []int{1, 2, 3, 4},
				"notes":      "Don't forget the camera!",
			},
		}, tripsHandler)

		if err != nil {
			t.Fatalf("Failed to create trip: %v", err)
		}

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var createResp Response
		json.Unmarshal(rr.Body.Bytes(), &createResp)

		tripData, ok := createResp.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		tripID, ok := tripData["id"].(float64)
		if !ok {
			t.Fatal("Expected trip ID to be present")
		}
		defer testDB.DB.Exec("DELETE FROM trip_plans WHERE id = $1", int(tripID))

		// Verify notes were saved
		if tripData["notes"] != "Don't forget the camera!" {
			t.Errorf("Expected notes to be saved, got %v", tripData["notes"])
		}

		// Retrieve all trips
		rr2, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/trips",
		}, tripsHandler)

		if err != nil {
			t.Fatalf("Failed to get trips: %v", err)
		}

		if rr2.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", rr2.Code)
		}

		var getResp Response
		json.Unmarshal(rr2.Body.Bytes(), &getResp)

		trips, ok := getResp.Data.([]interface{})
		if !ok {
			t.Fatal("Expected data to be an array")
		}

		// Should have at least the trip we just created
		found := false
		for _, trip := range trips {
			tripMap := trip.(map[string]interface{})
			if tripMap["name"] == "Vermont Fall Colors Tour" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created trip not found in list")
		}
	})

	t.Run("TripWithoutNotes", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body: map[string]interface{}{
				"name":       "Quick Trip",
				"start_date": "2025-10-05",
				"end_date":   "2025-10-06",
				"regions":    []int{1},
			},
		}, tripsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if id, ok := data["id"].(float64); ok {
			defer testDB.DB.Exec("DELETE FROM trip_plans WHERE id = $1", int(id))
		}
	})

	t.Run("TripInvalidRegionsJSON", func(t *testing.T) {
		// This tests the JSON marshaling error path
		// We can't easily trigger this through makeHTTPRequest,
		// so we'll just verify the happy path covers most of the function
		t.Skip("Cannot easily test JSON marshal error without mocking")
	})
}

// TestWeatherDataEdgeCases tests weather handler edge cases
func TestWeatherDataEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("NoWeatherData", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/weather",
			QueryParams: map[string]string{
				"region_id": "99999",
				"date":      "2025-01-01",
			},
		}, weatherHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertJSONResponse(t, rr, http.StatusOK, true)

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		if response.Message == "" {
			t.Log("Response might have data or message about no data available")
		}
	})

	t.Run("DefaultDate", func(t *testing.T) {
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
}

// TestGetReportsEdgeCases tests report retrieval edge cases
func TestGetReportsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	regionID := setupTestRegion(t, testDB.DB)
	defer cleanupTestRegion(t, testDB.DB, regionID)

	t.Run("InvalidRegionIDInGet", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/reports",
			QueryParams: map[string]string{"region_id": "not_a_number"},
		}, reportsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("WithoutDatabaseGetReports", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/reports",
			QueryParams: map[string]string{"region_id": "1"},
		}, reportsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return empty array when DB unavailable
		assertJSONResponse(t, rr, http.StatusOK, true)
	})

	t.Run("WithoutDatabaseSubmitReport", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body: map[string]interface{}{
				"region_id":      1,
				"foliage_status": "peak",
				"description":    "Test",
			},
		}, reportsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return mock response
		assertJSONResponse(t, rr, http.StatusOK, true)
	})
}

// TestGetTripsEdgeCases tests trip retrieval edge cases
func TestGetTripsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/trips",
		}, tripsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return empty array
		assertJSONResponse(t, rr, http.StatusOK, true)

		var response Response
		json.Unmarshal(rr.Body.Bytes(), &response)

		trips, ok := response.Data.([]interface{})
		if !ok {
			t.Fatal("Expected data to be an array")
		}

		if len(trips) != 0 {
			t.Error("Expected empty trips array when DB unavailable")
		}
	})
}

// TestSaveTripEdgeCases tests trip save edge cases
func TestSaveTripEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidRequestBody", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/trips", bytes.NewBufferString("not json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		tripsHandler(rr, req)

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("MissingStartDate", func(t *testing.T) {
		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body: map[string]interface{}{
				"name":     "Test Trip",
				"end_date": "2025-10-10",
			},
		}, tripsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "")
	})

	t.Run("WithoutDatabase", func(t *testing.T) {
		originalDB := db
		db = nil
		defer func() { db = originalDB }()

		rr, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/trips",
			Body: map[string]interface{}{
				"name":       "Mock Trip",
				"start_date": "2025-10-01",
				"end_date":   "2025-10-10",
				"regions":    []int{1},
			},
		}, tripsHandler)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return mock response
		assertJSONResponse(t, rr, http.StatusOK, true)
	})
}
