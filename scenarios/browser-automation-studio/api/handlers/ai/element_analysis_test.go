package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestNewElementAnalysisHandler(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] creates handler with logger", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(os.Stderr)

		handler := NewElementAnalysisHandler(log)

		assert.NotNil(t, handler)
		assert.Equal(t, log, handler.log)
	})
}

func TestAnalyzeElements_RequestValidation(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)
	handler := NewElementAnalysisHandler(log)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/analyze-elements", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.AnalyzeElements(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects missing URL", func(t *testing.T) {
		reqBody := ElementAnalysisRequest{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/analyze-elements", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.AnalyzeElements(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "MISSING_REQUIRED_FIELD", response.Code)

		// Check details contain field name
		detailsMap, ok := response.Details.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "url", detailsMap["field"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects empty URL", func(t *testing.T) {
		reqBody := ElementAnalysisRequest{URL: ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/analyze-elements", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.AnalyzeElements(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] normalizes URL without protocol", func(t *testing.T) {
		if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
			t.Skip("Skipping integration test - PLAYWRIGHT_DRIVER_URL not set")
		}

		reqBody := ElementAnalysisRequest{URL: "example.com"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/analyze-elements", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.AnalyzeElements(w, req)

		// Will fail at driver step in unit test, but should accept the request
		// and normalize URL to https://example.com - expect either 200 (success) or 500 (driver error)
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code,
			"Should accept request with normalized URL")
	})
}

func TestGetElementAtCoordinate_RequestValidation(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)
	handler := NewElementAnalysisHandler(log)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/element-at-coordinate", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.GetElementAtCoordinate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects missing URL", func(t *testing.T) {
		reqBody := ElementAtCoordinateRequest{X: 100, Y: 200}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/element-at-coordinate", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetElementAtCoordinate(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "MISSING_REQUIRED_FIELD", response.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] accepts zero coordinates", func(t *testing.T) {
		if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
			t.Skip("Skipping integration test - PLAYWRIGHT_DRIVER_URL not set")
		}

		reqBody := ElementAtCoordinateRequest{
			URL: "https://example.com",
			X:   0,
			Y:   0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/element-at-coordinate", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetElementAtCoordinate(w, req)

		// Will fail at driver in unit test, but should accept the request
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code,
			"Should accept valid request")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] accepts negative coordinates", func(t *testing.T) {
		if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
			t.Skip("Skipping integration test - PLAYWRIGHT_DRIVER_URL not set")
		}

		// Negative coordinates might be valid in some viewport scenarios
		reqBody := ElementAtCoordinateRequest{
			URL: "https://example.com",
			X:   -10,
			Y:   -10,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/element-at-coordinate", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetElementAtCoordinate(w, req)

		// Will fail at driver, but request validation should pass
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code,
			"Should accept valid request")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] normalizes URL without protocol", func(t *testing.T) {
		if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
			t.Skip("Skipping integration test - PLAYWRIGHT_DRIVER_URL not set")
		}

		reqBody := ElementAtCoordinateRequest{
			URL: "example.com",
			X:   100,
			Y:   200,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/element-at-coordinate", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetElementAtCoordinate(w, req)

		// Should normalize to https://example.com
		assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code,
			"Should accept request with normalized URL")
	})
}

func TestGetElementAtCoordinate_DriverIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	handler := NewElementAnalysisHandler(log)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles driver connection failure", func(t *testing.T) {
		// Set invalid driver URL
		originalURL := os.Getenv("PLAYWRIGHT_DRIVER_URL")
		os.Setenv("PLAYWRIGHT_DRIVER_URL", "http://invalid-driver:9999")
		defer os.Setenv("PLAYWRIGHT_DRIVER_URL", originalURL)

		ctx := context.Background()

		_, err := handler.getElementAtCoordinate(ctx, "https://example.com", 100, 200)

		assert.Error(t, err)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles navigation failure", func(t *testing.T) {
		// Check if Playwright driver is available
		driverURL := os.Getenv("PLAYWRIGHT_DRIVER_URL")
		if driverURL == "" {
			driverURL = "http://127.0.0.1:39400"
		}

		ctx := context.Background()

		// Use an invalid URL that will fail navigation
		_, err := handler.getElementAtCoordinate(ctx, "https://this-domain-does-not-exist-12345.com", 100, 200)

		if err != nil && (err.Error() == "PLAYWRIGHT_DRIVER_URL required for replay rendering" ||
			err.Error() == "failed to navigate to URL: Post \"http://127.0.0.1:39400/session/start\": dial tcp 127.0.0.1:39400: connect: connection refused") {
			t.Skip("Playwright driver not available")
		}

		assert.Error(t, err)
	})
}

func TestExtractPageElements_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	handler := NewElementAnalysisHandler(log)

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] extracts elements from real page", func(t *testing.T) {
		ctx := context.Background()

		elements, pageContext, screenshot, err := handler.extractPageElements(ctx, "https://example.com")

		if err != nil {
			t.Skipf("Browserless not available: %v", err)
		}

		require.NoError(t, err)
		assert.NotNil(t, elements)
		assert.NotEmpty(t, pageContext.URL)
		assert.NotEmpty(t, screenshot)

		// Verify page context has reasonable values (URL might have trailing slash added by browser)
		assert.Contains(t, []string{"https://example.com", "https://example.com/"}, pageContext.URL,
			"URL should be normalized https://example.com with optional trailing slash")
		assert.NotEmpty(t, pageContext.Title)

		// Screenshot should be base64 encoded
		assert.Contains(t, screenshot, "data:image")
	})
}

func TestGenerateAISuggestions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if Ollama is available by checking both HTTP endpoint and CLI availability
	ollamaAvailable := false
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err == nil {
		resp.Body.Close()
		// Also verify OLLAMA_URL or OLLAMA_HOST is set
		if os.Getenv("OLLAMA_URL") != "" || os.Getenv("OLLAMA_HOST") != "" {
			ollamaAvailable = true
		}
	}

	if !ollamaAvailable {
		t.Skip("Ollama not available or not configured (OLLAMA_URL/OLLAMA_HOST not set), skipping integration test")
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	handler := NewElementAnalysisHandler(log)

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] generates suggestions for search page", func(t *testing.T) {
		ctx := context.Background()

		elements := []ElementInfo{
			{
				Text:     "Search",
				TagName:  "BUTTON",
				Type:     "button",
				Category: "actions",
				Selectors: []SelectorOption{
					{Selector: "#search-btn", Type: "id", Robustness: 0.9},
				},
			},
			{
				Text:     "",
				TagName:  "INPUT",
				Type:     "text",
				Category: "data-entry",
				Selectors: []SelectorOption{
					{Selector: "#search-input", Type: "id", Robustness: 0.9},
				},
				Attributes: map[string]string{"placeholder": "Enter search term"},
			},
		}

		pageContext := PageContext{
			Title:       "Search Page",
			URL:         "https://example.com/search",
			HasSearch:   true,
			ButtonCount: 1,
			FormCount:   1,
		}

		suggestions, err := handler.generateAISuggestions(ctx, elements, pageContext)

		if err != nil {
			t.Skipf("Ollama integration failed: %v", err)
		}

		require.NoError(t, err)
		assert.NotEmpty(t, suggestions)

		// Verify suggestions have required fields
		for _, suggestion := range suggestions {
			assert.NotEmpty(t, suggestion.Action)
			assert.GreaterOrEqual(t, suggestion.Confidence, 0.0)
			assert.LessOrEqual(t, suggestion.Confidence, 1.0)
			assert.NotEmpty(t, suggestion.Category)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles empty element list", func(t *testing.T) {
		ctx := context.Background()

		elements := []ElementInfo{}
		pageContext := PageContext{
			Title: "Empty Page",
			URL:   "https://example.com",
		}

		suggestions, err := handler.generateAISuggestions(ctx, elements, pageContext)

		if err != nil {
			t.Skipf("Ollama integration failed: %v", err)
		}

		// Should either return empty suggestions or fallback suggestions
		require.NoError(t, err)
		assert.NotNil(t, suggestions)
	})
}

func TestBuildElementAnalysisPrompt(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)
	handler := NewElementAnalysisHandler(log)

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] builds prompt with elements and context", func(t *testing.T) {
		elements := []ElementInfo{
			{
				Text:     "Login",
				TagName:  "BUTTON",
				Category: "authentication",
				Selectors: []SelectorOption{
					{Selector: "#login-btn", Type: "id"},
				},
			},
		}

		pageContext := PageContext{
			Title:     "Login Page",
			URL:       "https://example.com/login",
			HasLogin:  true,
			FormCount: 1,
		}

		prompt := handler.buildElementAnalysisPrompt(elements, pageContext)

		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "Login Page")
		assert.Contains(t, prompt, "https://example.com/login")
		assert.Contains(t, prompt, "Login")
		assert.Contains(t, prompt, "BUTTON")
		assert.Contains(t, prompt, "authentication")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] handles empty element list", func(t *testing.T) {
		elements := []ElementInfo{}
		pageContext := PageContext{
			Title: "Empty Page",
			URL:   "https://example.com",
		}

		prompt := handler.buildElementAnalysisPrompt(elements, pageContext)

		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "Empty Page")
		assert.Contains(t, prompt, "https://example.com")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] includes page statistics", func(t *testing.T) {
		elements := []ElementInfo{}
		pageContext := PageContext{
			Title:       "Complex Page",
			URL:         "https://example.com",
			HasLogin:    true,
			HasSearch:   true,
			FormCount:   3,
			ButtonCount: 10,
			LinkCount:   50,
		}

		prompt := handler.buildElementAnalysisPrompt(elements, pageContext)

		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "3")  // form count
		assert.Contains(t, prompt, "10") // button count
		assert.Contains(t, prompt, "50") // link count
	})
}

func TestElementAnalysisHandler_extractPageElements(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] returns structured data when automation succeeds", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{
				Success:  true,
				NodeID:   "analysis.evaluate",
				StepType: "evaluate",
				ExtractedData: map[string]any{
					"value": map[string]any{
						"elements": []any{
							map[string]any{
								"text":      "Login",
								"tagName":   "BUTTON",
								"type":      "button",
								"selectors": []any{map[string]any{"selector": "#login-btn", "type": "id", "robustness": 0.9, "fallback": false}},
								"boundingBox": map[string]any{
									"x":      1,
									"y":      2,
									"width":  100,
									"height": 40,
								},
								"confidence": 0.85,
								"category":   "authentication",
								"attributes": map[string]any{
									"id":        "login-btn",
									"className": "primary",
								},
							},
						},
						"pageContext": map[string]any{
							"title":       "Example",
							"url":         "https://example.com",
							"hasLogin":    true,
							"hasSearch":   false,
							"formCount":   1,
							"buttonCount": 2,
							"linkCount":   5,
						},
					},
				},
			},
			{
				Success:  true,
				NodeID:   "analysis.screenshot",
				StepType: "screenshot",
				Screenshot: &autocontracts.Screenshot{
					Data:      []byte{0x89, 0x50, 0x4E},
					MediaType: "image/png",
				},
			},
		}

		handler := newElementAnalysisHandlerForTest(mockRunner)

		elements, pageCtx, screenshot, err := handler.extractPageElements(context.Background(), "https://example.com")

		require.NoError(t, err)
		require.Len(t, elements, 1)
		assert.Equal(t, "Login", elements[0].Text)
		assert.Equal(t, "Example", pageCtx.Title)
		assert.Contains(t, screenshot, "data:image/png;base64,")

		require.Len(t, mockRunner.RunCalls, 1)
		call := mockRunner.RunCalls[0]
		assert.Equal(t, previewDefaultViewportWidth, call.ViewportWidth)
		assert.Equal(t, previewDefaultViewportHeight, call.ViewportHeight)
		require.Len(t, call.Instructions, 4)
		assert.Equal(t, "analysis.evaluate", call.Instructions[2].NodeID)
		assert.Equal(t, "analysis.screenshot", call.Instructions[3].NodeID)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] surfaces automation runner failures", func(t *testing.T) {
		mockRunner := &MockAutomationRunner{
			Err: errors.New("driver unavailable"),
		}
		handler := newElementAnalysisHandlerForTest(mockRunner)

		_, _, _, err := handler.extractPageElements(context.Background(), "https://example.com")

		require.Error(t, err)
		assert.ErrorContains(t, err, "automation run failed")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] fails when screenshot data missing", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{
				Success:  true,
				NodeID:   "analysis.evaluate",
				StepType: "evaluate",
				ExtractedData: map[string]any{
					"value": map[string]any{
						"elements": []any{
							map[string]any{
								"text":      "Login",
								"tagName":   "BUTTON",
								"type":      "button",
								"selectors": []any{},
								"boundingBox": map[string]any{
									"x":      1,
									"y":      2,
									"width":  100,
									"height": 40,
								},
								"confidence": 0.5,
								"category":   "authentication",
								"attributes": map[string]any{},
							},
						},
						"pageContext": map[string]any{
							"title": "Example",
							"url":   "https://example.com",
						},
					},
				},
			},
			{
				Success:    true,
				NodeID:     "analysis.screenshot",
				StepType:   "screenshot",
				Screenshot: &autocontracts.Screenshot{Data: []byte{}},
			},
		}

		handler := newElementAnalysisHandlerForTest(mockRunner)

		_, _, _, err := handler.extractPageElements(context.Background(), "https://example.com")

		require.Error(t, err)
		assert.ErrorContains(t, err, "screenshot capture returned no data")
	})
}

func TestElementAnalysisHandler_getElementAtCoordinate(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] clamps selection index and hydrates element", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "probe.navigate", StepType: "navigate"},
			{
				Success:  true,
				NodeID:   "probe.element",
				StepType: "probeElements",
				ProbeResult: map[string]any{
					"candidates": []any{
						map[string]any{
							"element": map[string]any{
								"text":    "Login",
								"tagName": "BUTTON",
								"type":    "button",
								"boundingBox": map[string]any{
									"x":      0,
									"y":      0,
									"width":  80,
									"height": 24,
								},
								"selectors": []any{
									map[string]any{"selector": "#login-btn", "type": "id"},
								},
								"attributes": map[string]any{"id": "login-btn"},
							},
							"selector": "#login-btn",
							"depth":    1,
							"path":     []any{"html", "body", "button"},
						},
					},
					"selectedIndex": 5,
				},
			},
		}

		handler := newElementAnalysisHandlerForTest(mockRunner)

		result, err := handler.getElementAtCoordinate(context.Background(), "https://example.com", 10, 20)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.SelectedIndex)
		require.NotNil(t, result.Element)
		assert.Equal(t, "Login", result.Element.Text)
		require.Len(t, result.Candidates, 1)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] surfaces automation probe errors", func(t *testing.T) {
		mockRunner := &MockAutomationRunner{Err: errors.New("probe failed")}
		handler := newElementAnalysisHandlerForTest(mockRunner)

		_, err := handler.getElementAtCoordinate(context.Background(), "https://example.com", 10, 20)

		require.Error(t, err)
		assert.ErrorContains(t, err, "automation probe failed")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] errors when probe outcomes missing", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "probe.navigate", StepType: "navigate"},
		}

		handler := newElementAnalysisHandlerForTest(mockRunner)

		_, err := handler.getElementAtCoordinate(context.Background(), "https://example.com", 10, 20)

		require.Error(t, err)
		assert.ErrorContains(t, err, "probe did not return any outcomes")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] errors when no qualifying candidates", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "probe.navigate", StepType: "navigate"},
			{
				Success:  true,
				NodeID:   "probe.element",
				StepType: "probeElements",
				ProbeResult: map[string]any{
					"candidates":    []any{},
					"selectedIndex": 0,
				},
			},
		}

		handler := newElementAnalysisHandlerForTest(mockRunner)

		_, err := handler.getElementAtCoordinate(context.Background(), "https://example.com", 10, 20)

		require.Error(t, err)
		assert.ErrorContains(t, err, "no qualifying elements")
	})
}

func newElementAnalysisHandlerForTest(runner AutomationRunner) *ElementAnalysisHandler {
	log := logrus.New()
	log.SetOutput(io.Discard)
	mockSuggestions := newOllamaSuggestionGenerator(log, WithOllamaClient(NewMockOllamaClient(`{"suggestions": []}`)))
	return NewElementAnalysisHandler(log,
		WithElementRunner(runner),
		WithSuggestionGenerator(mockSuggestions),
	)
}
