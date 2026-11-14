package main

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/buildinfo"
)

func TestUnstampedEntrypointStillChecksUnresolvedActivationState(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_SOURCE_ROOT", root)
	t.Setenv("VROOLI_ROOT", root)
	old := buildinfo.RuntimeHomeSnapshotBase64
	buildinfo.RuntimeHomeSnapshotBase64 = ""
	t.Cleanup(func() { buildinfo.RuntimeHomeSnapshotBase64 = old })
	c, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := c.RuntimeHomeEntry(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	if err := guardRootActivationEntrypoint(false); err != nil {
		t.Fatalf("fresh development command was refused: %v", err)
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		t.Fatal("read-only development fence initialized runtime files")
	}
	directory := filepath.Join(entry.AbsPath, "core-activation")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	// Interrupted or damaged admitted state requires owner recovery, including
	// for a source-built CLI which cannot itself qualify as a native candidate.
	if err := os.WriteFile(filepath.Join(directory, "current.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := guardRootActivationEntrypoint(false); err == nil {
		t.Fatal("unstamped development CLI bypassed the root activation fence")
	}
}
