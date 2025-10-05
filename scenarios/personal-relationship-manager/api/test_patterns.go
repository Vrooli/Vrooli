
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
			req := pattern.Execute(t, setupData)
			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			suite.Handler(w, httpReq)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Additional validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// Common error patterns that can be reused across handlers

// invalidContactIDPattern tests handlers with invalid contact ID formats
func invalidContactIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidContactID",
		Description:    "Test handler with invalid contact ID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-id"},
			}
		},
	}
}

// nonExistentContactPattern tests handlers with non-existent contact IDs
func nonExistentContactPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentContact",
		Description:    "Test handler with non-existent contact ID",
		ExpectedStatus: http.StatusNotFound,
		Setup: func(t *testing.T) interface{} {
			return setupTestDB(t)
		},
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": "999999"},
			}
		},
		Cleanup: func(setupData interface{}) {
			if testDB, ok := setupData.(*TestDatabase); ok && testDB != nil {
				testDB.Cleanup()
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	}
}

// emptyBodyPattern tests handlers with empty request body
func emptyBodyPattern(urlPath string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "",
			}
		},
	}
}

// missingRequiredFieldsPattern tests handlers with missing required fields
func missingRequiredFieldsPattern(urlPath string, method string, invalidBody interface{}) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   invalidBody,
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

// AddInvalidContactID adds invalid contact ID test pattern
func (b *TestScenarioBuilder) AddInvalidContactID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidContactIDPattern(urlPath))
	return b
}

// AddNonExistentContact adds non-existent contact test pattern
func (b *TestScenarioBuilder) AddNonExistentContact(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentContactPattern(urlPath, method))
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath, method))
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyBodyPattern(urlPath, method))
	return b
}

// AddMissingRequiredFields adds missing required fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(urlPath string, method string, invalidBody interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldsPattern(urlPath, method, invalidBody))
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

// ContactHandlerErrorPatterns generates error test patterns for contact handlers
func ContactHandlerErrorPatterns(contactID int) []ErrorTestPattern {
	_ = contactID // May be used in future patterns

	return NewTestScenarioBuilder().
		AddInvalidContactID("/api/contacts/invalid").
		AddNonExistentContact("/api/contacts/999999", "GET").
		AddInvalidJSON("/api/contacts", "POST").
		AddEmptyBody("/api/contacts", "POST").
		AddMissingRequiredFields("/api/contacts", "POST", map[string]interface{}{
			"email": "test@example.com", // Missing required name field
		}).
		AddCustom(ErrorTestPattern{
			Name:           "UpdateNonExistentContact",
			Description:    "Test updating a non-existent contact",
			ExpectedStatus: http.StatusInternalServerError,
			Setup: func(t *testing.T) interface{} {
				return setupTestDB(t)
			},
			Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method:  "PUT",
					Path:    "/api/contacts/999999",
					URLVars: map[string]string{"id": "999999"},
					Body: Contact{
						Name:  "Updated Name",
						Email: "updated@example.com",
					},
				}
			},
			Cleanup: func(setupData interface{}) {
				if testDB, ok := setupData.(*TestDatabase); ok && testDB != nil {
					testDB.Cleanup()
				}
			},
		}).
		Build()
}

// InteractionHandlerErrorPatterns generates error test patterns for interaction handlers
func InteractionHandlerErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidContactID("/api/contacts/invalid/interactions").
		AddNonExistentContact("/api/contacts/999999/interactions", "GET").
		AddInvalidJSON("/api/interactions", "POST").
		AddMissingRequiredFields("/api/interactions", "POST", map[string]interface{}{
			"description": "test", // Missing contact_id
		}).
		Build()
}

// GiftHandlerErrorPatterns generates error test patterns for gift handlers
func GiftHandlerErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidContactID("/api/contacts/invalid/gifts").
		AddNonExistentContact("/api/contacts/999999/gifts", "GET").
		AddInvalidJSON("/api/gifts", "POST").
		AddMissingRequiredFields("/api/gifts", "POST", map[string]interface{}{
			"description": "test", // Missing contact_id and gift_name
		}).
		Build()
}

// ProcessorHandlerErrorPatterns generates error test patterns for relationship processor handlers
func ProcessorHandlerErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidContactID("/api/contacts/invalid/enrich").
		AddNonExistentContact("/api/contacts/999999/enrich", "POST").
		AddNonExistentContact("/api/contacts/999999/insights", "GET").
		AddCustom(ErrorTestPattern{
			Name:           "InvalidBirthdayDaysParam",
			Description:    "Test birthday endpoint with invalid days_ahead parameter",
			ExpectedStatus: http.StatusOK, // Should still work with default
			Setup:          nil,
			Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method:      "GET",
					Path:        "/api/birthdays",
					QueryParams: map[string]string{"days_ahead": "invalid"},
				}
			},
		}).
		AddCustom(ErrorTestPattern{
			Name:           "GiftSuggestionInvalidBody",
			Description:    "Test gift suggestion with invalid request body",
			ExpectedStatus: http.StatusBadRequest,
			Setup: func(t *testing.T) interface{} {
				return setupTestDB(t)
			},
			Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method:  "POST",
					Path:    "/api/contacts/1/gifts/suggest",
					URLVars: map[string]string{"id": "1"},
					Body:    `{"invalid json`,
				}
			},
			Cleanup: func(setupData interface{}) {
				if testDB, ok := setupData.(*TestDatabase); ok && testDB != nil {
					testDB.Cleanup()
				}
			},
		}).
		Build()
}
