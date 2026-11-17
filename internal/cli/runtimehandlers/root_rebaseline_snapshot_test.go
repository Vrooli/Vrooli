package runtimehandlers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestHubSnapshotPrunePreservesActiveAndRecoverable(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"mk", "tools"} {
		if err := os.Symlink(filepath.Join(repoRoot, name), filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	build := filepath.Join(dir, "build")
	if err := os.MkdirAll(build, 0700); err != nil {
		t.Fatal(err)
	}
	snapshotRoot := filepath.Join(dir, ".vrooli", "state", "root-rebaseline", "snapshots")
	if err := os.MkdirAll(snapshotRoot, 0700); err != nil {
		t.Fatal(err)
	}
	makeSnapshot := func(suffix, marker string) string {
		p := filepath.Join(snapshotRoot, "rebaseline-hub."+suffix)
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, ".lease"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		if marker != "" {
			if err := os.WriteFile(filepath.Join(p, marker), nil, 0600); err != nil {
				t.Fatal(err)
			}
		}
		old := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
		return p
	}
	completed := makeSnapshot("complete", ".completed")
	orphan := makeSnapshot("orphan00", "")
	recoverable := makeSnapshot("pending0", ".ready")
	active := makeSnapshot("active00", ".completed")
	current := makeSnapshot("current0", ".completed")
	retired := makeSnapshot("retired0", ".ready")
	if err := os.WriteFile(filepath.Join(retired, ".retire-approved"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(retired, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(build, "vrooli"), []byte("#!/bin/sh\ntest -f \"$4/.retire-approved\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snapshotRoot, "current"), []byte(current+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	release, err := platform.AcquireFileLock(filepath.Join(active, ".lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	cmd := exec.Command("make", "-f", filepath.Join(repoRoot, "Makefile"), "rebaseline-hub-prune", "BUILD_DIR=build", "VROOLI="+filepath.Join(build, "vrooli"), "BUNDLE_PUBLISHER=go -C "+repoRoot+" run ./cmd/vrooli-atomic-install")
	cmd.Env = append(os.Environ(), "HOME="+dir)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prune: %v %s", err, out)
	}
	for _, p := range []string{completed, orphan, retired} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("stale snapshot retained %s: %v", p, err)
		}
	}
	for _, p := range []string{recoverable, active, current} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("protected snapshot removed %s: %v", p, err)
		}
	}
}
