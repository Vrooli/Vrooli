package docs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRunner_Success(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	writeFile(t, filepath.Join(dir, "README.md"), fmt.Sprintf(
		"# Title\n\nSee [local](./local.md).\n\n```mermaid\ngraph TB\n  A --> B\n```\n\n[external](%s)\n",
		server.URL,
	))
	writeFile(t, filepath.Join(dir, "local.md"), "ok\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	}, WithHTTPClient(server.Client()))

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got failure: %+v", result)
	}
	if result.Summary.FilesChecked != 2 {
		t.Fatalf("expected 2 files checked, got %d", result.Summary.FilesChecked)
	}
	if result.Summary.BrokenLinks != 0 {
		t.Fatalf("expected no broken links, got %d", result.Summary.BrokenLinks)
	}
	if result.Summary.MermaidFailures != 0 {
		t.Fatalf("expected mermaid success, got %d failures", result.Summary.MermaidFailures)
	}
}

func TestRunner_UnclosedFenceFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```\nunclosed\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure due to unclosed fence")
	}
	if result.Summary.MarkdownFailures != 1 {
		t.Fatalf("expected 1 markdown failure, got %d", result.Summary.MarkdownFailures)
	}
}

func TestRunner_BrokenLocalLink(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "broken [link](missing.md)\n")

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     DefaultSettings(),
	})

	result := runner.Run(context.Background())
	if result.Success {
		t.Fatal("expected failure due to broken link")
	}
	if result.Summary.BrokenLinks != 1 {
		t.Fatalf("expected 1 broken link, got %d", result.Summary.BrokenLinks)
	}
}

func TestRunner_MermaidWarningWhenNotStrict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```mermaid\nnot a diagram\n```\n")

	settings := DefaultSettings()
	strict := false
	settings.Mermaid.Strict = &strict

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatal("expected success (warning only) when mermaid strict disabled")
	}
	if result.Summary.MarkdownWarnings == 0 {
		t.Fatalf("expected mermaid warning, got summary %+v", result.Summary)
	}
}

func TestRunner_MarkdownDisabledSkipsValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "```\nunclosed\n")

	settings := DefaultSettings()
	disabled := false
	settings.Markdown.Enabled = &disabled

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success when markdown validation disabled, got failure: %+v", result)
	}
	if result.Summary.MarkdownFailures != 0 {
		t.Fatalf("expected no markdown failures when disabled, got %d", result.Summary.MarkdownFailures)
	}
}

func TestRunner_AbsolutePathAllowlist(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "See /home/test/docs for details\n")

	settings := DefaultSettings()
	settings.Paths.Allow = []string{"/home/"}

	runner := New(Config{
		ScenarioDir:  dir,
		ScenarioName: "demo",
		Settings:     settings,
	})

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success with allowlisted absolute path, got failure: %+v", result)
	}
	if result.Summary.AbsolutePathHits != 1 {
		t.Fatalf("expected 1 absolute path hit, got %d", result.Summary.AbsolutePathHits)
	}
	if result.Summary.AbsoluteFailures != 0 {
		t.Fatalf("expected no absolute path failures when allowlisted, got %d", result.Summary.AbsoluteFailures)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
