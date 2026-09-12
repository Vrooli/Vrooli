package byok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audio-tools/internal/ai/summarizechain"
)

func TestOpenRouterContract(t *testing.T) {
	a := NewOpenRouterSummarize()
	if a.ID() != "openrouter" {
		t.Fatalf("id: %s", a.ID())
	}
	if a.Model() == "" {
		t.Fatalf("model must be set")
	}
	if a.IsAvailable(context.Background(), "") {
		t.Fatalf("empty key unavailable")
	}
}

func TestSummarizationSystemPrompt(t *testing.T) {
	if !strings.Contains(summarizationSystemPrompt("light"), "one sentence") {
		t.Fatal("light prompt missing one-sentence directive")
	}
	if !strings.Contains(summarizationSystemPrompt("heavy"), "compression") {
		t.Fatal("heavy prompt missing compression directive")
	}
	if !strings.Contains(summarizationSystemPrompt("moderate"), "2–4 sentences") && !strings.Contains(summarizationSystemPrompt(""), "2") {
		t.Fatal("default prompt missing 2-4 directive")
	}
}

func TestOpenRouterSummarizeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"summary"}}],"usage":{"prompt_tokens":10,"completion_tokens":3},"model":"m"}`))
	}))
	defer srv.Close()

	a := NewOpenRouterSummarize()
	a.Endpoint = srv.URL
	res, err := a.Summarize(context.Background(), "key", summarizechain.Request{Text: "hello", Level: "light"})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if res.Text != "summary" {
		t.Fatalf("text: %q", res.Text)
	}
	if res.PromptTokens != 10 || res.OutputTokens != 3 {
		t.Fatalf("tokens: %d/%d", res.PromptTokens, res.OutputTokens)
	}
}

func TestOpenRouterSummarizeMissingKey(t *testing.T) {
	a := NewOpenRouterSummarize()
	if _, err := a.Summarize(context.Background(), "", summarizechain.Request{}); err == nil {
		t.Fatalf("expected missing-key rejection")
	}
}

func TestOpenRouterSummarizeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	a := NewOpenRouterSummarize()
	a.Endpoint = srv.URL
	_, err := a.Summarize(context.Background(), "k", summarizechain.Request{Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500: %v", err)
	}
}
