package validation

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestCachedModuleEvidenceWalkReusesThePerRequestResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	original := walkModuleEvidenceFilesForEvidence
	walkModuleEvidenceFilesForEvidence = func(root string, dirs []string) ([]string, error) {
		calls.Add(1)
		return original(root, dirs)
	}
	t.Cleanup(func() { walkModuleEvidenceFilesForEvidence = original })

	ctx := withEvidenceWalkCache(context.Background())
	first, err := cachedWalkModuleEvidenceFiles(ctx, root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 {
		t.Fatalf("expected go.mod and main.go, got %v", first)
	}
	second, err := cachedWalkModuleEvidenceFiles(ctx, root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if &first[0] != &second[0] {
		t.Fatal("expected the second scanner plan to reuse the per-request file slice")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("module evidence walk count = %d, want 1", got)
	}
	otherRoot := t.TempDir()
	if _, err := cachedWalkModuleEvidenceFiles(ctx, otherRoot, []string{"."}); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("cross-target module evidence walk count = %d, want 2", got)
	}
}
