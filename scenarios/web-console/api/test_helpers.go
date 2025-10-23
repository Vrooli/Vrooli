package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestLogger provides a no-op for web-console (uses slog)
func setupTestLogger() func() {
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir := t.TempDir()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			// Cleanup is automatic with t.TempDir()
		},
	}
}

// setupTestConfig creates a test configuration
func setupTestConfig(tempDir string) config {
	return config{
		addr:                "127.0.0.1:0",
		defaultCommand:      "/bin/sleep",
		defaultArgs:         []string{"60"},
		sessionTTL:          30 * time.Minute,
		idleTimeout:         0,
		storagePath:         filepath.Join(tempDir, "sessions"),
		enableProxyGuard:    false, // Disabled for testing
		maxConcurrent:       4,
		panicKillGrace:      3 * time.Second,
		readBufferSizeBytes: 4096,
		defaultTTYRows:      32,
		defaultTTYCols:      120,
		defaultWorkingDir:   tempDir,
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) *http.Request {
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
				panic(fmt.Errorf("failed to marshal request body: %v", err))
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
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

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	return httpReq
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" && w.Body.Len() > 0 {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	if w.Body.Len() == 0 {
		t.Error("Empty response body")
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	message, exists := response["message"]
	if !exists {
		t.Error("Expected 'message' field in error response")
		return
	}

	messageStr, ok := message.(string)
	if !ok {
		t.Error("Expected 'message' to be a string")
		return
	}

	if expectedMessage != "" && messageStr != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, messageStr)
	}
}

// setupTestWorkspace creates a test workspace
func setupTestWorkspace(t *testing.T, tempDir string) *workspace {
	storagePath := filepath.Join(tempDir, "workspace.json")
	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}
	return ws
}

// setupTestSessionManager creates a test session manager
func setupTestSessionManager(t *testing.T, cfg config) (*sessionManager, *metricsRegistry, *workspace) {
	metrics := newMetricsRegistry()
	ws := setupTestWorkspace(t, cfg.storagePath)
	manager := newSessionManager(cfg, metrics, ws)
	manager.updateIdleTimeout(ws.idleTimeoutDuration())
	return manager, metrics, ws
}

// createTestSession creates a test session for testing
func createTestSession(t *testing.T, manager *sessionManager, req createSessionRequest) *session {
	s, err := manager.createSession(req)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}
	return s
}

// waitForSessionOutput waits for session output with timeout
func waitForSessionOutput(t *testing.T, session *session, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		hasOutput := session.replayBuffer != nil && session.replayBuffer.len() > 0
		if hasOutput {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// assertSessionState validates session state
func assertSessionState(t *testing.T, session *session, expectedCommand string) {
	if session.commandName != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, session.commandName)
	}
	if session.id == "" {
		t.Error("Session ID should not be empty")
	}
	if session.createdAt.IsZero() {
		t.Error("Session createdAt should not be zero")
	}
	if session.expiresAt.IsZero() {
		t.Error("Session expiresAt should not be zero")
	}
	if session.transcript == nil {
		t.Error("Session transcript manager should be initialized")
	}
}

// cleanupSession ensures session is properly cleaned up
func cleanupSession(session *session) {
	if session != nil {
		session.Close(reasonClientRequested)
		time.Sleep(50 * time.Millisecond) // Allow cleanup to complete
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateSessionRequest creates a test session creation request
func (g *TestDataGenerator) CreateSessionRequest(command string, args []string) createSessionRequest {
	return createSessionRequest{
		Operator: "test-operator",
		Reason:   "test",
		Command:  command,
		Args:     args,
		Metadata: json.RawMessage(`{"test": true}`),
	}
}

// CreateTabRequest creates a test tab creation request
func (g *TestDataGenerator) CreateTabRequest(id, label, colorID string) tabMeta {
	return tabMeta{
		ID:      id,
		Label:   label,
		ColorID: colorID,
		Order:   0,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// assertWorkspaceState validates workspace state
func assertWorkspaceState(t *testing.T, ws *workspace, expectedTabCount int) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	if len(ws.Tabs) != expectedTabCount {
		t.Errorf("Expected %d tabs, got %d", expectedTabCount, len(ws.Tabs))
	}
}

// addProxyHeaders adds required proxy headers to request
func addProxyHeaders(req *http.Request) {
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.Header.Set("X-Forwarded-Proto", "https")
}
