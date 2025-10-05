// +build testing

package main

import (
	"bytes"
	"context"
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
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Suppress logs during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "1" {
		log.SetOutput(ioutil.Discard)
		return func() {
			log.SetOutput(os.Stderr)
		}
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	DB         *sql.DB
	Server     *ApiServer
	Router     *mux.Router
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Create temporary directory
	tempDir, err := ioutil.TempDir("", "idea-generator-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Setup test database connection
	db := setupTestDB(t)

	// Create mock IdeaProcessor
	processor := &IdeaProcessor{
		db:        db,
		ollamaURL: "http://localhost:11434",
		qdrantURL: "http://localhost:6333",
	}

	server := &ApiServer{
		db:             db,
		ideaProcessor:  processor,
		windmillURL:    "http://localhost:5681",
		postgresURL:    os.Getenv("TEST_POSTGRES_URL"),
		qdrantURL:      "http://localhost:6333",
		minioURL:       "http://localhost:9000",
		redisURL:       "http://localhost:6379",
		ollamaURL:      "http://localhost:11434",
		unstructuredURL: "http://localhost:11450",
	}

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/health", server.healthHandler).Methods("GET")
	router.HandleFunc("/status", server.statusHandler).Methods("GET")
	router.HandleFunc("/campaigns", server.campaignsHandler).Methods("GET", "POST")
	router.HandleFunc("/ideas", server.ideasHandler).Methods("GET", "POST")
	router.HandleFunc("/api/ideas/generate", server.generateIdeasHandler).Methods("POST")
	router.HandleFunc("/ideas/refine", server.refineIdeaHandler).Methods("POST")
	router.HandleFunc("/search", server.searchHandler).Methods("POST")
	router.HandleFunc("/documents/process", server.processDocumentHandler).Methods("POST")
	router.HandleFunc("/workflows", server.workflowsHandler).Methods("GET")

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		DB:         db,
		Server:     server,
		Router:     router,
		Cleanup: func() {
			db.Close()
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory SQLite for tests or test PostgreSQL instance
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		// Skip database tests if no test database is configured
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Setup test schema
	setupTestSchema(t, db)

	return db
}

// setupTestSchema creates necessary tables for testing
func setupTestSchema(t *testing.T, db *sql.DB) {
	schema := `
	CREATE TABLE IF NOT EXISTS campaigns (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		context TEXT,
		color VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ideas (
		id VARCHAR(255) PRIMARY KEY,
		campaign_id VARCHAR(255) REFERENCES campaigns(id),
		title VARCHAR(500) NOT NULL,
		content TEXT,
		category VARCHAR(100),
		tags TEXT[],
		status VARCHAR(50) DEFAULT 'generated',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS documents (
		id VARCHAR(255) PRIMARY KEY,
		campaign_id VARCHAR(255) REFERENCES campaigns(id),
		original_name VARCHAR(500),
		extracted_text TEXT,
		processing_status VARCHAR(50) DEFAULT 'pending',
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
	db.Exec("DROP TABLE IF EXISTS documents")
	db.Exec("DROP TABLE IF EXISTS ideas")
	db.Exec("DROP TABLE IF EXISTS campaigns")
}

// TestCampaign provides a pre-configured campaign for testing
type TestCampaign struct {
	ID          string
	Name        string
	Description string
	Context     string
	Cleanup     func()
}

// createTestCampaign creates a test campaign in the database
func createTestCampaign(t *testing.T, db *sql.DB, name string) *TestCampaign {
	campaignID := uuid.New().String()
	description := fmt.Sprintf("Test campaign: %s", name)
	context := "Test context for idea generation"

	query := `INSERT INTO campaigns (id, name, description, context, color)
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, campaignID, name, description, context, "#3B82F6")
	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	return &TestCampaign{
		ID:          campaignID,
		Name:        name,
		Description: description,
		Context:     context,
		Cleanup: func() {
			db.Exec("DELETE FROM ideas WHERE campaign_id = $1", campaignID)
			db.Exec("DELETE FROM documents WHERE campaign_id = $1", campaignID)
			db.Exec("DELETE FROM campaigns WHERE id = $1", campaignID)
		},
	}
}

// createTestIdea creates a test idea in the database
func createTestIdea(t *testing.T, db *sql.DB, campaignID, title, content string) string {
	ideaID := uuid.New().String()

	query := `INSERT INTO ideas (id, campaign_id, title, content, category, status)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := db.Exec(query, ideaID, campaignID, title, content, "innovation", "generated")
	if err != nil {
		t.Fatalf("Failed to create test idea: %v", err)
	}

	return ideaID
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
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader *bytes.Buffer

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBody, _ := json.Marshal(v)
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	} else {
		bodyReader = bytes.NewBufferString("")
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT
	if (req.Method == "POST" || req.Method == "PUT") && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL vars if present
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if expectedBody != nil {
		var actualBody interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &actualBody); err != nil {
			t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
			return
		}

		// For complex validation, use type assertion
		switch expected := expectedBody.(type) {
		case map[string]interface{}:
			actual, ok := actualBody.(map[string]interface{})
			if !ok {
				t.Errorf("Expected map response, got %T", actualBody)
				return
			}
			// Validate expected fields exist
			for key := range expected {
				if _, exists := actual[key]; !exists {
					t.Errorf("Expected field '%s' not found in response", key)
				}
			}
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContain ...string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	body := w.Body.String()
	for _, s := range shouldContain {
		if !strings.Contains(body, s) {
			t.Errorf("Expected response to contain '%s', got: %s", s, body)
		}
	}
}

// mockOllamaServer creates a mock Ollama server for testing
type MockOllamaServer struct {
	Server *httptest.Server
	URL    string
}

func createMockOllamaServer() *MockOllamaServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/generate":
			response := map[string]interface{}{
				"response": `[{"title":"Test Idea","description":"Test description","category":"innovation","tags":["test"],"implementation_notes":"Test notes"}]`,
			}
			json.NewEncoder(w).Encode(response)
		case "/api/embeddings":
			response := map[string]interface{}{
				"embedding": make([]float64, 384), // Mock 384-dim embedding
			}
			json.NewEncoder(w).Encode(response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewServer(handler)
	return &MockOllamaServer{
		Server: server,
		URL:    server.URL,
	}
}

func (m *MockOllamaServer) Close() {
	m.Server.Close()
}

// mockQdrantServer creates a mock Qdrant server for testing
type MockQdrantServer struct {
	Server *httptest.Server
	URL    string
}

func createMockQdrantServer() *MockQdrantServer {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/points/search"):
			response := map[string]interface{}{
				"result": []map[string]interface{}{
					{
						"id":    "test-idea-1",
						"score": 0.95,
						"payload": map[string]interface{}{
							"title":       "Test Idea",
							"content":     "Test content",
							"category":    "innovation",
							"campaign_id": "test-campaign",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		case strings.Contains(r.URL.Path, "/points"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewServer(handler)
	return &MockQdrantServer{
		Server: server,
		URL:    server.URL,
	}
}

func (m *MockQdrantServer) Close() {
	m.Server.Close()
}

// createContextWithTimeout creates a context with timeout for testing
func createContextWithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
