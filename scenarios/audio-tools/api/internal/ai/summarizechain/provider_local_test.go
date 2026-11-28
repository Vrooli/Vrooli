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
	s := summarize.NewSummarizerWithRunner("resource-ollama-test", func(_ context.Context, args []string, _ string) ([]byte, error) {
		selector := ""
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--role" || args[i] == "--model" {
				selector = args[i+1]
				break
			}
		}
		resolved := map[string]string{
			summarize.DefaultSummarizeModel: "qwen3:1.7b",
			"summarize.audio_tools.quality": "qwen3.5:4b",
			"custom-model":                  "custom-model-v2",
		}[selector]
		return json.Marshal(map[string]any{"model": resolved, "response": "summary", "done_reason": "stop", "eval_count": 7})
	})
	p := summarizechain.NewLocalProvider(s, summarize.DefaultSummarizeModel)
	require.True(t, p.IsAvailable(context.Background()))
	res, err := p.Summarize(context.Background(), summarizechain.Request{Text: "hello", Level: "light"})
	require.NoError(t, err)
	require.Equal(t, "summary", res.Text)
	require.Equal(t, summarizechain.TierLocal, res.Tier)
	require.Equal(t, "ollama-local", res.ProviderID)
	require.Equal(t, "qwen3:1.7b", res.ModelID, "record the runtime-resolved tag rather than summarize.default")

	// Custom model override.
	res, err = p.Summarize(context.Background(), summarizechain.Request{Text: "x", Model: "custom-model"})
	require.NoError(t, err)
	require.Equal(t, "custom-model-v2", res.ModelID, "record the provider's resolved model tag")
	res, err = p.Summarize(context.Background(), summarizechain.Request{Text: "dense facts", LocalModelRole: "summarize.audio_tools.quality"})
	require.NoError(t, err)
	require.Equal(t, "qwen3.5:4b", res.ModelID)
}

func TestLocalProviderFallsBackToSelectorWhenGatewayOmitsResolvedModel(t *testing.T) {
	s := summarize.NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return json.Marshal(map[string]any{"response": "summary", "done_reason": "stop", "eval_count": 7})
	})
	p := summarizechain.NewLocalProvider(s, summarize.DefaultSummarizeModel)
	res, err := p.Summarize(context.Background(), summarizechain.Request{Text: "hello", Level: "light"})
	require.NoError(t, err)
	require.Equal(t, summarize.DefaultSummarizeModel, res.ModelID)
}

func TestLocalProvider_ErrorPath(t *testing.T) {
	s := summarize.NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	p := summarizechain.NewLocalProvider(s, "m")
	_, err := p.Summarize(context.Background(), summarizechain.Request{Text: "hi"})
	require.Error(t, err)
}
