package hostfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"storage-manager/internal/authority"
	"storage-manager/internal/cleanup"
)

// Every cleanup provider mutates through FS, so a redirected instance's scope
// confines removal, renames and directory creation to its own tree.
func TestRedirectedFSMutatesOnlyInsideItsTree(t *testing.T) {
	override := t.TempDir()
	host := t.TempDir()
	fs := New(Options{Scope: authority.Redirected(override)})
	ctx := context.Background()

	victim := filepath.Join(host, "native-resource-cache")
	if err := os.MkdirAll(victim, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fs.RemoveAll(ctx, victim); !errors.Is(err, authority.ErrOutOfScope) {
		t.Fatalf("RemoveAll outside the tree = %v, want ErrOutOfScope", err)
	}
	if err := fs.Rename(ctx, victim, filepath.Join(override, "moved")); !errors.Is(err, authority.ErrOutOfScope) {
		t.Fatalf("Rename out of the host tree = %v, want ErrOutOfScope", err)
	}
	if err := fs.MkdirAll(ctx, filepath.Join(host, "new")); !errors.Is(err, authority.ErrOutOfScope) {
		t.Fatalf("MkdirAll outside the tree = %v, want ErrOutOfScope", err)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("host directory was touched: %v", err)
	}

	inside := filepath.Join(override, "data", "stale")
	if err := fs.MkdirAll(ctx, inside); err != nil {
		t.Fatalf("MkdirAll inside the tree: %v", err)
	}
	if err := fs.RemoveAll(ctx, inside); err != nil {
		t.Fatalf("RemoveAll inside the tree: %v", err)
	}
}

func TestRedirectedProcessRunnerRefusesHostCommands(t *testing.T) {
	runner := ProcessRunner{scope: authority.Redirected(t.TempDir())}
	_, err := runner.Run(context.Background(), cleanup.ProcessCommand{Name: "docker", Args: []string{"image", "prune", "-f"}})
	if !errors.Is(err, authority.ErrOutOfScope) {
		t.Fatalf("Run = %v, want ErrOutOfScope before any process starts", err)
	}
}
