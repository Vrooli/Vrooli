package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageArtifactDirectorySwapsWithoutMutatingCurrentTree(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "dist")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, staged, cleanup, err := stageArtifact(target, true)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := os.WriteFile(filepath.Join(staged, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := swapArtifact(staged, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("target = %q, want new", got)
	}
}

func TestStageArtifactFileLeavesCurrentArtifactWhenBuildDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "app")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, staged, cleanup, err := stageArtifact(target, false)
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("staged path still exists after cleanup: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("target changed before publish: %q", got)
	}
}
