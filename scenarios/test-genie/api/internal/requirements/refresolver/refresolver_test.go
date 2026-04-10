package refresolver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSplitsFileAndFunction(t *testing.T) {
	parsed := Parse(" api/handlers_test.go :: TestListHandlers ")
	if parsed.FilePath != "api/handlers_test.go" {
		t.Fatalf("expected file path to be trimmed, got %q", parsed.FilePath)
	}
	if parsed.TestFunc != "TestListHandlers" {
		t.Fatalf("expected test function to be parsed, got %q", parsed.TestFunc)
	}
}

func TestResolveMarksExistingFunctionAsValid(t *testing.T) {
	scenarioDir := t.TempDir()
	filePath := filepath.Join(scenarioDir, "api", "handlers_test.go")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("package api\n\nfunc TestListHandlers(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	resolver := NewResolver(scenarioDir)
	result := resolver.Resolve("api/handlers_test.go::TestListHandlers")

	if !result.IsValid() || !result.FileExists || !result.FunctionExists {
		t.Fatalf("expected valid resolution, got %+v", result)
	}
}

func TestResolveSuggestsMovedFileAndSimilarFunction(t *testing.T) {
	scenarioDir := t.TempDir()
	filePath := filepath.Join(scenarioDir, "api", "handlers_test.go")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "package api\n\nfunc TestListHandlers(t *testing.T) {}\nfunc TestLoadHandler(t *testing.T) {}\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	resolver := NewResolver(scenarioDir)
	result := resolver.Resolve("internal/handlers_test.go::TestLoadHandlers")

	if result.Issue != IssueFileNotFound {
		t.Fatalf("expected missing-file issue, got %+v", result)
	}
	if result.Suggestion == nil {
		t.Fatal("expected a suggestion for relocated files")
	}
	if result.Suggestion.FoundFile != "api/handlers_test.go" {
		t.Fatalf("expected best match to be discovered, got %+v", result.Suggestion)
	}
	if !strings.Contains(result.Suggestion.Hint, "TestLoadHandler") {
		t.Fatalf("expected hint to mention similar functions, got %q", result.Suggestion.Hint)
	}
}
