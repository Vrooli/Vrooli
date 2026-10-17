//go:build !windows

package providers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTemporaryActivityProtectsParentOfOpenFile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux process observation")
	}
	root := t.TempDir()
	path := filepath.Join(root, "payload")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	provider := &FileProvider{leaseCheck: "temporary"}
	active := provider.activePredicate(root)
	if !active(path) || !active(root) {
		t.Fatal("open file did not protect its enclosing cleanup unit")
	}
}
