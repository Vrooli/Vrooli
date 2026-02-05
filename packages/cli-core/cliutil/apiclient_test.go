package cliutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIClientAppliesBaseAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("expected bearer token")
		}
		fmt.Fprintf(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := NewAPIClient(NewHTTPClient(HTTPClientOptions{}), func() APIBaseOptions {
		return APIBaseOptions{DefaultBase: server.URL}
	}, func() string { return "secret" })

	body, err := client.Get("/ping", nil)
	if err != nil {
		t.Fatalf("api get: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		response       string
		wantMessage    string
		wantCode       string
		wantStructured bool
	}{
		{
			name:           "structured error with all fields",
			statusCode:     400,
			response:       `{"error":"validation failed","code":"VALIDATION_ERROR","recovery_hint":"Fix the input","manual_steps":["Step 1","Step 2"]}`,
			wantMessage:    "validation failed",
			wantCode:       "VALIDATION_ERROR",
			wantStructured: true,
		},
		{
			name:           "structured error with auto_fix",
			statusCode:     500,
			response:       `{"error":"build failed","code":"BUILD_FAILED","auto_fix":{"command":"make rebuild","description":"Rebuild the project","safe":true}}`,
			wantMessage:    "build failed",
			wantCode:       "BUILD_FAILED",
			wantStructured: true,
		},
		{
			name:           "minimal structured error",
			statusCode:     404,
			response:       `{"error":"not found","code":"NOT_FOUND"}`,
			wantMessage:    "not found",
			wantCode:       "NOT_FOUND",
			wantStructured: true,
		},
		{
			name:           "unstructured error - plain text",
			statusCode:     500,
			response:       "Internal server error",
			wantMessage:    "Internal server error",
			wantCode:       "",
			wantStructured: false,
		},
		{
			name:           "unstructured error - no code field",
			statusCode:     400,
			response:       `{"error":"something went wrong"}`,
			wantMessage:    "something went wrong",
			wantCode:       "",
			wantStructured: false,
		},
		{
			name:           "empty response",
			statusCode:     503,
			response:       "",
			wantMessage:    "HTTP 503",
			wantCode:       "",
			wantStructured: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := ParseAPIError(tt.statusCode, []byte(tt.response))

			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantMessage)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", apiErr.Code, tt.wantCode)
			}
			if apiErr.IsStructured() != tt.wantStructured {
				t.Errorf("IsStructured() = %v, want %v", apiErr.IsStructured(), tt.wantStructured)
			}
		})
	}
}

func TestAPIErrorError(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 400,
		Message:    "invalid input",
	}
	got := apiErr.Error()
	want := "api error (400): invalid input"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIErrorFormatConcise(t *testing.T) {
	apiErr := &APIError{
		StatusCode:   400,
		Message:      "validation failed",
		Code:         "VALIDATION_ERROR",
		RecoveryHint: "Fix the input fields",
		AutoFix: &AutoFixInfo{
			Command:     "make fix",
			Description: "Auto-fix the issue",
			Safe:        true,
		},
		ManualSteps: []string{"Check field A", "Update field B"},
	}

	formatted := apiErr.FormatConcise()

	// Check that all expected parts are present
	if !strings.Contains(formatted, "Error: validation failed") {
		t.Error("missing error message")
	}
	if !strings.Contains(formatted, "Code: VALIDATION_ERROR") {
		t.Error("missing code")
	}
	if !strings.Contains(formatted, "Recovery: Fix the input fields") {
		t.Error("missing recovery hint")
	}
	if !strings.Contains(formatted, "Auto-fix (safe):") {
		t.Error("missing auto-fix header with safe indicator")
	}
	if !strings.Contains(formatted, "make fix") {
		t.Error("missing auto-fix command")
	}
	if !strings.Contains(formatted, "Manual steps:") {
		t.Error("missing manual steps header")
	}
	if !strings.Contains(formatted, "1. Check field A") {
		t.Error("missing first manual step")
	}
	if !strings.Contains(formatted, "2. Update field B") {
		t.Error("missing second manual step")
	}
}

func TestHTTPClientReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":         "scenario not found",
			"code":          "SCENARIO_NOT_FOUND",
			"recovery_hint": "Verify the scenario name is correct",
		})
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{DefaultBase: server.URL},
	})

	_, err := client.Do("GET", "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
	if apiErr.Message != "scenario not found" {
		t.Errorf("Message = %q, want 'scenario not found'", apiErr.Message)
	}
	if apiErr.Code != "SCENARIO_NOT_FOUND" {
		t.Errorf("Code = %q, want 'SCENARIO_NOT_FOUND'", apiErr.Code)
	}
	if !apiErr.IsStructured() {
		t.Error("expected structured error")
	}
}
