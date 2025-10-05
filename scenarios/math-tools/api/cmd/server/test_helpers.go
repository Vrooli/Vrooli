
package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
}

// setupTestLogger initializes logging for testing and returns cleanup function
func setupTestLogger() func() {
	// Redirect log output to discard during tests (unless verbose)
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(ioutil.Discard)
		return func() {
			log.SetOutput(os.Stderr)
		}
	}
	return func() {}
}

// TestServerConfig holds test server configuration
type TestServerConfig struct {
	APIToken    string
	DatabaseURL string
}

// setupTestServer creates a test server with the given configuration
func setupTestServer(t *testing.T, config *TestServerConfig) (*Server, func()) {
	// Set environment variables
	if config == nil {
		config = &TestServerConfig{
			APIToken: "test-token",
		}
	}

	os.Setenv("API_TOKEN", config.APIToken)
	os.Setenv("PORT", "0") // Random port

	// Don't use real database for tests - use invalid URL to skip DB connection
	if config.DatabaseURL == "" {
		// Use an invalid host/port that will fail quickly
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost:99999/test?connect_timeout=1")
	} else {
		os.Setenv("DATABASE_URL", config.DatabaseURL)
	}

	// Create server configuration manually to avoid slow DB connection
	serverConfig := &Config{
		Port:        getEnv("API_PORT", getEnv("PORT", "8095")),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		APIToken:    config.APIToken,
	}

	server := &Server{
		config: serverConfig,
		db:     nil, // No database for tests
		router: mux.NewRouter(),
	}

	server.setupRoutes()

	// Cleanup function
	cleanup := func() {
		if server.db != nil {
			server.db.Close()
		}
		os.Unsetenv("API_TOKEN")
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_URL")
	}

	return server, cleanup
}

// makeHTTPRequest is a helper to create HTTP requests for testing
func makeHTTPRequest(t *testing.T, method, path string, body interface{}, token string) *http.Request {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, validateFunc func(map[string]interface{}) error) {
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
		return
	}

	// Check content type
	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
		return
	}

	// Parse response
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Body: %s", err, recorder.Body.String())
		return
	}

	// Validate success flag
	expectedSuccess := expectedStatus < 400
	if response.Success != expectedSuccess {
		t.Errorf("Expected success=%v, got %v", expectedSuccess, response.Success)
	}

	// Custom validation
	if validateFunc != nil {
		responseMap := make(map[string]interface{})
		json.Unmarshal(recorder.Body.Bytes(), &responseMap)
		if err := validateFunc(responseMap); err != nil {
			t.Errorf("Response validation failed: %v", err)
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
		return
	}

	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse error response: %v", err)
		return
	}

	if response.Success {
		t.Errorf("Expected success=false for error response, got true")
	}

	if response.Error == "" {
		t.Errorf("Expected error message, got empty string")
	}

	if expectedErrorSubstring != "" && !contains(response.Error, expectedErrorSubstring) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedErrorSubstring, response.Error)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))))
}

// assertCalculationResult validates a calculation response
func assertCalculationResult(t *testing.T, response map[string]interface{}, operation string, expectedResult interface{}) {
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected data object in response")
		return
	}

	if op, ok := data["operation"].(string); !ok || op != operation {
		t.Errorf("Expected operation %s, got %v", operation, data["operation"])
	}

	result := data["result"]
	switch expected := expectedResult.(type) {
	case float64:
		if r, ok := result.(float64); !ok || !floatEquals(r, expected) {
			t.Errorf("Expected result %v, got %v", expected, result)
		}
	case []float64:
		if rSlice, ok := result.([]interface{}); ok {
			if len(rSlice) != len(expected) {
				t.Errorf("Expected result length %d, got %d", len(expected), len(rSlice))
			}
			for i, exp := range expected {
				if r, ok := rSlice[i].(float64); !ok || !floatEquals(r, exp) {
					t.Errorf("Expected result[%d] = %v, got %v", i, exp, rSlice[i])
				}
			}
		} else {
			t.Errorf("Expected slice result, got %T", result)
		}
	case map[string]interface{}:
		if rMap, ok := result.(map[string]interface{}); ok {
			for key, expVal := range expected {
				if val, exists := rMap[key]; !exists {
					t.Errorf("Expected key %s in result, not found", key)
				} else if expFloat, ok := expVal.(float64); ok {
					if valFloat, ok := val.(float64); ok {
						if !floatEquals(valFloat, expFloat) {
							t.Errorf("Expected %s = %v, got %v", key, expFloat, valFloat)
						}
					}
				}
			}
		} else {
			t.Errorf("Expected map result, got %T", result)
		}
	}
}

// floatEquals compares two floats with tolerance
func floatEquals(a, b float64) bool {
	tolerance := 0.0001
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < tolerance
}

// createCalculationRequest creates a calculation request payload
func createCalculationRequest(operation string, data []float64) CalculationRequest {
	return CalculationRequest{
		Operation: operation,
		Data:      data,
	}
}

// createMatrixRequest creates a matrix operation request
func createMatrixRequest(operation string, matrix [][]float64, matrixA [][]float64, matrixB [][]float64) CalculationRequest {
	return CalculationRequest{
		Operation: operation,
		Matrix:    matrix,
		MatrixA:   matrixA,
		MatrixB:   matrixB,
	}
}

// createStatisticsRequest creates a statistics request payload
func createStatisticsRequest(data []float64, analyses []string) StatisticsRequest {
	return StatisticsRequest{
		Data:     data,
		Analyses: analyses,
	}
}

// createSolveRequest creates an equation solving request
func createSolveRequest(equation string, variables []string, method string) SolveRequest {
	req := SolveRequest{
		Equations: equation,
		Variables: variables,
		Method:    method,
	}
	req.Options.Tolerance = 1e-6
	req.Options.MaxIterations = 1000
	return req
}

// createOptimizeRequest creates an optimization request
func createOptimizeRequest(objective string, variables []string, optimizationType string) OptimizeRequest {
	req := OptimizeRequest{
		ObjectiveFunction: objective,
		Variables:        variables,
		OptimizationType: optimizationType,
		Algorithm:        "gradient_descent",
	}
	req.Options.Tolerance = 1e-6
	req.Options.MaxIterations = 1000
	return req
}

// createForecastRequest creates a time series forecast request
func createForecastRequest(timeSeries []float64, horizon int, method string) ForecastRequest {
	return ForecastRequest{
		TimeSeries:      timeSeries,
		ForecastHorizon: horizon,
		Method:          method,
	}
}

// testEndpoint is a helper that tests an endpoint with a request and validates response
func testEndpoint(t *testing.T, server *Server, method, path string, body interface{}, token string, expectedStatus int, validator func(map[string]interface{}) error) {
	req := makeHTTPRequest(t, method, path, body, token)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
	assertJSONResponse(t, recorder, expectedStatus, validator)
}

// testEndpointError tests that an endpoint returns an error
func testEndpointError(t *testing.T, server *Server, method, path string, body interface{}, token string, expectedStatus int, errorSubstring string) {
	req := makeHTTPRequest(t, method, path, body, token)
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, req)
	assertErrorResponse(t, recorder, expectedStatus, errorSubstring)
}
