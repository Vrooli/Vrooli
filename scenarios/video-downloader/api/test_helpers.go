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
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Suppress logs during tests unless verbose mode
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(ioutil.Discard)
		return func() { log.SetOutput(os.Stderr) }
	}
	return func() {}
}

// TestDatabase manages test database lifecycle
type TestDatabase struct {
	DB      *sql.DB
	ConnStr string
	Cleanup func()
}

// setupTestDatabase creates an isolated test database
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database connection
	connStr := os.Getenv("TEST_POSTGRES_URL")
	if connStr == "" {
		// Build from components with test suffix
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" {
			t.Skip("Database configuration not available for testing")
			return nil
		}

		// Use test database name
		if dbName == "" {
			dbName = "video_downloader_test"
		} else {
			dbName = dbName + "_test"
		}

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		t.Skipf("Test database not available: %v", err)
		return nil
	}

	// Create test schema
	createTestSchema(t, testDB)

	return &TestDatabase{
		DB:      testDB,
		ConnStr: connStr,
		Cleanup: func() {
			cleanupTestData(testDB)
			testDB.Close()
		},
	}
}

// createTestSchema creates required tables for testing
func createTestSchema(t *testing.T, testDB *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS downloads (
			id SERIAL PRIMARY KEY,
			url TEXT NOT NULL,
			title TEXT,
			description TEXT,
			platform TEXT,
			duration INTEGER DEFAULT 0,
			format TEXT,
			quality TEXT,
			audio_format TEXT,
			audio_quality TEXT,
			audio_path TEXT,
			audio_file_size BIGINT DEFAULT 0,
			file_path TEXT,
			file_size BIGINT DEFAULT 0,
			status TEXT DEFAULT 'pending',
			progress INTEGER DEFAULT 0,
			has_transcript BOOLEAN DEFAULT FALSE,
			transcript_status TEXT,
			transcript_requested BOOLEAN DEFAULT FALSE,
			whisper_model TEXT,
			target_language TEXT,
			error_message TEXT,
			user_id TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP,
			transcript_started_at TIMESTAMP,
			transcript_completed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS download_queue (
			id SERIAL PRIMARY KEY,
			download_id INTEGER REFERENCES downloads(id) ON DELETE CASCADE,
			position INTEGER NOT NULL,
			priority INTEGER DEFAULT 1,
			retry_count INTEGER DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS transcripts (
			id SERIAL PRIMARY KEY,
			download_id INTEGER REFERENCES downloads(id) ON DELETE CASCADE,
			language TEXT,
			detected_language TEXT,
			confidence_score DOUBLE PRECISION DEFAULT 0.0,
			model_used TEXT,
			full_text TEXT,
			word_count INTEGER DEFAULT 0,
			processing_time_ms INTEGER DEFAULT 0,
			audio_duration_seconds DOUBLE PRECISION DEFAULT 0.0,
			whisper_version TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS transcript_segments (
			id SERIAL PRIMARY KEY,
			transcript_id INTEGER REFERENCES transcripts(id) ON DELETE CASCADE,
			start_time DOUBLE PRECISION,
			end_time DOUBLE PRECISION,
			text TEXT,
			confidence DOUBLE PRECISION DEFAULT 0.0,
			speaker_id TEXT,
			word_timestamps JSONB,
			sequence INTEGER,
			character_start INTEGER,
			character_end INTEGER
		);
	`

	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

// cleanupTestData removes all test data
func cleanupTestData(testDB *sql.DB) {
	testDB.Exec("TRUNCATE downloads, download_queue, transcripts, transcript_segments CASCADE")
}

// TestDownload provides a pre-configured download for testing
type TestDownload struct {
	Download *Download
	Cleanup  func()
}

// setupTestDownload creates a test download record
func setupTestDownload(t *testing.T, testDB *sql.DB, url string, options map[string]interface{}) *TestDownload {
	// Set defaults
	format := getStringOption(options, "format", "mp4")
	quality := getStringOption(options, "quality", "best")
	status := getStringOption(options, "status", "pending")
	transcriptRequested := getBoolOption(options, "transcript_requested", false)

	var downloadID int
	err := testDB.QueryRow(`
		INSERT INTO downloads (url, title, format, quality, status, transcript_requested)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		url, "Test Video", format, quality, status, transcriptRequested).Scan(&downloadID)

	if err != nil {
		t.Fatalf("Failed to create test download: %v", err)
	}

	download := &Download{
		ID:                  downloadID,
		URL:                 url,
		Title:               "Test Video",
		Format:              format,
		Quality:             quality,
		Status:              status,
		TranscriptRequested: transcriptRequested,
	}

	return &TestDownload{
		Download: download,
		Cleanup: func() {
			testDB.Exec("DELETE FROM downloads WHERE id = $1", downloadID)
		},
	}
}

// setupTestTranscript creates a test transcript for a download
func setupTestTranscript(t *testing.T, testDB *sql.DB, downloadID int) int {
	var transcriptID int
	err := testDB.QueryRow(`
		INSERT INTO transcripts (download_id, language, model_used, full_text, word_count, confidence_score)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		downloadID, "en", "base", "This is a test transcript.", 5, 0.95).Scan(&transcriptID)

	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Add some segments
	for i := 0; i < 3; i++ {
		_, err := testDB.Exec(`
			INSERT INTO transcript_segments (transcript_id, start_time, end_time, text, confidence, sequence, character_start, character_end)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			transcriptID, float64(i*10), float64(i*10+10), fmt.Sprintf("Segment %d text", i), 0.9, i, i*10, i*10+10)
		if err != nil {
			t.Fatalf("Failed to create test segment: %v", err)
		}
	}

	return transcriptID
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
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
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

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// assertHTMLErrorResponse validates plain text/HTML error responses
func assertHTMLErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	body := w.Body.String()
	if expectedMessage != "" && body != expectedMessage+"\n" {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, body)
	}
}

// Helper functions for option extraction
func getStringOption(options map[string]interface{}, key, defaultValue string) string {
	if options == nil {
		return defaultValue
	}
	if val, ok := options[key].(string); ok {
		return val
	}
	return defaultValue
}

func getBoolOption(options map[string]interface{}, key string, defaultValue bool) bool {
	if options == nil {
		return defaultValue
	}
	if val, ok := options[key].(bool); ok {
		return val
	}
	return defaultValue
}

// waitForCondition polls a condition until it's true or timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}
