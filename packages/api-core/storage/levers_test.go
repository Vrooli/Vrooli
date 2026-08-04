package storage

import (
	"strings"
	"testing"
)

func TestBuildLeverRegistryChecksSafetyClasses(t *testing.T) {
	if _, err := BuildLeverRegistry([]Lever{{Key: "VROOLI_BAD", Owner: "x", Entry: "e", Target: "/x", Scope: ScopeSession}}, func(string) string { return "" }); err == nil {
		t.Fatal("reserved prefix accepted")
	}
	if _, err := BuildLeverRegistry([]Lever{{Key: "GOCACHE", Owner: "x", Entry: "e", Target: "/x"}}, func(string) string { return "" }); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("missing scope not rejected: %v", err)
	}
	got, err := BuildLeverRegistry([]Lever{{Key: "GOCACHE", Owner: "x", Entry: "e", Target: "/x", Scope: ScopeProcessTree}}, func(string) string { return "/ambient" })
	if err != nil || len(got.Warnings) != 1 {
		t.Fatalf("ambient mismatch = %+v, err %v", got, err)
	}
}

func TestBuildLeverRegistryAllowsPerContainerReuse(t *testing.T) {
	registry, err := BuildLeverRegistry([]Lever{
		{Key: "RESOURCE_DATA_DIR", Owner: "reranker", Entry: "models", Target: "/one", Scope: ScopeContainer},
		{Key: "RESOURCE_DATA_DIR", Owner: "searxng", Entry: "cache", Target: "/two", Scope: ScopeContainer},
	}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("container-scoped reuse: %v", err)
	}
	if len(registry.Levers) != 2 {
		t.Fatalf("levers = %d, want 2", len(registry.Levers))
	}
}

func TestBuildLeverRegistryRejectsEnvironmentExportCollision(t *testing.T) {
	_, err := BuildLeverRegistryWithExports([]Lever{{Key: "OLLAMA_HOST", Owner: "scenario-a", Entry: "models", Target: "/shadow/ollama", Scope: ScopeProcessTree}}, func(string) string { return "" }, map[string]string{"OLLAMA_HOST": "http://ollama:11434"})
	if err == nil || !strings.Contains(err.Error(), "environment_exports") {
		t.Fatalf("expected cross-direction collision, got %v", err)
	}
}

func TestBuildLeverRegistryAllowsSameEnvironmentExportTarget(t *testing.T) {
	if _, err := BuildLeverRegistryWithExports([]Lever{{Key: "OLLAMA_HOST", Owner: "scenario-a", Entry: "models", Target: "http://ollama:11434", Scope: ScopeProcessTree}}, func(string) string { return "" }, map[string]string{"OLLAMA_HOST": "http://ollama:11434"}); err != nil {
		t.Fatalf("same target should be allowed: %v", err)
	}
}
