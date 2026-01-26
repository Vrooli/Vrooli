package docsearch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchFilesScenario(t *testing.T) {
	root, service := setupDocSearch(t)
	_ = root

	results, err := service.SearchFiles(context.Background(), FileSearchRequest{
		Pattern:        "README.md",
		Scope:          ScopeScenario,
		Scenario:       "alpha",
		IncludeContent: true,
		Limit:          5,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Scenario != "alpha" {
		t.Fatalf("expected scenario alpha, got %s", results[0].Scenario)
	}
	if results[0].RelativePath != "README.md" {
		t.Fatalf("expected relative path README.md, got %s", results[0].RelativePath)
	}
	if results[0].DocType != "readme" {
		t.Fatalf("expected doc type readme, got %s", results[0].DocType)
	}
	if !strings.Contains(results[0].ContentPreview, "Alpha Scenario") {
		t.Fatalf("expected preview to include content")
	}
}

func TestSearchTextScenario(t *testing.T) {
	_, service := setupDocSearch(t)
	results, err := service.SearchText(context.Background(), TextSearchRequest{
		Query:        "hello world",
		Scope:        ScopeScenario,
		Scenario:     "alpha",
		ContextLines: 1,
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected matches")
	}
	if results[0].LineNumber == 0 {
		t.Fatalf("expected line number")
	}
	if results[0].Scenario != "alpha" {
		t.Fatalf("expected scenario alpha, got %s", results[0].Scenario)
	}
}

func TestSearchUnifiedDefaults(t *testing.T) {
	_, service := setupDocSearch(t)
	resp, err := service.SearchUnified(context.Background(), UnifiedSearchRequest{
		Query:    "README.md",
		Scope:    ScopeScenario,
		Scenario: "alpha",
		Limit:    5,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || len(resp.Results) == 0 {
		t.Fatalf("expected unified results")
	}
}

func setupDocSearch(t *testing.T) (string, *Service) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("failed to create scenarios root: %v", err)
	}
	makeScenario(t, scenariosRoot, "alpha", []fixtureFile{
		{Path: "README.md", Content: "# Alpha Scenario"},
		{Path: "docs/guides/intro.md", Content: "Hello world\nMore lines"},
		{Path: "docs/internal/PROBLEMS.md", Content: "## 2026-01-01: Issue"},
	})
	makeScenario(t, scenariosRoot, "beta", []fixtureFile{
		{Path: "README.md", Content: "# Beta Scenario"},
		{Path: "docs/concepts/ARCHITECTURE.md", Content: "Architecture notes"},
	})
	service, err := NewService(scenariosRoot)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	return root, service
}

type fixtureFile struct {
	Path    string
	Content string
}

func makeScenario(t *testing.T, scenariosRoot, name string, files []fixtureFile) {
	path := filepath.Join(scenariosRoot, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}
	for _, file := range files {
		full := filepath.Join(path, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(full, []byte(file.Content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}
}
