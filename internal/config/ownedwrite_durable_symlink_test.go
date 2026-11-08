//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDurableOwnedWriteConfirmsPhysicalAncestorsBehindDirectoryAlias(t *testing.T) {
	logical := t.TempDir()
	physicalParent := t.TempDir()
	physical := filepath.Join(physicalParent, "new-physical-root")
	if err := os.Mkdir(physical, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(logical, "runtime-home")
	if err := os.Symlink(physical, alias); err != nil {
		t.Fatal(err)
	}
	physicalParent, err := filepath.EvalSymlinks(physicalParent)
	if err != nil {
		t.Fatal(err)
	}
	confirmed := false
	if err := writeOwnedFileDurableWith(filepath.Join(alias, "state", "intent.json"), []byte("intent"), 0o600, WriteOwnedFileAtomic, func(dir string) error {
		confirmed = confirmed || dir == physicalParent
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Fatal("logical ancestry was confirmed while the physical directory entry remained unconfirmed")
	}
}
