package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes a test logger with suppressed output
func setupTestLogger() func() {
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	// Suppress logging during tests unless VERBOSE_TESTS is set
	if os.Getenv("VERBOSE_TESTS") != "true" {
		log.SetOutput(io.Discard)
	}

	return func() {
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
		log.SetOutput(os.Stderr)
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database credentials from environment or defaults
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		dbHost := getEnvOrDefault("POSTGRES_HOST", "localhost")
		dbPort := getEnvOrDefault("POSTGRES_PORT", "5433")
		dbUser := getEnvOrDefault("POSTGRES_USER", "vrooli")
		dbPassword := getEnvOrDefault("POSTGRES_PASSWORD", "vrooli")
		dbName := getEnvOrDefault("POSTGRES_DB", "vrooli")

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return &TestDatabase{
		DB: db,
		Cleanup: func() {
			db.Close()
		},
	}
}

// TestApp wraps the App with test utilities
type TestApp struct {
	App     *App
	DB      *sql.DB
	Cleanup func()
}

// setupTestApp creates a fully configured test app
func setupTestApp(t *testing.T) *TestApp {
	testDB := setupTestDatabase(t)

	app := &App{
		DB:           testDB.DB,
		SmartPairing: NewSmartPairing(testDB.DB),
	}

	return &TestApp{
		App:     app,
		DB:      testDB.DB,
		Cleanup: testDB.Cleanup,
	}
}

// TestList provides a pre-configured list for testing
type TestList struct {
	List    *List
	Items   []*Item
	Cleanup func()
}

// setupTestList creates a test list with sample items
func setupTestList(t *testing.T, db *sql.DB, name string, itemCount int) *TestList {
	listID := uuid.New().String()

	// Create list
	_, err := db.Exec(`
		INSERT INTO elo_swipe.lists (id, name, description)
		VALUES ($1, $2, $3)
	`, listID, name, fmt.Sprintf("Test list: %s", name))
	if err != nil {
		t.Fatalf("Failed to create test list: %v", err)
	}

	// Create items
	items := make([]*Item, itemCount)
	for i := 0; i < itemCount; i++ {
		itemID := uuid.New().String()
		content := json.RawMessage(fmt.Sprintf(`{"name": "Item %d", "value": %d}`, i+1, i+1))

		_, err := db.Exec(`
			INSERT INTO elo_swipe.items (id, list_id, content)
			VALUES ($1, $2, $3)
		`, itemID, listID, content)
		if err != nil {
			t.Fatalf("Failed to create test item: %v", err)
		}

		items[i] = &Item{
			ID:              itemID,
			ListID:          listID,
			Content:         content,
			EloRating:       1500.0,
			ComparisonCount: 0,
			Wins:            0,
			Losses:          0,
			Confidence:      0.0,
		}
	}

	list := &List{
		ID:          listID,
		Name:        name,
		Description: fmt.Sprintf("Test list: %s", name),
		ItemCount:   itemCount,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return &TestList{
		List:  list,
		Items: items,
		Cleanup: func() {
			// Clean up comparisons first
			db.Exec(`DELETE FROM elo_swipe.comparisons WHERE list_id = $1`, listID)
			// Clean up pairing queue
			db.Exec(`DELETE FROM elo_swipe.pairing_queue WHERE list_id = $1`, listID)
			// Clean up items
			db.Exec(`DELETE FROM elo_swipe.items WHERE list_id = $1`, listID)
			// Clean up list
			db.Exec(`DELETE FROM elo_swipe.lists WHERE id = $1`, listID)
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader io.Reader

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
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
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

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
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

	return response
}

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	if arrayField == "" {
		// If no field specified, response itself should be an array
		var array []interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &array); err != nil {
			t.Errorf("Expected response to be an array, got error: %v", err)
			return nil
		}
		return array
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorFragment string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// Error responses might be plain text or JSON
	bodyStr := w.Body.String()

	if expectedErrorFragment != "" {
		if bodyStr == "" || len(bodyStr) == 0 {
			t.Errorf("Expected error message to contain '%s', but got empty response", expectedErrorFragment)
			return
		}
		// Just check if the error fragment is anywhere in the response
		// This handles both JSON {"error": "..."} and plain text responses
		if !contains(bodyStr, expectedErrorFragment) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorFragment, bodyStr)
		}
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		bytes.Contains([]byte(str), []byte(substr)))
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateListRequest creates a test CreateListRequest
func (g *TestDataGenerator) CreateListRequest(name string, itemCount int) CreateListRequest {
	items := make([]ItemInput, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = ItemInput{
			Content: json.RawMessage(fmt.Sprintf(`{"name": "Item %d", "value": %d}`, i+1, i+1)),
		}
	}

	return CreateListRequest{
		Name:        name,
		Description: fmt.Sprintf("Test list: %s", name),
		Items:       items,
	}
}

// CreateComparisonRequest creates a test CreateComparisonRequest
func (g *TestDataGenerator) CreateComparisonRequest(listID, winnerID, loserID string) CreateComparisonRequest {
	return CreateComparisonRequest{
		ListID:   listID,
		WinnerID: winnerID,
		LoserID:  loserID,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// getEnvOrDefault gets environment variable or returns default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createTestComparison creates a comparison between two items for testing
func createTestComparison(t *testing.T, db *sql.DB, listID, winnerID, loserID string) string {
	comparisonID := uuid.New().String()

	_, err := db.Exec(`
		INSERT INTO elo_swipe.comparisons
		(id, list_id, winner_id, loser_id, winner_rating_before, loser_rating_before,
		 winner_rating_after, loser_rating_after, k_factor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, comparisonID, listID, winnerID, loserID,
		1500.0, 1500.0, 1516.0, 1484.0, 32.0)

	if err != nil {
		t.Fatalf("Failed to create test comparison: %v", err)
	}

	return comparisonID
}

// withTimeout runs a test function with a timeout
func withTimeout(t *testing.T, timeout time.Duration, fn func()) {
	done := make(chan bool)
	go func() {
		fn()
		done <- true
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(timeout):
		t.Fatal("Test timed out")
	}
}

// withContext creates a context with timeout for testing
func withContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
