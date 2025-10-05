package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// SecurityTestPattern defines security-focused test scenarios
type SecurityTestPattern struct {
	Name           string
	Description    string
	Payload        string
	ExpectedStatus int
	ShouldBlock    bool
	Category       string // "sql-injection", "xss", "path-traversal", "command-injection"
}

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
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
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			reqData := pattern.Execute(t, setupData)
			req, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			w := executeRequest(suite.Handler, req)

			// Validate
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, w.Code)
			}

			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// RunSecurityTests executes security-focused test patterns
func (suite *HandlerTestSuite) RunSecurityTests(t *testing.T, patterns []SecurityTestPattern, testFunc func(*testing.T, SecurityTestPattern)) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_Security_%s_%s", suite.HandlerName, pattern.Category, pattern.Name), func(t *testing.T) {
			testFunc(t, pattern)
		})
	}
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidUUID adds invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "invalid")
		},
	})
	return b
}

// AddNonExistentResource adds non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(pathTemplate, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			path := fmt.Sprintf(pathTemplate, nonExistentID)
			return HTTPTestRequest{
				Method: method,
				Path:   path,
			}
		},
	})
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddMissingAuth adds missing authentication test pattern
func (b *TestScenarioBuilder) AddMissingAuth(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingAuthentication",
		Description:    "Test handler without authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
			}
		},
	})
	return b
}

// AddInvalidToken adds invalid token test pattern
func (b *TestScenarioBuilder) AddInvalidToken(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidToken",
		Description:    "Test handler with invalid authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: createAuthHeader("invalid.token.here"),
			}
		},
	})
	return b
}

// AddInsufficientPermissions adds insufficient permissions test pattern
func (b *TestScenarioBuilder) AddInsufficientPermissions(path, method string, userRole string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InsufficientPermissions",
		Description:    fmt.Sprintf("Test handler with %s role (insufficient permissions)", userRole),
		ExpectedStatus: http.StatusForbidden,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDatabase(t)
			user := createTestUser(t, "user@test.local", "Password123!", []string{userRole})
			return map[string]interface{}{
				"env":  env,
				"user": user,
			}
		},
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			data := setupData.(map[string]interface{})
			user := data["user"].(*TestUser)
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: createAuthHeader(user.Token),
			}
		},
		Cleanup: func(setupData interface{}) {
			if setupData != nil {
				data := setupData.(map[string]interface{})
				if user, ok := data["user"].(*TestUser); ok {
					user.Cleanup()
				}
				if env, ok := data["env"].(*TestEnvironment); ok {
					env.Cleanup()
				}
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds missing required fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(path, method string, invalidBody interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   invalidBody,
			}
		},
	})
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// SecurityTestBuilder provides a fluent interface for building security tests
type SecurityTestBuilder struct {
	patterns []SecurityTestPattern
}

// NewSecurityTestBuilder creates a new security test builder
func NewSecurityTestBuilder() *SecurityTestBuilder {
	return &SecurityTestBuilder{
		patterns: []SecurityTestPattern{},
	}
}

// AddSQLInjectionTests adds SQL injection test patterns
func (b *SecurityTestBuilder) AddSQLInjectionTests() *SecurityTestBuilder {
	payloads := GetSecurityTestPayloads().SQLInjection
	for i, payload := range payloads {
		b.patterns = append(b.patterns, SecurityTestPattern{
			Name:           fmt.Sprintf("SQLInjection_%d", i+1),
			Description:    "SQL injection attempt should be blocked",
			Payload:        payload,
			ExpectedStatus: http.StatusBadRequest,
			ShouldBlock:    true,
			Category:       "sql-injection",
		})
	}
	return b
}

// AddXSSTests adds XSS test patterns
func (b *SecurityTestBuilder) AddXSSTests() *SecurityTestBuilder {
	payloads := GetSecurityTestPayloads().XSS
	for i, payload := range payloads {
		b.patterns = append(b.patterns, SecurityTestPattern{
			Name:           fmt.Sprintf("XSS_%d", i+1),
			Description:    "XSS attempt should be blocked or sanitized",
			Payload:        payload,
			ExpectedStatus: http.StatusBadRequest,
			ShouldBlock:    true,
			Category:       "xss",
		})
	}
	return b
}

// AddPathTraversalTests adds path traversal test patterns
func (b *SecurityTestBuilder) AddPathTraversalTests() *SecurityTestBuilder {
	payloads := GetSecurityTestPayloads().PathTraversal
	for i, payload := range payloads {
		b.patterns = append(b.patterns, SecurityTestPattern{
			Name:           fmt.Sprintf("PathTraversal_%d", i+1),
			Description:    "Path traversal attempt should be blocked",
			Payload:        payload,
			ExpectedStatus: http.StatusBadRequest,
			ShouldBlock:    true,
			Category:       "path-traversal",
		})
	}
	return b
}

// AddCommandInjectionTests adds command injection test patterns
func (b *SecurityTestBuilder) AddCommandInjectionTests() *SecurityTestBuilder {
	payloads := GetSecurityTestPayloads().CommandInjection
	for i, payload := range payloads {
		b.patterns = append(b.patterns, SecurityTestPattern{
			Name:           fmt.Sprintf("CommandInjection_%d", i+1),
			Description:    "Command injection attempt should be blocked",
			Payload:        payload,
			ExpectedStatus: http.StatusBadRequest,
			ShouldBlock:    true,
			Category:       "command-injection",
		})
	}
	return b
}

// Build returns the configured security test patterns
func (b *SecurityTestBuilder) Build() []SecurityTestPattern {
	return b.patterns
}

// Common test pattern generators

// GenerateAuthenticationTests generates authentication test patterns for a handler
func GenerateAuthenticationTests(path, method string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth(path, method).
		AddInvalidToken(path, method).
		Build()
}

// GenerateValidationTests generates validation test patterns for a handler
func GenerateValidationTests(path, method string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON(path, method).
		AddMissingRequiredFields(path, method, map[string]interface{}{}).
		Build()
}

// GenerateResourceTests generates resource-based test patterns for a handler
func GenerateResourceTests(pathTemplate, method string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID(fmt.Sprintf(pathTemplate, "invalid-uuid"), method).
		AddNonExistentResource(pathTemplate, method).
		Build()
}

// GenerateSecurityTests generates comprehensive security test patterns
func GenerateSecurityTests() []SecurityTestPattern {
	return NewSecurityTestBuilder().
		AddSQLInjectionTests().
		AddXSSTests().
		Build()
}
