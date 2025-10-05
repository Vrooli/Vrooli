package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"github.com/vrooli/video-tools/internal/video"
)

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestLogger initializes controlled logging during tests
func setupTestLogger() func() {
	// Silence logs during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(io.Discard)
	}
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "video-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create necessary subdirectories
	subdirs := []string{"uploads", "processing", "frames", "thumbnails", "audio"}
	for _, dir := range subdirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create subdir %s: %v", dir, err)
		}
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestServer creates a test server with in-memory database
func setupTestServer(t *testing.T) (*Server, func()) {
	env := setupTestDirectory(t)

	// Use in-memory SQLite or test database
	testDB := os.Getenv("TEST_DATABASE_URL")
	if testDB == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", testDB)
	if err != nil {
		env.Cleanup()
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		env.Cleanup()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Create test processor (mock or real)
	processor, err := createTestProcessor(env.TempDir)
	if err != nil {
		db.Close()
		env.Cleanup()
		t.Fatalf("Failed to create test processor: %v", err)
	}

	config := &Config{
		Port:        "0", // Random port for testing
		DatabaseURL: testDB,
		WorkDir:     env.TempDir,
		APIToken:    "test-token",
	}

	server := &Server{
		config:    config,
		db:        db,
		router:    mux.NewRouter(),
		processor: processor,
	}

	server.setupRoutes()

	cleanup := func() {
		db.Close()
		env.Cleanup()
	}

	return server, cleanup
}

// createTestProcessor creates a processor for testing
func createTestProcessor(workDir string) (*video.Processor, error) {
	// Try to create real processor, fall back to mock if ffmpeg not available
	processor, err := video.NewProcessor(filepath.Join(workDir, "processing"))
	if err != nil {
		// Return mock processor for environments without ffmpeg
		return &video.Processor{}, nil
	}
	return processor, nil
}

// makeHTTPRequest creates and executes an HTTP request
func makeHTTPRequest(server *Server, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)

	// Add default headers
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Authorization"]; !ok {
		headers["Authorization"] = "Bearer " + server.config.APIToken
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	return w
}

// makeJSONRequest creates and executes a JSON HTTP request
func makeJSONRequest(server *Server, method, path string, payload interface{}) *httptest.ResponseRecorder {
	jsonData, _ := json.Marshal(payload)
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	return makeHTTPRequest(server, method, path, bytes.NewReader(jsonData), headers)
}

// makeMultipartRequest creates and executes a multipart form request
func makeMultipartRequest(server *Server, path string, fields map[string]string, files map[string][]byte) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, value := range fields {
		writer.WriteField(key, value)
	}

	// Add files
	for filename, content := range files {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil
		}
		part.Write(content)
	}

	writer.Close()

	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	return makeHTTPRequest(server, "POST", path, body, headers)
}

// assertJSONResponse validates JSON response structure and status
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	response := assertJSONResponse(t, w, expectedStatus)

	success, ok := response["success"].(bool)
	if !ok || success {
		t.Errorf("Expected success=false in error response")
	}

	if expectedMessage != "" {
		errorMsg, ok := response["error"].(string)
		if !ok {
			t.Errorf("Expected error message in response")
		} else if errorMsg != expectedMessage {
			t.Errorf("Expected error message '%s', got '%s'", expectedMessage, errorMsg)
		}
	}
}

// createTestVideo creates a minimal test video file
func createTestVideo(t *testing.T, path string) {
	t.Helper()

	// Create a minimal MP4 header (not a real video, but enough for basic tests)
	// For real video processing tests, you'd need actual video content
	testData := []byte{
		0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, // ftyp header
		0x69, 0x73, 0x6f, 0x6d, 0x00, 0x00, 0x02, 0x00,
		0x69, 0x73, 0x6f, 0x6d, 0x69, 0x73, 0x6f, 0x32,
		0x61, 0x76, 0x63, 0x31, 0x6d, 0x70, 0x34, 0x31,
	}

	if err := os.WriteFile(path, testData, 0644); err != nil {
		t.Fatalf("Failed to create test video: %v", err)
	}
}

// insertTestVideo inserts a test video record into the database
func insertTestVideo(t *testing.T, db *sql.DB) string {
	t.Helper()

	videoID := uuid.New().String()
	query := `INSERT INTO video_assets (
		id, name, description, format, duration_seconds,
		resolution_width, resolution_height, frame_rate,
		file_size_bytes, codec, bitrate_kbps, has_audio,
		audio_channels, status
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := db.Exec(query,
		videoID, "Test Video", "Test Description", "mp4", 10.5,
		1920, 1080, 30.0,
		1024000, "h264", 5000, true,
		2, "ready",
	)

	if err != nil {
		t.Fatalf("Failed to insert test video: %v", err)
	}

	return videoID
}

// insertTestJob inserts a test processing job into the database
func insertTestJob(t *testing.T, db *sql.DB, videoID, jobType string) string {
	t.Helper()

	jobID := uuid.New().String()
	query := `INSERT INTO processing_jobs (id, video_id, job_type, status)
		VALUES ($1, $2, $3, $4)`

	_, err := db.Exec(query, jobID, videoID, jobType, "pending")
	if err != nil {
		t.Fatalf("Failed to insert test job: %v", err)
	}

	return jobID
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	queries := []string{
		`DELETE FROM video_analytics WHERE 1=1`,
		`DELETE FROM processing_jobs WHERE 1=1`,
		`DELETE FROM streaming_sessions WHERE 1=1`,
		`DELETE FROM video_assets WHERE name LIKE 'Test%'`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Logf("Warning: cleanup query failed: %v", err)
		}
	}
}
