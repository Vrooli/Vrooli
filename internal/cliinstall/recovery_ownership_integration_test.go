//go:build ownership_integration && !windows

package cliinstall

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Run as root in an isolated environment: go test -tags ownership_integration ./internal/cliinstall
func TestSudoRecoveryArtifactsOwnedByInvoker(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "vrooli-nonexistent-test-user")
	t.Setenv("SUDO_UID", "65534")
	t.Setenv("SUDO_GID", "65534")
	source := t.TempDir()
	for _, name := range bundleNames {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := filepath.Join(home, ".vrooli", "snapshot")
	if err := os.MkdirAll(filepath.Dir(snapshot), 0700); err != nil {
		t.Fatal(err)
	}
	if err := PrepareBundle(source, snapshot); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(snapshot, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(data))
	retained, err := RetainRebaselineBundle(home, snapshot, sha)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	ordinary, err := beginOrdinaryBundle(home, snapshot, bin, sha, false, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{snapshot, retained, ordinary} {
		for _, name := range append([]string{"", "SHA256SUMS"}, bundleNames...) {
			path := filepath.Join(dir, name)
			info, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if uid := info.Sys().(*syscall.Stat_t).Uid; uid != 65534 {
				t.Errorf("%s owned by %d, want 65534", path, uid)
			}
		}
	}
}
