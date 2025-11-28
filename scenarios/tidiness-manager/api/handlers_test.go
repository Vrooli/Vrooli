package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// Test helpers for handler testing

// marshalBodyOrString marshals body to JSON bytes, or converts string directly
func marshalBodyOrString(t *testing.T, body interface{}) []byte {
	t.Helper()
	if str, ok := body.(string); ok {
		return []byte(str)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}
	return bodyBytes
}

// testHandlerRequest executes a handler with a test request and returns the response recorder
func testHandlerRequest(t *testing.T, method, url string, body interface{}, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes := marshalBodyOrString(t, body)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// assertHandlerStatus checks if handler returned expected status code
func assertHandlerStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int, handlerName string) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("%s status = %v, want %v", handlerName, rr.Code, expected)
	}
}

// assertHandlerError decodes error response and checks message
func assertHandlerError(t *testing.T, rr *httptest.ResponseRecorder, wantError, handlerName string) {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if response["error"] != wantError {
		t.Errorf("%s error = %v, want %v", handlerName, response["error"], wantError)
	}
}

// assertParseResponseFields validates parse endpoint response has required fields
func assertParseResponseFields(t *testing.T, rr *httptest.ResponseRecorder, handlerName string) {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode success response: %v", err)
	}
	if _, ok := response["issues"]; !ok {
		t.Errorf("%s response missing 'issues' field", handlerName)
	}
	if _, ok := response["count"]; !ok {
		t.Errorf("%s response missing 'count' field", handlerName)
	}
}

// Test respondJSON helper function
func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created response",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			respondJSON(rr, tt.status, tt.data)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("respondJSON() status = %v, want %v", status, tt.wantStatus)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("respondJSON() Content-Type = %v, want application/json", ct)
			}
		})
	}
}

// Test respondError helper function
func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request error",
			status:     http.StatusBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal server error",
			status:     http.StatusInternalServerError,
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			respondError(rr, tt.status, tt.message)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("respondError() status = %v, want %v", status, tt.wantStatus)
			}

			var response map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("respondError() message = %v, want %v", response["error"], tt.message)
			}
		})
	}
}

// Test handleParseLint
func TestHandleParseLint(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "valid lint parse request",
			body: ParseRequest{
				Scenario: "test-scenario",
				Tool:     "eslint",
				Output:   "src/main.ts:10:5: error: Unexpected token",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing scenario",
			body:       ParseRequest{Tool: "eslint", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "missing tool",
			body:       ParseRequest{Scenario: "test", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/parse/lint", tt.body, srv.handleParseLint)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleParseLint()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleParseLint()")
			} else {
				assertParseResponseFields(t, rr, "handleParseLint()")
			}
		})
	}
}

// Test handleParseType
func TestHandleParseType(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "valid type parse request",
			body: ParseRequest{
				Scenario: "test-scenario",
				Tool:     "tsc",
				Output:   "src/main.ts(10,5): error TS2304: Cannot find name 'foo'",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing scenario",
			body:       ParseRequest{Tool: "tsc", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "missing tool",
			body:       ParseRequest{Scenario: "test", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/parse/type", tt.body, srv.handleParseType)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleParseType()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleParseType()")
			} else {
				assertParseResponseFields(t, rr, "handleParseType()")
			}
		})
	}
}

// Test handleLightScan
func TestHandleLightScan(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
		db:     nil, // No DB needed for basic tests
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing scenario_path",
			body:       LightScanRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario_path is required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/scan/light", tt.body, srv.handleLightScan)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleLightScan()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleLightScan()")
			}
		})
	}
}

// Test handleSmartScan
func TestHandleSmartScan(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing scenario",
			body:       SmartScanRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario is required",
		},
		{
			name:       "missing files",
			body:       SmartScanRequest{Scenario: "test-scenario"},
			wantStatus: http.StatusBadRequest,
			wantError:  "files list cannot be empty",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/scan/smart", tt.body, srv.handleSmartScan)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleSmartScan()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleSmartScan()")
			}
		})
	}
}
