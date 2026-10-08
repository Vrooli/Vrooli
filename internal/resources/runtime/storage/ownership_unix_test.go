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
