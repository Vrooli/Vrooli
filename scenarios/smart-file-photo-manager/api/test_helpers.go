// +build testing

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

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TestLogger provides controlled logging during tests
func setupTestLogger() func() {
	// Suppress logs during testing unless debugging
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB          *sql.DB
	RedisClient *redis.Client
	App         *App
	Router      *gin.Engine
	Cleanup     func()
}

// setupTestEnvironment creates a test environment with database and dependencies
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Use test database connection
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		t.Skip("POSTGRES_URL not set, skipping integration test")
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Setup Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		opt = &redis.Options{Addr: "localhost:6379"}
	}

	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	}

	// Create processing queue and worker pool
	processingQueue := make(chan ProcessingJob, 100)
	ctx, cancel := context.WithCancel(context.Background())
	workerPool := &WorkerPool{
		workers: 2,
		jobs:    processingQueue,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Create app instance
	app := &App{
		DB:              db,
		RedisClient:     redisClient,
		QdrantURL:       "http://localhost:6333",
		MinioURL:        "localhost:9000",
		OllamaURL:       "http://localhost:11434",
		ProcessingQueue: processingQueue,
		WorkerPool:      workerPool,
	}

	// Set gin to test mode
	gin.SetMode(gin.TestMode)
	router := setupRouter(app)

	cleanup := func() {
		cancel()
		redisClient.Close()
		db.Close()
	}

	return &TestEnvironment{
		DB:          db,
		RedisClient: redisClient,
		App:         app,
		Router:      router,
		Cleanup:     cleanup,
	}
}

// TestFile represents a test file record
type TestFile struct {
	ID           string
	OriginalName string
	FileHash     string
	SizeBytes    int64
	MimeType     string
	StoragePath  string
	FolderPath   string
	Cleanup      func()
}

// createTestFile creates a test file in the database
func createTestFile(t *testing.T, env *TestEnvironment, filename, mimeType string) *TestFile {
	fileHash := fmt.Sprintf("hash_%s_%d", filename, time.Now().UnixNano())
	storagePath := fmt.Sprintf("/test/storage/%s", filename)

	// Check which columns exist
	var hasUserID, hasFilename, hasSizeColumn, hasPath bool
	env.DB.QueryRow(`
		SELECT
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'user_id'),
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'filename'),
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'size'),
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'path')
	`).Scan(&hasUserID, &hasFilename, &hasSizeColumn, &hasPath)

	var query string
	var args []interface{}

	// Build query based on actual schema
	baseColumns := ""
	baseValues := ""
	args = []interface{}{}

	// Always include original_name (fallback if filename doesn't exist)
	if hasFilename {
		baseColumns = "filename, original_name"
		baseValues = "$1, $1" // Use same value for both
		args = append(args, filename)
	} else {
		baseColumns = "original_name"
		baseValues = "$1"
		args = append(args, filename)
	}

	// Add remaining required columns
	argNum := len(args) + 1
	baseColumns += fmt.Sprintf(", file_hash, mime_type, storage_path, folder_path, status, processing_stage")
	baseValues += fmt.Sprintf(", $%d, $%d, $%d, $%d, 'pending', 'uploaded'", argNum, argNum+1, argNum+2, argNum+3)
	args = append(args, fileHash, mimeType, storagePath, "/")
	argNum += 4

	// Add size column if it exists (some schemas use size_bytes, some use size)
	if hasSizeColumn {
		baseColumns += ", size"
		baseValues += fmt.Sprintf(", $%d", argNum)
		args = append(args, int64(1024))
		argNum++
	} else {
		baseColumns += ", size_bytes"
		baseValues += fmt.Sprintf(", $%d", argNum)
		args = append(args, int64(1024))
		argNum++
	}

	// Add user_id if exists
	if hasUserID {
		baseColumns += ", user_id"
		baseValues += fmt.Sprintf(", $%d", argNum)
		args = append(args, "test_user")
		argNum++
	}

	// Add path if exists (some schemas have both path and storage_path)
	if hasPath {
		baseColumns += ", path"
		baseValues += fmt.Sprintf(", $%d", argNum)
		args = append(args, storagePath)
		argNum++
	}

	// Add timestamps
	baseColumns += ", uploaded_at, created_at, updated_at"
	baseValues += ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP"

	query = fmt.Sprintf(`
		INSERT INTO files (%s)
		VALUES (%s)
		RETURNING id
	`, baseColumns, baseValues)

	var fileID string
	err := env.DB.QueryRow(query, args...).Scan(&fileID)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	return &TestFile{
		ID:           fileID,
		OriginalName: filename,
		FileHash:     fileHash,
		SizeBytes:    1024,
		MimeType:     mimeType,
		StoragePath:  storagePath,
		FolderPath:   "/",
		Cleanup: func() {
			env.DB.Exec("DELETE FROM files WHERE id = $1", fileID)
		},
	}
}

// TestFolder represents a test folder record
type TestFolder struct {
	ID      string
	Path    string
	Name    string
	Cleanup func()
}

// createTestFolder creates a test folder in the database
func createTestFolder(t *testing.T, env *TestEnvironment, path, name string) *TestFolder {
	query := `
		INSERT INTO folders (path, name, created_at, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var folderID string
	err := env.DB.QueryRow(query, path, name).Scan(&folderID)
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}

	return &TestFolder{
		ID:   folderID,
		Path: path,
		Name: name,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM folders WHERE id = $1", folderID)
		},
	}
}

// makeHTTPRequest creates and executes an HTTP request against the test router
func makeHTTPRequest(env *TestEnvironment, method, path string, body interface{}) (*httptest.ResponseRecorder, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, path, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	return w, nil
}

// assertStatusCode verifies the HTTP status code
func assertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("Expected status %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()
	assertStatusCode(t, w, expectedStatus)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}
	return response
}

// assertErrorResponse validates error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()
	response := assertJSONResponse(t, w, expectedStatus)

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Fatalf("Response missing 'error' field: %v", response)
	}

	if expectedErrorContains != "" && !stringContains(errorMsg, expectedErrorContains) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedErrorContains, errorMsg)
	}
}

// stringContains checks if string contains substring
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		bytes.Contains([]byte(s), []byte(substr)))
}

// cleanupTestData removes all test data from database
func cleanupTestData(env *TestEnvironment) {
	env.DB.Exec("DELETE FROM suggestions WHERE created_at > NOW() - INTERVAL '1 hour'")
	env.DB.Exec("DELETE FROM files WHERE uploaded_at > NOW() - INTERVAL '1 hour'")
	env.DB.Exec("DELETE FROM folders WHERE created_at > NOW() - INTERVAL '1 hour'")
}
