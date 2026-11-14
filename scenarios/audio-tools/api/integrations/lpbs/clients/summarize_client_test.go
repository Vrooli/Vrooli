package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
)

func TestSummarizeClient_RoutesThroughMeteredInference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/features/audio-tools.summarize":
			require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
			var request struct {
				Instruction string                         `json:"instruction"`
				Turn        struct{ Role, Content string } `json:"turn"`
				Schema      json.RawMessage                `json:"schema"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, SummarizeInstruction, request.Instruction)
			require.Equal(t, "user", request.Turn.Role)
			require.JSONEq(t, `{"text":"long text","level":"light"}`, request.Turn.Content)
			require.JSONEq(t, SummarizeSchema, string(request.Schema))
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

// Executes the production Audio Tools client against the parent journey's real door.
func TestJourneyAudioSummary(t *testing.T) {
	base := os.Getenv("J4_DOOR_URL")
	if base == "" {
		t.Skip("journey child")
	}
	result, err := NewSummarizeClient(base).Summarize(context.Background(), os.Getenv("J4_DEVICE_KEY"), "", summarizechain.Request{OperationID: "j4-audio-summary", Text: "Our release is ready. Deploy it tomorrow.", Level: "heavy"})
	require.NoError(t, err)
	require.Equal(t, "Hello from router", result.Text)
	require.Equal(t, "fixture-model", result.ModelID)
	require.Equal(t, 4, result.PromptTokens)
	require.Equal(t, 3, result.OutputTokens)
}
