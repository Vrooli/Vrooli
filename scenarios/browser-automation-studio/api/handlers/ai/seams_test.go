package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// TestSeams_OllamaClientMock demonstrates using the MockOllamaClient for testing
// AI analysis without requiring a real Ollama instance.
func TestSeams_OllamaClientMock(t *testing.T) {
	t.Run("[SEAM:OLLAMA-CLIENT] mock returns configured response", func(t *testing.T) {
		mockResponse := `[{"text": "Login", "tagName": "BUTTON", "confidence": 0.9, "category": "auth"}]`
		mockClient := NewMockOllamaClient(mockResponse)

		response, err := mockClient.Query(context.Background(), "llama3.2", "test prompt")

		require.NoError(t, err)
		assert.Equal(t, mockResponse, response)
		assert.Len(t, mockClient.QueriesCalled, 1)
		assert.Equal(t, "llama3.2", mockClient.QueriesCalled[0].Model)
		assert.Equal(t, "test prompt", mockClient.QueriesCalled[0].Prompt)
	})

	t.Run("[SEAM:OLLAMA-CLIENT] mock returns configured error", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			Err: assert.AnError,
		}

		_, err := mockClient.Query(context.Background(), "llama3.2", "test prompt")

		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

// TestSeams_AutomationRunnerMock demonstrates using the MockAutomationRunner for testing
// handlers without requiring browser automation infrastructure.
func TestSeams_AutomationRunnerMock(t *testing.T) {
	t.Run("[SEAM:AUTOMATION-RUNNER] mock returns configured outcomes", func(t *testing.T) {
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{
				Success:   true,
				StepIndex: 0,
				NodeID:    "navigate",
				StepType:  "navigate",
			},
			{
				Success:   true,
				StepIndex: 1,
				NodeID:    "screenshot",
				StepType:  "screenshot",
				Screenshot: &autocontracts.Screenshot{
					Data: []byte("fake-screenshot-data"),
				},
			},
		}

		instructions := []autocontracts.CompiledInstruction{
			{Index: 0, NodeID: "navigate", Type: "navigate"},
			{Index: 1, NodeID: "screenshot", Type: "screenshot"},
		}

		outcomes, events, err := mockRunner.Run(context.Background(), 1920, 1080, instructions)

		require.NoError(t, err)
		assert.Len(t, outcomes, 2)
		assert.True(t, outcomes[0].Success)
		assert.Nil(t, events)
		assert.Len(t, mockRunner.RunCalls, 1)
		assert.Equal(t, 1920, mockRunner.RunCalls[0].ViewportWidth)
		assert.Equal(t, 1080, mockRunner.RunCalls[0].ViewportHeight)
	})

	t.Run("[SEAM:AUTOMATION-RUNNER] mock returns configured error", func(t *testing.T) {
		mockRunner := &MockAutomationRunner{
			Err: assert.AnError,
		}

		_, _, err := mockRunner.Run(context.Background(), 1920, 1080, nil)

		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

// TestSeams_DOMHandlerWithMockRunner demonstrates injecting a mock runner into DOMHandler.
func TestSeams_DOMHandlerWithMockRunner(t *testing.T) {
	t.Run("[SEAM:DOM-HANDLER] accepts injected automation runner", func(t *testing.T) {
		log := logrus.New()
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "dom.navigate", StepType: "navigate"},
			{Success: true, NodeID: "dom.wait", StepType: "wait"},
			{
				Success:  true,
				NodeID:   "dom.extract",
				StepType: "evaluate",
				ExtractedData: map[string]any{
					"value": map[string]any{
						"tagName":  "BODY",
						"children": []any{},
					},
				},
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))

		// Handler uses injected runner
		assert.NotNil(t, handler.runner)

		// Verify the mock runner tracks calls
		ctx := context.Background()
		_, _ = handler.ExtractDOMTree(ctx, "https://example.com")
		assert.Len(t, mockRunner.RunCalls, 1)
	})
}

// TestSeams_ElementAnalysisHandlerWithMocks demonstrates injecting mocks into ElementAnalysisHandler.
func TestSeams_ElementAnalysisHandlerWithMocks(t *testing.T) {
	t.Run("[SEAM:ELEMENT-ANALYSIS] accepts injected runner and suggestion generator", func(t *testing.T) {
		log := logrus.New()

		// Create mock runner
		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "analysis.navigate", StepType: "navigate"},
		}

		// Create mock Ollama client for suggestion generator
		mockOllama := NewMockOllamaClient(`{"suggestions": [{"action": "Login", "confidence": 0.9}]}`)
		suggestionGen := newOllamaSuggestionGenerator(log, WithOllamaClient(mockOllama))

		// Inject mocks
		handler := NewElementAnalysisHandler(log,
			WithElementRunner(mockRunner),
			WithSuggestionGenerator(suggestionGen),
		)

		assert.NotNil(t, handler.runner)
		assert.NotNil(t, handler.suggestionGenerator)
	})
}

// TestSeams_AIAnalysisHandlerWithMocks demonstrates injecting mocks into AIAnalysisHandler.
func TestSeams_AIAnalysisHandlerWithMocks(t *testing.T) {
	t.Run("[SEAM:AI-ANALYSIS] accepts injected DOM extractor and Ollama client", func(t *testing.T) {
		log := logrus.New()

		// Create mock DOM extractor
		mockDOMExtractor := &mockDOMExtractorForTest{
			response: `{"tagName": "BODY", "children": [{"tagName": "BUTTON", "text": "Search"}]}`,
		}

		// Create mock Ollama client
		mockOllama := NewMockOllamaClient(`[{"text": "Search", "tagName": "BUTTON", "confidence": 0.95}]`)

		// Inject mocks
		handler := NewAIAnalysisHandler(log, nil,
			WithDOMExtractor(mockDOMExtractor),
			WithAIAnalysisOllamaClient(mockOllama),
		)

		analyzer, ok := handler.analyzer.(*AIElementAnalyzer)
		require.True(t, ok, "handler should wire an AIElementAnalyzer")
		assert.Equal(t, mockDOMExtractor, analyzer.domExtractor)
		assert.Equal(t, mockOllama, analyzer.ollamaClient)
	})

	t.Run("[SEAM:AI-ANALYSIS] analyzes elements with mocked dependencies", func(t *testing.T) {
		log := logrus.New()

		mockDOMExtractor := &mockDOMExtractorForTest{
			response: `{"tagName": "BODY", "children": [{"tagName": "BUTTON", "text": "Submit"}]}`,
		}

		expectedElements := []ElementInfo{{
			Text:       "Submit",
			TagName:    "BUTTON",
			Confidence: 0.9,
			Category:   "actions",
		}}
		mockResponse, _ := json.Marshal(expectedElements)
		mockOllama := NewMockOllamaClient(string(mockResponse))

		handler := NewAIAnalysisHandler(log, nil,
			WithDOMExtractor(mockDOMExtractor),
			WithAIAnalysisOllamaClient(mockOllama),
		)

		ctx := context.Background()
		elements, err := handler.analyzeElementsWithAI(ctx, "https://example.com", "submit form")

		require.NoError(t, err)
		assert.Len(t, elements, 1)
		assert.Equal(t, "Submit", elements[0].Text)

		// Verify DOM was extracted
		assert.Equal(t, "https://example.com", mockDOMExtractor.lastURL)

		// Verify Ollama was queried
		assert.Len(t, mockOllama.QueriesCalled, 1)
		assert.Contains(t, mockOllama.QueriesCalled[0].Prompt, "submit form")
	})
}

// TestSeams_ScreenshotHandlerWithMock demonstrates injecting mock runner into ScreenshotHandler.
func TestSeams_ScreenshotHandlerWithMock(t *testing.T) {
	t.Run("[SEAM:SCREENSHOT] accepts injected automation runner", func(t *testing.T) {
		log := logrus.New()

		mockRunner := NewMockAutomationRunner()
		mockRunner.Outcomes = []autocontracts.StepOutcome{
			{Success: true, NodeID: "preview.navigate"},
			{
				Success:  true,
				NodeID:   "preview.screenshot",
				StepType: "screenshot",
				Screenshot: &autocontracts.Screenshot{
					Data: []byte("fake-png-data"),
				},
			},
		}
		mockRunner.Events = []autocontracts.EventEnvelope{
			{
				ExecutionID: uuid.New(),
				Kind:        autocontracts.EventKindExecutionCompleted,
			},
		}

		handler := NewScreenshotHandler(log, WithScreenshotRunner(mockRunner))

		assert.NotNil(t, handler.runner)
	})
}

// mockDOMExtractorForTest is a simple mock implementation of DOMExtractor for testing.
type mockDOMExtractorForTest struct {
	response string
	err      error
	lastURL  string
}

func (m *mockDOMExtractorForTest) ExtractDOMTree(_ context.Context, url string) (string, error) {
	m.lastURL = url
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// Compile-time interface enforcement
var _ DOMExtractor = (*mockDOMExtractorForTest)(nil)
