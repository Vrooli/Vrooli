package summarizechain_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/summarize"
)

func TestLocalProvider_HappyPath(t *testing.T) {
	s := summarize.NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return json.Marshal(map[string]any{"response": "summary", "done_reason": "stop", "eval_count": 7})
	})
	p := summarizechain.NewLocalProvider(s, summarize.DefaultSummarizeModel)
	require.True(t, p.IsAvailable(context.Background()))
	res, err := p.Summarize(context.Background(), summarizechain.Request{Text: "hello", Level: "light"})
	require.NoError(t, err)
	require.Equal(t, "summary", res.Text)
	require.Equal(t, summarizechain.TierLocal, res.Tier)
	require.Equal(t, "ollama-local", res.ProviderID)

	// Custom model override.
	res, err = p.Summarize(context.Background(), summarizechain.Request{Text: "x", Model: "custom-model"})
	require.NoError(t, err)
	require.Equal(t, "custom-model", res.ModelID)
}

func TestLocalProvider_ErrorPath(t *testing.T) {
	s := summarize.NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	p := summarizechain.NewLocalProvider(s, "m")
	_, err := p.Summarize(context.Background(), summarizechain.Request{Text: "hi"})
	require.Error(t, err)
}
