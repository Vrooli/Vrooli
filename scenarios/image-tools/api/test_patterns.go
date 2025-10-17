package main

import (
	"bytes"
	"fmt"
	"testing"
)

// ErrorTestScenario defines a systematic approach to testing error conditions
type ErrorTestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	Body           *bytes.Buffer
	ContentType    string
	ExpectedStatus int
	ExpectedError  string
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestScenario, 0),
	}
}

// AddMissingImage adds a test scenario for missing image file
func (b *TestScenarioBuilder) AddMissingImage(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "MissingImage",
		Description:    "Request without image file should return 400",
		Method:         "POST",
		Path:           path,
		Body:           bytes.NewBuffer([]byte{}),
		ContentType:    "application/json",
		ExpectedStatus: 400,
		ExpectedError:  "error",
	})
	return b
}

// AddInvalidFormat adds a test scenario for invalid/unsupported image format
func (b *TestScenarioBuilder) AddInvalidFormat(path string) *TestScenarioBuilder {
	body, contentType := createSimpleMultipart("image", "test.bmp", []byte("invalid format"))
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidFormat",
		Description:    "Unsupported image format should return 400",
		Method:         "POST",
		Path:           path,
		Body:           body,
		ContentType:    contentType,
		ExpectedStatus: 400,
		ExpectedError:  "error",
	})
	return b
}

// AddInvalidJSON adds a test scenario for malformed JSON in request
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidJSON",
		Description:    "Malformed JSON should return 400",
		Method:         "POST",
		Path:           path,
		Body:           bytes.NewBuffer([]byte("{invalid json")),
		ContentType:    "application/json",
		ExpectedStatus: 400,
		ExpectedError:  "error",
	})
	return b
}

// AddMissingParameter adds a test scenario for missing required parameter
func (b *TestScenarioBuilder) AddMissingParameter(path, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           fmt.Sprintf("Missing%s", paramName),
		Description:    fmt.Sprintf("Request without %s should return 400", paramName),
		Method:         "GET",
		Path:           path,
		Body:           nil,
		ContentType:    "",
		ExpectedStatus: 400,
		ExpectedError:  "error",
	})
	return b
}

// AddNonExistentResource adds a test scenario for accessing non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "NonExistentResource",
		Description:    "Request for non-existent resource should return 404",
		Method:         "GET",
		Path:           path,
		Body:           nil,
		ContentType:    "",
		ExpectedStatus: 404,
		ExpectedError:  "error",
	})
	return b
}

// AddInvalidQuality adds a test scenario for invalid quality parameter
func (b *TestScenarioBuilder) AddInvalidQuality(path string) *TestScenarioBuilder {
	body, contentType := createSimpleMultipartWithFields("image", "test.jpg", generateTestImageData("jpeg"), map[string]string{
		"quality": "invalid",
	})
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidQuality",
		Description:    "Invalid quality parameter should use default",
		Method:         "POST",
		Path:           path,
		Body:           body,
		ContentType:    contentType,
		ExpectedStatus: 200, // Should succeed with default quality
		ExpectedError:  "",
	})
	return b
}

// AddInvalidDimensions adds a test scenario for invalid resize dimensions
func (b *TestScenarioBuilder) AddInvalidDimensions(path string) *TestScenarioBuilder {
	body, contentType := createSimpleMultipartWithFields("image", "test.jpg", generateTestImageData("jpeg"), map[string]string{
		"width":  "-1",
		"height": "-1",
	})
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidDimensions",
		Description:    "Invalid dimensions should return 400",
		Method:         "POST",
		Path:           path,
		Body:           body,
		ContentType:    contentType,
		ExpectedStatus: 400,
		ExpectedError:  "error",
	})
	return b
}

// AddEmptyBatch adds a test scenario for empty batch request
func (b *TestScenarioBuilder) AddEmptyBatch(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "EmptyBatch",
		Description:    "Batch request with no images should return 400",
		Method:         "POST",
		Path:           path,
		Body:           bytes.NewBuffer([]byte{}),
		ContentType:    "multipart/form-data",
		ExpectedStatus: 400,
		ExpectedError:  "error",
	})
	return b
}

// Build returns the constructed test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// ErrorTestPattern provides systematic error testing
type ErrorTestPattern struct {
	scenarios []ErrorTestScenario
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern() *ErrorTestPattern {
	return &ErrorTestPattern{
		scenarios: make([]ErrorTestScenario, 0),
	}
}

// RunTests executes all test scenarios
func (p *ErrorTestPattern) RunTests(t *testing.T, server *Server, scenarios []ErrorTestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			headers := make(map[string]string)
			if scenario.ContentType != "" {
				headers["Content-Type"] = scenario.ContentType
			}

			var body *bytes.Buffer
			if scenario.Body != nil {
				body = scenario.Body
			} else {
				body = bytes.NewBuffer([]byte{})
			}

			resp, respBody, err := makeHTTPRequest(server.app, scenario.Method, scenario.Path, body, headers)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			if resp.StatusCode != scenario.ExpectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					scenario.Description, scenario.ExpectedStatus, resp.StatusCode, string(respBody))
			}

			if scenario.ExpectedError != "" && scenario.ExpectedStatus >= 400 {
				result := assertJSONResponse(t, resp, respBody, scenario.ExpectedStatus)
				if _, ok := result[scenario.ExpectedError]; !ok {
					t.Errorf("Expected '%s' key in error response, got: %v", scenario.ExpectedError, result)
				}
			}
		})
	}
}

// Helper function to create simple multipart data
func createSimpleMultipart(fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	return createSimpleMultipartWithFields(fieldName, filename, content, nil)
}

// Helper function to create multipart data with form fields
func createSimpleMultipartWithFields(fieldName, filename string, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	boundary := "----TestBoundary"

	// Write file
	body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	body.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldName, filename))
	body.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	body.Write(content)
	body.WriteString("\r\n")

	// Write fields
	for key, value := range fields {
		body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		body.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"\r\n\r\n", key))
		body.WriteString(value)
		body.WriteString("\r\n")
	}

	body.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
	return body, contentType
}

// HandlerTestSuite provides comprehensive handler testing
type HandlerTestSuite struct {
	Name        string
	Method      string
	Path        string
	Description string
}

// TestSuccessCase tests the happy path for a handler
func (s *HandlerTestSuite) TestSuccessCase(t *testing.T, server *Server, setupData interface{}) {
	t.Run(fmt.Sprintf("%s_Success", s.Name), func(t *testing.T) {
		// Test success cases are implemented in main_test.go
	})
}

// TestErrorCases tests various error conditions for a handler
func (s *HandlerTestSuite) TestErrorCases(t *testing.T, server *Server, scenarios []ErrorTestScenario) {
	pattern := NewErrorTestPattern()
	pattern.RunTests(t, server, scenarios)
}
