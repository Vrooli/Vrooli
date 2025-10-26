package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"app-monitor-api/repository"
)

// =============================================================================
// ReportAppIssue Tests
// =============================================================================

func TestReportAppIssue_RequiredFields(t *testing.T) {
	service := NewAppService(nil)

	testCases := []struct {
		name        string
		req         *IssueReportRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request payload is required",
		},
		{
			name: "missing app ID",
			req: &IssueReportRequest{
				AppID:   "",
				Message: "test message",
			},
			expectedErr: "app identifier is required",
		},
		{
			name: "whitespace app ID",
			req: &IssueReportRequest{
				AppID:   "   ",
				Message: "test message",
			},
			expectedErr: "app identifier is required",
		},
		{
			name: "missing message",
			req: &IssueReportRequest{
				AppID:   "test-app",
				Message: "",
			},
			expectedErr: "issue message is required",
		},
		{
			name: "whitespace message",
			req: &IssueReportRequest{
				AppID:   "test-app",
				Message: "   ",
			},
			expectedErr: "issue message is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.ReportAppIssue(context.Background(), tc.req)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("Expected error containing %q, got %q", tc.expectedErr, err.Error())
			}
		})
	}
}

func TestReportAppIssue_InvalidScreenshot(t *testing.T) {
	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{ID: "test-app", Name: "Test App", ScenarioName: "test-scenario"},
		},
	}

	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "message": "created", "data": {"issue_id": "issue-123"}}`)),
			}, nil
		},
	}

	service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)
	invalidBase64 := stringPtr("this is not valid base64!!!")

	req := &IssueReportRequest{
		AppID:          "test-app",
		Message:        "test message",
		ScreenshotData: invalidBase64,
	}

	// Should not error - invalid screenshot should be silently ignored
	result, err := service.ReportAppIssue(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestReportAppIssue_ValidScreenshot(t *testing.T) {
	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{ID: "test-app", Name: "Test App", ScenarioName: "test-scenario"},
		},
	}

	requestReceived := false
	var receivedPayload map[string]interface{}

	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			requestReceived = true
			bodyBytes, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(bodyBytes, &receivedPayload)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "message": "created", "data": {"issue_id": "issue-123"}}`)),
			}, nil
		},
	}

	service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

	// Create valid base64 encoded PNG
	validBase64 := base64.StdEncoding.EncodeToString([]byte("fake png data"))

	req := &IssueReportRequest{
		AppID:          "test-app",
		Message:        "test message",
		ScreenshotData: &validBase64,
	}

	result, err := service.ReportAppIssue(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !requestReceived {
		t.Fatal("HTTP request was not made")
	}

	// Verify screenshot was included in artifacts
	if artifacts, ok := receivedPayload["artifacts"].([]interface{}); ok {
		found := false
		for _, artifact := range artifacts {
			if artifactMap, ok := artifact.(map[string]interface{}); ok {
				if artifactMap["name"] == attachmentScreenshotName {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("Screenshot artifact was not included in payload")
		}
	}
}

func TestReportAppIssue_FullWorkflow(t *testing.T) {
	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{
				ID:           "test-app",
				Name:         "Test App",
				ScenarioName: "test-scenario",
			},
		},
	}

	requestReceived := false
	var receivedPayload map[string]interface{}

	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			requestReceived = true

			// Verify request structure
			if req.Method != http.MethodPost {
				t.Errorf("Expected POST, got %s", req.Method)
			}

			bodyBytes, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(bodyBytes, &receivedPayload)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "message": "Issue created", "data": {"issue_id": "ISSUE-456"}}`)),
			}, nil
		},
	}

	service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

	req := &IssueReportRequest{
		AppID:   "test-app",
		Message: "This is a test issue report with multiple sections.",
		Logs: []string{
			"[INFO] Application started",
			"[WARN] Slow query detected",
			"[ERROR] Connection timeout",
		},
		LogsTotal: intPtr(150),
	}

	result, err := service.ReportAppIssue(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.IssueID != "ISSUE-456" {
		t.Errorf("Expected issue ID 'ISSUE-456', got %q", result.IssueID)
	}

	if result.Message != "Issue created" {
		t.Errorf("Expected message 'Issue created', got %q", result.Message)
	}

	if !requestReceived {
		t.Fatal("HTTP request was not made")
	}

	// Verify payload structure
	if title, ok := receivedPayload["title"].(string); !ok || title == "" {
		t.Error("Expected title field to be populated")
	}

	if desc, ok := receivedPayload["description"].(string); !ok || desc == "" {
		t.Error("Expected description field to be populated")
	}

	if tags, ok := receivedPayload["tags"].([]interface{}); !ok || len(tags) == 0 {
		t.Error("Expected tags to be populated")
	}

	// Verify artifacts
	if artifacts, ok := receivedPayload["artifacts"].([]interface{}); ok {
		found := false
		for _, artifact := range artifacts {
			if artifactMap, ok := artifact.(map[string]interface{}); ok {
				if artifactMap["name"] == attachmentLifecycleName {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("Lifecycle logs artifact was not included")
		}
	}
}

func TestReportAppIssue_LogsTruncation(t *testing.T) {
	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{ID: "test-app", Name: "Test App", ScenarioName: "test-scenario"},
		},
	}

	var receivedPayload map[string]interface{}

	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			bodyBytes, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(bodyBytes, &receivedPayload)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "data": {"issue_id": "test"}}`)),
			}, nil
		},
	}

	service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

	// Create more logs than the max (300)
	logs := make([]string, 500)
	for i := 0; i < 500; i++ {
		logs[i] = "Log line"
	}

	req := &IssueReportRequest{
		AppID:   "test-app",
		Message: "test",
		Logs:    logs,
	}

	_, err := service.ReportAppIssue(context.Background(), req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify logs were truncated
	if artifacts, ok := receivedPayload["artifacts"].([]interface{}); ok {
		for _, artifact := range artifacts {
			if artifactMap, ok := artifact.(map[string]interface{}); ok {
				if artifactMap["name"] == attachmentLifecycleName {
					content := artifactMap["content"].(string)
					lines := strings.Split(content, "\n")
					if len(lines) > 300 {
						t.Errorf("Expected max 300 log lines, got %d", len(lines))
					}
				}
			}
		}
	}
}

// =============================================================================
// Sanitization Tests
// =============================================================================

func TestSanitizeConsoleLogs(t *testing.T) {
	testCases := []struct {
		name     string
		input    []IssueConsoleLogEntry
		maxItems int
		maxText  int
		expected int
	}{
		{
			name:     "empty input",
			input:    []IssueConsoleLogEntry{},
			maxItems: 10,
			maxText:  100,
			expected: 0,
		},
		{
			name: "normalize log levels",
			input: []IssueConsoleLogEntry{
				{Level: "LOG", Text: "test"},
				{Level: "info", Text: "test"},
				{Level: "WARN", Text: "test"},
				{Level: "invalid_level", Text: "test"},
			},
			maxItems: 10,
			maxText:  100,
			expected: 4,
		},
		{
			name: "truncate entries",
			input: []IssueConsoleLogEntry{
				{Level: "log", Text: "1"},
				{Level: "log", Text: "2"},
				{Level: "log", Text: "3"},
				{Level: "log", Text: "4"},
				{Level: "log", Text: "5"},
			},
			maxItems: 3,
			maxText:  100,
			expected: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeConsoleLogs(tc.input, tc.maxItems, tc.maxText)
			if len(result) != tc.expected {
				t.Errorf("Expected %d entries, got %d", tc.expected, len(result))
			}

			// Verify all levels are normalized
			for _, entry := range result {
				switch entry.Level {
				case "log", "info", "warn", "error", "debug":
					// valid
				default:
					t.Errorf("Invalid log level: %s", entry.Level)
				}
			}
		})
	}
}

func TestSanitizeNetworkRequests(t *testing.T) {
	testCases := []struct {
		name     string
		input    []IssueNetworkEntry
		maxItems int
		expected int
	}{
		{
			name:     "empty input",
			input:    []IssueNetworkEntry{},
			maxItems: 10,
			expected: 0,
		},
		{
			name: "normalize methods",
			input: []IssueNetworkEntry{
				{Method: "get", URL: "http://test.com"},
				{Method: "POST", URL: "http://test.com"},
				{Method: "", URL: "http://test.com"}, // Should default to GET
			},
			maxItems: 10,
			expected: 3,
		},
		{
			name: "truncate entries",
			input: []IssueNetworkEntry{
				{Method: "GET", URL: "http://1.com"},
				{Method: "GET", URL: "http://2.com"},
				{Method: "GET", URL: "http://3.com"},
			},
			maxItems: 2,
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeNetworkRequests(tc.input, tc.maxItems, 2048, 1500, 128)
			if len(result) != tc.expected {
				t.Errorf("Expected %d entries, got %d", tc.expected, len(result))
			}

			// Verify all methods are uppercase
			for _, entry := range result {
				if entry.Method != strings.ToUpper(entry.Method) {
					t.Errorf("Method should be uppercase: %s", entry.Method)
				}
			}
		})
	}
}

func TestSanitizeHealthChecks(t *testing.T) {
	testCases := []struct {
		name     string
		input    []IssueHealthCheckEntry
		expected int
	}{
		{
			name:     "empty input",
			input:    []IssueHealthCheckEntry{},
			expected: 0,
		},
		{
			name: "normalize status",
			input: []IssueHealthCheckEntry{
				{Status: "pass", Name: "Test 1"},
				{Status: "ok", Name: "Test 2"},
				{Status: "healthy", Name: "Test 3"},
				{Status: "fail", Name: "Test 4"},
				{Status: "error", Name: "Test 5"},
				{Status: "invalid", Name: "Test 6"}, // Should default to warn
			},
			expected: 6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeHealthChecks(tc.input, 20, 120, 512, 400, 120, 4000)
			if len(result) != tc.expected {
				t.Errorf("Expected %d entries, got %d", tc.expected, len(result))
			}

			// Verify all statuses are normalized
			for _, entry := range result {
				switch entry.Status {
				case "pass", "warn", "fail":
					// valid
				default:
					t.Errorf("Invalid status: %s", entry.Status)
				}
			}
		})
	}
}

func TestSanitizeIssueCaptures(t *testing.T) {
	validBase64 := base64.StdEncoding.EncodeToString([]byte("test data"))
	invalidBase64 := "not valid base64!!!"

	testCases := []struct {
		name     string
		input    []IssueCapture
		expected int
	}{
		{
			name:     "empty input",
			input:    []IssueCapture{},
			expected: 0,
		},
		{
			name: "filter invalid base64",
			input: []IssueCapture{
				{Data: validBase64, Type: "element"},
				{Data: invalidBase64, Type: "element"}, // Should be filtered out
				{Data: validBase64, Type: "page"},
			},
			expected: 2,
		},
		{
			name: "filter empty data",
			input: []IssueCapture{
				{Data: "", Type: "element"}, // Should be filtered out
				{Data: validBase64, Type: "element"},
			},
			expected: 1,
		},
		{
			name: "deduplicate IDs",
			input: []IssueCapture{
				{ID: "same-id", Data: validBase64, Type: "element"},
				{ID: "same-id", Data: validBase64, Type: "element"}, // Should get unique ID
			},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeIssueCaptures(tc.input, 12, 600, 160, 900)
			if len(result) != tc.expected {
				t.Errorf("Expected %d entries, got %d", tc.expected, len(result))
			}
		})
	}
}

// =============================================================================
// Title Generation Tests
// =============================================================================

func TestDeriveIssueTitle(t *testing.T) {
	testCases := []struct {
		name                   string
		primaryDescription     string
		message                string
		captures               []IssueCapture
		includeDiagnostics     bool
		includeScreenshot      bool
		expectedSubstring      string
	}{
		{
			name:               "primary description only",
			primaryDescription: "Button doesn't work\nAdditional details here",
			message:            "",
			captures:           nil,
			includeDiagnostics: false,
			includeScreenshot:  false,
			expectedSubstring:  "Button doesn't work",
		},
		{
			name:               "diagnostics with bridge keyword",
			primaryDescription: "",
			message:            "### Diagnostics Summary\n- bridge failure detected",
			captures:           nil,
			includeDiagnostics: true,
			includeScreenshot:  false,
			expectedSubstring:  "iframe bridge",
		},
		{
			name:               "diagnostics with localhost keyword",
			primaryDescription: "",
			message:            "localhost proxy issues detected",
			captures:           nil,
			includeDiagnostics: true,
			includeScreenshot:  false,
			expectedSubstring:  "localhost proxy",
		},
		{
			name:               "screenshot fallback",
			primaryDescription: "",
			message:            "Generic message",
			captures:           nil,
			includeDiagnostics: false,
			includeScreenshot:  true,
			expectedSubstring:  "Screenshot feedback",
		},
		{
			name:               "default fallback",
			primaryDescription: "",
			message:            "",
			captures:           nil,
			includeDiagnostics: false,
			includeScreenshot:  false,
			expectedSubstring:  "App Monitor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := deriveIssueTitle(tc.primaryDescription, tc.message, tc.captures, tc.includeDiagnostics, tc.includeScreenshot)
			if !strings.Contains(result, tc.expectedSubstring) {
				t.Errorf("Expected title to contain %q, got %q", tc.expectedSubstring, result)
			}
		})
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestCountConsoleErrors(t *testing.T) {
	logs := []IssueConsoleLogEntry{
		{Level: "log", Text: "Normal log"},
		{Level: "error", Text: "Error 1"},
		{Level: "ERROR", Text: "Error 2"}, // Case insensitive
		{Level: "warn", Text: "Warning"},
		{Level: "error", Text: "Error 3"},
	}

	result := countConsoleErrors(logs)
	if result != 3 {
		t.Errorf("Expected 3 errors, got %d", result)
	}
}

func TestCountFailedRequests(t *testing.T) {
	requests := []IssueNetworkEntry{
		{OK: boolPtr(true), Status: intPtr(200)},
		{OK: boolPtr(false), Status: intPtr(500)},
		{OK: boolPtr(true), Status: intPtr(404)}, // Status 404 should count as failed
		{OK: boolPtr(true), Status: intPtr(300)},
	}

	result := countFailedRequests(requests)
	if result != 2 {
		t.Errorf("Expected 2 failed requests, got %d", result)
	}
}

func TestCountPassingHealthChecks(t *testing.T) {
	checks := []IssueHealthCheckEntry{
		{Status: "pass"},
		{Status: "Pass"}, // Case insensitive
		{Status: "fail"},
		{Status: "warn"},
		{Status: "pass"},
	}

	result := countPassingHealthChecks(checks)
	if result != 3 {
		t.Errorf("Expected 3 passing checks, got %d", result)
	}
}

func TestNormalizePreviewURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid http URL",
			input:    "http://localhost:8080/path",
			expected: "http://localhost:8080/path",
		},
		{
			name:     "valid https URL",
			input:    "https://example.com/test",
			expected: "https://example.com/test",
		},
		{
			name:     "invalid scheme",
			input:    "ftp://example.com",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid URL",
			input:    "not a url",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizePreviewURL(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestExtractPreviewPath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path",
			input:    "http://localhost:8080/dashboard",
			expected: "/dashboard",
		},
		{
			name:     "path with query",
			input:    "http://localhost:8080/search?q=test",
			expected: "/search?q=test",
		},
		{
			name:     "path with fragment",
			input:    "http://localhost:8080/page#section",
			expected: "/page#section",
		},
		{
			name:     "root path",
			input:    "http://localhost:8080/",
			expected: "",
		},
		{
			name:     "proxy path extraction",
			input:    "http://localhost:8080/apps/test-app/proxy/dashboard",
			expected: "/dashboard",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractPreviewPath(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestSanitizeScenarioIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already clean",
			input:    "test-scenario",
			expected: "test-scenario",
		},
		{
			name:     "uppercase to lowercase",
			input:    "Test-Scenario",
			expected: "test-scenario",
		},
		{
			name:     "spaces to hyphens",
			input:    "test scenario",
			expected: "test-scenario",
		},
		{
			name:     "special characters removed",
			input:    "test@scenario!",
			expected: "test-scenario",
		},
		{
			name:     "multiple hyphens collapsed",
			input:    "test---scenario",
			expected: "test-scenario",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "scenario",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeScenarioIdentifier(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{
			name:     "shorter than limit",
			input:    "short",
			limit:    10,
			expected: "short",
		},
		{
			name:     "exactly at limit",
			input:    "exactly",
			limit:    7,
			expected: "exactly",
		},
		{
			name:     "longer than limit",
			input:    "this is a very long title that needs truncation",
			limit:    20,
			expected: "this is a very long ...",
		},
		{
			name:     "empty string",
			input:    "",
			limit:    10,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateTitle(tc.input, tc.limit)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "single line",
			expected: "single line",
		},
		{
			name:     "multiple lines with newline",
			input:    "first line\nsecond line",
			expected: "first line",
		},
		{
			name:     "multiple lines with carriage return",
			input:    "first line\rsecond line",
			expected: "first line",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace preserved",
			input:    "  first line  \nsecond",
			expected: "first line",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := firstLine(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestRoundFloat(t *testing.T) {
	testCases := []struct {
		name      string
		value     float64
		precision int
		expected  float64
	}{
		{
			name:      "round to 2 decimals",
			value:     3.14159,
			precision: 2,
			expected:  3.14,
		},
		{
			name:      "round to 0 decimals",
			value:     3.7,
			precision: 0,
			expected:  4.0,
		},
		{
			name:      "negative precision",
			value:     3.7,
			precision: -1,
			expected:  4.0,
		},
		{
			name:      "already rounded",
			value:     3.00,
			precision: 2,
			expected:  3.00,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := roundFloat(tc.value, tc.precision)
			if result != tc.expected {
				t.Errorf("Expected %.2f, got %.2f", tc.expected, result)
			}
		})
	}
}

func TestClampCaptureDimension(t *testing.T) {
	testCases := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "normal value",
			value:    1920,
			expected: 1920,
		},
		{
			name:     "negative value",
			value:    -100,
			expected: 0,
		},
		{
			name:     "exceeds maximum",
			value:    25000,
			expected: 20000,
		},
		{
			name:     "zero",
			value:    0,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := clampCaptureDimension(tc.value)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestSanitizeCaptureTimestamp(t *testing.T) {
	validTimestamp := time.Now().UTC().Format(time.RFC3339)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid RFC3339",
			input:    validTimestamp,
			expected: validTimestamp,
		},
		{
			name:     "invalid format",
			input:    "2024-01-01 12:00:00",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "garbage",
			input:    "not a timestamp",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeCaptureTimestamp(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// =============================================================================
// Integration-level tests
// =============================================================================

func TestReportAppIssue_IssueTrackerUnavailable(t *testing.T) {
	mockRepo := &mockAppRepository{
		apps: []repository.App{
			{ID: "test-app", Name: "Test App", ScenarioName: "test-scenario"},
		},
	}

	// Mock issue tracker being unavailable
	mockHTTP := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// First call: locating issue tracker (orchestrator) - simulate not found
			if strings.Contains(req.URL.String(), "orchestrator") {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"success": true, "scenarios": []}`)),
				}, nil
			}
			// Should never reach here because we can't locate the tracker
			t.Error("Should not attempt to submit issue when tracker is unavailable")
			return nil, nil
		},
	}

	service := NewAppServiceWithOptions(mockRepo, mockHTTP, nil)

	req := &IssueReportRequest{
		AppID:   "test-app",
		Message: "test message",
	}

	_, err := service.ReportAppIssue(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error when issue tracker is unavailable")
	}
}
