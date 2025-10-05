package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Silence logs during tests
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestDatabase manages test database connection
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *TestDatabase {
	// Save original db
	originalDB := db

	// Connect to test database
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5433")
	dbUser := getEnv("POSTGRES_USER", "vrooli")
	dbPass := getEnv("POSTGRES_PASSWORD", "vrooli")
	dbName := getEnv("POSTGRES_DB", "vrooli")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping database tests: %v", err)
	}

	if err = testDB.Ping(); err != nil {
		t.Skipf("Skipping database tests: cannot connect: %v", err)
	}

	// Set global db for handlers
	db = testDB

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			testDB.Close()
			db = originalDB
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	rr := httptest.NewRecorder()
	handler(rr, httpReq)

	return rr, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedData interface{}) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
		return
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	if expectedData != nil {
		var response Response
		// Use Unmarshal to preserve the body buffer
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to decode response: %v. Body: %s", err, rr.Body.String())
			return
		}

		// Validate response structure
		if response.Status == "" {
			t.Error("Response missing 'status' field")
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, errorContains string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
		return
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to decode error response: %v. Body: %s", err, rr.Body.String())
		return
	}

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", response.Status)
	}

	if errorContains != "" && response.Error == "" {
		t.Error("Expected error message but got none")
	}
}

// setupTestRegion creates a test region in the database
func setupTestRegion(t *testing.T, testDB *sql.DB) int {
	t.Helper()

	var regionID int
	err := testDB.QueryRow(`
		INSERT INTO regions (name, state, country, latitude, longitude, elevation_meters, typical_peak_week)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, "Test Region", "VT", "USA", 44.5, -72.7, 500, 41).Scan(&regionID)

	if err != nil {
		t.Fatalf("Failed to create test region: %v", err)
	}

	return regionID
}

// cleanupTestRegion removes a test region from the database
func cleanupTestRegion(t *testing.T, testDB *sql.DB, regionID int) {
	t.Helper()

	// Clean up related data first
	testDB.Exec("DELETE FROM foliage_observations WHERE region_id = $1", regionID)
	testDB.Exec("DELETE FROM foliage_predictions WHERE region_id = $1", regionID)
	testDB.Exec("DELETE FROM user_reports WHERE region_id = $1", regionID)
	testDB.Exec("DELETE FROM weather_data WHERE region_id = $1", regionID)
	testDB.Exec("DELETE FROM regions WHERE id = $1", regionID)
}

// setupTestFoliageData creates test foliage observation data
func setupTestFoliageData(t *testing.T, testDB *sql.DB, regionID int) {
	t.Helper()

	_, err := testDB.Exec(`
		INSERT INTO foliage_observations (region_id, observation_date, foliage_percentage, color_intensity, peak_status)
		VALUES ($1, $2, $3, $4, $5)
	`, regionID, "2025-10-01", 75, 8, "peak")

	if err != nil {
		t.Fatalf("Failed to create test foliage data: %v", err)
	}
}

// setupTestUserReport creates a test user report
func setupTestUserReport(t *testing.T, testDB *sql.DB, regionID int) int {
	t.Helper()

	var reportID int
	err := testDB.QueryRow(`
		INSERT INTO user_reports (region_id, report_date, foliage_status, description, photo_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, regionID, "2025-10-01", "peak", "Beautiful colors!", "https://example.com/photo.jpg").Scan(&reportID)

	if err != nil {
		t.Fatalf("Failed to create test user report: %v", err)
	}

	return reportID
}

// mockOllamaServer creates a mock Ollama server for testing
func mockOllamaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"response": `{"predicted_date": "2025-10-15", "confidence": 0.85}`,
		}
		json.NewEncoder(w).Encode(response)
	}))
}

// setTestEnv temporarily sets environment variables for tests
func setTestEnv(t *testing.T, key, value string) func() {
	t.Helper()
	original := os.Getenv(key)
	os.Setenv(key, value)
	return func() {
		if original == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, original)
		}
	}
}
