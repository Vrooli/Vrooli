// +build testing

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
	Request        HTTPTestRequest
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{"id": "invalid-uuid-format"},
		},
	})
	return b
}

// AddNonExistentMindMap adds a test for non-existent mind map
func (b *TestScenarioBuilder) AddNonExistentMindMap(path string, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentMindMap",
		Description:    "Test handler with non-existent mind map ID",
		ExpectedStatus: http.StatusNotFound,
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{"id": nonExistentID},
		},
	})
	return b
}

// AddInvalidJSON adds a test for invalid JSON payload
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "invalid json {{{",
		},
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, fieldName string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
	})
	return b
}

// AddEmptyBody adds a test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   nil,
		},
	})
	return b
}

// AddNonExistentNode adds a test for non-existent node
func (b *TestScenarioBuilder) AddNonExistentNode(path string, method string, mapID string) *TestScenarioBuilder {
	nonExistentNodeID := uuid.New().String()
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentNode",
		Description:    "Test handler with non-existent node ID",
		ExpectedStatus: http.StatusNotFound,
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			URLVars: map[string]string{
				"mapId":  mapID,
				"nodeId": nonExistentNodeID,
			},
		},
	})
	return b
}

// AddInvalidMethod adds a test for invalid HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(path string, invalidMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidMethod",
		Description:    "Test handler with invalid HTTP method",
		ExpectedStatus: http.StatusMethodNotAllowed,
		Request: HTTPTestRequest{
			Method: invalidMethod,
			Path:   path,
		},
	})
	return b
}

// Build returns the accumulated test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern, handler http.HandlerFunc) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			w, err := makeHTTPRequest(pattern.Request, handler)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, w.Code)
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

// Common test data builders

// buildCreateMindMapRequest builds a valid create mind map request
func buildCreateMindMapRequest(title, description, userID string) map[string]interface{} {
	return map[string]interface{}{
		"title":       title,
		"description": description,
		"userId":      userID,
		"isPublic":    false,
		"tags":        []string{},
		"metadata":    map[string]interface{}{},
	}
}

// buildUpdateMindMapRequest builds a valid update mind map request
func buildUpdateMindMapRequest(title, description string) map[string]interface{} {
	return map[string]interface{}{
		"title":       title,
		"description": description,
	}
}

// buildCreateNodeRequest builds a valid create node request
func buildCreateNodeRequest(content, nodeType string) map[string]interface{} {
	return map[string]interface{}{
		"content":    content,
		"type":       nodeType,
		"position_x": 100.0,
		"position_y": 200.0,
		"metadata":   map[string]interface{}{},
	}
}

// buildUpdateNodeRequest builds a valid update node request
func buildUpdateNodeRequest(content string, posX, posY float64) map[string]interface{} {
	return map[string]interface{}{
		"content":    content,
		"position_x": posX,
		"position_y": posY,
	}
}

// buildSearchRequest builds a valid search request
func buildSearchRequest(query, mode string) map[string]interface{} {
	return map[string]interface{}{
		"query": query,
		"mode":  mode,
	}
}

// buildOrganizeRequest builds a valid organize request
func buildOrganizeRequest(mindMapID, method string) map[string]interface{} {
	return map[string]interface{}{
		"mind_map_id": mindMapID,
		"method":      method,
	}
}

// buildDocumentToMindMapRequest builds a valid document to mind map request
func buildDocumentToMindMapRequest(title, content, docType, userID string) map[string]interface{} {
	return map[string]interface{}{
		"title":            title,
		"document_content": content,
		"document_type":    docType,
		"user_id":          userID,
	}
}
