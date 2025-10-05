package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Suppress log output during tests unless explicitly needed
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Config     Config
	OriginalDB *sql.DB
	Cleanup    func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *TestEnvironment {
	// Store original db
	originalDB := db

	// Create test database configuration
	testConfig := Config{
		Port:        "9999",
		ChatPort:    "9998",
		PostgresURL: os.Getenv("POSTGRES_URL"),
		QdrantURL:   "http://localhost:6333",
		OllamaURL:   "http://localhost:11434",
		N8NBaseURL:  "http://localhost:5678",
		MinioURL:    "http://localhost:9000",
	}

	if testConfig.PostgresURL == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	// Create test database connection
	testDB, err := sql.Open("postgres", testConfig.PostgresURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Set as global db for handlers
	db = testDB
	config = testConfig

	return &TestEnvironment{
		DB:         testDB,
		Config:     testConfig,
		OriginalDB: originalDB,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM conversations WHERE persona_id LIKE 'test-%'")
			testDB.Exec("DELETE FROM api_tokens WHERE persona_id LIKE 'test-%'")
			testDB.Exec("DELETE FROM training_jobs WHERE persona_id LIKE 'test-%'")
			testDB.Exec("DELETE FROM data_sources WHERE persona_id LIKE 'test-%'")
			testDB.Exec("DELETE FROM personas WHERE id LIKE 'test-%'")

			testDB.Close()
			db = originalDB
		},
	}
}

// TestPersona provides a pre-configured persona for testing
type TestPersona struct {
	Persona *Persona
	Cleanup func()
}

// setupTestPersona creates a test persona with sample data
func setupTestPersona(t *testing.T, name string) *TestPersona {
	personaID := "test-" + uuid.New().String()

	personalityTraits := map[string]interface{}{
		"openness":          0.8,
		"conscientiousness": 0.7,
		"extraversion":      0.6,
	}
	personalityTraitsJSON, _ := json.Marshal(personalityTraits)

	query := `
		INSERT INTO personas (id, name, description, personality_traits, created_at, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, last_updated`

	description := fmt.Sprintf("Test persona: %s", name)
	now := time.Now()

	var createdAt, lastUpdated time.Time
	err := db.QueryRow(query, personaID, name, description, personalityTraitsJSON, now, now).
		Scan(&createdAt, &lastUpdated)

	if err != nil {
		t.Fatalf("Failed to create test persona: %v", err)
	}

	persona := &Persona{
		ID:                personaID,
		Name:              name,
		Description:       description,
		PersonalityTraits: personalityTraits,
		CreatedAt:         createdAt,
		LastUpdated:       lastUpdated,
	}

	return &TestPersona{
		Persona: persona,
		Cleanup: func() {
			db.Exec("DELETE FROM personas WHERE id = $1", personaID)
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
	Headers     map[string]string
}

// makeGinRequest creates and executes a Gin test request
func makeGinRequest(c *gin.Context, req HTTPTestRequest, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	var bodyBytes []byte

	if req.Body != nil {
		var err error
		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
			}
		}
	}

	w := httptest.NewRecorder()
	ginReq := httptest.NewRequest(req.Method, req.Path, bytes.NewReader(bodyBytes))

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			ginReq.Header.Set(key, value)
		}
	}

	// Set default content type for requests with body
	if req.Body != nil && ginReq.Header.Get("Content-Type") == "" {
		ginReq.Header.Set("Content-Type", "application/json")
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := ginReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		ginReq.URL.RawQuery = q.Encode()
	}

	c.Request = ginReq
	handler(c)

	return w
}

// setupGinContext creates a test Gin context
func setupGinContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
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

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorSubstring != "" {
		errorStr := fmt.Sprintf("%v", errorMsg)
		if errorStr != expectedErrorSubstring {
			t.Errorf("Expected error message to be '%s', got '%s'", expectedErrorSubstring, errorStr)
		}
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreatePersonaRequest creates a test persona creation request
func (g *TestDataGenerator) CreatePersonaRequest(name string) CreatePersonaRequest {
	return CreatePersonaRequest{
		Name:        name,
		Description: fmt.Sprintf("Test description for %s", name),
		PersonalityTraits: map[string]interface{}{
			"creativity": 0.9,
			"analytical": 0.8,
		},
	}
}

// ConnectDataSourceRequest creates a test data source request
func (g *TestDataGenerator) ConnectDataSourceRequest(personaID, sourceType string) ConnectDataSourceRequest {
	return ConnectDataSourceRequest{
		PersonaID:  personaID,
		SourceType: sourceType,
		SourceConfig: map[string]interface{}{
			"path": "/test/data",
		},
	}
}

// StartTrainingRequest creates a test training request
func (g *TestDataGenerator) StartTrainingRequest(personaID string) StartTrainingRequest {
	return StartTrainingRequest{
		PersonaID: personaID,
		Model:     "llama2",
		Technique: "fine-tuning",
	}
}

// CreateTokenRequest creates a test token request
func (g *TestDataGenerator) CreateTokenRequest(personaID string) CreateTokenRequest {
	return CreateTokenRequest{
		PersonaID:   personaID,
		Name:        "test-token",
		Permissions: []string{"read", "write"},
	}
}

// ChatRequest creates a test chat request
func (g *TestDataGenerator) ChatRequest(personaID, message string) ChatRequest {
	return ChatRequest{
		PersonaID: personaID,
		Message:   message,
		SessionID: uuid.New().String(),
	}
}

// SearchRequest creates a test search request
func (g *TestDataGenerator) SearchRequest(personaID, query string) SearchRequest {
	return SearchRequest{
		PersonaID: personaID,
		Query:     query,
		Limit:     10,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
