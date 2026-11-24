package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/ai-provider-go/providererror"
)

func TestOpenRouterProviderUsesSharedClientAndKeepsRolePolicy(t *testing.T) {
	var gotRequest struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fixture-key" {
			t.Errorf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"router/served-model","choices":[{"message":{"content":"generated copy"}}]}`))
	}))
	defer server.Close()

	var resolved []string
	provider := NewOpenRouterProvider("fixture-key", "brand.copy")
	provider.BaseURL = server.URL + "/v1"
	provider.HTTPClient = server.Client()
	provider.Runner = func(_ context.Context, args []string) ([]byte, error) {
		resolved = append([]string(nil), args...)
		return []byte("vendor/policy-model\n"), nil
	}
	got, err := provider.GenerateText(context.Background(), TextRequest{Prompt: "write a tagline"})
	require.NoError(t, err)
	require.Equal(t, []string{"policy", "resolve", "--role", "brand.copy", "--field", "model"}, resolved)
	require.Equal(t, "vendor/policy-model", gotRequest.Model)
	require.Len(t, gotRequest.Messages, 1)
	require.Equal(t, "user", gotRequest.Messages[0].Role)
	require.Equal(t, "write a tagline", gotRequest.Messages[0].Content)
	require.Equal(t, TextResponse{Text: "generated copy", Provider: "openrouter", Model: "router/served-model"}, got)
}

func TestOpenRouterProviderHonorsExplicitModelWithoutPolicyLookup(t *testing.T) {
	var request struct {
		Model string `json:"model"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		_, _ = w.Write([]byte(`{"model":"explicit-model","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("fixture-key", "brand.copy")
	provider.BaseURL = server.URL
	provider.HTTPClient = server.Client()
	provider.Runner = func(context.Context, []string) ([]byte, error) {
		t.Fatal("policy lookup should not run for an explicit model")
		return nil, nil
	}
	got, err := provider.GenerateText(context.Background(), TextRequest{Prompt: "hello", Model: "operator/model"})
	require.NoError(t, err)
	require.Equal(t, "operator/model", request.Model)
	require.Equal(t, "explicit-model", got.Model)
}

func TestOpenRouterProviderClassifiesHTTPFailuresThroughSharedClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "11")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("fixture-key", "brand.copy")
	provider.BaseURL = server.URL
	provider.HTTPClient = server.Client()
	provider.Runner = func(context.Context, []string) ([]byte, error) { return []byte("resolved-model"), nil }
	_, err := provider.GenerateText(context.Background(), TextRequest{Prompt: "hello"})
	require.Error(t, err)
	typed, ok := providererror.As(err)
	require.True(t, ok)
	require.Equal(t, providererror.CodeRateLimited, typed.Code)
	require.Equal(t, "11", typed.RetryAfter)
}
