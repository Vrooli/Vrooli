package agentcatalog

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

func TestExtractModelExamplesReadsOnlyTheModelOption(t *testing.T) {
	help := "  --agents <json> JSON example 'not-a-model'\n" +
		"  --model <model> Model alias (e.g. 'future-fast', 'future-smart')\n" +
		"      Full model name (e.g. 'vendor/future-model')\n" +
		"  --name <name> Session name (e.g. 'not-a-model')\n"
	got := extractModelExamples(help)
	want := []string{"future-fast", "future-smart", "vendor/future-model"}
	if len(got) != len(want) {
		t.Fatalf("examples=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("examples=%v want=%v", got, want)
		}
	}
}

func TestDiscoverModelsFixtureCoversEveryRunnerAdapter(t *testing.T) {
	for _, runner := range []string{"codex", "claude-code", "opencode", "grok"} {
		t.Run(runner, func(t *testing.T) {
			env := discoveryInlineEnv(runner)
			t.Setenv(env, `{"models":["fixture-primary",{"slug":"fixture-fallback"}],"fetched_at":"2026-08-03T00:00:00Z"}`)
			catalog, err := DiscoverModels(context.Background(), runner)
			if err != nil || !catalog.Contains("fixture-primary") || !catalog.Contains("fixture-fallback") {
				t.Fatalf("catalog=%+v err=%v", catalog, err)
			}
		})
	}
}

func TestDiscoverModelsFixtureDegradationIsTypedForEveryRunner(t *testing.T) {
	for _, runner := range []string{"codex", "claude-code", "opencode", "grok"} {
		t.Run(runner, func(t *testing.T) {
			t.Setenv(discoveryInlineEnv(runner), "")
			t.Setenv(discoveryOverrideEnv(runner), filepath.Join(t.TempDir(), "missing.json"))
			_, err := DiscoverModels(context.Background(), runner)
			if err == nil || !errors.Is(err, ErrModelDiscoveryUnavailable) {
				t.Fatalf("err=%v, want typed discovery degradation", err)
			}
		})
	}
}
