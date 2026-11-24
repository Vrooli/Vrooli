package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicallyReplacesFileWithRequestedMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomically(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("replacement = %q, want new", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("replacement mode = %o, want 644", got)
	}
}

func TestWriteFileAtomicallyCleansTemporaryFileWhenRenameFails(t *testing.T) {
	parent := t.TempDir()
	path := filepath.Join(parent, "existing-directory")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomically(path, []byte("new"), 0o600); err == nil {
		t.Fatal("rename onto a directory unexpectedly succeeded")
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "existing-directory" {
		t.Fatalf("temporary files remain after failed rename: %#v", entries)
	}
}
