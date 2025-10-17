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
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes logging for tests
func setupTestLogger() func() {
	gin.SetMode(gin.TestMode)
	originalOutput := log.Writer()

	// Suppress logs during tests unless VERBOSE_TESTS is set
	if os.Getenv("VERBOSE_TESTS") != "true" {
		log.SetOutput(io.Discard)
	}

	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestDatabase manages test database connection
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Check if we should skip database tests
	if os.Getenv("SKIP_DB_TESTS") == "true" || os.Getenv("CI") == "true" {
		t.Skip("Skipping database test - database not available")
	}

	// Use test database URL or default to localhost
	dbURL := os.Getenv("TEST_POSTGRES_URL")
	if dbURL == "" {
		// Try to detect if postgres is running
		testDB, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/vrooli_test?sslmode=disable&search_path=react_component_library&connect_timeout=2")
		if err != nil || testDB.Ping() != nil {
			if testDB != nil {
				testDB.Close()
			}
			t.Skip("Skipping database test - PostgreSQL not available")
		}
		testDB.Close()
		dbURL = "postgres://postgres:postgres@localhost:5432/vrooli_test?sslmode=disable&search_path=react_component_library"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Failed to open test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Failed to ping test database: %v", err)
	}

	// Clean up test data before tests
	cleanupTestData(t, db)

	return &TestDatabase{
		DB: db,
		Cleanup: func() {
			cleanupTestData(t, db)
			db.Close()
		},
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, db *sql.DB) {
	tables := []string{
		"component_test_results",
		"component_versions",
		"component_usage",
		"components",
	}

	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM %s WHERE created_at > NOW() - INTERVAL '1 hour' OR author LIKE 'test-%%'", table)
		if _, err := db.Exec(query); err != nil {
			// Only warn, don't fail - table might not exist
			t.Logf("Warning: failed to clean %s: %v", table, err)
		}
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if req.Body != nil {
		bodyBytes, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, _ := http.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Default content type for JSON
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
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

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validationFunc func(map[string]interface{}) bool) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
		return
	}

	if validationFunc != nil && !validationFunc(response) {
		t.Errorf("Response validation failed. Response: %v", response)
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMsg string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse error response: %v", err)
		return
	}

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Errorf("Response missing 'error' field. Response: %v", response)
		return
	}

	if expectedErrorMsg != "" && errorMsg != expectedErrorMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedErrorMsg, errorMsg)
	}
}

// createTestComponent creates a test component in the database
func createTestComponent(t *testing.T, db *sql.DB, name string) uuid.UUID {
	id := uuid.New()
	query := `
		INSERT INTO components (
			id, name, category, description, code, props_schema,
			created_at, updated_at, version, author, usage_count,
			tags, is_active, dependencies, example_usage
		) VALUES (
			$1, $2, $3, $4, $5, $6, NOW(), NOW(), $7, $8, 0, $9::text[], true, $10::text[], $11
		)`

	code := fmt.Sprintf("const %s = () => <div>Test Component</div>;", name)
	_, err := db.Exec(
		query,
		id,
		name,
		"test",
		"Test component for testing purposes",
		code,
		`{"props": []}`,
		"1.0.0",
		"test-user",
		`{"test","component"}`,
		`{}`,
		fmt.Sprintf("<%s />", name),
	)

	if err != nil {
		t.Fatalf("Failed to create test component: %v", err)
	}

	return id
}

// TestComponentData provides sample component data for testing
type TestComponentData struct {
	Name         string
	Category     string
	Description  string
	Code         string
	PropsSchema  string
	Tags         []string
	Dependencies []string
}

// getValidComponentData returns valid component data for testing
func getValidComponentData() TestComponentData {
	return TestComponentData{
		Name:        "TestButton",
		Category:    "form",
		Description: "A test button component",
		Code:        "const TestButton = ({ label }) => <button>{label}</button>;",
		PropsSchema: `{"type": "object", "properties": {"label": {"type": "string"}}}`,
		Tags:        []string{"button", "form", "test"},
		Dependencies: []string{"react"},
	}
}

// getInvalidComponentData returns various invalid component data scenarios
func getInvalidComponentData() []struct {
	Name        string
	Data        TestComponentData
	Description string
} {
	return []struct {
		Name        string
		Data        TestComponentData
		Description string
	}{
		{
			Name: "EmptyName",
			Data: TestComponentData{
				Name:        "",
				Category:    "form",
				Description: "Test",
				Code:        "const Test = () => <div />;",
			},
			Description: "Component with empty name",
		},
		{
			Name: "EmptyCode",
			Data: TestComponentData{
				Name:        "Test",
				Category:    "form",
				Description: "Test",
				Code:        "",
			},
			Description: "Component with empty code",
		},
		{
			Name: "InvalidCode",
			Data: TestComponentData{
				Name:        "Test",
				Category:    "form",
				Description: "Test",
				Code:        "this is not valid javascript",
			},
			Description: "Component with invalid JavaScript code",
		},
	}
}
