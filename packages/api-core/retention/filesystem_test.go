package retention

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteContainedRejectsEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "capture")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := DeleteContained(context.Background(), root, link, nil); err == nil {
		t.Fatal("expected escaping symlink to be rejected")
	}
}

func TestDeleteContainedRejectsSymlinkIntoProtectedRoot(t *testing.T) {
	root := t.TempDir()
	protected := filepath.Join(root, "plan-artifacts")
	if err := os.MkdirAll(protected, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "cache-link")
	if err := os.Symlink(protected, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := DeleteContained(context.Background(), root, link, []string{protected}); err == nil {
		t.Fatal("expected symlink into protected root to be rejected")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("protected symlink was removed: %v", err)
	}
}

func TestDeleteContainedRemovesStrictChild(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "capture")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "artifact"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := DeleteContained(context.Background(), root, target, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target still exists or stat failed: %v", err)
	}
}
