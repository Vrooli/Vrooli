//go:build unix

package storage

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestEnsureAllDirsRejectsStorageUnderAnotherUsersExistingParent(t *testing.T) {
	root := t.TempDir()
	resolver, err := NewResolver(ResolverConfig{})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	originalEUID := currentEUID
	currentEUID = func() int { return originalEUID() + 1 }
	t.Cleanup(func() { currentEUID = originalEUID })

	_, err = EnsureAllDirs(resolver, Options{ResourceID: "vault", RootOverride: root}, 0o700)
	if err == nil {
		t.Fatal("EnsureAllDirs unexpectedly accepted a different effective user")
	}
	var storageErr *Error
	if !errors.As(err, &storageErr) || storageErr.Kind != ErrOwnership {
		t.Fatalf("EnsureAllDirs error = %#v, want ErrOwnership", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "config", "vrooli", "resources", "vault")); !os.IsNotExist(statErr) {
		t.Fatalf("storage directory was created despite owner mismatch: %v", statErr)
	}
}

func TestEnsureAllDirsUsesInvokingSudoUserForStorageOwnership(t *testing.T) {
	root := t.TempDir()
	resolver, err := NewResolver(ResolverConfig{})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	originalEUID := currentEUID
	originalIDs := invokingUserIDs
	originalHome := invokingUserHomeDir
	currentEUID = func() int { return originalEUID() + 1 }
	invokingUserIDs = func() (int, int, bool) { return originalEUID(), os.Getegid(), true }
	invokingUserHomeDir = func() (string, error) { return root, nil }
	t.Cleanup(func() {
		currentEUID = originalEUID
		invokingUserIDs = originalIDs
		invokingUserHomeDir = originalHome
	})

	paths, err := EnsureAllDirs(resolver, Options{ResourceID: "vault", RootOverride: root}, 0o700)
	if err != nil {
		t.Fatalf("EnsureAllDirs: %v", err)
	}
	info, err := os.Stat(paths.DataDir)
	if err != nil {
		t.Fatalf("Stat data directory: %v", err)
	}
	if got := info.Sys().(*syscall.Stat_t).Uid; got != uint32(originalEUID()) {
		t.Fatalf("data directory uid = %d, want invoking uid %d", got, originalEUID())
	}
}

func TestRepairRootOwnedFileForInvokingUserIsNarrowAndRootOnly(t *testing.T) {
	home := t.TempDir()
	state := filepath.Join(home, ".local", "state", "vrooli", "resource.json")
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	originalEUID, originalIDs, originalHome, originalChown := currentEUID, invokingUserIDs, invokingUserHomeDir, chownStoragePath
	currentEUID = func() int { return 0 }
	invokingUserIDs = func() (int, int, bool) { return os.Getuid() + 1, os.Getgid(), true }
	invokingUserHomeDir = func() (string, error) { return home, nil }
	chownCalls := 0
	chownStoragePath = func(path string, uid, gid int) error {
		chownCalls++
		return nil
	}
	t.Cleanup(func() {
		currentEUID, invokingUserIDs, invokingUserHomeDir, chownStoragePath = originalEUID, originalIDs, originalHome, originalChown
	})

	if _, err := RepairRootOwnedFileForInvokingUser(state); err == nil {
		t.Fatal("refused to reassign a non-root-owned file")
	}
	if chownCalls != 0 {
		t.Fatalf("non-root file triggered %d ownership changes", chownCalls)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if _, err := RepairRootOwnedFileForInvokingUser(outside); err == nil {
		t.Fatal("accepted a path outside the invoking user's home")
	}
	if chownCalls != 0 {
		t.Fatalf("out-of-home path triggered %d ownership changes", chownCalls)
	}
}

func TestRepairRootOwnedFileRejectsSymlink(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "target.json")
	link := filepath.Join(home, "state.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	originalEUID, originalIDs, originalHome := currentEUID, invokingUserIDs, invokingUserHomeDir
	currentEUID = func() int { return 0 }
	invokingUserIDs = func() (int, int, bool) { return os.Getuid() + 1, os.Getgid(), true }
	invokingUserHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { currentEUID, invokingUserIDs, invokingUserHomeDir = originalEUID, originalIDs, originalHome })
	if _, err := RepairRootOwnedFileForInvokingUser(link); err == nil {
		t.Fatal("accepted a symlink for ownership repair")
	}
}
