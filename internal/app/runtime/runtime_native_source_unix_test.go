//go:build linux || darwin

package runtimeapp

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestResolveNativeOperatorSourceRejectsFIFOWithoutOpeningIt(t *testing.T) {
	home := t.TempDir()
	path := repocontract.SourceRootPointerPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveNativeOperatorSource(home); err == nil {
		t.Fatal("FIFO accepted as native source pointer")
	}
}
