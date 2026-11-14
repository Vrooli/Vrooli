//go:build linux || darwin

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDarwinBrokerPublicationCommitNeedsNoAliasCleanup(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, ".staged"), filepath.Join(root, "sealed")
	if err := os.WriteFile(source, []byte("admitted"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := darwinSystemBrokerPublishExclusive(source, target); err != nil {
		t.Fatal(err)
	}
	// Model death immediately after the commit: no unlink/cleanup operation
	// follows publication before the recovered object is inspected.
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if uint64(info.Sys().(*syscall.Stat_t).Nlink) != 1 {
		t.Fatal("committed image needs alias cleanup before strict adoption")
	}
	if _, err = os.Lstat(source); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("committed staging alias survived: %v", err)
	}
	if err = os.WriteFile(source, []byte("replacement"), 0444); err != nil {
		t.Fatal(err)
	}
	if err = darwinSystemBrokerPublishExclusive(source, target); !errors.Is(err, os.ErrExist) {
		t.Fatalf("existing immutable target was not protected: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "admitted" {
		t.Fatalf("existing target changed: %q %v", data, err)
	}
	data, err = os.ReadFile(source)
	if err != nil || string(data) != "replacement" {
		t.Fatalf("failed publication lost its staged object: %q %v", data, err)
	}
}
