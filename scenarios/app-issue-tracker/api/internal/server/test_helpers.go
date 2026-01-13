package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"app-issue-tracker-api/internal/agentmanager"
	"app-issue-tracker-api/internal/logging"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	handler := slog.NewTextHandler(io.Discard, nil)
	return logging.WithLogger(slog.New(handler))
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir   string
	IssuesDir string
	Server    *Server
	AgentMock *testAgentManager
	Cleanup   func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	t.Helper()

	tempDir := t.TempDir()
	issuesDir := filepath.Join(tempDir, "issues")
	configDir := filepath.Join(tempDir, "initialization", "configuration")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create configuration directory: %v", err)
	}

	settingsPath := filepath.Join(configDir, "agent-settings.json")
	settingsPayload := []byte(`{
		"agent_manager": {
			"runner_type": "claude-code",
			"max_turns": 10,
			"allowed_tools": "Read,Write",
			"timeout_seconds": 60,
			"skip_permissions": true
		}
	}`)
	if err := os.WriteFile(settingsPath, settingsPayload, 0o644); err != nil {
		t.Fatalf("Failed to write test agent settings: %v", err)
	}

	// Create folder structure
	folders := append(ValidIssueStatuses(), "templates")
	for _, folder := range folders {
		if err := os.MkdirAll(filepath.Join(issuesDir, folder), 0o755); err != nil {
			t.Fatalf("Failed to create test folder %s: %v", folder, err)
		}
	}

	cfg := &Config{
		IssuesDir: issuesDir,
		Port:      "0", // Use dynamic port for testing
		// vrooli:env:optional - tests do not require vector search infrastructure
		QdrantURL:               os.Getenv("QDRANT_URL"),
		ScenarioRoot:            tempDir,
		VrooliRoot:              tempDir, // For tests, use same as ScenarioRoot
		WebsocketAllowedOrigins: []string{"*"},
	}

	agentMock := newTestAgentManager()
	server, _, err := NewServer(cfg, WithAgentManager(agentMock))
	if err != nil {
		t.Fatalf("failed to initialize server: %v", err)
	}

	server.Start()

	return &TestEnvironment{
		TempDir:   tempDir,
		IssuesDir: issuesDir,
		Server:    server,
		AgentMock: agentMock,
		Cleanup: func() {
			server.Stop()
		},
	}
}

type testAgentManager struct {
	mu          sync.Mutex
	run         *domainpb.Run
	events      []*domainpb.RunEvent
	createError error
	waitError   error
}

func newTestAgentManager() *testAgentManager {
	return &testAgentManager{
		run: &domainpb.Run{
			Id:     "run-test",
			Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
			Summary: &domainpb.RunSummary{
				Description:  "Test run completed",
				TokensUsed:   12,
				CostEstimate: 0.01,
			},
		},
	}
}

func (t *testAgentManager) IsAvailable(ctx context.Context) bool {
	return true
}

func (t *testAgentManager) Initialize(ctx context.Context, cfg agentmanager.ProfileConfig) error {
	return nil
}

func (t *testAgentManager) UpdateProfile(ctx context.Context, cfg agentmanager.ProfileConfig) error {
	return nil
}

func (t *testAgentManager) CreateRun(ctx context.Context, req agentmanager.RunRequest, cfg agentmanager.ProfileConfig) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.createError != nil {
		return "", t.createError
	}
	if t.run == nil {
		t.run = &domainpb.Run{Id: "run-test", Status: domainpb.RunStatus_RUN_STATUS_COMPLETE}
	}
	t.run.Id = "run-test"
	t.run.Tag = req.Tag
	return t.run.Id, nil
}

func (t *testAgentManager) WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*domainpb.Run, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.waitError != nil {
		return nil, t.waitError
	}
	return t.run, nil
}

func (t *testAgentManager) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.run, nil
}

func (t *testAgentManager) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.events, nil
}

func (t *testAgentManager) StopRun(ctx context.Context, runID string) error {
	return nil
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
func makeHTTPRequest(handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
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
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
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
	handler(w, httpReq)
	return w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	t.Helper()

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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	t.Helper()

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
		if fallback, ok := response["message"]; ok {
			errorMsg = fallback
		} else {
			t.Error("Expected error field in response")
			return
		}
	}

	if expectedErrorSubstring != "" {
		errorStr := fmt.Sprintf("%v", errorMsg)
		if !strings.Contains(errorStr, expectedErrorSubstring) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorStr)
		}
	}
}

// createTestIssue creates a test issue with minimal required fields
func createTestIssue(id, title, issueType, priority, appID string) *Issue {
	now := time.Now().UTC().Format(time.RFC3339)
	issue := &Issue{
		ID:          id,
		Title:       title,
		Description: fmt.Sprintf("Test issue: %s", title),
		Type:        issueType,
		Priority:    priority,
		Targets: []Target{
			{Type: "scenario", ID: appID},
		},
		Status: "open",
	}
	issue.Reporter.Name = "Test Suite"
	issue.Reporter.Email = "test@vrooli.local"
	issue.Reporter.Timestamp = now
	issue.Metadata.CreatedAt = now
	issue.Metadata.UpdatedAt = now
	issue.Metadata.Extra = make(map[string]string)

	return issue
}

// createTestIssueWithError creates a test issue with error context
func createTestIssueWithError(id, title string, errorMsg, stackTrace string, affectedFiles []string) *Issue {
	issue := createTestIssue(id, title, "bug", "high", "test-app")
	issue.ErrorContext.ErrorMessage = errorMsg
	issue.ErrorContext.StackTrace = stackTrace
	issue.ErrorContext.AffectedFiles = affectedFiles
	issue.ErrorContext.EnvironmentInfo = map[string]string{
		"os":      "linux",
		"version": "test-1.0.0",
	}
	return issue
}

// assertIssueExists checks if an issue exists in the expected folder
func assertIssueExists(t *testing.T, server *Server, issueID, expectedFolder string) *Issue {
	t.Helper()

	issueDir, folder, err := server.findIssueDirectory(issueID)
	if err != nil {
		t.Fatalf("Failed to find issue %s: %v", issueID, err)
	}

	if folder != expectedFolder {
		t.Errorf("Expected issue in folder '%s', found in '%s'", expectedFolder, folder)
	}

	issue, err := server.loadIssueFromDir(issueDir)
	if err != nil {
		t.Fatalf("Failed to load issue %s: %v", issueID, err)
	}

	return issue
}

// assertIssueNotExists checks that an issue does not exist
func assertIssueNotExists(t *testing.T, server *Server, issueID string) {
	t.Helper()

	_, _, err := server.findIssueDirectory(issueID)
	if err == nil {
		t.Errorf("Expected issue %s to not exist, but it was found", issueID)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// CreateIssueRequest generates a create issue request
func (g *TestDataGenerator) CreateIssueRequest(title, description, appID string) map[string]interface{} {
	return map[string]interface{}{
		"title":       title,
		"description": description,
		"app_id":      appID,
		"type":        "bug",
		"priority":    "medium",
	}
}

// UpdateIssueRequest generates an update issue request
func (g *TestDataGenerator) UpdateIssueRequest(fields map[string]interface{}) map[string]interface{} {
	return fields
}

// InvestigateRequest generates an investigation request
func (g *TestDataGenerator) InvestigateRequest(issueID, agentID string, autoResolve bool) map[string]interface{} {
	return map[string]interface{}{
		"issue_id":     issueID,
		"agent_id":     agentID,
		"auto_resolve": autoResolve,
	}
}
