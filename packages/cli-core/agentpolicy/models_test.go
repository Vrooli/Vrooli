package agentpolicy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverModelsReadsCodexCacheAndDeduplicatesSlugs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "models.json")
	if err := os.WriteFile(path, []byte(`{"fetched_at":"2026-08-04T00:00:00Z","models":[{"slug":"gpt-new"},{"slug":"gpt-new"},{"id":"fallback"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_CODEX_MODELS_FILE", path)
	catalog, err := DiscoverModels(context.Background(), "codex")
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if !catalog.Contains("gpt-new") || !catalog.Contains("fallback") || len(catalog.Models) != 2 {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	if catalog.FetchedAt != "2026-08-04T00:00:00Z" {
		t.Fatalf("fetched_at=%q", catalog.FetchedAt)
	}
}

func TestDiscoverModelsReturnsDistinctUnavailableError(t *testing.T) {
	t.Setenv("VROOLI_CODEX_MODELS_FILE", filepath.Join(t.TempDir(), "missing.json"))
	_, err := DiscoverModels(context.Background(), "codex")
	if err == nil || !errors.Is(err, ErrModelDiscoveryUnavailable) {
		t.Fatalf("error=%v, want ErrModelDiscoveryUnavailable", err)
	}
}
