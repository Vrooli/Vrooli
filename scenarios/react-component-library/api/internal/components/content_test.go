package components

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestFSContentStore_RoundTrip exercises the happy path: read a known
// file, write it back with a new body, read again to confirm the new
// digest sticks.
func TestFSContentStore_RoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Button.tsx"), []byte("export const Button = () => null;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFSContentStore(root)
	c := Component{SourcePath: "Button.tsx"}

	got, err := store.Read(context.Background(), c)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Body == "" || got.SHA256 == "" {
		t.Fatalf("Read returned empty body or sha")
	}

	newBody := "export const Button = () => <button>x</button>;\n"
	written, err := store.Write(context.Background(), c, WriteContentInput{Body: newBody, ExpectedSHA256: got.SHA256})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if written.Body != newBody {
		t.Fatalf("written body mismatch")
	}

	after, err := store.Read(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if after.Body != newBody {
		t.Fatalf("re-read body mismatch")
	}
	if after.SHA256 != written.SHA256 {
		t.Fatalf("sha mismatch after roundtrip")
	}
}

// TestFSContentStore_PathEscape — the guard must reject paths that
// resolve outside the configured root.
func TestFSContentStore_PathEscape(t *testing.T) {
	root := t.TempDir()
	store := NewFSContentStore(root)

	cases := []string{
		"../etc/passwd",
		"a/../../escape.tsx",
		"/etc/passwd",
		"",
		"../" + filepath.Base(root) + "_sibling/x.tsx",
	}
	for _, sp := range cases {
		t.Run(sp, func(t *testing.T) {
			_, err := store.Read(context.Background(), Component{SourcePath: sp})
			var esc ErrPathEscape
			if !errors.As(err, &esc) {
				t.Fatalf("expected ErrPathEscape for %q, got %v", sp, err)
			}
			_, err = store.Write(context.Background(), Component{SourcePath: sp}, WriteContentInput{Body: "x"})
			if !errors.As(err, &esc) {
				t.Fatalf("Write expected ErrPathEscape for %q, got %v", sp, err)
			}
		})
	}
}

// TestFSContentStore_OptimisticConcurrency — when the caller pins an
// expected sha that doesn't match disk, Write must reject without
// modifying the file.
func TestFSContentStore_OptimisticConcurrency(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Card.tsx")
	original := []byte("v1")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewFSContentStore(root)
	c := Component{SourcePath: "Card.tsx"}

	_, err := store.Write(context.Background(), c, WriteContentInput{
		Body:           "v2",
		ExpectedSHA256: "deadbeef",
	})
	var conflict ErrContentConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ErrContentConflict, got %v", err)
	}
	on, _ := os.ReadFile(path)
	if string(on) != "v1" {
		t.Fatalf("file mutated despite conflict; got %q", on)
	}
}

// TestFSContentStore_ReadMissingFile — surfaces a wrapped os.* error
// (mapped to Internal at the transport edge); we just verify the error
// is non-nil and not a typed ErrPathEscape.
func TestFSContentStore_ReadMissingFile(t *testing.T) {
	store := NewFSContentStore(t.TempDir())
	_, err := store.Read(context.Background(), Component{SourcePath: "no-such.tsx"})
	if err == nil {
		t.Fatal("expected error")
	}
	var esc ErrPathEscape
	if errors.As(err, &esc) {
		t.Fatalf("missing file must not classify as ErrPathEscape")
	}
}
