// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *http.Response, setupData interface{})
	Cleanup        func(setupData interface{})
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration
	Cleanup        func(setupData interface{})
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
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": "invalid-uuid"},
			}
		},
	})
	return b
}

// AddNonExistentCampaign adds non-existent campaign test pattern
func (b *TestScenarioBuilder) AddNonExistentCampaign(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentCampaign",
		Description:    "Test handler with non-existent campaign ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": uuid.New().String()},
			}
		},
	})
	return b
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds missing required fields test pattern
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, bodyTemplate map[string]interface{}, fieldToOmit string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldToOmit),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldToOmit),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			// Create a copy of bodyTemplate without the specified field
			body := make(map[string]interface{})
			for k, v := range bodyTemplate {
				if k != fieldToOmit {
					body[k] = v
				}
			}
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   body,
			}
		},
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   "",
			}
		},
	})
	return b
}

// AddCustomPattern adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the built test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common test data generators

// generateTestCampaignRequest generates a test campaign request
func generateTestCampaignRequest() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Test Campaign",
		"description": "Test campaign description",
		"color":       "#3B82F6",
	}
}

// generateTestIdeaRequest generates a test idea generation request
func generateTestIdeaRequest(campaignID string) map[string]interface{} {
	return map[string]interface{}{
		"campaign_id": campaignID,
		"prompt":      "Generate innovative ideas for testing",
		"count":       1,
	}
}

// generateTestSearchRequest generates a test semantic search request
func generateTestSearchRequest() map[string]interface{} {
	return map[string]interface{}{
		"query": "test search query",
		"limit": 5,
	}
}

// generateTestRefinementRequest generates a test refinement request
func generateTestRefinementRequest(ideaID string) map[string]interface{} {
	return map[string]interface{}{
		"idea_id":     ideaID,
		"refinement":  "Please improve this idea with more details",
		"user_id":     uuid.New().String(),
	}
}

// generateTestDocumentRequest generates a test document processing request
func generateTestDocumentRequest(campaignID string) map[string]interface{} {
	return map[string]interface{}{
		"document_id": uuid.New().String(),
		"campaign_id": campaignID,
		"file_path":   "/tmp/test-document.pdf",
	}
}

// Edge case test patterns

// createInvalidDataPatterns creates patterns for invalid data testing
func createInvalidDataPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "NullCampaignID",
			Description:    "Test with null campaign ID",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/ideas",
					Body: map[string]interface{}{
						"campaign_id": nil,
						"prompt":      "test",
						"count":       1,
					},
				}
			},
		},
		{
			Name:           "NegativeCount",
			Description:    "Test with negative count value",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/ideas",
					Body: map[string]interface{}{
						"campaign_id": uuid.New().String(),
						"prompt":      "test",
						"count":       -1,
					},
				}
			},
		},
		{
			Name:           "ExcessiveCount",
			Description:    "Test with excessive count value",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/ideas",
					Body: map[string]interface{}{
						"campaign_id": uuid.New().String(),
						"prompt":      "test",
						"count":       1000,
					},
				}
			},
		},
		{
			Name:           "EmptyPrompt",
			Description:    "Test with empty prompt",
			ExpectedStatus: http.StatusOK, // Should use default prompt
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
				campaign := setupData.(*TestCampaign)
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/ideas",
					Body: map[string]interface{}{
						"campaign_id": campaign.ID,
						"prompt":      "",
						"count":       1,
					},
				}
			},
			Setup: func(t *testing.T, env *TestEnvironment) interface{} {
				return createTestCampaign(t, env.DB, "empty-prompt-test")
			},
			Cleanup: func(setupData interface{}) {
				if campaign, ok := setupData.(*TestCampaign); ok {
					campaign.Cleanup()
				}
			},
		},
	}
}

// createBoundaryTestPatterns creates patterns for boundary testing
func createBoundaryTestPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "MaxPromptLength",
			Description:    "Test with maximum allowed prompt length",
			ExpectedStatus: http.StatusOK,
			Setup: func(t *testing.T, env *TestEnvironment) interface{} {
				return createTestCampaign(t, env.DB, "max-prompt-test")
			},
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
				campaign := setupData.(*TestCampaign)
				// Create a very long prompt (e.g., 10KB)
				longPrompt := ""
				for i := 0; i < 500; i++ {
					longPrompt += "Generate ideas for innovative solutions. "
				}
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/ideas",
					Body: map[string]interface{}{
						"campaign_id": campaign.ID,
						"prompt":      longPrompt[:10000],
						"count":       1,
					},
				}
			},
			Cleanup: func(setupData interface{}) {
				if campaign, ok := setupData.(*TestCampaign); ok {
					campaign.Cleanup()
				}
			},
		},
		{
			Name:           "ZeroLimit",
			Description:    "Test semantic search with zero limit",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "POST",
					Path:   "/search",
					Body: map[string]interface{}{
						"query": "test",
						"limit": 0,
					},
				}
			},
		},
	}
}

// ConcurrencyTestPattern defines concurrency testing scenarios
type ConcurrencyTestPattern struct {
	Name           string
	Description    string
	Concurrency    int
	Iterations     int
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}, iteration int) error
	Validate       func(t *testing.T, env *TestEnvironment, setupData interface{}, results []error)
	Cleanup        func(setupData interface{})
}

// createConcurrencyPatterns creates concurrency test patterns
func createConcurrencyPatterns() []ConcurrencyTestPattern {
	return []ConcurrencyTestPattern{
		{
			Name:        "ConcurrentIdeaGeneration",
			Description: "Test concurrent idea generation requests",
			Concurrency: 10,
			Iterations:  5,
			Setup: func(t *testing.T, env *TestEnvironment) interface{} {
				return createTestCampaign(t, env.DB, "concurrent-test")
			},
			Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}, iteration int) error {
				campaign := setupData.(*TestCampaign)
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/ideas",
					Body: map[string]interface{}{
						"campaign_id": campaign.ID,
						"prompt":      fmt.Sprintf("Concurrent test %d", iteration),
						"count":       1,
					},
				}
				w := makeHTTPRequest(env, req)
				if w.Code != http.StatusCreated && w.Code != http.StatusOK {
					return fmt.Errorf("request failed with status %d", w.Code)
				}
				return nil
			},
			Validate: func(t *testing.T, env *TestEnvironment, setupData interface{}, results []error) {
				errorCount := 0
				for _, err := range results {
					if err != nil {
						errorCount++
						t.Logf("Concurrent request error: %v", err)
					}
				}
				// Allow up to 10% failure rate in concurrent tests
				maxErrors := len(results) / 10
				if errorCount > maxErrors {
					t.Errorf("Too many concurrent errors: %d out of %d", errorCount, len(results))
				}
			},
			Cleanup: func(setupData interface{}) {
				if campaign, ok := setupData.(*TestCampaign); ok {
					campaign.Cleanup()
				}
			},
		},
	}
}
