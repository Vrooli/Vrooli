package phases

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

func TestRunDocsPhase_NoMarkdown(t *testing.T) {
	dir := t.TempDir()
	env := workspace.Environment{
		ScenarioName: "demo",
		ScenarioDir:  dir,
		TestDir:      filepath.Join(dir, "test"),
	}
	if err := os.MkdirAll(env.TestDir, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	report := runDocsPhase(context.Background(), env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected no error, got %v", report.Err)
	}
	if report.FailureClassification != "" {
		t.Fatalf("expected no failure classification, got %s", report.FailureClassification)
	}
}
