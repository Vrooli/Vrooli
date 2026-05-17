package summarizechain_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/summarize"
)

func TestLocalProvider_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":"summary"},"done_reason":"stop","eval_count":7}`))
	}))
	defer srv.Close()
	s := summarize.NewSummarizer(srv.URL)
	p := summarizechain.NewLocalProvider(s, "qwen3")
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	s := summarize.NewSummarizer(srv.URL)
	p := summarizechain.NewLocalProvider(s, "m")
	_, err := p.Summarize(context.Background(), summarizechain.Request{Text: "hi"})
	require.Error(t, err)
}
