package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// Setup test environment
func setupLogoTestEnv(t *testing.T) string {
	// Create temporary directory for test uploads
	tempDir := filepath.Join(os.TempDir(), "invoice-generator-test-logos")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	return tempDir
}

// Cleanup test environment
func cleanupLogoTestEnv(t *testing.T, tempDir string) {
	if err := os.RemoveAll(tempDir); err != nil {
		t.Logf("Warning: Failed to cleanup test directory: %v", err)
	}
}

// Helper to create multipart form request with proper content type
func createMultipartRequest(t *testing.T, fieldName, fileName, content string) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Determine content type from file extension
	var contentType string
	if strings.HasSuffix(fileName, ".svg") {
		contentType = "image/svg+xml"
	} else if strings.HasSuffix(fileName, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(fileName, ".jpg") || strings.HasSuffix(fileName, ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(fileName, ".webp") {
		contentType = "image/webp"
	} else {
		contentType = "text/plain"
	}

	// Create form file with proper MIME type
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName)}
	h["Content-Type"] = []string{contentType}

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := io.WriteString(part, content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/api/logos/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func TestUploadLogoHandler_Success(t *testing.T) {
	// Create test SVG content
	svgContent := `<svg width="100" height="100"><rect width="100" height="100" fill="#2563EB"/></svg>`

	req := createMultipartRequest(t, "logo", "test.svg", svgContent)
	rr := httptest.NewRecorder()

	// Set required environment variable for test
	os.Setenv("API_PORT", "8100")
	defer os.Unsetenv("API_PORT")

	// Execute
	UploadLogoHandler(rr, req)

	// Verify
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusCreated, status, rr.Body.String())
	}

	var response LogoUploadResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, rr.Body.String())
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false")
	}

	if response.LogoID == "" {
		t.Errorf("Expected non-empty logo_id")
	}

	if !strings.HasSuffix(response.LogoURL, ".svg") {
		t.Errorf("Expected logo URL to end with .svg, got %s", response.LogoURL)
	}

	if response.FileSize != int64(len(svgContent)) {
		t.Errorf("Expected file size %d, got %d", len(svgContent), response.FileSize)
	}

	// Cleanup - delete the uploaded test file
	testFile := filepath.Join(LogoStoragePath, response.FileName)
	os.Remove(testFile)
}

func TestUploadLogoHandler_InvalidFileType(t *testing.T) {
	// Setup
	req := createMultipartRequest(t, "logo", "test.txt", "This is not an image")
	rr := httptest.NewRecorder()

	// Execute
	UploadLogoHandler(rr, req)

	// Verify
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, exists := response["error"]; !exists {
		t.Errorf("Expected error field in response")
	}
}

func TestUploadLogoHandler_MissingFile(t *testing.T) {
	// Setup
	req := httptest.NewRequest("POST", "/api/logos/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	rr := httptest.NewRecorder()

	// Execute
	UploadLogoHandler(rr, req)

	// Verify
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}
}

func TestUploadLogoHandler_FileTooLarge(t *testing.T) {
	// Create content larger than 5MB
	largeContent := strings.Repeat("a", MaxLogoSize+1)

	req := createMultipartRequest(t, "logo", "large.png", largeContent)
	rr := httptest.NewRecorder()

	// Execute
	UploadLogoHandler(rr, req)

	// Verify
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d for oversized file, got %d", http.StatusBadRequest, status)
	}
}

func TestListLogosHandler(t *testing.T) {
	// Setup
	req := httptest.NewRequest("GET", "/api/logos", nil)
	rr := httptest.NewRecorder()

	// Set required environment variable for test
	os.Setenv("API_PORT", "8100")
	defer os.Unsetenv("API_PORT")

	// Execute
	ListLogosHandler(rr, req)

	// Verify
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Just verify the response structure is correct (count may vary based on test state)
	if _, ok := response["count"]; !ok {
		t.Errorf("Expected 'count' field in response")
	}

	if _, ok := response["logos"]; !ok {
		t.Errorf("Expected 'logos' field in response")
	}
}

func TestGetLogoHandler_NotFound(t *testing.T) {
	// Setup
	req := httptest.NewRequest("GET", "/api/logos/nonexistent.svg", nil)
	rr := httptest.NewRecorder()

	// Use mux router to properly set path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/logos/{filename}", GetLogoHandler).Methods("GET")

	// Execute
	router.ServeHTTP(rr, req)

	// Verify - Should return 404
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status %d for nonexistent file, got %d", http.StatusNotFound, status)
	}
}

func TestGetLogoHandler_InvalidFilename(t *testing.T) {
	// Test with actual path traversal characters in the filename parameter
	req := httptest.NewRequest("GET", "/api/logos/..%2F..%2F..%2Fetc%2Fpasswd", nil)
	rr := httptest.NewRecorder()

	// Use mux router to properly set path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/logos/{filename}", GetLogoHandler).Methods("GET")

	// Execute
	router.ServeHTTP(rr, req)

	// Verify - mux router redirects path traversal attempts (301) or handler rejects (400)
	if status := rr.Code; status != http.StatusBadRequest && status != http.StatusMovedPermanently {
		t.Errorf("Expected status %d or %d for invalid filename, got %d", http.StatusBadRequest, http.StatusMovedPermanently, status)
	}
}

func TestDeleteLogoHandler_InvalidFilename(t *testing.T) {
	// Test with actual path traversal characters in the filename parameter
	req := httptest.NewRequest("DELETE", "/api/logos/..%2F..%2F..%2Fetc%2Fpasswd", nil)
	rr := httptest.NewRecorder()

	// Use mux router to properly set path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/logos/{filename}", DeleteLogoHandler).Methods("DELETE")

	// Execute
	router.ServeHTTP(rr, req)

	// Verify - mux router redirects path traversal attempts (301) or handler rejects (400)
	if status := rr.Code; status != http.StatusBadRequest && status != http.StatusMovedPermanently {
		t.Errorf("Expected status %d or %d for invalid filename, got %d", http.StatusBadRequest, http.StatusMovedPermanently, status)
	}
}

func TestAllowedLogoTypes(t *testing.T) {
	// Verify allowed types configuration
	expectedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/svg+xml", "image/webp"}

	for _, mimeType := range expectedTypes {
		if _, exists := allowedLogoTypes[mimeType]; !exists {
			t.Errorf("Expected MIME type %s to be allowed", mimeType)
		}
	}

	// Verify correct file extensions
	if allowedLogoTypes["image/png"] != ".png" {
		t.Errorf("Expected PNG extension to be .png, got %s", allowedLogoTypes["image/png"])
	}

	if allowedLogoTypes["image/svg+xml"] != ".svg" {
		t.Errorf("Expected SVG extension to be .svg, got %s", allowedLogoTypes["image/svg+xml"])
	}
}

func TestMaxLogoSize(t *testing.T) {
	// Verify max size is 5MB as documented
	expected := 5 * 1024 * 1024
	if MaxLogoSize != expected {
		t.Errorf("Expected MaxLogoSize to be %d (5MB), got %d", expected, MaxLogoSize)
	}
}
