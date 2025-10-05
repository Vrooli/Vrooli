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
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// setupTestLogger initializes a test logger with controlled output
func setupTestLogger() func() {
	// Silence logs during tests or redirect to test output
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// setupTestDBWithProcessor creates a test database connection and initializes the processor
func setupTestDBWithProcessor(t *testing.T) (*sql.DB, func()) {
	testDB, cleanup := setupTestDB(t)
	if testDB != nil {
		// Initialize global mindMapProcessor for handlers
		mindMapProcessor = setupTestProcessor(t, testDB)
	}
	return testDB, cleanup
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use test database configuration
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")

		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "5432"
		}
		if user == "" {
			user = "postgres"
		}
		if password == "" {
			password = "postgres"
		}
		if dbname == "" {
			dbname = "mind_maps_test"
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil, func() {}
	}

	// Configure connection pool
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(2)
	testDB.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test: database not available: %v", err)
		return nil, func() {}
	}

	// Initialize schema if needed
	initTestSchema(t, testDB)

	cleanup := func() {
		// Clean up test data
		cleanupTestData(testDB)
		testDB.Close()
	}

	return testDB, cleanup
}

// initTestSchema creates necessary tables for testing
func initTestSchema(t *testing.T, db *sql.DB) {
	// Drop existing tables to ensure clean schema
	dropSchema := `
	DROP TABLE IF EXISTS mind_map_nodes CASCADE;
	DROP TABLE IF EXISTS mind_maps CASCADE;
	`

	if _, err := db.Exec(dropSchema); err != nil {
		t.Logf("Warning: Could not drop existing schema: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS mind_maps (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(255) NOT NULL,
		description TEXT,
		campaign_id VARCHAR(255),
		owner_id VARCHAR(255) NOT NULL DEFAULT 'default-user',
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		is_public BOOLEAN DEFAULT false,
		share_token VARCHAR(255) UNIQUE
	);

	CREATE TABLE IF NOT EXISTS mind_map_nodes (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		mind_map_id UUID NOT NULL REFERENCES mind_maps(id) ON DELETE CASCADE,
		map_id UUID REFERENCES mind_maps(id) ON DELETE CASCADE,
		title VARCHAR(255),
		content TEXT NOT NULL,
		node_type VARCHAR(50) DEFAULT 'child',
		type VARCHAR(50) DEFAULT 'child',
		position_x FLOAT DEFAULT 0,
		position_y FLOAT DEFAULT 0,
		parent_id UUID REFERENCES mind_map_nodes(id) ON DELETE CASCADE,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_mind_maps_owner ON mind_maps(owner_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_mind_map ON mind_map_nodes(mind_map_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_parent ON mind_map_nodes(parent_id);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Logf("Warning: Could not initialize schema: %v", err)
	}
}

// cleanupTestData removes all test data from the database
func cleanupTestData(db *sql.DB) {
	db.Exec("DELETE FROM mind_map_nodes")
	db.Exec("DELETE FROM mind_maps")
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		// Check if Body is a string (for invalid JSON tests)
		if bodyStr, ok := req.Body.(string); ok {
			bodyReader = bytes.NewBufferString(bodyStr)
		} else {
			bodyBytes, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewBuffer(bodyBytes)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars for mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Execute request
	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedData interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if expectedData != nil {
		var response Response
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Additional validation can be added here based on expectedData
	}
}

// assertErrorResponse validates an error response (supports both JSON and plain text)
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContainMessage string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Try to decode as JSON first
	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		// If not JSON, it might be plain text error (from http.Error)
		bodyText := w.Body.String()
		if bodyText == "" {
			t.Logf("Empty error response body")
		} else {
			t.Logf("Plain text error response: %s", bodyText)
		}
		return
	}

	if response.Status != "error" && response.Status != "fail" {
		t.Errorf("Expected error status, got %s", response.Status)
	}

	if shouldContainMessage != "" && response.Message == "" {
		t.Error("Expected error message, got empty string")
	}
}

// createTestMindMap creates a test mind map in the database
func createTestMindMap(t *testing.T, db *sql.DB, title, userID string) *MindMap {
	t.Helper()

	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO mind_maps (id, title, description, owner_id, metadata, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, title, description, owner_id, created_at, updated_at
	`

	var mindMap MindMap
	var createdAt, updatedAt time.Time

	err := db.QueryRow(query, id, title, "Test description", userID, "{}", false, now).Scan(
		&mindMap.ID, &mindMap.Title, &mindMap.Description, &mindMap.OwnerID,
		&createdAt, &updatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to create test mind map: %v", err)
	}

	mindMap.CreatedAt = createdAt.Format(time.RFC3339)
	mindMap.UpdatedAt = updatedAt.Format(time.RFC3339)
	mindMap.Metadata = make(map[string]interface{})

	return &mindMap
}

// createTestNode creates a test node in the database
func createTestNode(t *testing.T, db *sql.DB, mindMapID, content, nodeType string) *Node {
	t.Helper()

	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO mind_map_nodes (id, mind_map_id, map_id, content, type, position_x, position_y, metadata, created_at, updated_at)
		VALUES ($1, $2, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id, mind_map_id, content, type, position_x, position_y, metadata
	`

	var node Node
	var metadataJSON string

	err := db.QueryRow(query, id, mindMapID, content, nodeType, 0.0, 0.0, "{}", now).Scan(
		&node.ID, &node.MindMapID, &node.Content, &node.Type,
		&node.PositionX, &node.PositionY, &metadataJSON,
	)

	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	node.Metadata = make(map[string]interface{})

	return &node
}

// setupTestProcessor creates a test processor instance
func setupTestProcessor(t *testing.T, db *sql.DB) *MindMapProcessor {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	return NewMindMapProcessor(db, ollamaURL, qdrantURL)
}
