package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines systematic error test scenarios
type ErrorTestPattern struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	Description    string
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds test for invalid UUID parameter
func (b *TestScenarioBuilder) AddInvalidUUID(pathTemplate string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Invalid UUID",
		Method:         "GET",
		Path:           fmt.Sprintf(pathTemplate, "invalid-uuid-format"),
		Body:           nil,
		ExpectedStatus: http.StatusNotFound,
		Description:    "Should reject malformed UUID",
	})
	return b
}

// AddNonExistentFunnel adds test for non-existent funnel
func (b *TestScenarioBuilder) AddNonExistentFunnel(pathTemplate string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Non-existent Funnel",
		Method:         "GET",
		Path:           fmt.Sprintf(pathTemplate, nonExistentID),
		Body:           nil,
		ExpectedStatus: http.StatusNotFound,
		Description:    "Should return 404 for non-existent funnel",
	})
	return b
}

// AddInvalidJSON adds test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Malformed JSON",
		Method:         "POST",
		Path:           path,
		Body:           "this is not valid json",
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject malformed JSON",
	})
	return b
}

// AddMissingRequiredField adds test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, incompleteData map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Missing Required Field",
		Method:         "POST",
		Path:           path,
		Body:           incompleteData,
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject request with missing required fields",
	})
	return b
}

// AddEmptyBody adds test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Empty Request Body",
		Method:         "POST",
		Path:           path,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should reject empty request body",
	})
	return b
}

// AddNullValues adds test for null value handling
func (b *TestScenarioBuilder) AddNullValues(path string, dataWithNulls map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Null Values",
		Method:         "POST",
		Path:           path,
		Body:           dataWithNulls,
		ExpectedStatus: http.StatusBadRequest,
		Description:    "Should handle null values appropriately",
	})
	return b
}

// AddInvalidMethod adds test for wrong HTTP method
func (b *TestScenarioBuilder) AddInvalidMethod(path string, wrongMethod string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Invalid HTTP Method",
		Method:         wrongMethod,
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusMethodNotAllowed,
		Description:    "Should reject invalid HTTP method",
	})
	return b
}

// AddUnauthorized adds test for unauthorized access
func (b *TestScenarioBuilder) AddUnauthorized(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Unauthorized Access",
		Method:         "GET",
		Path:           path,
		Body:           nil,
		ExpectedStatus: http.StatusUnauthorized,
		Description:    "Should reject unauthorized access",
	})
	return b
}

// AddDuplicateSlug adds test for duplicate slug
func (b *TestScenarioBuilder) AddDuplicateSlug(path string, duplicateData map[string]interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "Duplicate Slug",
		Method:         "POST",
		Path:           path,
		Body:           duplicateData,
		ExpectedStatus: http.StatusInternalServerError,
		Description:    "Should handle duplicate slug appropriately",
	})
	return b
}

// Build returns the completed pattern list
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides comprehensive HTTP handler testing
type HandlerTestSuite struct {
	Server  *Server
	T       *testing.T
	Cleanup func()
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T) *HandlerTestSuite {
	testServer := setupTestServer(t)
	if testServer == nil {
		t.Skip("Test server not available")
		return nil
	}

	return &HandlerTestSuite{
		Server:  testServer.Server,
		T:       t,
		Cleanup: testServer.Cleanup,
	}
}

// TestEndpoint tests a single endpoint with given pattern
func (suite *HandlerTestSuite) TestEndpoint(pattern ErrorTestPattern) {
	suite.T.Run(pattern.Name, func(t *testing.T) {
		req, err := makeHTTPRequest(pattern.Method, pattern.Path, pattern.Body)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		recorder := httptest.NewRecorder()
		suite.Server.router.ServeHTTP(recorder, req)

		if recorder.Code != pattern.ExpectedStatus {
			t.Errorf("%s: Expected status %d, got %d. Body: %s",
				pattern.Description, pattern.ExpectedStatus, recorder.Code, recorder.Body.String())
		}
	})
}

// TestAllPatterns executes all patterns in the suite
func (suite *HandlerTestSuite) TestAllPatterns(patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		suite.TestEndpoint(pattern)
	}
}

// EdgeCaseTestBuilder builds edge case test scenarios
type EdgeCaseTestBuilder struct {
	scenarios []ErrorTestPattern
}

// NewEdgeCaseTestBuilder creates a new edge case builder
func NewEdgeCaseTestBuilder() *EdgeCaseTestBuilder {
	return &EdgeCaseTestBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddEmptyString adds test for empty string handling
func (b *EdgeCaseTestBuilder) AddEmptyString(path string, field string) *EdgeCaseTestBuilder {
	data := map[string]interface{}{
		field: "",
	}
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Empty %s", field),
		Method:         "POST",
		Path:           path,
		Body:           data,
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Should reject empty %s", field),
	})
	return b
}

// AddVeryLongString adds test for oversized input
func (b *EdgeCaseTestBuilder) AddVeryLongString(path string, field string) *EdgeCaseTestBuilder {
	longString := make([]byte, 10000)
	for i := range longString {
		longString[i] = 'a'
	}
	data := map[string]interface{}{
		field: string(longString),
	}
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Very Long %s", field),
		Method:         "POST",
		Path:           path,
		Body:           data,
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Should handle very long %s", field),
	})
	return b
}

// AddSpecialCharacters adds test for special character handling
func (b *EdgeCaseTestBuilder) AddSpecialCharacters(path string, field string) *EdgeCaseTestBuilder {
	data := map[string]interface{}{
		field: "!@#$%^&*()[]{}|\\;:'\",<>?/",
	}
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Special Characters in %s", field),
		Method:         "POST",
		Path:           path,
		Body:           data,
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Should handle special characters in %s", field),
	})
	return b
}

// AddUnicodeCharacters adds test for unicode handling
func (b *EdgeCaseTestBuilder) AddUnicodeCharacters(path string, field string) *EdgeCaseTestBuilder {
	data := map[string]interface{}{
		field: "测试 テスト 테스트 тест",
	}
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Unicode in %s", field),
		Method:         "POST",
		Path:           path,
		Body:           data,
		ExpectedStatus: http.StatusCreated,
		Description:    fmt.Sprintf("Should accept unicode in %s", field),
	})
	return b
}

// AddNegativeNumbers adds test for negative number handling
func (b *EdgeCaseTestBuilder) AddNegativeNumbers(path string, field string) *EdgeCaseTestBuilder {
	data := map[string]interface{}{
		field: -1,
	}
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Negative %s", field),
		Method:         "POST",
		Path:           path,
		Body:           data,
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Should reject negative %s", field),
	})
	return b
}

// AddZeroValue adds test for zero value handling
func (b *EdgeCaseTestBuilder) AddZeroValue(path string, field string) *EdgeCaseTestBuilder {
	data := map[string]interface{}{
		field: 0,
	}
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Zero %s", field),
		Method:         "POST",
		Path:           path,
		Body:           data,
		ExpectedStatus: http.StatusBadRequest,
		Description:    fmt.Sprintf("Should handle zero %s", field),
	})
	return b
}

// Build returns the completed scenarios
func (b *EdgeCaseTestBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}
