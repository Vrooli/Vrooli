package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(os.Stdout)
	log.SetPrefix("[test] ")
	return func() {
		log.SetOutput(originalOutput)
		log.SetPrefix("")
	}
}

// TestDB manages test database connection
type TestDB struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates an isolated test database environment
func setupTestDB(t *testing.T) *TestDB {
	// Store original db connection
	originalDB := db

	// Create test database connection
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	if dbName == "" {
		dbName = "travel_map_test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open test database connection: %v", err)
	}

	// Try to ping - if fails, skip test requiring DB
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test requiring database: %v", err)
	}

	// Set global db to test db
	db = testDB

	// Clean up test data
	cleanupTestData(t, testDB)

	return &TestDB{
		DB: testDB,
		Cleanup: func() {
			cleanupTestData(t, testDB)
			testDB.Close()
			db = originalDB
		},
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, testDB *sql.DB) {
	tables := []string{"travels", "travel_stats", "achievements", "bucket_list"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE user_id LIKE 'test_%%'", table)
		if _, err := testDB.Exec(query); err != nil {
			// Table might not exist, that's ok for tests
			t.Logf("Warning: Failed to clean table %s: %v", table, err)
		}
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	return httptest.NewRecorder()
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertJSONArray validates that response contains an array
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	bodyStr := w.Body.String()
	if expectedErrorMessage != "" && !strings.Contains(bodyStr, expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, bodyStr)
	}
}

// createTestTravel creates a test travel entry in the database
func createTestTravel(t *testing.T, testDB *sql.DB, userID string) *Travel {
	travel := &Travel{
		ID:           time.Now().UnixNano(),
		UserID:       userID,
		Location:     "Paris, France",
		Lat:          48.8566,
		Lng:          2.3522,
		Date:         "2024-01-15",
		Type:         "vacation",
		Notes:        "Visited the Eiffel Tower",
		Country:      "France",
		City:         "Paris",
		Continent:    "Europe",
		DurationDays: 5,
		Rating:       5,
		Photos:       []string{"photo1.jpg", "photo2.jpg"},
		Tags:         []string{"europe", "culture"},
		CreatedAt:    time.Now(),
	}

	photosJSON, _ := json.Marshal(travel.Photos)
	tagsJSON, _ := json.Marshal(travel.Tags)

	query := `INSERT INTO travels (id, user_id, location, lat, lng, date, type, notes,
			  country, city, continent, duration_days, rating, photos, tags, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := testDB.Exec(query, travel.ID, travel.UserID, travel.Location, travel.Lat, travel.Lng,
		travel.Date, travel.Type, travel.Notes, travel.Country, travel.City, travel.Continent,
		travel.DurationDays, travel.Rating, photosJSON, tagsJSON, travel.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test travel: %v", err)
	}

	return travel
}

// createTestBucketItem creates a test bucket list item in the database
func createTestBucketItem(t *testing.T, testDB *sql.DB, userID string) *BucketListItem {
	item := &BucketListItem{
		ID:             int(time.Now().UnixNano() % 1000000),
		Location:       "Tokyo, Japan",
		Country:        "Japan",
		City:           "Tokyo",
		Priority:       1,
		Notes:          "Experience cherry blossom season",
		EstimatedDate:  "2025-04-01",
		BudgetEstimate: 3000.00,
		Tags:           []string{"asia", "culture"},
		Completed:      false,
	}

	tagsJSON, _ := json.Marshal(item.Tags)

	query := `INSERT INTO bucket_list (id, user_id, location, country, city, priority, notes,
			  estimated_date, budget_estimate, tags, completed, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())`

	_, err := testDB.Exec(query, item.ID, userID, item.Location, item.Country, item.City,
		item.Priority, item.Notes, item.EstimatedDate, item.BudgetEstimate, tagsJSON, item.Completed)

	if err != nil {
		t.Fatalf("Failed to create test bucket item: %v", err)
	}

	return item
}

// createTestAchievement creates a test achievement in the database
func createTestAchievement(t *testing.T, testDB *sql.DB, userID string, achievementType string) {
	query := `INSERT INTO achievements (user_id, achievement_type, achievement_name, description, icon, unlocked_at)
			  VALUES ($1, $2, $3, $4, $5, NOW())
			  ON CONFLICT (user_id, achievement_type) DO NOTHING`

	_, err := testDB.Exec(query, userID, achievementType, "Test Achievement", "Test description", "🏆")
	if err != nil {
		t.Fatalf("Failed to create test achievement: %v", err)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// TravelRequest creates a test travel request
func (g *TestDataGenerator) TravelRequest(userID, location string) map[string]interface{} {
	return map[string]interface{}{
		"user_id":       userID,
		"location":      location,
		"lat":           40.7128,
		"lng":           -74.0060,
		"date":          "2024-01-01",
		"type":          "vacation",
		"notes":         "Test travel",
		"country":       "USA",
		"city":          "New York",
		"continent":     "North America",
		"duration_days": 3,
		"rating":        4,
		"photos":        []string{},
		"tags":          []string{"test"},
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
