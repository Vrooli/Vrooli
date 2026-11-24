package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
			_, _ = w.Write([]byte(`{"model":"vendor/served-model","choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":3,"completion_tokens":2,"cost":0.001}}`))
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
	if result.Text != "hello" || result.Model != "vendor/served-model" || result.Usage.PromptTokens == nil || *result.Usage.PromptTokens != 3 {
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

func TestProbeAPIKeyUsesSharedAuthenticatedTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/auth/key" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-synthetic" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "key-probe-test" {
			t.Errorf("User-Agent = %q", got)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("probe response"))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "sk-synthetic", UserAgent: "key-probe-test"})
	status, err := client.ProbeAPIKey(context.Background(), "/api/v1/auth/key")
	if err != nil || status != http.StatusForbidden {
		t.Fatalf("ProbeAPIKey() = (%d, %v), want (403, nil)", status, err)
	}
}

func TestProbeAPIKeyPreservesCancellationWithoutLeakingKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	const key = "sk-synthetic-secret"
	client := New(Config{BaseURL: "http://127.0.0.1", APIKey: key})
	_, err := client.ProbeAPIKey(ctx, "/api/v1/auth/key")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ProbeAPIKey() error = %v, want canceled context", err)
	}
	if err != nil && strings.Contains(err.Error(), key) {
		t.Fatalf("ProbeAPIKey() leaked key in error: %v", err)
	}
}

func TestOpenRawRequestPreservesProviderOwnedPayloadAndResponse(t *testing.T) {
	const requestBody = `{"model":"openai/model","tools":[{"type":"function"}],"stream":false}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/chat/completions" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-synthetic" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("HTTP-Referer"); got != "https://fixture.example" {
			t.Errorf("HTTP-Referer = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != requestBody {
			t.Errorf("request body = %q, %v", body, err)
		}
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("provider-specific error detail"))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL, APIKey: "sk-synthetic"})
	response, err := client.OpenRawRequest(context.Background(), http.MethodPost, "/api/v1/chat/completions", []byte(requestBody), http.Header{"HTTP-Referer": []string{"https://fixture.example"}})
	if err != nil {
		t.Fatalf("OpenRawRequest() error: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnprocessableEntity || response.Header.Get("Retry-After") != "7" {
		t.Fatalf("response = status %d headers %#v", response.StatusCode, response.Header)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != "provider-specific error detail" {
		t.Fatalf("response body = %q, %v", body, err)
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
