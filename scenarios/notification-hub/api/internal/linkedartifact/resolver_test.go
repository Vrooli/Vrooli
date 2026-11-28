package linkedartifact

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRequiresAnAllowedRegularFileAndEnforcesSize(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "decision.md")
	if err := os.WriteFile(inside, []byte("readable decision context"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "escape.txt")); err != nil {
		t.Fatal(err)
	}

	resolver := New(root)
	file, info, err := resolver.Open(inside)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len("readable decision context")) {
		t.Fatalf("size = %d", info.Size())
	}
	_ = file.Close()
	if _, _, err := resolver.Open(secret); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outside file error = %v", err)
	}
	if _, _, err := resolver.Open(filepath.Join(root, "escape.txt")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("symlink escape error = %v", err)
	}
	if _, _, err := New().Open(inside); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing source error = %v", err)
	}

	large := filepath.Join(root, "large.md")
	f, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, _, err := resolver.Open(large); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("large file error = %v", err)
	}
}
