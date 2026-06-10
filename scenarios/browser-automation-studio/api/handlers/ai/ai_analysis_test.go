package ai

import (
	"context"
	"errors"
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

func TestRunAIAnalyze_RequestValidation(t *testing.T) {
	log := logrus.New()
	makeHandler := func(analyzer ElementAnalyzer) *AIAnalysisHandler {
		return NewAIAnalysisHandler(log, nil, WithElementAnalyzer(analyzer), WithAIAnalysisTimeout(time.Second))
	}

	t.Run("rejects empty URL", func(t *testing.T) {
		analyzer := &mockElementAnalyzer{}
		handler := makeHandler(analyzer)
		_, err := handler.RunAIAnalyze(context.Background(), "", "search", "", false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingURL)
		assert.Empty(t, analyzer.calls)
	})

	t.Run("rejects empty intent", func(t *testing.T) {
		analyzer := &mockElementAnalyzer{}
		handler := makeHandler(analyzer)
		_, err := handler.RunAIAnalyze(context.Background(), "https://example.com", "", "", false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingIntent)
		assert.Empty(t, analyzer.calls)
	})
}

func TestRunAIAnalyze_DelegatesToAnalyzer(t *testing.T) {
	log := logrus.New()
	suggestions := []ElementInfo{{Text: "Search", TagName: "BUTTON", Confidence: 0.9}}
	analyzer := &mockElementAnalyzer{suggestions: suggestions}
	handler := NewAIAnalysisHandler(log, nil, WithElementAnalyzer(analyzer))

	got, err := handler.RunAIAnalyze(context.Background(), "https://example.com", "search products", "", false)

	require.NoError(t, err)
	assert.Equal(t, suggestions, got)
	require.Len(t, analyzer.calls, 1)
	assert.Equal(t, "https://example.com", analyzer.calls[0].url)
	assert.Equal(t, "search products", analyzer.calls[0].intent)
}

func TestRunAIAnalyze_AnalyzerError(t *testing.T) {
	log := logrus.New()
	analyzer := &mockElementAnalyzer{err: errors.New("analysis failed")}
	handler := NewAIAnalysisHandler(log, nil, WithElementAnalyzer(analyzer))

	_, err := handler.RunAIAnalyze(context.Background(), "https://example.com", "search", "", false)
	require.Error(t, err)
}

func TestAIElementAnalyzer_ExtractFailure(t *testing.T) {
	log := logrus.New()
	mockDOM := &recordingDOMExtractor{err: errors.New("failed to connect")}
	mockOllama := NewMockOllamaClient(`[{"text": "Search"}]`)

	analyzer := &AIElementAnalyzer{
		log:          log,
		domExtractor: mockDOM,
		ollamaClient: mockOllama,
		role:         "chat.small",
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
		role:         "chat.small",
	}

	results, err := analyzer.Analyze(context.Background(), "https://example.com", "search")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Search", results[0].Text)
	assert.Len(t, mockDOM.calls, 1)
	assert.Len(t, mockOllama.QueriesCalled, 1)
	assert.Equal(t, "chat.small", mockOllama.QueriesCalled[0].Role)
}

func TestAIElementAnalyzer_FallbackOnBadJSON(t *testing.T) {
	log := logrus.New()
	mockDOM := &recordingDOMExtractor{response: "<html></html>"}
	mockOllama := NewMockOllamaClient("not-json")

	analyzer := &AIElementAnalyzer{
		log:          log,
		domExtractor: mockDOM,
		ollamaClient: mockOllama,
		role:         "chat.small",
	}

	results, err := analyzer.Analyze(context.Background(), "https://example.com", "search")

	require.NoError(t, err)
	assert.NotEmpty(t, results, "fallback suggestion should be returned")
	assert.Len(t, mockOllama.QueriesCalled, 1)
}
