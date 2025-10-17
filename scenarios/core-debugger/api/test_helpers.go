// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Silence logs during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(ioutil.Discard)
		return func() {
			log.SetOutput(os.Stderr)
		}
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	DataDir    string
	OriginalWD string
	OriginalDataDir string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "core-debugger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create data directory structure
	testDataDir := filepath.Join(tempDir, "data")
	dirs := []string{
		filepath.Join(testDataDir, "issues"),
		filepath.Join(testDataDir, "health"),
		filepath.Join(testDataDir, "workarounds"),
		filepath.Join(testDataDir, "patterns"),
		filepath.Join(testDataDir, "logs"),
		filepath.Join(testDataDir, "templates"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create components.json
	componentsFile := filepath.Join(testDataDir, "components.json")
	componentsData := `{
  "components": [
    {
      "id": "cli",
      "name": "Test CLI",
      "description": "Test component",
      "health_check": "echo 'ok'",
      "critical": true,
      "timeout_ms": 1000
    },
    {
      "id": "test-component",
      "name": "Test Component",
      "description": "Non-critical test component",
      "health_check": "exit 0",
      "critical": false,
      "timeout_ms": 500
    }
  ]
}`
	if err := ioutil.WriteFile(componentsFile, []byte(componentsData), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create components.json: %v", err)
	}

	// Create common workarounds file
	workaroundsFile := filepath.Join(testDataDir, "workarounds", "common.json")
	workaroundsData := `{
  "workarounds": []
}`
	if err := ioutil.WriteFile(workaroundsFile, []byte(workaroundsData), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create common.json: %v", err)
	}

	// Save original data directory and set new one
	originalDataDir := dataDir
	dataDir = testDataDir

	// Reload components with test data
	loadComponents()

	return &TestEnvironment{
		TempDir:         tempDir,
		DataDir:         testDataDir,
		OriginalDataDir: originalDataDir,
		Cleanup: func() {
			dataDir = originalDataDir
			loadComponents() // Restore original components
			os.RemoveAll(tempDir)
		},
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    string
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest, handler http.Handler) *httptest.ResponseRecorder {
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, body)

	// Set default headers
	if req.Body != "" && req.Headers["Content-Type"] == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validateFunc func(map[string]interface{}) bool) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got '%s'", contentType)
		return
	}

	if validateFunc != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
			t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
			return
		}

		if !validateFunc(data) {
			t.Errorf("JSON validation failed for response: %s", w.Body.String())
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// createTestIssue creates a test issue file
func createTestIssue(t *testing.T, env *TestEnvironment, issue CoreIssue) string {
	t.Helper()

	issueFile := filepath.Join(env.DataDir, "issues", issue.ID+".json")
	data, err := json.MarshalIndent(issue, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test issue: %v", err)
	}

	if err := ioutil.WriteFile(issueFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test issue: %v", err)
	}

	return issueFile
}

// createTestWorkaround creates a test workaround
func createTestWorkaround(issueID string) Workaround {
	return Workaround{
		ID:          fmt.Sprintf("workaround-%d", time.Now().Unix()),
		IssueID:     issueID,
		Description: "Test workaround",
		Commands:    []string{"echo 'test'"},
		SuccessRate: 0.95,
		CreatedAt:   time.Now(),
		Validated:   true,
	}
}

// parseJSONResponse parses a JSON response into a map
func parseJSONResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return data
}

// assertHealthStatus validates health status response
func assertHealthStatus(t *testing.T, data map[string]interface{}, expectedStatus string) {
	t.Helper()

	status, ok := data["status"].(string)
	if !ok {
		t.Error("Response missing 'status' field")
		return
	}

	if status != expectedStatus {
		t.Errorf("Expected status '%s', got '%s'", expectedStatus, status)
	}

	// Validate other required fields
	if _, ok := data["components"]; !ok {
		t.Error("Response missing 'components' field")
	}
	if _, ok := data["active_issues"]; !ok {
		t.Error("Response missing 'active_issues' field")
	}
	if _, ok := data["last_check"]; !ok {
		t.Error("Response missing 'last_check' field")
	}
}

// countIssueFiles counts issue files in the test environment
func countIssueFiles(env *TestEnvironment) int {
	issuesDir := filepath.Join(env.DataDir, "issues")
	files, err := ioutil.ReadDir(issuesDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			count++
		}
	}
	return count
}

// waitForCondition waits for a condition to be true or times out
func waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// readIssueFile reads an issue from file
func readIssueFile(t *testing.T, env *TestEnvironment, issueID string) *CoreIssue {
	t.Helper()

	issueFile := filepath.Join(env.DataDir, "issues", issueID+".json")
	data, err := ioutil.ReadFile(issueFile)
	if err != nil {
		t.Fatalf("Failed to read issue file: %v", err)
	}

	var issue CoreIssue
	if err := json.Unmarshal(data, &issue); err != nil {
		t.Fatalf("Failed to parse issue: %v", err)
	}

	return &issue
}

// createJSONBody creates a JSON body for requests
func createJSONBody(t *testing.T, data interface{}) string {
	t.Helper()

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	return string(jsonData)
}

// TestIssueBuilder provides a fluent interface for building test issues
type TestIssueBuilder struct {
	issue CoreIssue
}

// NewTestIssue creates a new test issue builder
func NewTestIssue(id, component string) *TestIssueBuilder {
	return &TestIssueBuilder{
		issue: CoreIssue{
			ID:              id,
			Component:       component,
			Severity:        "medium",
			Status:          "active",
			Description:     "Test issue",
			ErrorSignature:  fmt.Sprintf("error-%s", id),
			FirstSeen:       time.Now(),
			LastSeen:        time.Now(),
			OccurrenceCount: 1,
			Workarounds:     []Workaround{},
			FixAttempts:     []FixAttempt{},
			Metadata:        make(map[string]interface{}),
		},
	}
}

func (b *TestIssueBuilder) WithSeverity(severity string) *TestIssueBuilder {
	b.issue.Severity = severity
	return b
}

func (b *TestIssueBuilder) WithStatus(status string) *TestIssueBuilder {
	b.issue.Status = status
	return b
}

func (b *TestIssueBuilder) WithDescription(desc string) *TestIssueBuilder {
	b.issue.Description = desc
	return b
}

func (b *TestIssueBuilder) WithWorkaround(workaround Workaround) *TestIssueBuilder {
	b.issue.Workarounds = append(b.issue.Workarounds, workaround)
	return b
}

func (b *TestIssueBuilder) Build() CoreIssue {
	return b.issue
}
