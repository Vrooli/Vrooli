package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() { log.SetOutput(originalOutput) }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir string
	Cleanup func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "image-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set environment variables for test
	os.Setenv("MINIO_DISABLED", "true")
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	return &TestEnvironment{
		TempDir: tempDir,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestServer creates a test server instance
func setupTestServer() *Server {
	os.Setenv("MINIO_DISABLED", "true")
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	server := NewServer()
	server.SetupRoutes()
	return server
}

// makeHTTPRequest creates and executes an HTTP request against the test server
func makeHTTPRequest(app *fiber.App, method, path string, body io.Reader, headers map[string]string) (*http.Response, []byte, error) {
	req := httptest.NewRequest(method, path, body)

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req, -1)
	if err != nil {
		return nil, nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}

	return resp, respBody, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, resp *http.Response, body []byte, expectedStatus int) map[string]interface{} {
	t.Helper()

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, string(body))
	}

	return result
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, resp *http.Response, body []byte, expectedStatus int, expectedErrorKey string) {
	t.Helper()

	result := assertJSONResponse(t, resp, body, expectedStatus)

	if _, ok := result[expectedErrorKey]; !ok {
		t.Errorf("Expected error key '%s' in response, got: %v", expectedErrorKey, result)
	}
}

// createTestImageFile creates a test image file for upload tests
func createTestImageFile(t *testing.T, filename string, content []byte) string {
	t.Helper()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, filename)

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to create test image file: %v", err)
	}

	return path
}

// createMultipartFormData creates multipart form data for file uploads
func createMultipartFormData(t *testing.T, fieldName, filename string, content []byte, formValues map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add form values
	for key, value := range formValues {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("Failed to write form field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

// createMultipartBatchData creates multipart form data for batch uploads
func createMultipartBatchData(t *testing.T, files map[string][]byte, operations string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add files
	for filename, content := range files {
		part, err := writer.CreateFormFile("images", filename)
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}

		if _, err := part.Write(content); err != nil {
			t.Fatalf("Failed to write file content: %v", err)
		}
	}

	// Add operations
	if err := writer.WriteField("operations", operations); err != nil {
		t.Fatalf("Failed to write operations field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

// generateTestImageData generates minimal test image data
func generateTestImageData(format string) []byte {
	switch format {
	case "png":
		// Minimal valid PNG (1x1 red pixel)
		return []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
			0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
			0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
			0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
			0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
			0x44, 0xAE, 0x42, 0x60, 0x82,
		}
	case "jpeg", "jpg":
		// Minimal valid JPEG
		return []byte{
			0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
			0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
			0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
			0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
			0x07, 0x07, 0x07, 0x09, 0x09, 0x08, 0x0A, 0x0C,
			0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
			0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D,
			0x1A, 0x1C, 0x1C, 0x20, 0x24, 0x2E, 0x27, 0x20,
			0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
			0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27,
			0x39, 0x3D, 0x38, 0x32, 0x3C, 0x2E, 0x33, 0x34,
			0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
			0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4,
			0x00, 0x14, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x03, 0xFF, 0xC4, 0x00, 0x14,
			0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01,
			0x00, 0x00, 0x3F, 0x00, 0x37, 0xFF, 0xD9,
		}
	default:
		return []byte("fake image data")
	}
}

// createTestFileHeader creates a test multipart.FileHeader for testing
func createTestFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	// Create a temporary file
	tempFile := createTestImageFile(t, filename, content)

	// Read the file back
	fileContent, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	// Create buffer and multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("test", filename)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	writer.Close()

	// Parse the multipart data to extract FileHeader
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}

	files := form.File["test"]
	if len(files) == 0 {
		t.Fatal("No files in form")
	}

	return files[0]
}
