package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// TestScenario represents a single test scenario
type TestScenario struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ValidateFunc   func(t *testing.T, response map[string]interface{})
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidJSON",
		Description: "Test handler with malformed JSON",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body:   `{"invalid": "json`,
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddEmptyData adds a test for empty data array
func (b *TestScenarioBuilder) AddEmptyData(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "EmptyData",
		Description: "Test handler with empty data array",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     "bar",
				"data":           []map[string]interface{}{},
				"export_formats": []string{"png"},
			},
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidChartType adds a test for invalid chart type
func (b *TestScenarioBuilder) AddInvalidChartType(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidChartType",
		Description: "Test handler with unsupported chart type",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     "invalid_type",
				"data":           generateTestChartData("bar", 3),
				"export_formats": []string{"png"},
			},
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredFields adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, missingField string) *TestScenarioBuilder {
	body := map[string]interface{}{}

	// Intentionally omit the specified field
	if missingField != "chart_type" {
		body["chart_type"] = "bar"
	}
	if missingField != "data" {
		body["data"] = generateTestChartData("bar", 3)
	}

	b.scenarios = append(b.scenarios, TestScenario{
		Name:        fmt.Sprintf("Missing_%s", missingField),
		Description: fmt.Sprintf("Test handler with missing %s field", missingField),
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body:   body,
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidDataStructure adds a test for invalid data structure
func (b *TestScenarioBuilder) AddInvalidDataStructure(path string, chartType string) *TestScenarioBuilder {
	// Create data that doesn't match the chart type requirements
	var invalidData []map[string]interface{}

	switch chartType {
	case "pie":
		// Pie chart needs name/value, but we'll give it x/y
		invalidData = []map[string]interface{}{
			{"x": "A", "y": 10},
		}
	case "candlestick":
		// Candlestick needs date, open, high, low, close
		invalidData = []map[string]interface{}{
			{"x": "A", "y": 10},
		}
	case "heatmap":
		// Heatmap needs x, y, value
		invalidData = []map[string]interface{}{
			{"x": "A"},
		}
	default:
		// Bar/line needs x and y
		invalidData = []map[string]interface{}{
			{"label": "A"},
		}
	}

	b.scenarios = append(b.scenarios, TestScenario{
		Name:        fmt.Sprintf("InvalidDataStructure_%s", chartType),
		Description: fmt.Sprintf("Test %s chart with invalid data structure", chartType),
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     chartType,
				"data":           invalidData,
				"export_formats": []string{"png"},
			},
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidExportFormat adds a test for invalid export format
func (b *TestScenarioBuilder) AddInvalidExportFormat(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "InvalidExportFormat",
		Description: "Test handler with unsupported export format",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     "bar",
				"data":           generateTestChartData("bar", 3),
				"export_formats": []string{"invalid_format"},
			},
		},
		ExpectedStatus: http.StatusCreated, // Should succeed but filter invalid formats
		ValidateFunc: func(t *testing.T, response map[string]interface{}) {
			// Verify that invalid format was filtered out
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if formats, ok := metadata["formats_generated"].([]interface{}); ok {
					if len(formats) > 0 {
						// Should have defaulted to png
						if formats[0] != "png" {
							t.Errorf("Expected default format 'png', got %v", formats[0])
						}
					}
				}
			}
		},
	})
	return b
}

// AddExtremeValues adds tests for extreme input values
func (b *TestScenarioBuilder) AddExtremeValues(path string) *TestScenarioBuilder {
	// Very large width
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "ExtremeWidth",
		Description: "Test with extremely large width value",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     "bar",
				"data":           generateTestChartData("bar", 3),
				"width":          10000,
				"height":         600,
				"export_formats": []string{"png"},
			},
		},
		ExpectedStatus: http.StatusCreated,
		ValidateFunc: func(t *testing.T, response map[string]interface{}) {
			// Width should be constrained to max (2000)
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if dimensions, ok := metadata["dimensions"].(map[string]interface{}); ok {
					if width, ok := dimensions["width"].(float64); ok {
						if width > 2000 {
							t.Errorf("Width should be constrained to 2000, got %v", width)
						}
					}
				}
			}
		},
	})

	// Very small dimensions
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        "MinimumDimensions",
		Description: "Test with very small dimensions",
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     "bar",
				"data":           generateTestChartData("bar", 3),
				"width":          50,
				"height":         50,
				"export_formats": []string{"png"},
			},
		},
		ExpectedStatus: http.StatusCreated,
		ValidateFunc: func(t *testing.T, response map[string]interface{}) {
			// Dimensions should be constrained to min (200)
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if dimensions, ok := metadata["dimensions"].(map[string]interface{}); ok {
					if width, ok := dimensions["width"].(float64); ok {
						if width < 200 {
							t.Errorf("Width should be at least 200, got %v", width)
						}
					}
				}
			}
		},
	})

	return b
}

// AddLargeDataset adds a test with large dataset
func (b *TestScenarioBuilder) AddLargeDataset(path string, size int) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:        fmt.Sprintf("LargeDataset_%d_points", size),
		Description: fmt.Sprintf("Test with %d data points", size),
		Request: HTTPTestRequest{
			Method: "POST",
			Path:   path,
			Body: map[string]interface{}{
				"chart_type":     "line",
				"data":           generateTestChartData("line", size),
				"export_formats": []string{"png"},
			},
		},
		ExpectedStatus: http.StatusCreated,
		ValidateFunc: func(t *testing.T, response map[string]interface{}) {
			assertResponseHasField(t, response, "chart_id")
			assertResponseHasField(t, response, "files")
		},
	})
	return b
}

// Build returns all the scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines systematic error testing
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, rr *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
}

// RunScenarios executes all test scenarios
func (suite *HandlerTestSuite) RunScenarios(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			rr, err := makeHTTPRequest(scenario.Request)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Execute handler
			suite.Handler(rr, mustCreateRequest(t, scenario.Request))

			// Validate status code
			if rr.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, rr.Code, rr.Body.String())
			}

			// Parse response if JSON
			var response map[string]interface{}
			if rr.Header().Get("Content-Type") == "application/json" {
				response = assertJSONResponse(t, rr, scenario.ExpectedStatus)
			}

			// Run custom validation if provided
			if scenario.ValidateFunc != nil && response != nil {
				scenario.ValidateFunc(t, response)
			}
		})
	}
}

// mustCreateRequest creates an http.Request for testing (panics on error)
func mustCreateRequest(t *testing.T, req HTTPTestRequest) *http.Request {
	var body []byte
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			body = []byte(v)
		case []byte:
			body = v
		}
	}

	// Use httptest.NewRequest to create proper request
	httpReq := httptest.NewRequest(req.Method, req.Path, nil)
	if len(body) > 0 {
		httpReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}
	return httpReq
}
