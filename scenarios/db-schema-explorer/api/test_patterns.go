// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

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
	Router      *mux.Router
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
			req := pattern.Execute(t, setupData)
			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler
			suite.Router.ServeHTTP(w, httpReq)

			// Validate
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(method, urlPath string, body interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredField",
		Description:    "Test handler with missing required field",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(method, urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// invalidQueryParameterPattern tests handlers with invalid query parameters
func invalidQueryParameterPattern(urlPath string, params map[string]string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidQueryParameter",
		Description:    "Test handler with invalid query parameter",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:      "GET",
				Path:        urlPath,
				QueryParams: params,
			}
		},
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

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(method, urlPath))
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, urlPath string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(method, urlPath, body))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(method, urlPath))
	return b
}

// AddInvalidQueryParameter adds invalid query parameter test pattern
func (b *TestScenarioBuilder) AddInvalidQueryParameter(urlPath string, params map[string]string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidQueryParameterPattern(urlPath, params))
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

// Common test scenarios for schema operations

// SchemaConnectErrorPatterns returns error patterns for schema connect endpoint
func SchemaConnectErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/schema/connect").
		AddEmptyBody("POST", "/api/v1/schema/connect").
		AddMissingRequiredField("POST", "/api/v1/schema/connect", map[string]string{
			"connection_string": "",
			// missing database_name
		}).
		Build()
}

// QueryGenerateErrorPatterns returns error patterns for query generation endpoint
func QueryGenerateErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/query/generate").
		AddEmptyBody("POST", "/api/v1/query/generate").
		AddMissingRequiredField("POST", "/api/v1/query/generate", map[string]interface{}{
			"database_context": "main",
			// missing natural_language
		}).
		Build()
}

// QueryExecuteErrorPatterns returns error patterns for query execution endpoint
func QueryExecuteErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/query/execute").
		AddEmptyBody("POST", "/api/v1/query/execute").
		AddMissingRequiredField("POST", "/api/v1/query/execute", map[string]interface{}{
			"database_name": "main",
			// missing sql
		}).
		AddCustom(ErrorTestPattern{
			Name:           "InvalidSQL",
			Description:    "Test with invalid SQL syntax",
			ExpectedStatus: http.StatusOK, // Returns success:false with error message
			Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/query/execute",
					Body: map[string]interface{}{
						"sql":           "INVALID SQL SYNTAX",
						"database_name": "main",
						"limit":         10,
					},
				}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				// Should return 200 with success:false
				response := assertJSONResponse(t, w, http.StatusOK, nil)
				if response != nil {
					if success, ok := response["success"].(bool); ok && success {
						t.Error("Expected success to be false for invalid SQL")
					}
					if _, ok := response["error"].(string); !ok {
						t.Error("Expected error message for invalid SQL")
					}
				}
			},
		}).
		Build()
}

// SchemaExportErrorPatterns returns error patterns for schema export endpoint
func SchemaExportErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/schema/export").
		AddEmptyBody("POST", "/api/v1/schema/export").
		AddMissingRequiredField("POST", "/api/v1/schema/export", map[string]interface{}{
			"format": "json",
			// missing database_name
		}).
		Build()
}

// SchemaDiffErrorPatterns returns error patterns for schema diff endpoint
func SchemaDiffErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/schema/diff").
		AddEmptyBody("POST", "/api/v1/schema/diff").
		AddMissingRequiredField("POST", "/api/v1/schema/diff", map[string]interface{}{
			"source": "db1",
			// missing target
		}).
		Build()
}

// LayoutSaveErrorPatterns returns error patterns for layout save endpoint
func LayoutSaveErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/layout/save").
		AddEmptyBody("POST", "/api/v1/layout/save").
		AddMissingRequiredField("POST", "/api/v1/layout/save", map[string]interface{}{
			"database_name": "main",
			"layout_type":   "graph",
			// missing name
		}).
		Build()
}

// QueryOptimizeErrorPatterns returns error patterns for query optimize endpoint
func QueryOptimizeErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/query/optimize").
		AddEmptyBody("POST", "/api/v1/query/optimize").
		AddMissingRequiredField("POST", "/api/v1/query/optimize", map[string]interface{}{
			"database_name": "main",
			// missing sql
		}).
		Build()
}

// EdgeCaseTestPattern defines edge case testing scenarios
type EdgeCaseTestPattern struct {
	Name        string
	Description string
	Execute     func(t *testing.T, server *TestServer)
}

// RunEdgeCaseTests executes edge case tests
func RunEdgeCaseTests(t *testing.T, server *TestServer, patterns []EdgeCaseTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			pattern.Execute(t, server)
		})
	}
}

// Common edge cases

// LargeLimitEdgeCase tests query execution with very large limit
func LargeLimitEdgeCase() EdgeCaseTestPattern {
	return EdgeCaseTestPattern{
		Name:        "LargeLimit",
		Description: "Test query execution with very large limit",
		Execute: func(t *testing.T, server *TestServer) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/query/execute",
				Body: map[string]interface{}{
					"sql":           "SELECT 1",
					"database_name": "main",
					"limit":         999999,
				},
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			server.Router.ServeHTTP(w, httpReq)

			// Should handle gracefully
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		},
	}
}

// EmptyDatabaseNameEdgeCase tests with empty database name
func EmptyDatabaseNameEdgeCase() EdgeCaseTestPattern {
	return EdgeCaseTestPattern{
		Name:        "EmptyDatabaseName",
		Description: "Test with empty database name",
		Execute: func(t *testing.T, server *TestServer) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/schema/connect",
				Body: map[string]interface{}{
					"connection_string": "",
					"database_name":     "",
				},
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			server.Router.ServeHTTP(w, httpReq)

			// Should use default or return error
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			if response != nil {
				dbName, _ := response["database_name"].(string)
				if dbName == "" {
					t.Error("Expected database_name to be set to default")
				}
			}
		},
	}
}

// SpecialCharactersEdgeCase tests with special characters in input
func SpecialCharactersEdgeCase() EdgeCaseTestPattern {
	return EdgeCaseTestPattern{
		Name:        "SpecialCharacters",
		Description: "Test with special characters in query",
		Execute: func(t *testing.T, server *TestServer) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/query/generate",
				Body: map[string]interface{}{
					"natural_language":    "show users with name containing <script>alert('xss')</script>",
					"database_context":    "main",
					"include_explanation": false,
				},
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			server.Router.ServeHTTP(w, httpReq)

			// Should handle special characters safely
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		},
	}
}
