//go:build !windows

package lightingruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/config"
)

func TestLightingStateDirectoryWriterHealsLegacyAndReaderStaysPassive(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "host-lighting")
	if err = EnsureStateDirectory(path, true); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil || before.Mode().Perm() != 0o700 {
		t.Fatalf("new private directory %v %v", before, err)
	}
	if err = os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err = EnsureStateDirectory(path, false); err == nil {
		t.Fatal("nonprivate read unexpectedly qualified")
	}
	read, _ := os.Stat(path)
	if read.Mode().Perm() != 0o755 {
		t.Fatal("reader changed storage")
	}
	uid, _ := config.RepairIdentity()
	if err = verifyLightingStateDirectoryForOwner(path, true, uid+1); err == nil {
		t.Fatal("foreign owner was reassigned or chmodded")
	}
	foreign, _ := os.Stat(path)
	if foreign.Mode().Perm() != 0o755 {
		t.Fatal("foreign-owner refusal changed directory")
	}
	if err = EnsureStateDirectory(path, true); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	if after.Mode().Perm() != 0o700 || !os.SameFile(before, after) {
		t.Fatal("writer replaced owner directory instead of safely tightening it")
	}
}

func TestLightingStateDirectoryRefusesAliasAndForeignWrites(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "target")
	if err = os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "host-lighting")
	if err = os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	if err = EnsureStateDirectory(alias, true); err == nil {
		t.Fatal("symlink owner directory admitted")
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm() != 0o755 {
		t.Fatal("alias refusal chmodded target")
	}
	if err = os.Chmod(target, 0o777); err != nil {
		t.Fatal(err)
	}
	if err = EnsureStateDirectory(target, true); err == nil {
		t.Fatal("foreign-writable directory silently adopted")
	}
	info, _ = os.Stat(target)
	if info.Mode().Perm() != 0o777 {
		t.Fatal("unsafe-directory refusal changed storage")
	}
}
