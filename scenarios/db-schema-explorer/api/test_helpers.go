// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
}

// setupTestLogger initializes a test logger that suppresses verbose output
func setupTestLogger() func() {
	// Redirect log output to discard during tests
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestDatabase manages test database connection and cleanup
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates an isolated test database environment
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database environment variables or defaults
	connStr := os.Getenv("TEST_POSTGRES_URL")
	if connStr == "" {
		// Build from individual components with test defaults
		host := getEnv("TEST_POSTGRES_HOST", "localhost")
		port := getEnv("TEST_POSTGRES_PORT", "5432")
		user := getEnv("TEST_POSTGRES_USER", "postgres")
		password := getEnv("TEST_POSTGRES_PASSWORD", "postgres")
		dbName := getEnv("TEST_POSTGRES_DB", "test_db_explorer")

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbName)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
		return nil
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test - database not reachable: %v", err)
		return nil
	}

	// Setup test schema
	setupTestSchema(t, db)

	return &TestDatabase{
		DB: db,
		Cleanup: func() {
			cleanupTestSchema(db)
			db.Close()
		},
	}
}

// setupTestSchema creates necessary test tables
func setupTestSchema(t *testing.T, db *sql.DB) {
	schema := `
	CREATE SCHEMA IF NOT EXISTS db_explorer;

	CREATE TABLE IF NOT EXISTS db_explorer.schema_snapshots (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		database_name VARCHAR(255) NOT NULL,
		schema_name VARCHAR(255) NOT NULL,
		tables_count INT DEFAULT 0,
		columns_count INT DEFAULT 0,
		relationships_count INT DEFAULT 0,
		schema_data JSONB,
		snapshot_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS db_explorer.query_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		natural_language TEXT NOT NULL,
		generated_sql TEXT NOT NULL,
		database_name VARCHAR(255) NOT NULL,
		query_type VARCHAR(50) DEFAULT 'SELECT',
		execution_time_ms BIGINT,
		result_count BIGINT,
		user_feedback TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS db_explorer.visualization_layouts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		database_name VARCHAR(255) NOT NULL,
		layout_type VARCHAR(50) NOT NULL,
		layout_data JSONB,
		is_shared BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to setup test schema: %v", err)
	}
}

// cleanupTestSchema removes test data
func cleanupTestSchema(db *sql.DB) {
	db.Exec("TRUNCATE TABLE db_explorer.schema_snapshots CASCADE")
	db.Exec("TRUNCATE TABLE db_explorer.query_history CASCADE")
	db.Exec("TRUNCATE TABLE db_explorer.visualization_layouts CASCADE")
}

// TestServer wraps the server for testing
type TestServer struct {
	Server *Server
	Router *mux.Router
}

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T, db *sql.DB) *TestServer {
	server := &Server{
		db:     db,
		router: mux.NewRouter(),
	}
	server.setupRoutes()

	return &TestServer{
		Server: server,
		Router: server.router,
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
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
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

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	bodyStr := w.Body.String()

	// Check if it's JSON error format
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		// JSON format
		if errorMsg, exists := response["error"]; exists {
			if expectedErrorSubstring != "" && !strings.Contains(errorMsg.(string), expectedErrorSubstring) {
				t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorMsg)
			}
		}
	} else {
		// Plain text error
		if expectedErrorSubstring != "" && !strings.Contains(bodyStr, expectedErrorSubstring) {
			t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorSubstring, bodyStr)
		}
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// SchemaConnectRequest creates a test schema connection request
func (g *TestDataGenerator) SchemaConnectRequest(dbName string) map[string]interface{} {
	return map[string]interface{}{
		"connection_string": "",
		"database_name":     dbName,
	}
}

// QueryGenerateRequest creates a test query generation request
func (g *TestDataGenerator) QueryGenerateRequest(naturalLang, dbContext string, includeExplanation bool) QueryGenerateRequest {
	return QueryGenerateRequest{
		NaturalLanguage:    naturalLang,
		DatabaseContext:    dbContext,
		IncludeExplanation: includeExplanation,
	}
}

// QueryExecuteRequest creates a test query execution request
func (g *TestDataGenerator) QueryExecuteRequest(sqlQuery, dbName string, limit int) QueryExecuteRequest {
	return QueryExecuteRequest{
		SQL:          sqlQuery,
		DatabaseName: dbName,
		Limit:        limit,
	}
}

// LayoutSaveRequest creates a test layout save request
func (g *TestDataGenerator) LayoutSaveRequest(name, dbName, layoutType string, isShared bool) map[string]interface{} {
	return map[string]interface{}{
		"name":          name,
		"database_name": dbName,
		"layout_type":   layoutType,
		"layout_data":   json.RawMessage(`{"nodes": [], "edges": []}`),
		"is_shared":     isShared,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// createTestSchemaSnapshot inserts a test schema snapshot
func createTestSchemaSnapshot(t *testing.T, db *sql.DB, dbName string) uuid.UUID {
	id := uuid.New()
	schemaData := json.RawMessage(`{"tables": [{"name": "users"}]}`)

	query := `
		INSERT INTO db_explorer.schema_snapshots
		(id, database_name, schema_name, tables_count, columns_count, schema_data)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.Exec(query, id, dbName, "public", 1, 5, schemaData)
	if err != nil {
		t.Fatalf("Failed to create test schema snapshot: %v", err)
	}

	return id
}

// createTestQueryHistory inserts a test query history entry
func createTestQueryHistory(t *testing.T, db *sql.DB, dbName string) uuid.UUID {
	id := uuid.New()

	query := `
		INSERT INTO db_explorer.query_history
		(id, natural_language, generated_sql, database_name, query_type)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := db.Exec(query, id, "show all users", "SELECT * FROM users", dbName, "SELECT")
	if err != nil {
		t.Fatalf("Failed to create test query history: %v", err)
	}

	return id
}

// createTestLayout inserts a test visualization layout
func createTestLayout(t *testing.T, db *sql.DB, dbName string) uuid.UUID {
	id := uuid.New()
	layoutData := json.RawMessage(`{"nodes": [], "edges": []}`)

	query := `
		INSERT INTO db_explorer.visualization_layouts
		(id, name, database_name, layout_type, layout_data, is_shared)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.Exec(query, id, "Test Layout", dbName, "graph", layoutData, false)
	if err != nil {
		t.Fatalf("Failed to create test layout: %v", err)
	}

	return id
}

// assertHealthyServices validates health check response
func assertHealthyServices(t *testing.T, response map[string]interface{}) {
	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	services, ok := response["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'services' field in response")
	}

	if dbStatus, ok := services["database"].(string); !ok || dbStatus == "" {
		t.Error("Expected database status in services")
	}
}
