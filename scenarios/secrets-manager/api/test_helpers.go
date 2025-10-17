package main

import (
	"bytes"
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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// setupTestLogger initializes the logger for testing
func setupTestLogger() func() {
	// Save original logger output and redirect to discard during tests
	originalOutput := log.Writer()
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(originalOutput)
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
	tempDir, err := ioutil.TempDir("", "secrets-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create necessary test subdirectories
	testDirs := []string{
		filepath.Join(tempDir, "data"),
		filepath.Join(tempDir, "resources"),
		filepath.Join(tempDir, "scenarios"),
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default Content-Type if not specified
	if httpReq.Header.Get("Content-Type") == "" && req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL vars for mux
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields ...string) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Expected field '%s' in JSON response, but it was missing. Response: %v", field, result)
		}
	}

	return result
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		// Error responses might be plain text
		if !strings.Contains(w.Body.String(), "error") && !strings.Contains(w.Body.String(), "invalid") {
			t.Logf("Warning: Error response is not JSON and doesn't contain error keywords: %s", w.Body.String())
		}
		return
	}

	if _, hasError := result["error"]; !hasError {
		if _, hasMessage := result["message"]; !hasMessage {
			t.Errorf("Error response missing 'error' or 'message' field: %v", result)
		}
	}
}

// createTestResourceSecret creates a test resource secret
func createTestResourceSecret(resourceName, secretKey string) ResourceSecret {
	description := fmt.Sprintf("Test secret for %s", secretKey)
	return ResourceSecret{
		ID:           uuid.New().String(),
		ResourceName: resourceName,
		SecretKey:    secretKey,
		SecretType:   "api_key",
		Required:     true,
		Description:  &description,
	}
}

// createTestSecretValidation creates a test secret validation
func createTestSecretValidation(resourceSecretID string, status string) SecretValidation {
	return SecretValidation{
		ID:                  uuid.New().String(),
		ResourceSecretID:    resourceSecretID,
		ValidationStatus:    status,
		ValidationMethod:    "pattern_match",
		ErrorMessage:        nil,
		ValidationDetails:   nil,
	}
}

// createTestScanResult creates a test scan result
func createTestScanResult(scanType string, resourcesScanned []string, secretsFound int) SecretScan {
	return SecretScan{
		ID:                uuid.New().String(),
		ScanType:          scanType,
		ResourcesScanned:  resourcesScanned,
		SecretsDiscovered: secretsFound,
		ScanDurationMs:    100,
		ScanStatus:        "completed",
		ErrorMessage:      nil,
	}
}

// createTestVaultSecret creates a test vault secret
func createTestVaultSecret(name string, required bool) VaultSecret {
	return VaultSecret{
		Name:        name,
		Description: fmt.Sprintf("Test vault secret: %s", name),
		Required:    required,
		Configured:  false,
		SecretType:  "api_key",
	}
}

// createTestFile creates a test file with content
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	if err := ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

// createTestResourceDir creates a test resource directory structure
func createTestResourceDir(t *testing.T, baseDir, resourceName string, files map[string]string) string {
	resourceDir := filepath.Join(baseDir, "resources", resourceName)
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		t.Fatalf("Failed to create resource directory %s: %v", resourceDir, err)
	}

	for filename, content := range files {
		createTestFile(t, resourceDir, filename, content)
	}

	return resourceDir
}

// createTestScenarioDir creates a test scenario directory structure
func createTestScenarioDir(t *testing.T, baseDir, scenarioName string, files map[string]string) string {
	scenarioDir := filepath.Join(baseDir, "scenarios", scenarioName)
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory %s: %v", scenarioDir, err)
	}

	for filename, content := range files {
		createTestFile(t, scenarioDir, filename, content)
	}

	return scenarioDir
}

// mockVaultCLIOutput returns mock vault CLI output for testing
func mockVaultCLIOutput(resourceName string, configured bool) string {
	if configured {
		return fmt.Sprintf(`Resource: %s
Status: Configured
Secrets Found: 3
- API_KEY (configured)
- ENDPOINT (configured)
- SECRET_TOKEN (configured)
`, resourceName)
	}
	return fmt.Sprintf(`Resource: %s
Status: Missing
Missing Secrets:
- API_KEY (required)
- ENDPOINT (required)
- SECRET_TOKEN (optional)
`, resourceName)
}

// validateSecretType checks if a secret type is valid
func validateSecretType(secretType string) bool {
	validTypes := map[string]bool{
		"api_key":     true,
		"endpoint":    true,
		"password":    true,
		"token":       true,
		"certificate": true,
		"quota":       true,
		"config":      true,
	}
	return validTypes[secretType]
}

// validateValidationStatus checks if a validation status is valid
func validateValidationStatus(status string) bool {
	validStatuses := map[string]bool{
		"valid":     true,
		"invalid":   true,
		"missing":   true,
		"pending":   true,
		"error":     true,
	}
	return validStatuses[status]
}
