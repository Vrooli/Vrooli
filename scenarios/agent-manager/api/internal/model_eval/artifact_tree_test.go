package model_eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureArtifactTreeHashesExactTaskFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "receipt.json"), []byte(`{"state":"refused"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bytesBefore, digestBefore, err := measureArtifactTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if bytesBefore == 0 || digestBefore == "" {
		t.Fatalf("measurement = bytes %d digest %q", bytesBefore, digestBefore)
	}
	if err := os.WriteFile(filepath.Join(root, "receipt.json"), []byte(`{"state":"failed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bytesAfter, digestAfter, err := measureArtifactTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if bytesAfter == bytesBefore || digestAfter == digestBefore {
		t.Fatalf("changed file was not reflected: before=(%d,%s) after=(%d,%s)", bytesBefore, digestBefore, bytesAfter, digestAfter)
	}
}

func TestMeasureArtifactTreeRefusesSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "outside.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := measureArtifactTree(root); err == nil {
		t.Fatal("expected symlinked artifact to be refused")
	}
}
