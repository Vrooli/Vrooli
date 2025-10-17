// +build testing

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

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// setupTestLogger initializes controlled logging during tests
func setupTestLogger() func() {
	// Silence logs during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(io.Discard)
		gin.SetMode(gin.ReleaseMode)
	}

	return func() {
		log.SetOutput(os.Stderr)
		gin.SetMode(gin.DebugMode)
	}
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use test database URL from environment
	postgresURL := os.Getenv("POSTGRES_TEST_URL")
	if postgresURL == "" {
		postgresURL = os.Getenv("POSTGRES_URL")
		if postgresURL == "" {
			t.Skip("No test database available - set POSTGRES_TEST_URL or POSTGRES_URL")
		}
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

// setupTestAPI creates a test API instance with test database
func setupTestAPI(t *testing.T) (*API, func()) {
	db, cleanup := setupTestDB(t)
	api := &API{db: db}
	return api, cleanup
}

// setupTestRouter creates a test router with all routes configured
func setupTestRouter(t *testing.T) (*gin.Engine, *API, func()) {
	gin.SetMode(gin.TestMode)

	api, cleanup := setupTestAPI(t)
	router := gin.New()

	// Health check endpoint
	router.GET("/health", api.healthCheck)

	// API routes
	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/search", api.searchCharacters)
		apiGroup.GET("/character/:codepoint", api.getCharacterDetail)
		apiGroup.GET("/categories", api.getCategories)
		apiGroup.GET("/blocks", api.getBlocks)
		apiGroup.POST("/bulk/range", api.getBulkRange)
	}

	return router, api, cleanup
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
}

// makeHTTPRequest executes an HTTP test request and returns the response
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyJSON, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewBuffer(bodyJSON)
	}

	httpReq, _ := http.NewRequest(req.Method, req.Path, bodyReader)
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

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	return w
}

// assertJSONResponse validates a JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields ...string) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field '%s' not found in response: %v", field, response)
		}
	}

	return response
}

// assertSearchResponse validates a search response
func assertSearchResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) SearchResponse {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response SearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse search response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertCharacterResponse validates a character detail response
func assertCharacterResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) CharacterDetailResponse {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response CharacterDetailResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse character response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v. Body: %s", err, w.Body.String())
	}

	if _, exists := response["error"]; !exists {
		t.Errorf("Expected 'error' field in error response, got: %v", response)
	}
}

// seedTestCharacter adds a test character to the database for testing
func seedTestCharacter(t *testing.T, db *sql.DB, codepoint string, decimal int, name string) {
	query := `
		INSERT INTO characters
		(codepoint, decimal, name, category, block, unicode_version, description, html_entity, css_content, properties)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (codepoint) DO NOTHING
	`

	description := fmt.Sprintf("Test character: %s", name)
	htmlEntity := fmt.Sprintf("&#%d;", decimal)
	cssContent := fmt.Sprintf("\\%X", decimal)
	properties := []byte(`{}`)

	_, err := db.Exec(query, codepoint, decimal, name, "So", "Test Block", "1.0",
		description, htmlEntity, cssContent, properties)
	if err != nil {
		t.Fatalf("Failed to seed test character: %v", err)
	}
}

// cleanupTestCharacter removes a test character from the database
func cleanupTestCharacter(t *testing.T, db *sql.DB, codepoint string) {
	_, err := db.Exec("DELETE FROM characters WHERE codepoint = $1", codepoint)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test character %s: %v", codepoint, err)
	}
}

// countCharactersInDB returns the total number of characters in the database
func countCharactersInDB(t *testing.T, db *sql.DB) int {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM characters").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count characters: %v", err)
	}
	return count
}

// verifyTableExists checks if a table exists in the database
func verifyTableExists(t *testing.T, db *sql.DB, tableName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = $1
		)
	`
	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to verify table existence: %v", err)
	}
	return exists
}
