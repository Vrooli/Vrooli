package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ErrorTestPattern defines a systematic error test case
type ErrorTestPattern struct {
	Name           string
	URL            string
	Method         string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidJSON adds a test case for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(url string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Invalid JSON",
		URL:            url,
		Method:         "POST",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid JSON",
	})
	return b
}

// AddMissingRequiredField adds test case for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(url, fieldName string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Missing " + fieldName,
		URL:            url,
		Method:         "POST",
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidBackupType adds test case for invalid backup type
func (b *TestScenarioBuilder) AddInvalidBackupType(url string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:   "Invalid backup type",
		URL:    url,
		Method: "POST",
		Body: map[string]interface{}{
			"type":    "invalid",
			"targets": []string{"postgres"},
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid backup type. Must be 'full', 'incremental', or 'differential'",
	})
	return b
}

// AddEmptyTargets adds test case for empty targets
func (b *TestScenarioBuilder) AddEmptyTargets(url string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:   "Empty targets",
		URL:    url,
		Method: "POST",
		Body: map[string]interface{}{
			"type":    "full",
			"targets": []string{},
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "At least one target must be specified",
	})
	return b
}

// AddMissingRestoreID adds test case for missing restore point or backup job ID
func (b *TestScenarioBuilder) AddMissingRestoreID(url string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:   "Missing restore point ID",
		URL:    url,
		Method: "POST",
		Body: map[string]interface{}{
			"targets": []string{"postgres"},
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Either restore_point_id or backup_job_id must be specified",
	})
	return b
}

// AddCustom adds a custom error test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns all accumulated test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	Router  *http.Handler
	BaseURL string
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(router *http.Handler, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		Router:  router,
		BaseURL: baseURL,
	}
}

// ExecuteErrorPatterns runs all error patterns against the handler
func ExecuteErrorPatterns(t *testing.T, router http.Handler, patterns []ErrorTestPattern) {
	t.Helper()

	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			var req *http.Request
			if bodyStr, ok := pattern.Body.(string); ok {
				// For invalid JSON strings
				req = httptest.NewRequest(pattern.Method, pattern.URL, nil)
				req.Header.Set("Content-Type", "application/json")
				if bodyStr != "" {
					req.Body = http.NoBody
				}
			} else {
				req = makeHTTPRequest(pattern.Method, pattern.URL, pattern.Body)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if pattern.ExpectedError != "" {
				body := rr.Body.String()
				if body != pattern.ExpectedError+"\n" {
					t.Errorf("Expected error %q, got %q", pattern.ExpectedError, body)
				}
			}
		})
	}
}

// TableDrivenTest represents a table-driven test case
type TableDrivenTest struct {
	Name           string
	Input          interface{}
	ExpectedOutput interface{}
	ExpectedError  bool
}

// RunTableDrivenTests executes table-driven tests
func RunTableDrivenTests(t *testing.T, tests []TableDrivenTest, testFunc func(interface{}) (interface{}, error)) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, err := testFunc(tt.Input)

			if tt.ExpectedError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.ExpectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.ExpectedOutput != nil && output != tt.ExpectedOutput {
				t.Errorf("Expected output %v, got %v", tt.ExpectedOutput, output)
			}
		})
	}
}

// BackupTestScenario represents a complete backup test scenario
type BackupTestScenario struct {
	Name           string
	BackupType     string
	Targets        []string
	Description    string
	ExpectSuccess  bool
	ExpectedStatus int
	Validate       func(*testing.T, *httptest.ResponseRecorder)
}

// ExecuteBackupScenarios runs backup test scenarios
func ExecuteBackupScenarios(t *testing.T, router http.Handler, scenarios []BackupTestScenario) {
	t.Helper()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			reqBody := BackupCreateRequest{
				Type:        scenario.BackupType,
				Targets:     scenario.Targets,
				Description: scenario.Description,
			}

			req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if scenario.Validate != nil {
				scenario.Validate(t, rr)
			}
		})
	}
}

// RestoreTestScenario represents a complete restore test scenario
type RestoreTestScenario struct {
	Name           string
	RestorePointID string
	BackupJobID    string
	Targets        []string
	Destination    string
	Verify         bool
	ExpectedStatus int
	Validate       func(*testing.T, *httptest.ResponseRecorder)
}

// ExecuteRestoreScenarios runs restore test scenarios
func ExecuteRestoreScenarios(t *testing.T, router http.Handler, scenarios []RestoreTestScenario) {
	t.Helper()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			reqBody := RestoreCreateRequest{
				RestorePointID:      scenario.RestorePointID,
				BackupJobID:         scenario.BackupJobID,
				Targets:             scenario.Targets,
				Destination:         scenario.Destination,
				VerifyBeforeRestore: scenario.Verify,
			}

			req := makeHTTPRequest("POST", "/api/v1/restore/create", reqBody)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, rr.Code, rr.Body.String())
			}

			if scenario.Validate != nil {
				scenario.Validate(t, rr)
			}
		})
	}
}
