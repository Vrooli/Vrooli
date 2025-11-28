package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// requireIntegrationTest skips test if running in short mode
func requireIntegrationTest(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// setupTestServerOrSkip sets up test server and skips test if setup fails
func setupTestServerOrSkip(t *testing.T) *Server {
	t.Helper()
	requireIntegrationTest(t)
	srv := setupTestServer(t)
	if srv == nil {
		t.Skip("Test server setup failed")
	}
	return srv
}

// makeRequest creates and executes an HTTP request, returning the recorder
func makeRequest(t *testing.T, router http.Handler, method, url string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// unmarshalResponse unmarshals JSON response into target
func unmarshalResponse(t *testing.T, rr *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), target); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
}

// assertStatusCode checks if response status matches expected
func assertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("Handler returned status %d, want %d. Body: %s", rr.Code, expected, rr.Body.String())
	}
}

// setupTestServer creates a test server with database connection and routing
func setupTestServer(t *testing.T) *Server {
	srv := setupTestServerNoData(t)
	if srv == nil {
		return nil
	}

	// Create issues table if not exists
	_, err := srv.db.Exec(`
		CREATE TABLE IF NOT EXISTS issues (
			id SERIAL PRIMARY KEY,
			scenario VARCHAR(255) NOT NULL,
			file_path TEXT NOT NULL,
			category VARCHAR(100) NOT NULL,
			severity VARCHAR(50) NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			line_number INTEGER,
			column_number INTEGER,
			agent_notes TEXT,
			remediation_steps TEXT,
			status VARCHAR(50) DEFAULT 'open',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			campaign_id INTEGER,
			session_id VARCHAR(255)
		)
	`)
	if err != nil {
		t.Logf("Warning: Could not create issues table: %v", err)
	}

	// Clean up test data
	_, _ = srv.db.Exec("DELETE FROM issues WHERE scenario LIKE 'test-%'")

	return srv
}

// setupTestServerNoData creates a minimal test server without creating tables
func setupTestServerNoData(t *testing.T) *Server {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
		return nil
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Could not open database connection")
		return nil
	}

	if err := db.Ping(); err != nil {
		t.Skip("Could not ping database, skipping integration test")
		return nil
	}

	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: dbURL,
		},
		db:     db,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	return srv
}

// insertTestIssue inserts a test issue into the database
func insertTestIssue(t *testing.T, db *sql.DB, scenario, filePath, category, severity, title, description string) {
	_, err := db.Exec(`
		INSERT INTO issues (
			scenario, file_path, category, severity, title, description, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, scenario, filePath, category, severity, title, description, "open", time.Now())

	if err != nil {
		t.Fatalf("Failed to insert test issue: %v", err)
	}
}

// File system test helpers

// setupTestDir creates a temp directory with a subdirectory structure
func setupTestDir(t *testing.T, subDir string) (rootDir, targetDir string) {
	t.Helper()
	rootDir = t.TempDir()
	targetDir = filepath.Join(rootDir, subDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", targetDir, err)
	}
	return rootDir, targetDir
}

// writeTestFile writes content to a file path, creating parent dirs as needed
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// File metrics test helpers

// findMetricsByPath searches metrics slice for a specific file path
func findMetricsByPath(metrics []DetailedFileMetrics, filePath string) *DetailedFileMetrics {
	for i := range metrics {
		if metrics[i].FilePath == filePath {
			return &metrics[i]
		}
	}
	return nil
}

// requireMetricsByPath finds metrics or fails the test
func requireMetricsByPath(t *testing.T, metrics []DetailedFileMetrics, filePath string) *DetailedFileMetrics {
	t.Helper()
	m := findMetricsByPath(metrics, filePath)
	if m == nil {
		t.Fatalf("Expected to find metrics for %s", filePath)
	}
	return m
}

// collectAndValidateMetrics collects metrics and validates the count
func collectAndValidateMetrics(t *testing.T, tmpDir string, files []string, expectedCount int) []DetailedFileMetrics {
	t.Helper()
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)
	if err != nil {
		t.Fatalf("CollectDetailedFileMetrics failed: %v", err)
	}
	if len(metrics) != expectedCount {
		t.Fatalf("Expected %d file metrics, got %d", expectedCount, len(metrics))
	}
	return metrics
}

// Database test helpers

// setupDBTest sets up and validates a database connection for integration tests
func setupDBTest(t *testing.T) (*Server, *sql.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Could not open database connection")
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skip("Could not ping database, skipping integration test")
	}
	srv := &Server{
		config: &Config{DatabaseURL: dbURL},
		db:     db,
	}
	return srv, db
}

// cleanupDBTest cleans up test data from the database
func cleanupDBTest(db *sql.DB, scenario string) {
	_, _ = db.Exec("DELETE FROM file_metrics WHERE scenario = $1", scenario)
}
