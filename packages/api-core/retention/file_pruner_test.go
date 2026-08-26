package retention

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFilePrunerEnforcesByteBudgetWithoutPartialWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.bin")
	if err := os.WriteFile(path, []byte("123456"), 0o600); err != nil {
		t.Fatal(err)
	}
	pruner, err := NewFilePruner(FileConfig{Path: path, Now: func() time.Time { return time.Unix(100, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	result, err := pruner.Prune(context.Background(), Budget{Name: "cache", MaxBytes: 3})
	if err != nil {
		t.Fatal(err)
	}
	if result.BoundBy != BoundBytes || result.Deleted != 1 || result.FreedBytes != 6 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists, stat error = %v", err)
	}
}

func TestFilePrunerLeavesRecentFileWithinBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.bin")
	if err := os.WriteFile(path, []byte("12"), 0o600); err != nil {
		t.Fatal(err)
	}
	pruner, err := NewFilePruner(FileConfig{Path: path, Now: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	result, err := pruner.Prune(context.Background(), Budget{Name: "cache", MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 0 || result.After.Bytes != 2 {
		t.Fatalf("result = %+v", result)
	}
}

func TestFilePrunerRefusesProtectedExecutable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".vrooli", "bin", "vrooli")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("live"), 0o755); err != nil {
		t.Fatal(err)
	}
	pruner, err := NewFilePruner(FileConfig{Path: path, ProtectedRoots: []string{filepath.Dir(path)}, Now: func() time.Time { return time.Unix(100, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pruner.Prune(context.Background(), Budget{Name: "runtime", MaxBytes: 1})
	if err == nil || !strings.Contains(err.Error(), "protected path") {
		t.Fatalf("Prune error = %v, want protected-path refusal", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("protected executable was removed: %v", statErr)
	}
}
