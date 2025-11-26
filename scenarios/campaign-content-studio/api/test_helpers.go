// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLoggerHelper provides controlled logging during tests
type TestLoggerHelper struct {
	originalLogger *Logger
	cleanup        func()
}

// setupTestLogger initializes a test logger
func setupTestLogger() func() {
	testLogger := NewLogger()
	testLogger.Logger.SetOutput(io.Discard) // Suppress log output during tests
	return func() {
		// Cleanup if needed
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Service    *CampaignService
	Router     *mux.Router
	Cleanup    func()
}

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use environment variable or fallback to test database
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		// Skip database tests if no test database is available
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
		return nil, func() {}
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Create test schema
	createSchema := `
		CREATE TABLE IF NOT EXISTS campaigns (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			settings JSONB DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS campaign_documents (
			id UUID PRIMARY KEY,
			campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
			filename VARCHAR(255) NOT NULL,
			file_path TEXT NOT NULL,
			content_type VARCHAR(100),
			processed_text TEXT,
			embedding_id VARCHAR(255),
			upload_date TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS generated_content (
			id UUID PRIMARY KEY,
			campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
			content_type VARCHAR(100) NOT NULL,
			prompt TEXT NOT NULL,
			generated_content TEXT NOT NULL,
			used_documents TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`

	if _, err := db.Exec(createSchema); err != nil {
		db.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("TRUNCATE campaigns CASCADE")
		db.Close()
	}

	return db, cleanup
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	db, dbCleanup := setupTestDB(t)

	// Create mock service URLs
	n8nURL := "http://localhost:5678"
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	qdrantURL := "http://localhost:6333"
	minioURL := "http://localhost:9000"

	service := NewCampaignService(db, n8nURL, postgresURL, qdrantURL, minioURL)

	// Create router
	r := mux.NewRouter()
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/campaigns", service.ListCampaigns).Methods("GET")
	r.HandleFunc("/campaigns", service.CreateCampaign).Methods("POST")
	r.HandleFunc("/campaigns/{campaignId}/documents", service.ListDocuments).Methods("GET")
	r.HandleFunc("/campaigns/{campaignId}/search", service.SearchDocuments).Methods("POST")
	r.HandleFunc("/generate", service.GenerateContent).Methods("POST")

	return &TestEnvironment{
		DB:      db,
		Service: service,
		Router:  r,
		Cleanup: dbCleanup,
	}
}

// TestCampaign provides a pre-configured campaign for testing
type TestCampaign struct {
	Campaign *Campaign
	Cleanup  func()
}

// setupTestCampaign creates a test campaign with sample data
func setupTestCampaign(t *testing.T, env *TestEnvironment, name string) *TestCampaign {
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        name,
		Description: fmt.Sprintf("Test campaign: %s", name),
		Settings: map[string]interface{}{
			"target_audience": "test",
			"tone":           "professional",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	settingsJSON, _ := json.Marshal(campaign.Settings)

	_, err := env.DB.Exec(`
		INSERT INTO campaigns (id, name, description, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		campaign.ID, campaign.Name, campaign.Description, settingsJSON,
		campaign.CreatedAt, campaign.UpdatedAt)

	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	return &TestCampaign{
		Campaign: campaign,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM campaigns WHERE id = $1", campaign.ID)
		},
	}
}

// TestDocument provides a pre-configured document for testing
type TestDocument struct {
	Document *Document
	Cleanup  func()
}

// setupTestDocument creates a test document
func setupTestDocument(t *testing.T, env *TestEnvironment, campaignID uuid.UUID, filename string) *TestDocument {
	doc := &Document{
		ID:          uuid.New(),
		CampaignID:  campaignID,
		Filename:    filename,
		FilePath:    fmt.Sprintf("/test/path/%s", filename),
		ContentType: "text/plain",
		UploadDate:  time.Now().UTC(),
	}

	_, err := env.DB.Exec(`
		INSERT INTO campaign_documents (id, campaign_id, filename, file_path, content_type, upload_date)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		doc.ID, doc.CampaignID, doc.Filename, doc.FilePath, doc.ContentType, doc.UploadDate)

	if err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}

	return &TestDocument{
		Document: doc,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM campaign_documents WHERE id = $1", doc.ID)
		},
	}
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
func makeHTTPRequest(router *mux.Router, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if req.Body != nil {
		var bodyBytes []byte

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			var err error
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				log.Printf("Failed to marshal request body: %v", err)
				bodyBytes = []byte{}
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
	router.ServeHTTP(w, httpReq)
	return w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
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

// assertJSONArray validates that response is an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var array []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &array); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
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

	if expectedErrorMessage != "" && !strings.Contains(errorMsg.(string), expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, errorMsg)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CampaignRequest creates a test campaign creation request
func (g *TestDataGenerator) CampaignRequest(name, description string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": description,
		"settings": map[string]interface{}{
			"target_audience": "test",
			"tone":           "professional",
		},
	}
}

// GenerateContentRequest creates a test content generation request
func (g *TestDataGenerator) GenerateContentRequest(campaignID, contentType, prompt string) map[string]interface{} {
	return map[string]interface{}{
		"campaign_id":    campaignID,
		"content_type":   contentType,
		"prompt":         prompt,
		"include_images": false,
	}
}

// SearchRequest creates a test search request
func (g *TestDataGenerator) SearchRequest(query string, limit int) map[string]interface{} {
	return map[string]interface{}{
		"query": query,
		"limit": limit,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
