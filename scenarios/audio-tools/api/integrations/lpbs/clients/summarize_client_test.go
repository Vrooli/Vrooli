package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
)

func TestSummarizeClient_RoutesThroughMeteredInference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/ai/inference":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
			var request struct {
				Role     string `json:"role"`
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
				Metadata struct {
					AppBundleKey string `json:"app_bundle_key"`
					Operation    string `json:"operation"`
				} `json:"metadata"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, "audio.summarize", request.Role)
			require.Len(t, request.Messages, 2)
			require.Equal(t, "audio-tools", request.Metadata.AppBundleKey)
			require.Equal(t, "audio.summarize", request.Metadata.Operation)
			_, _ = w.Write([]byte(`{"content":"short version","prompt_tokens":12,"completion_tokens":4,"model":"openai/gpt-4o-mini"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	c := NewSummarizeClient(server.URL)
	c.Doer = server.Client()
	require.True(t, c.IsAvailable(context.Background()))
	result, err := c.Summarize(context.Background(), "access-token", "user@example.com", summarizechain.Request{Text: "long text", Level: "light"})
	require.NoError(t, err)
	require.Equal(t, "short version", result.Text)
	require.Equal(t, 12, result.PromptTokens)
	require.Equal(t, 4, result.OutputTokens)
	require.Equal(t, summarizechain.TierVrooli, result.Tier)
	require.Equal(t, "lpbs", result.ProviderID)
	require.Equal(t, "openai/gpt-4o-mini", result.ModelID)
}

func TestSummarizeClient_RequiresTokenAndText(t *testing.T) {
	c := NewSummarizeClient("http://lpbs.test")
	_, err := c.Summarize(context.Background(), "", "user", summarizechain.Request{Text: "text"})
	require.ErrorContains(t, err, "access token is required")
	_, err = c.Summarize(context.Background(), "tok", "user", summarizechain.Request{})
	require.ErrorContains(t, err, "text is required")
}
