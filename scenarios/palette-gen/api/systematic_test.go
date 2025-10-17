package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGenerateHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestGenerateHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Build systematic error test scenarios
	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/generate", "GET", generateHandler).
		AddMethodNotAllowed("/generate", "PUT", generateHandler).
		AddMethodNotAllowed("/generate", "DELETE", generateHandler).
		AddInvalidJSON("/generate", generateHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "GenerateHandler",
		Handler: generateHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestSuggestHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestSuggestHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/suggest", "GET", suggestHandler).
		AddMethodNotAllowed("/suggest", "PUT", suggestHandler).
		AddMethodNotAllowed("/suggest", "DELETE", suggestHandler).
		AddInvalidJSON("/suggest", suggestHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "SuggestHandler",
		Handler: suggestHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestExportHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestExportHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/export", "GET", exportHandler).
		AddMethodNotAllowed("/export", "PUT", exportHandler).
		AddMethodNotAllowed("/export", "DELETE", exportHandler).
		AddInvalidJSON("/export", exportHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "ExportHandler",
		Handler: exportHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestAccessibilityHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestAccessibilityHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/accessibility", "GET", accessibilityHandler).
		AddMethodNotAllowed("/accessibility", "PUT", accessibilityHandler).
		AddMethodNotAllowed("/accessibility", "DELETE", accessibilityHandler).
		AddInvalidJSON("/accessibility", accessibilityHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "AccessibilityHandler",
		Handler: accessibilityHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestHarmonyHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestHarmonyHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/harmony", "GET", harmonyHandler).
		AddMethodNotAllowed("/harmony", "PUT", harmonyHandler).
		AddMethodNotAllowed("/harmony", "DELETE", harmonyHandler).
		AddInvalidJSON("/harmony", harmonyHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "HarmonyHandler",
		Handler: harmonyHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestColorblindHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestColorblindHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/colorblind", "GET", colorblindHandler).
		AddMethodNotAllowed("/colorblind", "PUT", colorblindHandler).
		AddMethodNotAllowed("/colorblind", "DELETE", colorblindHandler).
		AddInvalidJSON("/colorblind", colorblindHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "ColorblindHandler",
		Handler: colorblindHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestHistoryHandlerSystematic uses TestScenarioBuilder for systematic error testing
func TestHistoryHandlerSystematic(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	errorScenarios := NewTestScenarioBuilder().
		AddMethodNotAllowed("/history", "POST", historyHandler).
		AddMethodNotAllowed("/history", "PUT", historyHandler).
		AddMethodNotAllowed("/history", "DELETE", historyHandler).
		Build()

	suite := &HandlerTestSuite{
		Name:    "HistoryHandler",
		Handler: historyHandler,
	}

	suite.RunErrorTests(t, errorScenarios)
}

// TestHandlerSuccessScenarios uses HandlerTestSuite for success path testing
func TestHandlerSuccessScenarios(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	successCases := []SuccessTestCase{
		{
			Name:           "GenerateOceanPalette",
			Method:         "POST",
			Path:           "/generate",
			Body:           generateTestPaletteRequest("ocean", "vibrant", 5, ""),
			Handler:        generateHandler,
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				response := assertJSONResponse(t, w, http.StatusOK)
				assertPaletteResponse(t, response, 5)
			},
			Description: "Generate ocean theme palette",
		},
		{
			Name:           "CheckAccessibility",
			Method:         "POST",
			Path:           "/accessibility",
			Body:           generateTestAccessibilityRequest("#000000", "#FFFFFF"),
			Handler:        accessibilityHandler,
			ExpectedStatus: http.StatusOK,
			Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				response := assertJSONResponse(t, w, http.StatusOK)
				if _, ok := response["contrast_ratio"]; !ok {
					t.Error("Expected contrast_ratio in response")
				}
			},
			Description: "Check color accessibility",
		},
	}

	suite := &HandlerTestSuite{
		Name:    "SuccessScenarios",
		Handler: nil, // Will use handler from test case
	}

	suite.RunSuccessTests(t, successCases)
}
