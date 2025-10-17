package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// setupTestLogger initializes the global logger for testing with controlled output
func setupTestLogger() func() {
	// Create a discarding logger to avoid noise during tests
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
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
	tempDir, err := os.MkdirTemp("", "chart-generator-test-*")
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
	var body io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			body = bytes.NewBufferString(v)
		case []byte:
			body = bytes.NewBuffer(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(jsonData)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, body)
	if err != nil {
		return nil, err
	}

	// Set default content type for POST/PUT requests
	if req.Method == "POST" || req.Method == "PUT" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	rr := httptest.NewRecorder()
	return rr, nil
}

// assertJSONResponse validates that the response is valid JSON and matches expected structure
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if status := rr.Code; status != expectedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v", status, expectedStatus)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v\nBody: %s", err, rr.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates that the response contains an error with expected message
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedMessageContains string) {
	t.Helper()

	response := assertJSONResponse(t, rr, expectedStatus)
	if response == nil {
		return
	}

	// Check for error field
	if errField, ok := response["error"]; ok {
		if errMap, ok := errField.(map[string]interface{}); ok {
			if message, ok := errMap["message"].(string); ok {
				if expectedMessageContains != "" {
					// Simple string contains check
					found := false
					for i := 0; i <= len(message)-len(expectedMessageContains); i++ {
						if message[i:i+len(expectedMessageContains)] == expectedMessageContains {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Error message '%s' does not contain '%s'", message, expectedMessageContains)
					}
				}
				return
			}
		}
	}

	// Alternative: check for message field directly
	if message, ok := response["message"].(string); ok {
		if expectedMessageContains != "" {
			// Simple string contains check
			found := false
			for i := 0; i <= len(message)-len(expectedMessageContains); i++ {
				if message[i:i+len(expectedMessageContains)] == expectedMessageContains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Error message '%s' does not contain '%s'", message, expectedMessageContains)
			}
		}
		return
	}

	// If success is false, that's also an error response
	if success, ok := response["success"].(bool); ok && !success {
		return
	}

	t.Errorf("Response does not contain expected error structure: %+v", response)
}

// assertSuccessResponse validates that the response indicates success
func assertSuccessResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	response := assertJSONResponse(t, rr, expectedStatus)
	if response == nil {
		return nil
	}

	// Check for success field
	if success, ok := response["success"].(bool); ok && !success {
		t.Errorf("Expected success=true, got success=false")
	}

	return response
}

// createTempChartFile creates a temporary file for testing chart generation
func createTempChartFile(t *testing.T, filename string, content []byte) string {
	t.Helper()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, filename)

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return filePath
}

// assertFileExists checks if a file exists at the given path
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at path: %s", path)
	}
}

// assertFileNotExists checks if a file does NOT exist at the given path
func assertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expected file to NOT exist at path: %s", path)
	}
}

// generateTestChartData creates sample chart data for testing
func generateTestChartData(chartType string, numPoints int) []map[string]interface{} {
	data := make([]map[string]interface{}, numPoints)

	switch chartType {
	case "bar", "line", "area":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"x": string(rune('A' + i)),
				"y": float64((i + 1) * 10),
			}
		}
	case "pie":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"name":  string(rune('A' + i)),
				"value": float64((i + 1) * 10),
			}
		}
	case "scatter":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"x": float64(i * 10),
				"y": float64((i + 1) * 15),
			}
		}
	case "candlestick":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"date":  "2024-01-" + string(rune('0'+i+1)),
				"open":  float64(100 + i*5),
				"high":  float64(110 + i*5),
				"low":   float64(95 + i*5),
				"close": float64(105 + i*5),
			}
		}
	case "heatmap":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"x":     string(rune('A' + i%5)),
				"y":     string(rune('1' + i/5)),
				"value": float64((i + 1) * 10),
			}
		}
	case "treemap":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"name":  "Category " + string(rune('A'+i)),
				"value": float64((i + 1) * 20),
			}
		}
	case "gantt":
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"task":     "Task " + string(rune('A'+i)),
				"start":    float64(i * 10),
				"duration": float64(20 + i*5),
			}
		}
	default:
		for i := 0; i < numPoints; i++ {
			data[i] = map[string]interface{}{
				"x": string(rune('A' + i)),
				"y": float64((i + 1) * 10),
			}
		}
	}

	return data
}

// assertResponseField checks if a response contains a specific field with expected value
func assertResponseField(t *testing.T, response map[string]interface{}, field string, expectedValue interface{}) {
	t.Helper()

	value, ok := response[field]
	if !ok {
		t.Errorf("Response missing expected field: %s", field)
		return
	}

	if expectedValue != nil && value != expectedValue {
		t.Errorf("Field %s: expected %v, got %v", field, expectedValue, value)
	}
}

// assertResponseHasField checks if a response contains a specific field (value doesn't matter)
func assertResponseHasField(t *testing.T, response map[string]interface{}, field string) {
	t.Helper()

	if _, ok := response[field]; !ok {
		t.Errorf("Response missing expected field: %s", field)
	}
}
