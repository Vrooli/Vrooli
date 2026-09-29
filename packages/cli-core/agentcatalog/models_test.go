package agentcatalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoverModelsReadsConfiguredCodexCatalogAndDeduplicatesSlugs(t *testing.T) {
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

func TestParseModelCatalogPreservesRunnerModelsOmittedByLegacyCache(t *testing.T) {
	catalog, err := parseModelCatalog("codex", []byte(`{"models":[{"slug":"gpt-6-luna","display_name":"GPT-6-Luna"},{"slug":"gpt-6-sol","display_name":"GPT-6-Sol"}],"source":"codex debug models","exhaustive":true}`), "codex debug models")
	if err != nil {
		t.Fatalf("parseModelCatalog: %v", err)
	}
	if !catalog.Contains("gpt-6-luna") || !catalog.Contains("gpt-6-sol") {
		t.Fatalf("catalog=%+v; runner models were lost", catalog)
	}
	if !catalog.Exhaustive {
		t.Fatalf("catalog=%+v; explicit exhaustive marker was lost", catalog)
	}
}

func TestDiscoverModelsPrefersLiveCodexCatalogOverStaleCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable fixture uses a POSIX shell")
	}
	home := t.TempDir()
	cacheDir := filepath.Join(home, ".codex")
	if err := os.Mkdir(cacheDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "models_cache.json"), []byte(`{"models":[{"slug":"gpt-5.6-luna"}],"source":"stale cache"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runnerDir := t.TempDir()
	codexPath := filepath.Join(runnerDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nprintf '%s\\n' '{\"models\":[{\"slug\":\"gpt-6-luna\"},{\"slug\":\"gpt-6-sol\"}],\"source\":\"live runner\"}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", runnerDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	catalog, err := DiscoverModels(context.Background(), "codex")
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if catalog.Source != "live runner (codex debug models)" || !catalog.Contains("gpt-6-luna") || !catalog.Contains("gpt-6-sol") {
		t.Fatalf("catalog=%+v; live runner catalog did not supersede stale cache", catalog)
	}
	if !catalog.Authoritative {
		t.Fatalf("catalog=%+v; live runner result must be authoritative", catalog)
	}
}

func TestDiscoverModelsRecordsTheProducingRunnerIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable fixture uses a POSIX shell")
	}
	runnerDir := t.TempDir()
	codexPath := filepath.Join(runnerDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then printf '%s\\n' 'codex-cli 0.156.1'; else printf '%s\\n' '{\"models\":[{\"slug\":\"gpt-6-luna\"}]}' ; fi\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", runnerDir)
	catalog, err := DiscoverModels(context.Background(), "codex")
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if catalog.BinaryPath != codexPath || catalog.RunnerVersion != "codex-cli 0.156.1" {
		t.Fatalf("catalog identity=%+v", catalog)
	}
}

func TestResolveRunnerBinaryDoesNotReplaceAnUnmanagedCodex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path fixture uses POSIX semantics")
	}
	runnerDir := t.TempDir()
	codexPath := filepath.Join(runnerDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", runnerDir)
	got, err := resolveRunnerBinary("codex")
	if err != nil || got != codexPath {
		t.Fatalf("resolved=%q err=%v", got, err)
	}
}

func TestResolveRunnerBinaryUsesTheLaunchBinaryBehindAVrooliShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path fixture uses POSIX semantics")
	}
	root := t.TempDir()
	shimDir := filepath.Join(root, ".vrooli", "shims")
	if err := os.MkdirAll(shimDir, 0o700); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(shimDir, "codex")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shimDir)
	got, err := resolveRunnerBinary("codex")
	if err != nil {
		t.Fatal(err)
	}
	want := shim
	for _, candidate := range []string{"/usr/bin/codex", "/bin/codex"} {
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			want = candidate
			break
		}
	}
	if got != want {
		t.Fatalf("resolved=%q want=%q", got, want)
	}
}

func TestDiscoverModelsMarksCompatibilityCacheNonAuthoritative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable fixture uses a POSIX shell")
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(`{"models":["stale-model"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runnerDir := t.TempDir()
	// The command exists but cannot provide a catalog, forcing the documented
	// compatibility-cache path without making the test depend on a host CLI.
	codexPath := filepath.Join(runnerDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", runnerDir)

	catalog, err := DiscoverModels(context.Background(), "codex")
	if err != nil {
		t.Fatalf("DiscoverModels: %v", err)
	}
	if catalog.Authoritative || catalog.Source != filepath.Join(home, ".codex", "models_cache.json") {
		t.Fatalf("catalog=%+v; compatibility cache must be marked non-authoritative", catalog)
	}
	if catalog.IsAuthoritative() {
		t.Fatalf("catalog=%+v; compatibility cache must not be treated as authoritative", catalog)
	}
}

func TestIsAuthoritativePreservesLegacyInMemoryFixtures(t *testing.T) {
	if !(LiveModelCatalog{Models: []string{"fixture"}}).IsAuthoritative() {
		t.Fatal("legacy in-memory fixture should remain compatible")
	}
	if (LiveModelCatalog{Models: []string{"fixture"}, Source: "compatibility cache"}).IsAuthoritative() {
		t.Fatal("named fallback source must not be treated as authoritative")
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
