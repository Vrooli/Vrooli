package ai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOpenRouterProvider_ResolvesRole verifies the provider resolves a policy
// role (not a hard-coded slug) through the resource-openrouter seam.
func TestOpenRouterProvider_ResolvesRole(t *testing.T) {
	var gotArgs []string
	p := &OpenRouterProvider{
		Role: "chat.quality",
		Runner: func(_ context.Context, args []string) ([]byte, error) {
			gotArgs = args
			return []byte("vendor/some-model\n"), nil
		},
	}
	model, err := p.resolveModel(context.Background(), p.Role)
	if err != nil {
		t.Fatalf("resolveModel: %v", err)
	}
	if model != "vendor/some-model" {
		t.Errorf("expected trimmed resolved slug, got %q", model)
	}
	want := []string{"policy", "resolve", "--role", "chat.quality", "--field", "model"}
	if strings.Join(gotArgs, " ") != strings.Join(want, " ") {
		t.Errorf("unexpected resolve args: %v", gotArgs)
	}
}

// TestOpenRouterProvider_ResolveError surfaces resolver failures instead of
// falling back to a concrete model slug.
func TestOpenRouterProvider_ResolveError(t *testing.T) {
	p := &OpenRouterProvider{
		Role: "chat.default",
		Runner: func(_ context.Context, _ []string) ([]byte, error) {
			return nil, fmt.Errorf("boom")
		},
	}
	if _, err := p.resolveModel(context.Background(), p.Role); err == nil {
		t.Fatal("expected error when resolver fails")
	}
}

// TestNewOpenRouterProvider_DefaultRole verifies the default role is chat.default
// and no concrete model slug is baked in.
func TestNewOpenRouterProvider_DefaultRole(t *testing.T) {
	t.Setenv("WC_OPENROUTER_ROLE", "")
	p := NewOpenRouterProvider()
	if p.Role != "chat.default" {
		t.Errorf("expected default role chat.default, got %q", p.Role)
	}
}

func TestCheckProviderResponse_OK(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
	if err := checkProviderResponse(resp, "test"); err != nil {
		t.Errorf("200 should return nil, got %v", err)
	}
}

func TestCheckProviderResponse_NonOK(t *testing.T) {
	rec := httptest.NewRecorder()
	_, _ = rec.WriteString("rate limited")
	resp := rec.Result()
	resp.StatusCode = http.StatusTooManyRequests
	err := checkProviderResponse(resp, "openrouter")
	if err == nil {
		t.Fatal("non-200 should return error")
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Errorf("error should mention provider name, got %v", err)
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention status code, got %v", err)
	}
}
