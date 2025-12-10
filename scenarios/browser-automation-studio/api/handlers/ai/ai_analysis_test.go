package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockElementAnalyzer struct {
	suggestions []ElementInfo
	err         error
	calls       []struct {
		url    string
		intent string
	}
}

func (m *mockElementAnalyzer) Analyze(_ context.Context, url, intent string) ([]ElementInfo, error) {
	m.calls = append(m.calls, struct {
		url    string
		intent string
	}{url: url, intent: intent})
	if m.err != nil {
		return nil, m.err
	}
	return m.suggestions, nil
}

type recordingDOMExtractor struct {
	response string
	err      error
	calls    []string
}

func (d *recordingDOMExtractor) ExtractDOMTree(_ context.Context, url string) (string, error) {
	d.calls = append(d.calls, url)
	if d.err != nil {
		return "", d.err
	}
	return d.response, nil
}

func TestNewAIAnalysisHandler(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] creates handler with default analyzer wiring", func(t *testing.T) {
		log := logrus.New()
		domHandler := NewDOMHandler(log)

		handler := NewAIAnalysisHandler(log, domHandler)

		require.NotNil(t, handler)
		require.NotNil(t, handler.analyzer)

		defaultAnalyzer, ok := handler.analyzer.(*AIElementAnalyzer)
		require.True(t, ok, "default analyzer should be AIElementAnalyzer")
		assert.Equal(t, domHandler, defaultAnalyzer.domExtractor)
		assert.NotNil(t, defaultAnalyzer.ollamaClient)
	})
}

func TestAIAnalyzeElements_RequestValidation(t *testing.T) {
	log := logrus.New()

	makeHandler := func(analyzer ElementAnalyzer) *AIAnalysisHandler {
		return NewAIAnalysisHandler(log, nil, WithElementAnalyzer(analyzer), WithAIAnalysisTimeout(time.Second))
	}

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects invalid JSON", func(t *testing.T) {
		analyzer := &mockElementAnalyzer{}
		handler := makeHandler(analyzer)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-analyze-elements", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.AIAnalyzeElements(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Empty(t, analyzer.calls)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects missing URL", func(t *testing.T) {
		analyzer := &mockElementAnalyzer{}
		handler := makeHandler(analyzer)

		body, _ := json.Marshal(AIAnalyzeRequest{Intent: "search"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-analyze-elements", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.AIAnalyzeElements(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Empty(t, analyzer.calls)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects missing intent", func(t *testing.T) {
		analyzer := &mockElementAnalyzer{}
		handler := makeHandler(analyzer)

		body, _ := json.Marshal(AIAnalyzeRequest{URL: "https://example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-analyze-elements", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.AIAnalyzeElements(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Empty(t, analyzer.calls)
	})
}

func TestAIAnalyzeElements_DelegatesToAnalyzer(t *testing.T) {
	log := logrus.New()
	suggestions := []ElementInfo{{
		Text:       "Search",
		TagName:    "BUTTON",
		Confidence: 0.9,
	}}

	analyzer := &mockElementAnalyzer{suggestions: suggestions}
	handler := NewAIAnalysisHandler(log, nil, WithElementAnalyzer(analyzer))

	body, _ := json.Marshal(AIAnalyzeRequest{
		URL:    "https://example.com",
		Intent: "search products",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-analyze-elements", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.AIAnalyzeElements(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, analyzer.calls, 1)
	assert.Equal(t, "https://example.com", analyzer.calls[0].url)
	assert.Equal(t, "search products", analyzer.calls[0].intent)

	var response []ElementInfo
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, suggestions, response)
}

func TestAIAnalyzeElements_AnalyzerError(t *testing.T) {
	log := logrus.New()
	analyzer := &mockElementAnalyzer{err: errors.New("analysis failed")}
	handler := NewAIAnalysisHandler(log, nil, WithElementAnalyzer(analyzer))

	body, _ := json.Marshal(AIAnalyzeRequest{
		URL:    "https://example.com",
		Intent: "search",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai-analyze-elements", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	handler.AIAnalyzeElements(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Len(t, analyzer.calls, 1)

	var response APIError
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, "INTERNAL_SERVER_ERROR", response.Code)
}

func TestAIElementAnalyzer_ExtractFailure(t *testing.T) {
	log := logrus.New()
	mockDOM := &recordingDOMExtractor{err: errors.New("failed to connect")}
	mockOllama := NewMockOllamaClient(`[{"text": "Search"}]`)

	analyzer := &AIElementAnalyzer{
		log:          log,
		domExtractor: mockDOM,
		ollamaClient: mockOllama,
		model:        "test-model",
	}

	_, err := analyzer.Analyze(context.Background(), "https://example.com", "search")
	require.Error(t, err)
	assert.Empty(t, mockOllama.QueriesCalled, "should not call Ollama when DOM extraction fails")
}

func TestAIElementAnalyzer_ParsesSuggestions(t *testing.T) {
	log := logrus.New()
	mockDOM := &recordingDOMExtractor{response: "<html><body><button>Search</button></body></html>"}
	mockOllama := NewMockOllamaClient(`[{"text": "Search", "tagName": "BUTTON", "confidence": 0.95}]`)

	analyzer := &AIElementAnalyzer{
		log:          log,
		domExtractor: mockDOM,
		ollamaClient: mockOllama,
		model:        "test-model",
	}

	results, err := analyzer.Analyze(context.Background(), "https://example.com", "search")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Search", results[0].Text)
	assert.Len(t, mockDOM.calls, 1)
	assert.Len(t, mockOllama.QueriesCalled, 1)
	assert.Equal(t, "test-model", mockOllama.QueriesCalled[0].Model)
}

func TestAIElementAnalyzer_FallbackOnBadJSON(t *testing.T) {
	log := logrus.New()
	mockDOM := &recordingDOMExtractor{response: "<html></html>"}
	mockOllama := NewMockOllamaClient("not-json")

	analyzer := &AIElementAnalyzer{
		log:          log,
		domExtractor: mockDOM,
		ollamaClient: mockOllama,
		model:        "test-model",
	}

	results, err := analyzer.Analyze(context.Background(), "https://example.com", "search")

	require.NoError(t, err)
	assert.NotEmpty(t, results, "fallback suggestion should be returned")
	assert.Len(t, mockOllama.QueriesCalled, 1)
}
