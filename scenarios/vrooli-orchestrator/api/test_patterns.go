package main

import (
	"fmt"
	"net/http"
	"testing"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ErrorSubstring string
}

// TestScenarioBuilder helps build comprehensive test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	// We'll use a special marker to indicate invalid JSON in tests
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "InvalidJSON",
		Description: "Test with malformed JSON request body",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "invalid-json-string", // This will be handled specially in tests
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorSubstring: "JSON",
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        fmt.Sprintf("MissingRequired_%s", fieldName),
		Description: fmt.Sprintf("Test with missing required field: %s", fieldName),
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   map[string]interface{}{}, // Empty body
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorSubstring: "required",
	})
	return b
}

// AddNonExistentProfile adds a test for non-existent profile
func (b *TestScenarioBuilder) AddNonExistentProfile(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "NonExistentProfile",
		Description: "Test with non-existent profile name",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
		ExpectedStatus: http.StatusNotFound,
		ErrorSubstring: "not found",
	})
	return b
}

// AddEmptyProfileName adds a test for empty profile name in URL
func (b *TestScenarioBuilder) AddEmptyProfileName(method, pathTemplate string) *TestScenarioBuilder {
	// Replace {profileName} with empty string
	path := pathTemplate
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "EmptyProfileName",
		Description: "Test with empty profile name",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorSubstring: "required",
	})
	return b
}

// AddDuplicateProfile adds a test for duplicate profile creation
func (b *TestScenarioBuilder) AddDuplicateProfile(path string, profileName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "DuplicateProfile",
		Description: "Test creating a profile with duplicate name",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"name":         profileName,
				"display_name": "Duplicate Test",
			},
		},
		ExpectedStatus: http.StatusConflict,
		ErrorSubstring: "already exists",
	})
	return b
}

// AddDeleteActiveProfile adds a test for deleting an active profile
func (b *TestScenarioBuilder) AddDeleteActiveProfile(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "DeleteActiveProfile",
		Description: "Test deleting an active profile",
		Request: HTTPTestRequest{
			Method: "DELETE",
			Path:   path,
		},
		ExpectedStatus: http.StatusConflict,
		ErrorSubstring: "cannot delete active",
	})
	return b
}

// Build returns the built error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name        string
	Description string
	Setup       func(t *testing.T, env *TestEnvironment)
	Cleanup     func(t *testing.T, env *TestEnvironment)
}

// RunTests executes a suite of tests
func (suite *HandlerTestSuite) RunTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			if suite.Setup != nil {
				suite.Setup(t, env)
			}

			// Cleanup
			if suite.Cleanup != nil {
				defer suite.Cleanup(t, env)
			}

			// Execute request
			rr, err := makeHTTPRequest(env.Router, pattern.Request)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate response
			assertErrorResponse(t, rr, pattern.ExpectedStatus, pattern.ErrorSubstring)
		})
	}
}

// Common test patterns for reuse

// ProfileCRUDErrorPatterns returns common error patterns for profile CRUD operations
func ProfileCRUDErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddNonExistentProfile("GET", "/api/v1/profiles/nonexistent").
		AddNonExistentProfile("PUT", "/api/v1/profiles/nonexistent").
		AddNonExistentProfile("DELETE", "/api/v1/profiles/nonexistent").
		AddMissingRequiredField("POST", "/api/v1/profiles", "name").
		Build()
}

// ProfileActivationErrorPatterns returns error patterns for profile activation
func ProfileActivationErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddNonExistentProfile("POST", "/api/v1/profiles/nonexistent/activate").
		Build()
}
