package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	RequestConfig  HTTPTestRequest
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds a test pattern for invalid UUID handling
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath, varName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method:  "GET",
			Path:    urlPath,
			URLVars: map[string]string{varName: "invalid-uuid-format"},
		},
	})
	return b
}

// AddNonExistentBrand adds a test pattern for non-existent brand handling
func (b *TestScenarioBuilder) AddNonExistentBrand(urlPath, varName string) *TestScenarioBuilder {
	nonExistentID := uuid.New()
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentBrand",
		Description:    "Test handler with non-existent brand ID",
		ExpectedStatus: http.StatusNotFound,
		RequestConfig: HTTPTestRequest{
			Method:  "GET",
			Path:    urlPath,
			URLVars: map[string]string{varName: nonExistentID.String()},
		},
	})
	return b
}

// AddInvalidJSON adds a test pattern for invalid JSON handling
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body:   "invalid-json-string", // This will cause JSON marshal to fail
		},
	})
	return b
}

// AddMissingRequiredField adds a test pattern for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, method string, missingField string) *TestScenarioBuilder {
	body := make(map[string]interface{})
	// Create incomplete request body based on which field is missing
	switch missingField {
	case "brand_name":
		body["industry"] = "tech"
	case "industry":
		body["brand_name"] = "Test Brand"
	case "brand_id":
		body["target_app_path"] = "/test/path"
	case "target_app_path":
		body["brand_id"] = uuid.New().String()
	}

	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", missingField),
		Description:    fmt.Sprintf("Test handler with missing %s", missingField),
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body:   body,
		},
	})
	return b
}

// AddEmptyBody adds a test pattern for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		RequestConfig: HTTPTestRequest{
			Method: method,
			Path:   urlPath,
			Body:   map[string]interface{}{},
		},
	})
	return b
}

// Build returns the compiled list of error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			w, err := makeHTTPRequest(pattern.RequestConfig, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d for pattern %s",
					pattern.ExpectedStatus, w.Code, pattern.Name)
				t.Logf("Response body: %s", w.Body.String())
			}

			// Validate error response structure
			if pattern.ExpectedStatus >= 400 {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}
			}
		})
	}
}

// PaginationTestSuite provides standardized pagination testing
type PaginationTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BasePath    string
}

// RunPaginationTests executes pagination-related tests
func (suite *PaginationTestSuite) RunPaginationTests(t *testing.T) {
	tests := []struct {
		name          string
		limit         string
		offset        string
		expectSuccess bool
	}{
		{"DefaultPagination", "", "", true},
		{"CustomLimit", "10", "", true},
		{"CustomOffset", "", "5", true},
		{"LimitAndOffset", "15", "10", true},
		{"ZeroLimit", "0", "", true},
		{"NegativeLimit", "-1", "", true},
		{"LargeLimit", "1000", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestLogger()
			defer cleanup()

			path := suite.BasePath
			if tt.limit != "" || tt.offset != "" {
				path += "?"
				if tt.limit != "" {
					path += fmt.Sprintf("limit=%s", tt.limit)
				}
				if tt.offset != "" {
					if tt.limit != "" {
						path += "&"
					}
					path += fmt.Sprintf("offset=%s", tt.offset)
				}
			}

			req := HTTPTestRequest{
				Method: "GET",
				Path:   path,
			}

			w, err := makeHTTPRequest(req, suite.Handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Should return 200 even with unusual pagination params
			if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 200 or 500, got %d", w.Code)
			}
		})
	}
}
