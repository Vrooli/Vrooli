// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"io"
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

// setupTestLogger initializes logging for testing with cleanup
func setupTestLogger() func() {
	// Redirect logs to /dev/null during tests to reduce noise
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Server  *Server
	Config  *Config
	Cleanup func()
}

// setupTestEnvironment creates a test server with mock configuration
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	config := &Config{
		Port:        "15001",
		DatabaseURL: "mock", // Use mock to avoid DB dependency
		N8NURL:      "http://mock-n8n:5678",
		APIToken:    "test-token-12345",
	}

	server := &Server{
		config: config,
		db:     nil, // No database for unit tests
		router: mux.NewRouter(),
	}

	server.setupRoutes()

	return &TestEnvironment{
		Server:  server,
		Config:  config,
		Cleanup: func() {
			// Cleanup resources if needed
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
func makeHTTPRequest(server *Server, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if using mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validateData func(data map[string]interface{})) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v. Body: %s", err, w.Body.String())
	}

	if validateData != nil {
		dataMap, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Response data is not a map: %T", response.Data)
		}
		validateData(dataMap)
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v. Body: %s", err, w.Body.String())
	}

	if response.Success {
		t.Error("Expected error response to have success=false")
	}

	if response.Error == "" {
		t.Error("Expected error message to be present")
	}

	if expectedErrorContains != "" && response.Error != expectedErrorContains {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorContains, response.Error)
	}
}

// assertHashResponse validates a hash operation response
func assertHashResponse(t *testing.T, data map[string]interface{}, expectedAlgorithm string) {
	t.Helper()

	hash, ok := data["hash"].(string)
	if !ok || hash == "" {
		t.Error("Expected hash to be present and non-empty")
	}

	algorithm, ok := data["algorithm"].(string)
	if !ok || algorithm != expectedAlgorithm {
		t.Errorf("Expected algorithm '%s', got '%s'", expectedAlgorithm, algorithm)
	}

	executionTime, ok := data["execution_time_ms"].(float64)
	if !ok || executionTime < 0 {
		t.Error("Expected execution_time_ms to be a positive number")
	}
}

// assertEncryptResponse validates an encryption response
func assertEncryptResponse(t *testing.T, data map[string]interface{}, expectedAlgorithm string) {
	t.Helper()

	encryptedData, ok := data["encrypted_data"].(string)
	if !ok || encryptedData == "" {
		t.Error("Expected encrypted_data to be present and non-empty")
	}

	algorithm, ok := data["algorithm"].(string)
	if !ok || algorithm != expectedAlgorithm {
		t.Errorf("Expected algorithm '%s', got '%s'", expectedAlgorithm, algorithm)
	}

	iv, ok := data["initialization_vector"].(string)
	if !ok || iv == "" {
		t.Error("Expected initialization_vector to be present and non-empty")
	}
}

// assertKeyGenResponse validates a key generation response
func assertKeyGenResponse(t *testing.T, data map[string]interface{}, expectedKeyType string, expectedKeySize int) {
	t.Helper()

	keyID, ok := data["key_id"].(string)
	if !ok || keyID == "" {
		t.Error("Expected key_id to be present and non-empty")
	}

	keyType, ok := data["key_type"].(string)
	if !ok || keyType != expectedKeyType {
		t.Errorf("Expected key_type '%s', got '%s'", expectedKeyType, keyType)
	}

	keySize, ok := data["key_size"].(float64)
	if !ok || int(keySize) != expectedKeySize {
		t.Errorf("Expected key_size %d, got %v", expectedKeySize, keySize)
	}

	fingerprint, ok := data["fingerprint"].(string)
	if !ok || fingerprint == "" {
		t.Error("Expected fingerprint to be present and non-empty")
	}

	createdAt, ok := data["created_at"].(string)
	if !ok || createdAt == "" {
		t.Error("Expected created_at to be present and non-empty")
	}

	// Parse and validate timestamp
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Errorf("Expected created_at to be valid RFC3339 timestamp, got parse error: %v", err)
	}
}

// assertSignResponse validates a signature response
func assertSignResponse(t *testing.T, data map[string]interface{}) {
	t.Helper()

	signature, ok := data["signature"].(string)
	if !ok || signature == "" {
		t.Error("Expected signature to be present and non-empty")
	}

	algorithm, ok := data["algorithm"].(string)
	if !ok || algorithm == "" {
		t.Error("Expected algorithm to be present and non-empty")
	}

	keyID, ok := data["key_id"].(string)
	if !ok || keyID == "" {
		t.Error("Expected key_id to be present and non-empty")
	}
}

// assertVerifyResponse validates a verification response
func assertVerifyResponse(t *testing.T, data map[string]interface{}, expectedValid bool) {
	t.Helper()

	isValid, ok := data["is_valid"].(bool)
	if !ok {
		t.Error("Expected is_valid to be a boolean")
	}

	if isValid != expectedValid {
		t.Errorf("Expected is_valid to be %v, got %v", expectedValid, isValid)
	}

	verificationDetails, ok := data["verification_details"].(map[string]interface{})
	if !ok {
		t.Error("Expected verification_details to be present")
	} else {
		signatureValid, ok := verificationDetails["signature_valid"].(bool)
		if !ok {
			t.Error("Expected verification_details.signature_valid to be a boolean")
		}
		if signatureValid != expectedValid {
			t.Errorf("Expected signature_valid to be %v, got %v", expectedValid, signatureValid)
		}
	}
}

// createAuthHeaders creates headers with authentication token
func createAuthHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}
}

// createTestHashRequest creates a test hash request
func createTestHashRequest(data, algorithm string) HashRequest {
	return HashRequest{
		Data:      data,
		Algorithm: algorithm,
		Format:    "hex",
	}
}

// createTestEncryptRequest creates a test encryption request
func createTestEncryptRequest(data, algorithm string) EncryptRequest {
	return EncryptRequest{
		Data:      data,
		Algorithm: algorithm,
		Format:    "base64",
	}
}

// createTestKeyGenRequest creates a test key generation request
func createTestKeyGenRequest(keyType string, keySize int) KeyGenRequest {
	return KeyGenRequest{
		KeyType: keyType,
		KeySize: keySize,
		Usage:   []string{"encryption", "signing"},
		Name:    "test-key",
	}
}
