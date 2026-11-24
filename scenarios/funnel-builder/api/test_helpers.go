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
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

var testFunnelProjects sync.Map

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

func createTestProject(t *testing.T, server *Server, name string) string {
	t.Helper()

	projectData := map[string]interface{}{
		"name":        name,
		"description": fmt.Sprintf("Project for %s", name),
	}

	req, _ := makeHTTPRequest("POST", "/api/v1/projects", projectData)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("Failed to create test project: status %d, body: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse project response: %v", err)
	}

	projectID, ok := response["id"].(string)
	if !ok || projectID == "" {
		t.Fatalf("Project response missing id: %v", response)
	}

	return projectID
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, server *Server, funnelID string) {
	t.Helper()

	if funnelID != "" {
		ctx := context.Background()

		var projectID string
		if value, ok := testFunnelProjects.LoadAndDelete(funnelID); ok {
			if id, ok := value.(string); ok {
				projectID = id
			}
		}

		if projectID == "" {
			var project sql.NullString
			if err := server.db.QueryRow(ctx, "SELECT project_id FROM funnel_builder.funnels WHERE id = $1", funnelID).Scan(&project); err == nil && project.Valid {
				projectID = project.String
			}
		}

		// Delete in correct order due to foreign keys
		server.db.Exec(ctx, "DELETE FROM funnel_builder.analytics_events WHERE funnel_id = $1", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.step_responses WHERE lead_id IN (SELECT id FROM funnel_builder.leads WHERE funnel_id = $1)", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.leads WHERE funnel_id = $1", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.funnel_steps WHERE funnel_id = $1", funnelID)
		server.db.Exec(ctx, "DELETE FROM funnel_builder.funnels WHERE id = $1", funnelID)

		if projectID != "" {
			server.db.Exec(ctx, "DELETE FROM funnel_builder.projects WHERE id = $1", projectID)
		}
	}
}
