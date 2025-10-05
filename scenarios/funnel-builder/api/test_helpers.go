package main

import (
	"bytes"
	"context"
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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalMode string
}

// setupTestLogger initializes gin for testing mode
func setupTestLogger() func() {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard) // Suppress logs during tests
	return func() {
		gin.SetMode(originalMode)
		log.SetOutput(os.Stderr)
	}
}

// TestDatabase manages test database connection
type TestDatabase struct {
	Pool    *pgxpool.Pool
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	// Use test database configuration
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			t.Skip("Database configuration not available - skipping database tests")
			return nil
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Skipf("Failed to parse database URL: %v", err)
		return nil
	}

	config.MaxConns = 10
	config.MinConns = 2

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Skipf("Failed to connect to test database: %v", err)
		return nil
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Failed to ping test database: %v", err)
		return nil
	}

	return &TestDatabase{
		Pool: pool,
		Cleanup: func() {
			pool.Close()
		},
	}
}

// TestServer wraps the server for testing
type TestServer struct {
	Server   *Server
	Recorder *httptest.ResponseRecorder
	Cleanup  func()
}

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T) *TestServer {
	t.Helper()

	cleanup := setupTestLogger()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Test database not available")
		return nil
	}

	server := &Server{
		db:     testDB.Pool,
		router: gin.New(),
	}
	server.router.SetTrustedProxies(nil)
	server.setupRoutes()

	return &TestServer{
		Server:   server,
		Recorder: httptest.NewRecorder(),
		Cleanup: func() {
			testDB.Cleanup()
			cleanup()
		},
	}
}

// makeHTTPRequest creates and executes an HTTP request
func makeHTTPRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
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

	return req, nil
}

// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedKeys []string) map[string]interface{} {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, recorder.Body.String())
	}

	for _, key := range expectedKeys {
		if _, exists := response[key]; !exists {
			t.Errorf("Expected key '%s' not found in response: %v", key, response)
		}
	}

	return response
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v. Body: %s", err, recorder.Body.String())
	}

	if _, exists := response["error"]; !exists {
		t.Errorf("Expected 'error' field in error response: %v", response)
	}
}

// createTestFunnel creates a funnel for testing
func createTestFunnel(t *testing.T, server *Server, name string) string {
	t.Helper()

	funnelData := map[string]interface{}{
		"name":        name,
		"description": "Test funnel for automated testing",
		"steps": []map[string]interface{}{
			{
				"type":     "form",
				"position": 0,
				"title":    "Contact Info",
				"content":  json.RawMessage(`{"fields":[{"name":"email","type":"email","required":true}]}`),
			},
			{
				"type":     "quiz",
				"position": 1,
				"title":    "Quick Quiz",
				"content":  json.RawMessage(`{"question":"What is your goal?","options":["Option A","Option B"]}`),
			},
		},
	}

	req, _ := makeHTTPRequest("POST", "/api/v1/funnels", funnelData)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("Failed to create test funnel: status %d, body: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &response)
	return response["id"].(string)
}

// createTestLead creates a lead for testing
func createTestLead(t *testing.T, server *Server, funnelID string) (string, string) {
	t.Helper()

	sessionID := uuid.New().String()

	// Get funnel to find slug
	req, _ := makeHTTPRequest("GET", "/api/v1/funnels/"+funnelID, nil)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	var funnel map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &funnel)

	if funnel["slug"] == nil {
		t.Fatalf("No slug in funnel response: %+v", funnel)
	}
	slug := funnel["slug"].(string)

	// First update funnel to active status
	updateData := map[string]interface{}{
		"name":   funnel["name"],
		"status": "active",
	}
	req, _ = makeHTTPRequest("PUT", "/api/v1/funnels/"+funnelID, updateData)
	recorder = httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	// Execute funnel to create lead with proper headers
	req, _ = makeHTTPRequest("GET", fmt.Sprintf("/api/v1/execute/%s?session_id=%s", slug, sessionID), nil)
	req.RemoteAddr = "127.0.0.1:12345"  // Set test IP address
	recorder = httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Failed to create test lead: status %d, body: %s", recorder.Code, recorder.Body.String())
	}

	return sessionID, slug
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, server *Server, funnelID string) {
	t.Helper()

	if funnelID != "" {
		ctx := context.Background()
		// Delete in correct order due to foreign keys
		server.db.Exec(ctx, "DELETE FROM funnel_builder.analytics_events WHERE funnel_id = $1", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.step_responses WHERE lead_id IN (SELECT id FROM funnel_builder.leads WHERE funnel_id = $1)", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.leads WHERE funnel_id = $1", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.funnel_steps WHERE funnel_id = $1", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.funnels WHERE id = $1", funnelID)
	}
}

// waitForCondition polls until condition is met or timeout
func waitForCondition(timeout time.Duration, condition func() bool) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("condition not met within timeout")
}

// assertResponseTime validates response time is within limit
func assertResponseTime(t *testing.T, start time.Time, maxDuration time.Duration, operation string) {
	t.Helper()

	elapsed := time.Since(start)
	if elapsed > maxDuration {
		t.Errorf("%s took %v, expected < %v", operation, elapsed, maxDuration)
	}
}
