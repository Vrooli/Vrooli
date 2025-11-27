package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

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
			var bodyBytes []byte
			var err error

			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/api/v1/parse/lint", bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			srv.handleParseLint(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handleParseLint() status = %v, want %v", status, tt.wantStatus)
			}

			if tt.wantError != "" {
				var response map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if response["error"] != tt.wantError {
					t.Errorf("handleParseLint() error = %v, want %v", response["error"], tt.wantError)
				}
			} else {
				var response map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode success response: %v", err)
				}
				if _, ok := response["issues"]; !ok {
					t.Error("handleParseLint() response missing 'issues' field")
				}
				if _, ok := response["count"]; !ok {
					t.Error("handleParseLint() response missing 'count' field")
				}
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
			var bodyBytes []byte
			var err error

			if str, ok := tt.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/api/v1/parse/type", bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			srv.handleParseType(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handleParseType() status = %v, want %v", status, tt.wantStatus)
			}

			if tt.wantError != "" {
				var response map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if response["error"] != tt.wantError {
					t.Errorf("handleParseType() error = %v, want %v", response["error"], tt.wantError)
				}
			} else {
				var response map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode success response: %v", err)
				}
				if _, ok := response["issues"]; !ok {
					t.Error("handleParseType() response missing 'issues' field")
				}
				if _, ok := response["count"]; !ok {
					t.Error("handleParseType() response missing 'count' field")
				}
			}
		})
	}
}
