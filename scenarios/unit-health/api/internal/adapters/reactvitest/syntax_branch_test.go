package reactvitest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

func TestCollectSyntaxReportsMissingAndDryRunInputs(t *testing.T) {
	if rows, reason := (Analyzer{}).CollectSyntax(context.Background(), adapters.QualityInput{Root: t.TempDir(), Workspace: "ui"}); len(rows) != 0 || reason != testquality.MissingInput {
		t.Fatalf("empty syntax = %v, %v", rows, reason)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "case.test.ts"), []byte("test('case', () => {})"), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, reason := (Analyzer{}).CollectSyntax(context.Background(), adapters.QualityInput{Root: root, Workspace: "ui", Executed: false})
	if reason != testquality.ReasonNone || len(rows) != 4 {
		t.Fatalf("dry syntax = %v, %v", rows, reason)
	}
}

func TestCollectSyntaxMarksOwnerUnavailableWhenExecutionCannotResolveOwner(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "case.test.ts"), []byte("test('case', () => {})"), 0o600); err != nil {
		t.Fatal(err)
	}
	rows, reason := (Analyzer{}).CollectSyntax(context.Background(), adapters.QualityInput{Root: root, Workspace: "ui", Executed: true})
	if reason != testquality.ReasonNone || len(rows) != 4 || rows[0].Status == "" {
		t.Fatalf("owner unavailable = %v, %v", rows, reason)
	}
}
