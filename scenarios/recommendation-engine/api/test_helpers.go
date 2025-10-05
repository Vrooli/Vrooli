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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Silence logs during tests
	return func() { log.SetOutput(originalOutput) }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB             *sql.DB
	Service        *RecommendationService
	Router         *gin.Engine
	QdrantConn     *grpc.ClientConn
	TestScenarioID string
	Cleanup        func()
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	gin.SetMode(gin.TestMode)

	// Use test database connection
	db := setupTestDB(t)

	// Create test scenario ID
	testScenarioID := uuid.New().String()

	// Mock Qdrant connection (nil for tests that don't need it)
	var qdrantConn *grpc.ClientConn
	var service *RecommendationService

	// Try to connect to Qdrant if available
	qdrantHost := os.Getenv("QDRANT_HOST")
	qdrantPort := os.Getenv("QDRANT_PORT")
	if qdrantHost != "" && qdrantPort != "" {
		conn, err := grpc.Dial(fmt.Sprintf("%s:%s", qdrantHost, qdrantPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			qdrantConn = conn
		}
	}

	service = NewRecommendationService(db, qdrantConn)

	// Setup router
	router := setupTestRouter(service)

	return &TestEnvironment{
		DB:             db,
		Service:        service,
		Router:         router,
		QdrantConn:     qdrantConn,
		TestScenarioID: testScenarioID,
		Cleanup: func() {
			// Clean up test data
			db.Exec("DELETE FROM user_interactions WHERE user_id IN (SELECT id FROM users WHERE scenario_id = $1)", testScenarioID)
			db.Exec("DELETE FROM users WHERE scenario_id = $1", testScenarioID)
			db.Exec("DELETE FROM items WHERE scenario_id = $1", testScenarioID)
			if qdrantConn != nil {
				qdrantConn.Close()
			}
			db.Close()
		},
	}
}

// setupTestDB creates a test database connection and initializes schema
func setupTestDB(t *testing.T) *sql.DB {
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")

		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			t.Skip("Database configuration not available, skipping database tests")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Database not available: %v", err)
	}

	// Initialize database schema
	initTestSchema(t, db)

	return db
}

// initTestSchema creates necessary database tables for testing
func initTestSchema(t *testing.T, db *sql.DB) {
	// Enable extensions
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)

	// Drop existing tables to ensure clean state for testing
	_, err := db.Exec(`
		DROP TABLE IF EXISTS user_interactions CASCADE;
		DROP TABLE IF EXISTS items CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
	`)
	if err != nil {
		t.Logf("Warning: Failed to drop tables: %v", err)
	}

	// Create tables (simplified version for testing)
	schema := `
	CREATE TABLE users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		scenario_id VARCHAR(255) NOT NULL,
		external_id VARCHAR(255) NOT NULL,
		preferences JSONB DEFAULT '{}',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(scenario_id, external_id)
	);

	CREATE TABLE items (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		scenario_id VARCHAR(255) NOT NULL,
		external_id VARCHAR(255) NOT NULL,
		title VARCHAR(500) NOT NULL,
		description TEXT,
		category VARCHAR(255),
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(scenario_id, external_id)
	);

	CREATE TABLE user_interactions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		interaction_type VARCHAR(50) NOT NULL,
		interaction_value FLOAT DEFAULT 1.0,
		context JSONB DEFAULT '{}',
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
}

// setupTestRouter creates a test Gin router
func setupTestRouter(service *RecommendationService) *gin.Engine {
	router := gin.New()

	v1 := router.Group("/api/v1")
	{
		recommendations := v1.Group("/recommendations")
		{
			recommendations.POST("/ingest", service.IngestHandler)
			recommendations.POST("/get", service.RecommendHandler)
			recommendations.POST("/similar", service.SimilarHandler)
		}
	}

	router.GET("/health", service.HealthHandler)

	return router
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	httpReq, _ := http.NewRequest(req.Method, req.Path, bodyReader)
	httpReq.Header.Set("Content-Type", "application/json")

	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validate func(map[string]interface{}) bool) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if validate == nil {
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	if !validate(response) {
		t.Errorf("Response validation failed: %+v", response)
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); !ok || errorMsg == "" {
		t.Errorf("Expected error message, got: %+v", response)
	}
}

// createTestItem creates a test item in the database
func createTestItem(t *testing.T, env *TestEnvironment, externalID, title, category string) *Item {
	t.Helper()

	item := &Item{
		ScenarioID:  env.TestScenarioID,
		ExternalID:  externalID,
		Title:       title,
		Description: fmt.Sprintf("Test description for %s", title),
		Category:    category,
		Metadata:    map[string]interface{}{"test": true},
	}

	if err := env.Service.CreateItem(item); err != nil {
		t.Fatalf("Failed to create test item: %v", err)
	}

	return item
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, env *TestEnvironment, externalID string) string {
	t.Helper()

	userID := uuid.New().String()
	query := `
		INSERT INTO users (id, scenario_id, external_id, preferences, created_at)
		VALUES ($1, $2, $3, '{}', NOW())
	`
	_, err := env.DB.Exec(query, userID, env.TestScenarioID, externalID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return userID
}

// createTestInteraction creates a test interaction
func createTestInteraction(t *testing.T, env *TestEnvironment, userExternalID, itemExternalID, interactionType string, value float64) {
	t.Helper()

	interaction := &UserInteraction{
		UserID:           userExternalID,
		ItemID:           itemExternalID,
		InteractionType:  interactionType,
		InteractionValue: value,
		Context:          map[string]interface{}{"test": true},
	}

	if err := env.Service.CreateUserInteraction(interaction); err != nil {
		t.Fatalf("Failed to create test interaction: %v", err)
	}
}
