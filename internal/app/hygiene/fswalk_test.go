package hygiene

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type fakeDirEntry struct{ dir bool }

func (f fakeDirEntry) Name() string               { return "x" }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestSkipUnreadable(t *testing.T) {
	perm := &fs.PathError{Op: "open", Path: "/root-owned", Err: fs.ErrPermission}
	other := errors.New("disk exploded")

	cases := []struct {
		name    string
		walkErr error
		entry   os.DirEntry
		want    error
	}{
		{"no error", nil, fakeDirEntry{dir: true}, nil},
		{"permission on dir skips subtree", perm, fakeDirEntry{dir: true}, filepath.SkipDir},
		{"permission on file skips file", perm, fakeDirEntry{dir: false}, nil},
		{"permission with nil entry skips", perm, nil, nil},
		{"non-permission error propagates", other, fakeDirEntry{dir: true}, other},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skipUnreadable(tc.walkErr, tc.entry); got != tc.want {
				t.Fatalf("skipUnreadable = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsUnreadable(t *testing.T) {
	if !isUnreadable(&fs.PathError{Err: fs.ErrPermission}) {
		t.Fatal("wrapped permission error must be unreadable")
	}
	if isUnreadable(errors.New("nope")) {
		t.Fatal("non-permission error must not be unreadable")
	}
}

// A root-owned, unreadable directory under a scanned root (the real failure that
// aborted the whole hygiene run) must degrade to a skip, letting the readable
// siblings still be reported.
func TestFindEmptyDirectories_SkipsUnreadableSubtree(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission bits; cannot simulate an unreadable directory")
	}
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "docs", "empty-ok"))
	locked := filepath.Join(root, "docs", "locked")
	mustMkdir(t, filepath.Join(locked, "child"))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	empty, err := findEmptyDirectories(root, []string{"docs"})
	if err != nil {
		t.Fatalf("findEmptyDirectories aborted on an unreadable path: %v", err)
	}
	rel := make([]string, 0, len(empty))
	for _, p := range empty {
		r, _ := filepath.Rel(root, p)
		rel = append(rel, filepath.ToSlash(r))
	}
	if !slices.Contains(rel, "docs/empty-ok") {
		t.Fatalf("readable empty sibling was lost: %v", rel)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
