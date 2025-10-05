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
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing with proper cleanup
func setupTestLogger() func() {
	gin.SetMode(gin.TestMode)
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Suppress logs during tests
	return func() {
		log.SetOutput(originalOutput)
		gin.SetMode(gin.DebugMode)
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection with cleanup
func setupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	// Use environment variables or defaults for test database
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("TEST_DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("TEST_DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("TEST_DB_NAME")
	if dbName == "" {
		dbName = "email_outreach_test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping database tests: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping database tests: cannot connect: %v", err)
		return nil
	}

	// Store original db and replace with test db
	originalDB := db
	db = testDB

	// Setup test schema
	setupTestSchema(t, testDB)

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			cleanupTestData(testDB)
			db = originalDB
			testDB.Close()
		},
	}
}

// setupTestSchema creates necessary database tables for testing
func setupTestSchema(t *testing.T, testDB *sql.DB) {
	t.Helper()

	schema := `
		CREATE TABLE IF NOT EXISTS campaigns (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			template_id TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			send_schedule TIMESTAMP,
			total_recipients INTEGER NOT NULL DEFAULT 0,
			sent_count INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			subject TEXT NOT NULL,
			html_content TEXT NOT NULL,
			text_content TEXT NOT NULL,
			style_category TEXT NOT NULL DEFAULT 'professional',
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS email_recipients (
			id TEXT PRIMARY KEY,
			campaign_id TEXT NOT NULL,
			email TEXT NOT NULL,
			name TEXT,
			pronouns TEXT,
			personalization_level TEXT NOT NULL DEFAULT 'template_only',
			send_status TEXT NOT NULL DEFAULT 'pending'
		);
	`

	if _, err := testDB.Exec(schema); err != nil {
		t.Logf("Warning: Failed to create test schema: %v", err)
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(testDB *sql.DB) {
	tables := []string{"email_recipients", "campaigns", "templates"}
	for _, table := range tables {
		testDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
}

// TestRouter creates a test Gin router with all routes configured
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Health check endpoint
	router.GET("/health", healthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/campaigns", listCampaigns)
		v1.POST("/campaigns", createCampaign)
		v1.GET("/campaigns/:id", getCampaign)
		v1.POST("/campaigns/:id/send", sendCampaign)
		v1.GET("/campaigns/:id/analytics", getCampaignAnalytics)

		v1.POST("/templates/generate", generateTemplate)
		v1.GET("/templates", listTemplates)
	}

	return router
}

// HTTPTestRequest represents an HTTP test request configuration
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Create response recorder
	recorder := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(recorder, httpReq)

	return recorder, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, validateBody func(map[string]interface{}) bool) {
	t.Helper()

	// Check status code
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
		return
	}

	// Parse JSON response
	var responseBody map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Body: %s", err, recorder.Body.String())
		return
	}

	// Validate body if validator provided
	if validateBody != nil && !validateBody(responseBody) {
		t.Errorf("Response body validation failed. Body: %v", responseBody)
	}
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	// Check status code
	if recorder.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
		return
	}

	// Parse error response
	var errorBody map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &errorBody); err != nil {
		t.Errorf("Failed to parse error response: %v. Body: %s", err, recorder.Body.String())
		return
	}

	// Check for error field
	errorMsg, ok := errorBody["error"].(string)
	if !ok {
		t.Errorf("Response missing 'error' field. Body: %v", errorBody)
		return
	}

	// Validate error message content
	if expectedErrorContains != "" {
		if !contains(errorMsg, expectedErrorContains) {
			t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorContains, errorMsg)
		}
	}
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		bytes.Contains([]byte(s), []byte(substr)))
}

// createTestCampaign creates a test campaign in the database
func createTestCampaign(t *testing.T, testDB *sql.DB, name string) *Campaign {
	t.Helper()

	campaign := &Campaign{
		ID:              uuid.New().String(),
		Name:            name,
		Description:     "Test campaign description",
		Status:          "draft",
		TotalRecipients: 0,
		SentCount:       0,
	}

	_, err := testDB.Exec(`
		INSERT INTO campaigns (id, name, description, status, total_recipients, sent_count)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, campaign.ID, campaign.Name, campaign.Description, campaign.Status,
		campaign.TotalRecipients, campaign.SentCount)

	if err != nil {
		t.Fatalf("Failed to create test campaign: %v", err)
	}

	return campaign
}

// createTestTemplate creates a test template in the database
func createTestTemplate(t *testing.T, testDB *sql.DB, name string, tone string) *Template {
	t.Helper()

	if tone == "" {
		tone = "professional"
	}

	template := &Template{
		ID:            uuid.New().String(),
		Name:          name,
		Subject:       "Test Subject: " + name,
		HTMLContent:   "<html><body><h1>Test</h1></body></html>",
		TextContent:   "Test email content",
		StyleCategory: tone,
	}

	_, err := testDB.Exec(`
		INSERT INTO templates (id, name, subject, html_content, text_content, style_category)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, template.ID, template.Name, template.Subject, template.HTMLContent,
		template.TextContent, template.StyleCategory)

	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	return template
}

// assertCampaignExists verifies a campaign exists in the database
func assertCampaignExists(t *testing.T, testDB *sql.DB, campaignID string) bool {
	t.Helper()

	var exists bool
	err := testDB.QueryRow("SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1)", campaignID).Scan(&exists)
	if err != nil {
		t.Errorf("Failed to check campaign existence: %v", err)
		return false
	}

	return exists
}

// assertTemplateExists verifies a template exists in the database
func assertTemplateExists(t *testing.T, testDB *sql.DB, templateID string) bool {
	t.Helper()

	var exists bool
	err := testDB.QueryRow("SELECT EXISTS(SELECT 1 FROM templates WHERE id = $1)", templateID).Scan(&exists)
	if err != nil {
		t.Errorf("Failed to check template existence: %v", err)
		return false
	}

	return exists
}
