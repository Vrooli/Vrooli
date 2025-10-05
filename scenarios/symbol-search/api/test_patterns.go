// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestScenario{},
	}
}

// ErrorTestScenario defines a systematic error test case
type ErrorTestScenario struct {
	Name           string
	Description    string
	Method         string
	Path           string
	QueryParams    map[string]string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
}

// AddInvalidCodepoint adds test for invalid codepoint format
func (b *TestScenarioBuilder) AddInvalidCodepoint(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidCodepoint",
		Description:    "Test with invalid codepoint format",
		Method:         "GET",
		Path:           endpoint,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid codepoint format",
	})
	return b
}

// AddNonExistentCharacter adds test for non-existent character
func (b *TestScenarioBuilder) AddNonExistentCharacter(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "NonExistentCharacter",
		Description:    "Test with non-existent character",
		Method:         "GET",
		Path:           endpoint,
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "Character not found",
	})
	return b
}

// AddInvalidQueryParams adds test for invalid query parameters
func (b *TestScenarioBuilder) AddInvalidQueryParams(endpoint string, params map[string]string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidQueryParams",
		Description:    "Test with invalid query parameters",
		Method:         "GET",
		Path:           endpoint,
		QueryParams:    params,
		ExpectedStatus: http.StatusOK, // May still return 200 with empty results
	})
	return b
}

// AddInvalidJSON adds test for malformed JSON request
func (b *TestScenarioBuilder) AddInvalidJSON(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON",
		Method:         "POST",
		Path:           endpoint,
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredFields adds test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "MissingRequiredFields",
		Description:    "Test with missing required fields",
		Method:         "POST",
		Path:           endpoint,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidRange adds test for invalid range parameters
func (b *TestScenarioBuilder) AddInvalidRange(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "InvalidRange",
		Description:    "Test with invalid range (start > end)",
		Method:         "POST",
		Path:           endpoint,
		Body: BulkRangeRequest{
			Ranges: []struct {
				Start  string `json:"start"`
				End    string `json:"end"`
				Format string `json:"format,omitempty"`
			}{
				{Start: "U+007F", End: "U+0000"}, // Invalid: start > end
			},
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddTooManyRanges adds test for exceeding range limits
func (b *TestScenarioBuilder) AddTooManyRanges(endpoint string) *TestScenarioBuilder {
	ranges := make([]struct {
		Start  string `json:"start"`
		End    string `json:"end"`
		Format string `json:"format,omitempty"`
	}, 11) // More than allowed limit of 10

	for i := 0; i < 11; i++ {
		ranges[i] = struct {
			Start  string `json:"start"`
			End    string `json:"end"`
			Format string `json:"format,omitempty"`
		}{
			Start: fmt.Sprintf("U+%04X", i*100),
			End:   fmt.Sprintf("U+%04X", i*100+50),
		}
	}

	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "TooManyRanges",
		Description:    "Test with too many ranges (>10)",
		Method:         "POST",
		Path:           endpoint,
		Body:           BulkRangeRequest{Ranges: ranges},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Must specify 1-10 ranges",
	})
	return b
}

// AddEmptyRanges adds test for empty ranges array
func (b *TestScenarioBuilder) AddEmptyRanges(endpoint string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:           "EmptyRanges",
		Description:    "Test with empty ranges array",
		Method:         "POST",
		Path:           endpoint,
		Body:           BulkRangeRequest{Ranges: []struct {
			Start  string `json:"start"`
			End    string `json:"end"`
			Format string `json:"format,omitempty"`
		}{}},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Must specify 1-10 ranges",
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// HandlerTestSuite provides comprehensive testing for HTTP handlers
type HandlerTestSuite struct {
	Name        string
	Router      *gin.Engine
	API         *API
	Cleanup     func()
	Scenarios   []ErrorTestScenario
}

// RunErrorTests executes all error test scenarios
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T) {
	for _, scenario := range suite.Scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:      scenario.Method,
				Path:        scenario.Path,
				QueryParams: scenario.QueryParams,
				Body:        scenario.Body,
			}

			w := makeHTTPRequest(suite.Router, req)

			if w.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, w.Code, w.Body.String())
			}

			// If error is expected, verify error message is present
			if scenario.ExpectedError != "" && w.Code >= 400 {
				assertErrorResponse(t, w, scenario.ExpectedStatus)
			}
		})
	}
}

// Performance test helpers

// BenchmarkConfig configures performance benchmark tests
type BenchmarkConfig struct {
	Name            string
	Description     string
	WarmupRounds    int
	MeasureRounds   int
	MaxResponseTime int // milliseconds
}

// PerformanceTestResult captures performance test metrics
type PerformanceTestResult struct {
	MinTime       int64
	MaxTime       int64
	AvgTime       int64
	P95Time       int64
	P99Time       int64
	TotalRequests int
	SuccessRate   float64
}

// EdgeCasePattern defines common edge case test scenarios
type EdgeCasePattern struct {
	Name        string
	Description string
	Test        func(t *testing.T, router *gin.Engine, api *API)
}

// CommonEdgeCases returns a set of common edge case tests
func CommonEdgeCases() []EdgeCasePattern {
	return []EdgeCasePattern{
		{
			Name:        "EmptyQuery",
			Description: "Test search with empty query string",
			Test: func(t *testing.T, router *gin.Engine, api *API) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/search",
					QueryParams: map[string]string{
						"q": "",
					},
				}
				w := makeHTTPRequest(router, req)
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for empty query, got %d", w.Code)
				}
			},
		},
		{
			Name:        "NegativeLimit",
			Description: "Test search with negative limit parameter",
			Test: func(t *testing.T, router *gin.Engine, api *API) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/search",
					QueryParams: map[string]string{
						"q":     "test",
						"limit": "-1",
					},
				}
				w := makeHTTPRequest(router, req)
				// Should default to 100
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 with negative limit, got %d", w.Code)
				}
			},
		},
		{
			Name:        "ExcessiveLimit",
			Description: "Test search with limit exceeding maximum",
			Test: func(t *testing.T, router *gin.Engine, api *API) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/search",
					QueryParams: map[string]string{
						"q":     "test",
						"limit": "10000",
					},
				}
				w := makeHTTPRequest(router, req)
				// Should cap at 1000
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 with excessive limit, got %d", w.Code)
				}
			},
		},
		{
			Name:        "NegativeOffset",
			Description: "Test search with negative offset parameter",
			Test: func(t *testing.T, router *gin.Engine, api *API) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/search",
					QueryParams: map[string]string{
						"q":      "test",
						"offset": "-1",
					},
				}
				w := makeHTTPRequest(router, req)
				// Should default to 0
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 with negative offset, got %d", w.Code)
				}
			},
		},
		{
			Name:        "SpecialCharactersInQuery",
			Description: "Test search with special characters",
			Test: func(t *testing.T, router *gin.Engine, api *API) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/search",
					QueryParams: map[string]string{
						"q": "!@#$%^&*()",
					},
				}
				w := makeHTTPRequest(router, req)
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 with special chars, got %d", w.Code)
				}
			},
		},
		{
			Name:        "UnicodeInQuery",
			Description: "Test search with Unicode characters",
			Test: func(t *testing.T, router *gin.Engine, api *API) {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/search",
					QueryParams: map[string]string{
						"q": "你好世界",
					},
				}
				w := makeHTTPRequest(router, req)
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 with Unicode, got %d", w.Code)
				}
			},
		},
	}
}
