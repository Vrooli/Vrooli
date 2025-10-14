package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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

// TestDB provides a test database connection
type TestDB struct {
	db      *sql.DB
	cleanup func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *TestDB {
	// TEST_DATABASE_URL is required for database tests
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping test: TEST_DATABASE_URL not set")
		return nil
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Skipping test: database connection failed: %v", err)
		return nil
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping test: database ping failed: %v", err)
		return nil
	}

	// Create test tables if they don't exist
	initTestSchema(t, db)

	return &TestDB{
		db: db,
		cleanup: func() {
			// Clean up test data
			db.Exec("DELETE FROM processed_emails")
			db.Exec("DELETE FROM triage_rules")
			db.Exec("DELETE FROM email_accounts")
			db.Close()
		},
	}
}

// initTestSchema creates necessary test tables
func initTestSchema(t *testing.T, db *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS email_accounts (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			email_address VARCHAR(255) NOT NULL,
			imap_settings JSONB,
			smtp_settings JSONB,
			sync_enabled BOOLEAN DEFAULT true,
			last_sync TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);

		CREATE TABLE IF NOT EXISTS processed_emails (
			id VARCHAR(36) PRIMARY KEY,
			account_id VARCHAR(36) REFERENCES email_accounts(id),
			message_id VARCHAR(255),
			subject VARCHAR(500),
			sender_email VARCHAR(255),
			full_body TEXT,
			priority_score FLOAT,
			processed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS triage_rules (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			conditions JSONB,
			actions JSONB,
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Logf("Warning: Failed to create test schema: %v", err)
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
	Context     map[string]interface{}
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

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
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

// CreateTestAccount creates a test email account in the database
func (g *TestDataGenerator) CreateTestAccount(t *testing.T, db *sql.DB, userID string) string {
	accountID := uuid.New().String()
	emailAddress := fmt.Sprintf("test-%s@example.com", accountID[:8])

	imapConfig := map[string]interface{}{
		"server":   "imap.example.com",
		"port":     993,
		"username": emailAddress,
		"password": "test-password",
		"use_tls":  true,
	}

	smtpConfig := map[string]interface{}{
		"server":   "smtp.example.com",
		"port":     587,
		"username": emailAddress,
		"password": "test-password",
		"use_tls":  true,
	}

	imapJSON, _ := json.Marshal(imapConfig)
	smtpJSON, _ := json.Marshal(smtpConfig)

	query := `
		INSERT INTO email_accounts (
			id, user_id, email_address, imap_settings, smtp_settings,
			sync_enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(query, accountID, userID, emailAddress, imapJSON, smtpJSON,
		true, time.Now(), time.Now())

	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}

	return accountID
}

// CreateTestEmail creates a test email in the database
func (g *TestDataGenerator) CreateTestEmail(t *testing.T, db *sql.DB, accountID string) string {
	emailID := uuid.New().String()

	query := `
		INSERT INTO processed_emails (
			id, account_id, message_id, subject, sender_email, full_body, priority_score, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := db.Exec(query, emailID, accountID, "msg-"+emailID, "Test Subject", "sender@example.com",
		"Test email body", 0.5, time.Now())

	if err != nil {
		t.Fatalf("Failed to create test email: %v", err)
	}

	return emailID
}

// CreateTestRule creates a test triage rule in the database
func (g *TestDataGenerator) CreateTestRule(t *testing.T, db *sql.DB, userID string) string {
	ruleID := uuid.New().String()

	conditions := map[string]interface{}{
		"sender_contains": []string{"important@example.com"},
	}

	actions := map[string]interface{}{
		"priority": "high",
		"tag":      "important",
	}

	conditionsJSON, _ := json.Marshal(conditions)
	actionsJSON, _ := json.Marshal(actions)

	query := `
		INSERT INTO triage_rules (
			id, user_id, name, description, conditions, actions, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := db.Exec(query, ruleID, userID, "Test Rule", "Test rule description",
		conditionsJSON, actionsJSON, true, time.Now(), time.Now())

	if err != nil {
		t.Fatalf("Failed to create test rule: %v", err)
	}

	return ruleID
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// MockServer provides a mock HTTP server for testing external services
type MockServer struct {
	server   *httptest.Server
	handlers map[string]http.HandlerFunc
}

// NewMockServer creates a new mock server
func NewMockServer() *MockServer {
	ms := &MockServer{
		handlers: make(map[string]http.HandlerFunc),
	}

	ms.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, exists := ms.handlers[r.URL.Path]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	}))

	return ms
}

// SetHandler sets a handler for a specific path
func (ms *MockServer) SetHandler(path string, handler http.HandlerFunc) {
	ms.handlers[path] = handler
}

// URL returns the mock server URL
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Close closes the mock server
func (ms *MockServer) Close() {
	ms.server.Close()
}

// MockAuthServer creates a mock authentication server
func MockAuthServer(t *testing.T, validToken string, userID string) *MockServer {
	ms := NewMockServer()

	ms.SetHandler("/api/v1/auth/validate", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "Bearer "+validToken {
			response := map[string]interface{}{
				"valid":      true,
				"user_id":    userID,
				"roles":      []string{"user"},
				"expires_at": time.Now().Add(24 * time.Hour),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		}
	})

	ms.SetHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	return ms
}
