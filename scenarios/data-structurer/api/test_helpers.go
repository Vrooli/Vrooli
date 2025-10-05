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
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// setupTestLogger initializes a test logger with controlled output
func setupTestLogger() func() {
	// Save original gin mode
	originalMode := gin.Mode()

	// Set gin to test mode to reduce noise
	gin.SetMode(gin.TestMode)

	// Disable default logging for cleaner test output
	gin.DefaultWriter = io.Discard

	return func() {
		gin.SetMode(originalMode)
		gin.DefaultWriter = os.Stdout
	}
}

// TestDatabase manages test database lifecycle
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates an isolated test database environment
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database connection or in-memory if available
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "data_structurer_test"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping database test: %v", err)
		return nil
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctx); err != nil {
		testDB.Close()
		t.Skipf("Skipping database test - database unavailable: %v", err)
		return nil
	}

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM processed_data WHERE source_file_name LIKE 'test-%'")
			testDB.Exec("DELETE FROM schema_templates WHERE name LIKE 'test-%'")
			testDB.Exec("DELETE FROM schemas WHERE name LIKE 'test-%'")
			testDB.Close()
		},
	}
}

// TestEnvironment manages complete test environment
type TestEnvironment struct {
	TempDir    string
	Database   *TestDatabase
	Router     *gin.Engine
	OriginalWD string
	Cleanup    func()
}

// setupTestEnvironment creates a complete isolated test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	cleanup := setupTestLogger()

	tempDir, err := os.MkdirTemp("", "data-structurer-test-*")
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		cleanup()
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Setup test database
	testDB := setupTestDatabase(t)

	// Create test router
	router := gin.New()
	router.Use(gin.Recovery())

	env := &TestEnvironment{
		TempDir:    tempDir,
		Database:   testDB,
		Router:     router,
		OriginalWD: originalWD,
		Cleanup: func() {
			if testDB != nil {
				testDB.Cleanup()
			}
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
			cleanup()
		},
	}

	return env
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				log.Printf("Failed to marshal request body: %v", err)
				return nil
			}
			bodyReader = bytes.NewBuffer(jsonData)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set content-type for JSON if not specified
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, checkFields map[string]interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if checkFields == nil {
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	for field, expectedValue := range checkFields {
		if actualValue, ok := response[field]; ok {
			if actualValue != expectedValue {
				t.Errorf("Field '%s': expected %v, got %v", field, expectedValue, actualValue)
			}
		} else {
			t.Errorf("Expected field '%s' not found in response", field)
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContainError bool) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if !shouldContainError {
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Body: %s", err, w.Body.String())
	}

	if _, hasError := response["error"]; !hasError {
		if _, hasMessage := response["message"]; !hasMessage {
			t.Errorf("Error response should contain 'error' or 'message' field")
		}
	}
}

// createTestSchema creates a test schema in the database
func createTestSchema(t *testing.T, testDB *TestDatabase, name string) *Schema {
	t.Helper()

	if testDB == nil || testDB.DB == nil {
		t.Skip("Database not available")
		return nil
	}

	schema := &Schema{
		ID:          uuid.New(),
		Name:        "test-" + name,
		Description: "Test schema for " + name,
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string"},
				"age":  map[string]interface{}{"type": "integer"},
			},
		},
		Version:   1,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "test-user",
	}

	schemaJSON, err := json.Marshal(schema.SchemaDefinition)
	if err != nil {
		t.Fatalf("Failed to marshal schema definition: %v", err)
	}

	_, err = testDB.DB.Exec(`
		INSERT INTO schemas (id, name, description, schema_definition, version, is_active, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, schema.ID, schema.Name, schema.Description, schemaJSON, schema.Version, schema.IsActive,
		schema.CreatedAt, schema.UpdatedAt, schema.CreatedBy)

	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return schema
}

// createTestProcessedData creates test processed data in the database
func createTestProcessedData(t *testing.T, testDB *TestDatabase, schemaID uuid.UUID) *ProcessedData {
	t.Helper()

	if testDB == nil || testDB.DB == nil {
		t.Skip("Database not available")
		return nil
	}

	confidence := 0.95
	processingTime := 100

	processedData := &ProcessedData{
		ID:               uuid.New(),
		SchemaID:         schemaID,
		SourceFileName:   "test-file.txt",
		SourceFilePath:   "/tmp/test-file.txt",
		SourceFileType:   "text",
		SourceFileSize:   1024,
		RawContent:       "Test content",
		StructuredData:   map[string]interface{}{"name": "Test", "age": 30},
		ConfidenceScore:  &confidence,
		ProcessingStatus: "completed",
		ProcessingTimeMs: &processingTime,
		CreatedAt:        time.Now(),
	}

	structuredJSON, err := json.Marshal(processedData.StructuredData)
	if err != nil {
		t.Fatalf("Failed to marshal structured data: %v", err)
	}

	_, err = testDB.DB.Exec(`
		INSERT INTO processed_data (
			id, schema_id, source_file_name, source_file_path, source_file_type,
			source_file_size, raw_content, structured_data, confidence_score,
			processing_status, processing_time_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, processedData.ID, processedData.SchemaID, processedData.SourceFileName,
		processedData.SourceFilePath, processedData.SourceFileType,
		processedData.SourceFileSize, processedData.RawContent, structuredJSON,
		processedData.ConfidenceScore, processedData.ProcessingStatus,
		processedData.ProcessingTimeMs, processedData.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test processed data: %v", err)
	}

	return processedData
}

// createTestFile creates a temporary test file
func createTestFile(t *testing.T, tempDir, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return filePath
}

// cleanupTestFiles removes test files
func cleanupTestFiles(paths ...string) {
	for _, path := range paths {
		os.Remove(path)
	}
}

// skipIfNoDatabase skips the test if database is not available
func skipIfNoDatabase(t *testing.T, testDB *TestDatabase) {
	if testDB == nil || testDB.DB == nil {
		t.Skip("Database not available for testing")
	}
}

// skipIfNoOllama skips the test if Ollama is not available
func skipIfNoOllama(t *testing.T) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(ollamaHost + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skip("Ollama not available for testing")
	}
}
