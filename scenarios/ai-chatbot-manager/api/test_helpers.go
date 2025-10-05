package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// Global logger variable
var logger *Logger

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	if logger == nil {
		logger = NewLogger()
	}
	originalLogger := logger
	logger = NewLogger()
	logger.SetOutput(io.Discard) // Silence logs during tests
	return func() { logger = originalLogger }
}

// TestDatabase manages test database connection
type TestDatabase struct {
	db      *Database
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database URL from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = os.Getenv("DATABASE_URL")
	}

	if testDBURL == "" {
		t.Skip("DATABASE_URL not set, skipping database tests")
	}

	cfg := &Config{
		DatabaseURL: testDBURL,
	}

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}

	return &TestDatabase{
		db: db,
		Cleanup: func() {
			db.Close()
		},
	}
}

// TestChatbot provides a pre-configured chatbot for testing
type TestChatbot struct {
	Chatbot *Chatbot
	Cleanup func()
}

// setupTestChatbot creates a test chatbot with sample data
func setupTestChatbot(t *testing.T, db *Database, name string) *TestChatbot {
	chatbot := &Chatbot{
		ID:          uuid.New().String(),
		Name:        name,
		Description: fmt.Sprintf("Test chatbot: %s", name),
		Personality: "friendly and helpful",
		KnowledgeBase: "Test knowledge base",
		ModelConfig: map[string]interface{}{
			"model":       "llama2",
			"temperature": 0.7,
		},
		WidgetConfig: map[string]interface{}{
			"theme": "light",
			"position": "bottom-right",
		},
		EscalationConfig: map[string]interface{}{
			"enabled": true,
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save chatbot to database if provided
	if db != nil {
		err := db.CreateChatbot(chatbot)
		if err != nil {
			t.Fatalf("Failed to create test chatbot: %v", err)
		}
	}

	return &TestChatbot{
		Chatbot: chatbot,
		Cleanup: func() {
			if db != nil {
				db.DeleteChatbot(chatbot.ID)
			}
		},
	}
}

// TestTenant provides a pre-configured tenant for testing
type TestTenant struct {
	Tenant  *Tenant
	Cleanup func()
}

// setupTestTenant creates a test tenant with sample data
func setupTestTenant(t *testing.T, db *Database, name string) *TestTenant {
	tenant := &Tenant{
		ID:                       uuid.New().String(),
		Name:                     name,
		Slug:                     strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Description:              fmt.Sprintf("Test tenant: %s", name),
		Plan:                     "pro",
		MaxChatbots:              10,
		MaxConversationsPerMonth: 10000,
		APIKey:                   uuid.New().String(),
		IsActive:                 true,
		Config:                   map[string]interface{}{},
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Note: Tenant creation not implemented in Database yet
	// This creates an in-memory tenant for testing
	return &TestTenant{
		Tenant: tenant,
		Cleanup: func() {
			// Cleanup will be implemented when database methods are available
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
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader io.Reader

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

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
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

// ChatRequest creates a test chat request
func (g *TestDataGenerator) ChatRequest(message, sessionID string) ChatRequest {
	return ChatRequest{
		Message:   message,
		SessionID: sessionID,
		Context:   map[string]interface{}{},
	}
}

// CreateChatbotRequest creates a test chatbot creation request
func (g *TestDataGenerator) CreateChatbotRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":          name,
		"description":   fmt.Sprintf("Test chatbot: %s", name),
		"personality":   "friendly and helpful",
		"knowledge_base": "Test knowledge base",
		"model_config": map[string]interface{}{
			"model":       "llama2",
			"temperature": 0.7,
		},
		"is_active": true,
	}
}

// CreateTenantRequest creates a test tenant creation request
func (g *TestDataGenerator) CreateTenantRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"slug":        strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		"description": fmt.Sprintf("Test tenant: %s", name),
		"plan":        "pro",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// skipIfOllamaUnavailable skips the test if Ollama is not available
func skipIfOllamaUnavailable(t *testing.T) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		t.Skip("OLLAMA_URL not set, skipping Ollama-dependent tests")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(ollamaURL)
	if err != nil {
		t.Skipf("Ollama not available at %s: %v", ollamaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Ollama returned unexpected status: %d", resp.StatusCode)
	}
}

// assertContains checks if a string contains a substring
func assertContains(t *testing.T, haystack, needle string) {
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected string to contain '%s', got '%s'", needle, haystack)
	}
}

// assertNotContains checks if a string does not contain a substring
func assertNotContains(t *testing.T, haystack, needle string) {
	if strings.Contains(haystack, needle) {
		t.Errorf("Expected string not to contain '%s', got '%s'", needle, haystack)
	}
}
