package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for testing
func setupTestLogger() func() {
	// Redirect logs to discard by default
	// Tests can enable verbose logging if needed
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir     string
	WorkDir     string
	DataDir     string
	OriginalWD  string
	Cleanup     func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "audio-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	workDir := filepath.Join(tempDir, "work")
	dataDir := filepath.Join(tempDir, "data")

	if err := os.MkdirAll(workDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create work dir: %v", err)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create data dir: %v", err)
	}

	originalWD, _ := os.Getwd()

	return &TestEnvironment{
		TempDir:    tempDir,
		WorkDir:    workDir,
		DataDir:    dataDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// createTestAudioFile creates a simple WAV file for testing
// Note: This creates a minimal valid WAV header + silence
func createTestAudioFile(t *testing.T, dir, filename string, durationSeconds float64) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)

	// Simple WAV header for 16-bit mono 44100 Hz
	sampleRate := 44100
	numSamples := int(durationSeconds * float64(sampleRate))
	dataSize := numSamples * 2 // 16-bit = 2 bytes per sample
	fileSize := 44 + dataSize - 8

	header := make([]byte, 44)

	// RIFF header
	copy(header[0:4], "RIFF")
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	header[16] = 16 // chunk size
	header[20] = 1  // audio format (PCM)
	header[22] = 1  // num channels
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)

	byteRate := sampleRate * 2
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)

	header[32] = 2  // block align
	header[34] = 16 // bits per sample

	// data chunk
	copy(header[36:40], "data")
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}
	defer file.Close()

	// Write header
	if _, err := file.Write(header); err != nil {
		t.Fatalf("Failed to write WAV header: %v", err)
	}

	// Write silence (zeros)
	silence := make([]byte, dataSize)
	if _, err := file.Write(silence); err != nil {
		t.Fatalf("Failed to write audio data: %v", err)
	}

	return filePath
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	URLVars     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates an HTTP test request from HTTPTestRequest
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBody, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default content type for JSON
	if req.Headers == nil || req.Headers["Content-Type"] == "" {
		if req.Body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// createMultipartRequest creates a multipart form request for file upload
func createMultipartRequest(t *testing.T, filePath, fieldName string, formFields map[string]string) (*http.Request, *multipart.Writer, error) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	if filePath != "" {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := io.Copy(part, file); err != nil {
			return nil, nil, fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// Add form fields
	for key, value := range formFields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, nil, fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, writer, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); ok {
		if expectedErrorContains != "" && !contains(errorMsg, expectedErrorContains) {
			t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorContains, errorMsg)
		}
	} else {
		t.Error("Expected error field in response")
	}
}

// assertFieldExists checks if a field exists in the response
func assertFieldExists(t *testing.T, response map[string]interface{}, fieldName string) interface{} {
	t.Helper()

	value, ok := response[fieldName]
	if !ok {
		t.Errorf("Expected field '%s' in response, but it was missing", fieldName)
		return nil
	}

	return value
}

// assertFieldEquals checks if a field equals the expected value
func assertFieldEquals(t *testing.T, response map[string]interface{}, fieldName string, expected interface{}) {
	t.Helper()

	value := assertFieldExists(t, response, fieldName)
	if value != expected {
		t.Errorf("Expected field '%s' to be %v, got %v", fieldName, expected, value)
	}
}

// generateTestID generates a test UUID
func generateTestID() string {
	return uuid.New().String()
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
