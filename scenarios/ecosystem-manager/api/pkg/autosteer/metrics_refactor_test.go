package autosteer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEstimateDuplicationCountsDuplicates(t *testing.T) {
	tempDir := t.TempDir()
	scenarioPath := filepath.Join(tempDir, "scenarios", "demo")
	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	content := []byte("function test() {}\n")
	if err := os.WriteFile(filepath.Join(scenarioPath, "a.ts"), content, 0o644); err != nil {
		t.Fatalf("failed to write duplicate file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "b.ts"), content, 0o644); err != nil {
		t.Fatalf("failed to write duplicate file: %v", err)
	}

	collector := NewRefactorMetricsCollector(tempDir, "")
	score := collector.estimateDuplication(scenarioPath)

	if score < 40 || score > 60 {
		t.Fatalf("expected duplication score around 50, got %.2f", score)
	}
}

func TestEstimateDuplicationNoFiles(t *testing.T) {
	tempDir := t.TempDir()
	scenarioPath := filepath.Join(tempDir, "scenarios", "empty")
	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	collector := NewRefactorMetricsCollector(tempDir, "")
	score := collector.estimateDuplication(scenarioPath)

	if score != 0 {
		t.Fatalf("expected 0 duplication score, got %.2f", score)
	}
}
