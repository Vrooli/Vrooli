package gotest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

func TestAnalyzerIdentityAndSourceAnalysis(t *testing.T) {
	analyzer := Analyzer{}
	if analyzer.Identity().ID != "go" || !analyzer.Matches(adapters.Match{Language: "GO"}) || analyzer.Matches(adapters.Match{Language: "rust"}) {
		t.Fatal("Go analyzer identity or matching is wrong")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "case_test.go"), []byte("package p\nimport \"testing\"\nfunc TestCase(t *testing.T){ t.Fatal(\"expected\") }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vendor", "ignored.go"), []byte("package ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, reason := analyzer.AnalyzeSource(context.Background(), adapters.QualityInput{Root: root, Workspace: "api"})
	if reason != testquality.ReasonNone || len(rows) == 0 || rows[0].RuleID == "" {
		t.Fatalf("analysis = %v, %v", rows, reason)
	}
}

func TestAnalyzerReportsWalkAndCancellationFailures(t *testing.T) {
	analyzer := Analyzer{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	if rows, reason := analyzer.AnalyzeSource(ctx, adapters.QualityInput{Root: root}); len(rows) != 0 || reason == "" {
		t.Fatalf("cancelled analysis = %v, %v", rows, reason)
	}
	if rows, reason := analyzer.AnalyzeSource(context.Background(), adapters.QualityInput{Root: filepath.Join(root, "missing")}); len(rows) != 0 || reason == "" {
		t.Fatalf("missing-root analysis = %v, %v", rows, reason)
	}
}
