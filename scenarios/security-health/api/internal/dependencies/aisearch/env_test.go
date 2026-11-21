package aisearch

import (
	"context"
	"testing"

	sharedsearch "github.com/vrooli/ai-go/search"
)

func TestResolveConfigEmbeddingUsesPolicyDimensions(t *testing.T) {
	old := resolveEmbeddingPolicy
	t.Cleanup(func() { resolveEmbeddingPolicy = old })

	resolveEmbeddingPolicy = func(_ context.Context, role string) (sharedsearch.EmbeddingPolicy, error) {
		if role != "embedding.default" {
			t.Fatalf("role = %q", role)
		}
		return sharedsearch.EmbeddingPolicy{
			Role:       "embedding.default",
			Model:      "resolved-test-embed:latest",
			Dimensions: 1234,
		}, nil
	}

	cfg, err := ResolveConfigEmbedding(context.Background(), Config{EmbedRole: "embedding.default"})
	if err != nil {
		t.Fatalf("ResolveConfigEmbedding returned error: %v", err)
	}
	if cfg.EmbedRole != "embedding.default" || cfg.EmbeddingPolicy.Dimensions != 1234 {
		t.Fatalf("cfg = %#v", cfg)
	}
}

func TestResolveConfigEmbeddingSkipsDisabledConfig(t *testing.T) {
	called := false
	old := resolveEmbeddingPolicy
	t.Cleanup(func() { resolveEmbeddingPolicy = old })
	resolveEmbeddingPolicy = func(context.Context, string) (sharedsearch.EmbeddingPolicy, error) {
		called = true
		return sharedsearch.EmbeddingPolicy{}, nil
	}

	cfg, err := ResolveConfigEmbedding(context.Background(), Config{Disabled: true})
	if err != nil {
		t.Fatalf("ResolveConfigEmbedding returned error: %v", err)
	}
	if !cfg.Disabled {
		t.Fatalf("cfg = %#v", cfg)
	}
	if called {
		t.Fatal("policy resolver was called for disabled config")
	}
}
