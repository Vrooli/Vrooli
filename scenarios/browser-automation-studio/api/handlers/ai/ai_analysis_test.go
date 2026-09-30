package ai

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
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

func TestOllamaSuggestionGeneratorAcceptsArrayResponse(t *testing.T) {
	log := logrus.New()
	generator := newOllamaSuggestionGenerator(log, WithOllamaClient(NewMockOllamaClient(`[{"action":"Search","confidence":0.95,"category":"actions"}]`)))

	suggestions, err := generator.generateAISuggestions(context.Background(), []ElementInfo{{Text: "Search", TagName: "BUTTON"}}, PageContext{URL: "https://example.com"})
	require.NoError(t, err)
	require.Len(t, suggestions, 1)
	assert.Equal(t, "Search", suggestions[0].Action)
}

// [REQ:BAS-AI-GENERATION-VALIDATION] Incomplete provider output must not become
// successful suggestions or an indistinguishable empty result.
func TestOllamaSuggestionResponseContract(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		valid         bool
		count         int
	}{
		{"object", `{"suggestions":[{"action":"Search","confidence":0.9,"category":"data-entry"}]}`, true, 1},
		{"array", `[{"action":"Search","confidence":0,"category":"actions"}]`, true, 1},
		{"absent optional text", `{"suggestions":[{"action":"Search","confidence":0.9,"category":"data-entry","description":null,"elementText":null,"selector":null,"reasoning":null}]}`, true, 1},
		{"null required action", `[{"action":null,"confidence":0.9,"category":"actions"}]`, false, 0},
		{"null required confidence", `[{"action":"Search","confidence":null,"category":"actions"}]`, false, 0},
		{"null required category", `[{"action":"Search","confidence":0.9,"category":null}]`, false, 0},
		{"numeric optional text", `[{"action":"Search","confidence":0.9,"category":"actions","elementText":42}]`, false, 0},
		{"boolean optional text", `[{"action":"Search","confidence":0.9,"category":"actions","selector":false}]`, false, 0},
		{"object optional text", `[{"action":"Search","confidence":0.9,"category":"actions","reasoning":{}}]`, false, 0},
		{"empty", `{"suggestions":[]}`, true, 0},
		{"malformed", `not JSON`, false, 0},
		{"missing list", `{}`, false, 0},
		{"null list", `{"suggestions":null}`, false, 0},
		{"missing category", `[{"action":"Search","confidence":0.9}]`, false, 0},
		{"unknown category", `[{"action":"Search","confidence":0.9,"category":"guess"}]`, false, 0},
		{"missing action", `[{"confidence":0.9,"category":"actions"}]`, false, 0},
		{"blank action", `[{"action":"  ","confidence":0.9,"category":"actions"}]`, false, 0},
		{"missing confidence", `[{"action":"Search","category":"actions"}]`, false, 0},
		{"negative confidence", `[{"action":"Search","confidence":-0.1,"category":"actions"}]`, false, 0},
		{"excess confidence", `[{"action":"Search","confidence":1.1,"category":"actions"}]`, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			generator := newOllamaSuggestionGenerator(logrus.New(), WithOllamaClient(NewMockOllamaClient(tc.payload)))
			got, err := generator.generateAISuggestions(context.Background(), []ElementInfo{{Text: "Search", TagName: "BUTTON"}}, PageContext{URL: "https://example.test"})
			if !tc.valid {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, tc.count)
			if tc.name == "absent optional text" {
				require.Equal(t, AISuggestion{Action: "Search", Confidence: 0.9, Category: "data-entry"}, got[0], "absent metadata must not manufacture text or change required fields")
			}
		})
	}
}

func TestOllamaSuggestionsRequestStructuredGatewayOutput(t *testing.T) {
	client := NewDefaultOllamaClient(logrus.New(), WithOllamaRunner(func(_ context.Context, args []string, prompt string) ([]byte, error) {
		require.Equal(t, []string{"gateway", "generate"}, args[:2])
		require.Contains(t, args, "--prompt-stdin")
		require.Contains(t, prompt, "Search")
		pos := slices.Index(args, "--format")
		require.GreaterOrEqual(t, pos, 0, "structured generation must constrain the provider response")
		var schema map[string]any
		require.NoError(t, json.Unmarshal([]byte(args[pos+1]), &schema))
		require.Equal(t, "object", schema["type"])
		return []byte(`{"response":"{\"suggestions\":[{\"action\":\"Search\",\"confidence\":0.9,\"category\":\"actions\"}]}"}`), nil
	}))
	generator := newOllamaSuggestionGenerator(logrus.New(), WithOllamaClient(client))
	got, err := generator.generateAISuggestions(context.Background(), []ElementInfo{{Text: "Search", TagName: "BUTTON"}}, PageContext{URL: "https://example.test"})
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestOllamaSuggestionsWithNoElementsDoNotCallProvider(t *testing.T) {
	client := NewMockOllamaClient("")
	client.Err = errors.New("provider must not be called without elements")
	generator := newOllamaSuggestionGenerator(logrus.New(), WithOllamaClient(client))
	got, err := generator.generateAISuggestions(context.Background(), nil, PageContext{URL: "https://example.test"})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.Empty(t, client.QueriesCalled)
}
