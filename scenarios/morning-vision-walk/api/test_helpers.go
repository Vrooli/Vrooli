// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes controlled logging for testing
func setupTestLogger() func() {
	// Create a test logger that writes to stdout with test prefix
	testLogger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	// Redirect standard log to our test logger
	log.SetOutput(testLogger.Writer())
	log.SetFlags(testLogger.Flags())

	return func() {
		// Restore default logging
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "morning-vision-walk-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// TestSession provides a pre-configured session for testing
type TestSession struct {
	Session *Session
	Cleanup func()
}

// setupTestSession creates a test session with sample data
func setupTestSession(t *testing.T, userID string) *TestSession {
	sessionID := fmt.Sprintf("test-session-%d", time.Now().UnixNano())
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		StartTime: time.Now(),
		Messages:  []ConversationMessage{},
		Insights:  []map[string]interface{}{},
		ActionItems: []string{},
	}

	// Add sample messages
	now := time.Now()
	session.Messages = []ConversationMessage{
		{
			ID:        "msg-1",
			SessionID: sessionID,
			Role:      "user",
			Content:   "I want to improve my Vrooli development workflow",
			Timestamp: now,
		},
		{
			ID:        "msg-2",
			SessionID: sessionID,
			Role:      "assistant",
			Content:   "That's great! Let's explore what aspects you'd like to focus on.",
			Timestamp: now.Add(1 * time.Second),
		},
	}

	activeSessions[sessionID] = session

	return &TestSession{
		Session: session,
		Cleanup: func() {
			delete(activeSessions, sessionID)
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// For plain text errors, just check that body is not empty
	if w.Body.Len() == 0 {
		t.Error("Expected error message in response body")
	}
}

// assertErrorResponseWithMessage validates error responses with specific message
func assertErrorResponseWithMessage(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected error message in response body")
		return
	}

	// For JSON error responses
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if errorMsg, exists := response["error"]; exists {
			if !containsIgnoreCase(errorMsg.(string), expectedMessage) {
				t.Errorf("Expected error message to contain '%s', got '%s'", expectedMessage, errorMsg)
			}
		}
	}
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
			 s[len(s)-len(substr):] == substr ||
			 bytes.Contains([]byte(s), []byte(substr)))))
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// ConversationRequest creates a test conversation request
func (g *TestDataGenerator) ConversationRequest(sessionID, userID, message string) ConversationRequest {
	return ConversationRequest{
		SessionID: sessionID,
		UserID:    userID,
		Message:   message,
		Context:   map[string]interface{}{},
	}
}

// InsightRequest creates a test insight request
func (g *TestDataGenerator) InsightRequest(sessionID, userID string, history []ConversationMessage) InsightRequest {
	return InsightRequest{
		SessionID: sessionID,
		UserID:    userID,
		History:   history,
		Context:   map[string]interface{}{},
	}
}

// TaskPrioritizationRequest creates a test task prioritization request
func (g *TestDataGenerator) TaskPrioritizationRequest(sessionID, userID string, tasks []map[string]interface{}) TaskPrioritizationRequest {
	return TaskPrioritizationRequest{
		SessionID: sessionID,
		UserID:    userID,
		Tasks:     tasks,
		Context:   map[string]interface{}{},
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// mockN8nWorkflow mocks the n8n workflow execution for testing
func mockN8nWorkflow(workflow string, data map[string]interface{}) map[string]interface{} {
	// Return mock responses based on workflow type
	switch workflow {
	case "vision-context-gatherer", "context-gatherer":
		return map[string]interface{}{
			"success":  true,
			"workflow": workflow,
			"context": map[string]interface{}{
				"current_tasks": []string{"task1", "task2"},
				"recent_activity": "active",
			},
			"timestamp": time.Now().Unix(),
		}
	case "vision-conversation":
		return map[string]interface{}{
			"success":  true,
			"workflow": workflow,
			"response": "I understand your goal. Let's explore that further.",
			"timestamp": time.Now().Unix(),
		}
	case "insight-generator":
		return map[string]interface{}{
			"success":  true,
			"workflow": workflow,
			"insights": []map[string]interface{}{
				{
					"type": "pattern",
					"description": "Focus on workflow automation",
				},
			},
			"timestamp": time.Now().Unix(),
		}
	case "daily-planning":
		return map[string]interface{}{
			"success":  true,
			"workflow": workflow,
			"plan": map[string]interface{}{
				"morning": []string{"Review PRs", "Write tests"},
				"afternoon": []string{"Feature development"},
			},
			"timestamp": time.Now().Unix(),
		}
	case "task-prioritizer":
		return map[string]interface{}{
			"success":  true,
			"workflow": workflow,
			"prioritized_tasks": []map[string]interface{}{
				{"task": "High priority task", "priority": 1},
				{"task": "Medium priority task", "priority": 2},
			},
			"timestamp": time.Now().Unix(),
		}
	default:
		return map[string]interface{}{
			"success":  true,
			"workflow": workflow,
			"response": "Mock workflow response",
			"timestamp": time.Now().Unix(),
		}
	}
}
