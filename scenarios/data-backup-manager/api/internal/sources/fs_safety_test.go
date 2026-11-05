package sources

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:DBM-WORKSPACE-FIDELITY] Synthetic files only; no engine or service.
func TestFilesystemRestorePreservesModesAndRefusesOverwrite(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "restored")
	if err := os.Mkdir(filepath.Join(src, "empty"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "run"), []byte("synthetic"), 0751); err != nil {
		t.Fatal(err)
	}
	c := newFilesystemCapturer()
	if err := c.Restore(context.Background(), RestoreSpec{ArtifactPath: src, Target: dst}); err != nil {
		t.Fatal(err)
	}
	for p, mode := range map[string]os.FileMode{"run": 0751, "empty": 0700} {
		info, err := os.Stat(filepath.Join(dst, p))
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("%s mode lost: %v %v", p, info, err)
		}
	}
	if err := c.Restore(context.Background(), RestoreSpec{ArtifactPath: src, Target: dst}); err == nil {
		t.Fatal("non-empty destination accepted")
	}
}

func TestFilesystemRestoreRejectsSymlinkDestination(t *testing.T) {
	src, outside := t.TempDir(), t.TempDir()
	target := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	if err := newFilesystemCapturer().Restore(context.Background(), RestoreSpec{ArtifactPath: src, Target: target}); err == nil {
		t.Fatal("symlink destination accepted")
	}
}
