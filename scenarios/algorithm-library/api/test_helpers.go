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
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestLogger initializes the global logger for testing with controlled output
func setupTestLogger() func() {
	originalLogger := logger
	// Suppress logger output during tests unless VERBOSE_TESTS is set
	if os.Getenv("VERBOSE_TESTS") != "true" {
		logger = log.New(ioutil.Discard, "", 0)
	} else {
		logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	}
	return func() { logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "algorithm-library-test")
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

// HTTPTestRequest represents a simplified HTTP request for testing
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	URLVars     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates an HTTP test request with optional JSON body
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Buffer
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	} else {
		bodyReader = bytes.NewBuffer([]byte{})
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default Content-Type for requests with body
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add URL vars for mux router
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, dest interface{}) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code, "Response status code mismatch")

	if dest != nil {
		err := json.NewDecoder(w.Body).Decode(dest)
		require.NoError(t, err, "Failed to decode JSON response")
	}
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessageContains string) {
	t.Helper()

	assert.Equal(t, expectedStatus, w.Code, "Response status code mismatch")

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode error response")

	if expectedMessageContains != "" {
		message, ok := result["error"].(string)
		if !ok {
			message, _ = result["message"].(string)
		}
		assert.Contains(t, message, expectedMessageContains, "Error message does not contain expected text")
	}
}

// setupMockDB creates a mock database connection for testing
func setupMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create mock database")

	oldDB := db
	db = mockDB

	cleanup := func() {
		db = oldDB
		mockDB.Close()
	}

	return mock, cleanup
}

// createTestFile creates a temporary file with given content for testing
func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	require.NoError(t, err, "Failed to create directory")

	err = ioutil.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err, "Failed to write test file")

	return filePath
}

// TestAlgorithmData provides sample algorithm data for tests
type TestAlgorithmData struct {
	ID                  string
	Name                string
	DisplayName         string
	Category            string
	Description         string
	ComplexityTime      string
	ComplexitySpace     string
	Difficulty          string
	Tags                string
	LanguageCount       int
	TestCaseCount       int
	HasValidatedImpl    bool
}

// getTestAlgorithm returns a standard test algorithm for reuse
func getTestAlgorithm() TestAlgorithmData {
	return TestAlgorithmData{
		ID:                  "123-456-789",
		Name:                "quicksort",
		DisplayName:         "QuickSort",
		Category:            "sorting",
		Description:         "Efficient divide-and-conquer sorting algorithm",
		ComplexityTime:      "O(n log n)",
		ComplexitySpace:     "O(log n)",
		Difficulty:          "medium",
		Tags:                `["divide-conquer","in-place"]`,
		LanguageCount:       3,
		TestCaseCount:       5,
		HasValidatedImpl:    true,
	}
}

// setupMockAlgorithmRows prepares sqlmock rows for algorithm queries
func setupMockAlgorithmRows(mock sqlmock.Sqlmock, algo TestAlgorithmData) {
	rows := sqlmock.NewRows([]string{
		"id", "name", "display_name", "category", "description",
		"complexity_time", "complexity_space", "difficulty", "tags",
		"language_count", "test_case_count", "has_validated_impl",
	}).AddRow(
		algo.ID, algo.Name, algo.DisplayName, algo.Category, algo.Description,
		algo.ComplexityTime, algo.ComplexitySpace, algo.Difficulty, algo.Tags,
		algo.LanguageCount, algo.TestCaseCount, algo.HasValidatedImpl,
	)

	mock.ExpectQuery(`SELECT DISTINCT a\.id`).WillReturnRows(rows)
}

// TestImplementationData provides sample implementation data for tests
type TestImplementationData struct {
	ID          string
	AlgorithmID string
	Language    string
	Code        string
	Validated   bool
}

// getTestImplementation returns a standard test implementation
func getTestImplementation() TestImplementationData {
	return TestImplementationData{
		ID:          "impl-123",
		AlgorithmID: "123-456-789",
		Language:    "python",
		Code:        "def quicksort(arr): return sorted(arr)",
		Validated:   true,
	}
}

// TestCaseData provides sample test case data
type TestCaseData struct {
	ID             string
	AlgorithmID    string
	Input          string
	ExpectedOutput string
	TimeoutMs      int
}

// getTestCases returns standard test cases for algorithm testing
func getTestCases() []TestCaseData {
	return []TestCaseData{
		{
			ID:             "test-1",
			AlgorithmID:    "123-456-789",
			Input:          `{"arr": [3, 1, 2]}`,
			ExpectedOutput: `[1, 2, 3]`,
			TimeoutMs:      1000,
		},
		{
			ID:             "test-2",
			AlgorithmID:    "123-456-789",
			Input:          `{"arr": [5, 2, 8, 1]}`,
			ExpectedOutput: `[1, 2, 5, 8]`,
			TimeoutMs:      1000,
		},
		{
			ID:             "test-3",
			AlgorithmID:    "123-456-789",
			Input:          `{"arr": []}`,
			ExpectedOutput: `[]`,
			TimeoutMs:      1000,
		},
	}
}

// setupMockTestCases prepares sqlmock rows for test case queries
func setupMockTestCases(mock sqlmock.Sqlmock, testCases []TestCaseData, algorithmID string) {
	rows := sqlmock.NewRows([]string{"id", "input", "expected_output", "timeout_ms"})
	for _, tc := range testCases {
		rows.AddRow(tc.ID, tc.Input, tc.ExpectedOutput, tc.TimeoutMs)
	}

	mock.ExpectQuery(`SELECT id, input, expected_output, timeout_ms`).
		WithArgs(algorithmID).
		WillReturnRows(rows)
}

// expectDatabasePing sets up expectation for database health check
func expectDatabasePing(mock sqlmock.Sqlmock) {
	mock.ExpectPing()
}

// expectAlgorithmUsageLog sets up expectation for usage logging
func expectAlgorithmUsageLog(mock sqlmock.Sqlmock, algorithmID, language string) {
	mock.ExpectExec(`INSERT INTO algorithm_usage`).
		WithArgs(algorithmID, language, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

// assertMockExpectations verifies all mock expectations were met
func assertMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	err := mock.ExpectationsWereMet()
	assert.NoError(t, err, "Mock database expectations were not met")
}
