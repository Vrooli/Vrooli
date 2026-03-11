// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// Package testutil tests verify the test utilities work correctly.
// These meta-tests ensure our testing infrastructure is reliable.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAssertStatus verifies the status assertion helper.
func TestAssertStatus(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   int
		expectedCode   int
		shouldPass     bool
		category       string
	}{
		{
			name:         "matching_status",
			responseCode: http.StatusOK,
			expectedCode: http.StatusOK,
			shouldPass:   true,
			category:     "happy_path",
		},
		{
			name:         "created_status",
			responseCode: http.StatusCreated,
			expectedCode: http.StatusCreated,
			shouldPass:   true,
			category:     "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tc.responseCode)

			// This is a "happy path only" test - we only test successful assertions
			if tc.shouldPass {
				AssertStatus(t, rec, tc.expectedCode)
			}
		})
	}
}

// TestAssertContentType verifies the content type assertion helper.
func TestAssertContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		expected     string
		shouldPass   bool
		category     string
	}{
		{
			name:        "exact_match",
			contentType: "application/json",
			expected:    "application/json",
			shouldPass:  true,
			category:    "happy_path",
		},
		{
			name:        "prefix_match",
			contentType: "application/json; charset=utf-8",
			expected:    "application/json",
			shouldPass:  true,
			category:    "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", tc.contentType)

			if tc.shouldPass {
				AssertContentType(t, rec, tc.expected)
			}
		})
	}
}

// TestMakeRequest verifies the request factory helper.
func TestMakeRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantMethod string
		category   string
	}{
		{
			name:       "get_request",
			method:     http.MethodGet,
			path:       "/api/v1/test",
			wantMethod: http.MethodGet,
			category:   "happy_path",
		},
		{
			name:       "post_request",
			method:     http.MethodPost,
			path:       "/api/v1/test",
			wantMethod: http.MethodPost,
			category:   "happy_path",
		},
		{
			name:       "empty_method_defaults_to_get",
			method:     "",
			path:       "/api/v1/test",
			wantMethod: http.MethodGet,
			category:   "edge_case",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			req := MakeRequest(t, tc.method, tc.path, nil)

			// ASSERT
			if req.Method != tc.wantMethod {
				t.Errorf("expected method %q, got %q", tc.wantMethod, req.Method)
			}
			if req.URL.Path != tc.path {
				t.Errorf("expected path %q, got %q", tc.path, req.URL.Path)
			}
			if ct := req.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", ct)
			}
		})
	}
}

// TestMakeJSONRequest verifies the JSON request factory helper.
func TestMakeJSONRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		category string
	}{
		{
			name:     "nil_body",
			method:   http.MethodPost,
			path:     "/api/v1/test",
			body:     nil,
			category: "boundary",
		},
		{
			name:     "map_body",
			method:   http.MethodPost,
			path:     "/api/v1/test",
			body:     map[string]string{"key": "value"},
			category: "happy_path",
		},
		{
			name:     "struct_body",
			method:   http.MethodPost,
			path:     "/api/v1/test",
			body:     struct{ Name string }{"test"},
			category: "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			req := MakeJSONRequest(t, tc.method, tc.path, tc.body)

			// ASSERT
			if req.Method != tc.method {
				t.Errorf("expected method %q, got %q", tc.method, req.Method)
			}
		})
	}
}

// TestStringPtr verifies the string pointer helper.
func TestStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		category string
	}{
		{
			name:     "non_empty_string",
			input:    "test value",
			category: "happy_path",
		},
		{
			name:     "empty_string",
			input:    "",
			category: "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			result := StringPtr(tc.input)

			// ASSERT
			if result == nil {
				t.Fatal("expected non-nil pointer")
			}
			if *result != tc.input {
				t.Errorf("expected %q, got %q", tc.input, *result)
			}
		})
	}
}

// TestMustParseJSON verifies the JSON parsing helper.
func TestMustParseJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantName string
		category string
	}{
		{
			name:     "simple_object",
			json:     `{"name":"test"}`,
			wantName: "test",
			category: "happy_path",
		},
		{
			name:     "nested_object",
			json:     `{"name":"nested","extra":"value"}`,
			wantName: "nested",
			category: "happy_path",
		},
	}

	type testStruct struct {
		Name string `json:"name"`
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			result := MustParseJSON[testStruct](t, tc.json)

			// ASSERT
			if result.Name != tc.wantName {
				t.Errorf("expected name %q, got %q", tc.wantName, result.Name)
			}
		})
	}
}

// TestAssertJSON verifies the JSON assertion helper.
func TestAssertJSON(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantKey  string
		category string
	}{
		{
			name:     "valid_json",
			body:     `{"key":"value"}`,
			wantKey:  "value",
			category: "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteString(tc.body)

			var result map[string]string
			AssertJSON(t, rec, &result)

			if result["key"] != tc.wantKey {
				t.Errorf("expected key %q, got %q", tc.wantKey, result["key"])
			}
		})
	}
}
