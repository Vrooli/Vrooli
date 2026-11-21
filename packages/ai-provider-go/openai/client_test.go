package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/ai-provider-go/providererror"
)

func TestChatAndModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":3,"completion_tokens":2,"cost":0.001}}`))
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"vendor/model","owned_by":"vendor"},{"id":"other/model"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(Config{ProviderName: "example", BaseURL: server.URL + "/v1", APIKey: "sk-test"})
	result, err := client.Chat(context.Background(), ChatRequest{
		Model:    "vendor/model",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if result.Text != "hello" || result.Usage.PromptTokens == nil || *result.Usage.PromptTokens != 3 {
		t.Fatalf("result = %+v", result)
	}
	raw, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	models, err := NormalizeModels(raw, "test", "now", 1)
	if err != nil {
		t.Fatalf("NormalizeModels: %v", err)
	}
	if models.Count != 2 || len(models.Models) != 1 || models.Models[0].Provider != "other" {
		t.Fatalf("models = %+v", models)
	}
}

func TestErrorClassification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
	}))
	defer server.Close()

	client := New(Config{ProviderName: "example", BaseURL: server.URL + "/v1", APIKey: "k"})
	_, err := client.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "x"}}})
	typed, ok := providererror.As(err)
	if !ok {
		t.Fatalf("error not typed: %v", err)
	}
	if typed.Code != providererror.CodeRateLimited || typed.RetryAfter != "9" || typed.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("typed = %+v", typed)
	}
}

func TestMessagesFromEnvelope(t *testing.T) {
	messages := MessagesFromEnvelope("describe", []InlineImage{{MediaType: "image/png", DataB64: "abc"}})
	if len(messages) != 1 {
		t.Fatalf("messages = %d", len(messages))
	}
	parts, ok := messages[0].Content.([]ContentPart)
	if !ok || len(parts) != 2 || parts[1].ImageURL == nil {
		t.Fatalf("parts = %#v", messages[0].Content)
	}
	if parts[1].ImageURL.URL != "data:image/png;base64,abc" {
		t.Fatalf("url = %q", parts[1].ImageURL.URL)
	}
}
