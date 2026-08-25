package hostfs

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"storage-manager/internal/cleanup"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func write(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func walkPaths(t *testing.T, f *FS, root string) []string {
	t.Helper()
	var got []string
	if err := f.Walk(context.Background(), root, func(info cleanup.FileInfo) error {
		got = append(got, info.Path)
		return nil
	}); err != nil {
		t.Fatalf("Walk(%s): %v", root, err)
	}
	sort.Strings(got)
	return got
}

// TestWalk_ReportsFilesAndDirectories asserts directories are visited too.
// The top-level-entry providers aggregate a subtree by its directory, so a walk
// that reported only files would under-count an empty directory's existence.
func TestWalk_ReportsFilesAndDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write(t, filepath.Join(root, "staging", "payload.bin"), "data")
	if err := os.MkdirAll(filepath.Join(root, "empty-dir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got := walkPaths(t, New(Options{}), root)

	want := []string{
		root,
		filepath.Join(root, "empty-dir"),
		filepath.Join(root, "staging"),
		filepath.Join(root, "staging", "payload.bin"),
	}
	if len(got) != len(want) {
		t.Fatalf("walked %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestWalk_DoesNotFollowSymlinks asserts a symlink is never descended into and
// never reported as a candidate.
//
// This matters because the providers only check that a path lies within a
// configured root. A symlink inside /tmp pointing at a home directory would
// otherwise present that home directory's contents as reclaimable temp files.
func TestWalk_DoesNotFollowSymlinks(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "symlink creation on windows requires elevation or developer mode")
	}

	root := t.TempDir()
	outside := t.TempDir()
	write(t, filepath.Join(outside, "precious.txt"), "do not touch")

	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	for _, path := range walkPaths(t, New(Options{}), root) {
		if path != root {
			t.Errorf("walk reported %q; a symlink must be neither followed nor listed", path)
		}
	}
}

// TestWalk_SkipsUnreadableSubtreeAndContinues asserts a permission error does
// not abort the walk.
//
// A cleanup pass over a shared /tmp meets unreadable directories constantly —
// the incident host had system-service private mounts among 24,882 entries. If
// one of those aborted the walk, the reclaimable space behind it would never be
// found, which is precisely how a cleanup tool reports zero while the disk is
// full.
func TestWalk_SkipsUnreadableSubtreeAndContinues(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "unix permission bits do not model windows ACLs")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypasses permission bits, so the failure cannot be provoked")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	write(t, filepath.Join(locked, "hidden.bin"), "unreachable")
	reachable := filepath.Join(root, "reachable.bin")
	write(t, reachable, "findable")

	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	got := walkPaths(t, New(Options{}), root)

	var sawReachable bool
	for _, path := range got {
		if path == reachable {
			sawReachable = true
		}
		if path == filepath.Join(locked, "hidden.bin") {
			t.Errorf("walk descended into an unreadable directory")
		}
	}
	if !sawReachable {
		t.Errorf("walk did not reach %q past the unreadable directory; got %v", reachable, got)
	}
}

// TestRemoveAll_MissingPathIsNotAnError asserts removal is idempotent.
// Two safeguards can report the same pressure event and race to reclaim the
// same entry; the loser must not report a failure for work that got done.
func TestRemoveAll_MissingPathIsNotAnError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := New(Options{}).RemoveAll(context.Background(), filepath.Join(root, "never-existed")); err != nil {
		t.Errorf("RemoveAll on a missing path = %v, want nil", err)
	}
}

// TestRemoveAll_DeletesSubtree asserts the ordinary path works end to end.
func TestRemoveAll_DeletesSubtree(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	write(t, filepath.Join(staging, "nested", "payload.bin"), "data")

	if err := New(Options{}).RemoveAll(context.Background(), staging); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Errorf("staging directory still present after RemoveAll (stat err = %v)", err)
	}
}

// TestRemoveAll_RemovesSymlinkNotTarget asserts deleting a link inside a root
// does not delete what it points at.
func TestRemoveAll_RemovesSymlinkNotTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "symlink creation on windows requires elevation or developer mode")
	}

	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "precious.txt")
	write(t, target, "do not touch")

	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if err := New(Options{}).RemoveAll(context.Background(), link); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("removing a symlink destroyed its target: %v", err)
	}
}

// TestStat_ReportsSymlinkItselfNotTarget asserts Lstat semantics.
func TestStat_ReportsSymlinkItselfNotTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "symlink creation on windows requires elevation or developer mode")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target")
	write(t, target, "0123456789")
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	info, err := New(Options{}).Stat(context.Background(), link)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.IsDir {
		t.Error("symlink reported as a directory")
	}
	if info.Size == 10 {
		t.Error("Stat reported the target's size; it must describe the link itself")
	}
}

// TestWalk_HonoursContextCancellation asserts a cancelled cleanup stops
// promptly rather than walking a very large tree to completion.
func TestWalk_HonoursContextCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for i := 0; i < 50; i++ {
		write(t, filepath.Join(root, "entry", string(rune('a'+i%26))+".bin"), "data")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := New(Options{}).Walk(ctx, root, func(cleanup.FileInfo) error { return nil }); err == nil {
		t.Error("Walk with a cancelled context returned nil, want a context error")
	}
}

func TestWithinRoot(t *testing.T) {
	t.Parallel()

	root := filepath.Join(string(filepath.Separator), "tmp")
	cases := []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "entry"), true},
		{filepath.Join(root, "entry", "nested"), true},
		// The classic prefix bug: /tmpfoo is not inside /tmp.
		{root + "foo", false},
		{filepath.Join(string(filepath.Separator), "var", "tmp"), false},
	}
	for _, tc := range cases {
		if got := WithinRoot(root, tc.path); got != tc.want {
			t.Errorf("WithinRoot(%q, %q) = %v, want %v", root, tc.path, got, tc.want)
		}
	}
}
