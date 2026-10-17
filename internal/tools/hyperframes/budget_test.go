package hyperframes

import (
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type budgetEntry struct {
	fs.DirEntry
	info func() (fs.FileInfo, error)
}

func (e budgetEntry) Info() (fs.FileInfo, error) { return e.info() }

type budgetFileInfo struct {
	fs.FileInfo
	size int64
}

func (i budgetFileInfo) Size() int64 { return i.size }

func TestTemporaryBudgetAllowsChildDeletedBetweenReadAndInfo(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "chromium-temporary")
	if err := os.WriteFile(child, []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	walk := func(_ string, visit fs.WalkDirFunc) error {
		return visit(child, budgetEntry{DirEntry: entries[0], info: func() (fs.FileInfo, error) {
			if err := os.Remove(child); err != nil {
				t.Fatal(err)
			}
			return os.Lstat(child)
		}}, nil)
	}
	if err := checkDiskBudgetWalk(context.Background(), root, walk); err != nil {
		t.Fatalf("concurrently deleted child failed sampled budget: %v", err)
	}
}

func TestTemporaryBudgetWalkErrorsRemainFailClosed(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name string
		path string
		err  error
		want error
	}{
		{"vanished-child", filepath.Join(root, "gone"), fs.ErrNotExist, nil},
		{"missing-root", root, fs.ErrNotExist, fs.ErrNotExist},
		{"permission", filepath.Join(root, "private"), fs.ErrPermission, fs.ErrPermission},
		{"other-io", filepath.Join(root, "broken"), fs.ErrInvalid, fs.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			walk := func(_ string, visit fs.WalkDirFunc) error {
				return visit(tc.path, nil, &fs.PathError{Op: "readdir", Path: tc.path, Err: tc.err})
			}
			err := checkDiskBudgetWalk(context.Background(), root, walk)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
	if err := checkDiskBudget(filepath.Join(root, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing actual root accepted: %v", err)
	}
}

func TestTemporaryBudgetInfoErrorsAndSymlinkPolicy(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.WriteFile(child, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, sentinel := range []error{fs.ErrPermission, fs.ErrInvalid} {
		walk := func(_ string, visit fs.WalkDirFunc) error {
			return visit(child, budgetEntry{DirEntry: entries[0], info: func() (fs.FileInfo, error) {
				return nil, &fs.PathError{Op: "lstat", Path: child, Err: sentinel}
			}}, nil)
		}
		if err := checkDiskBudgetWalk(context.Background(), root, walk); !errors.Is(err, sentinel) {
			t.Fatalf("Info error suppressed: %v", err)
		}
	}
	// WalkDir does not follow symlinks. An external target must remain outside
	// the sampled temporary tree, including a broken target.
	if err := os.Symlink(filepath.Join(root, "absent-target"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := checkDiskBudget(root); err != nil {
		t.Fatalf("unchanged no-follow policy rejected symlink: %v", err)
	}
}

func TestTemporaryBudgetRejectsSizeOverflowAndEntryFlood(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fixture")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		sizes []int64
	}{
		{"overflow", []int64{1, math.MaxInt64}},
		{"negative", []int64{-1}},
		{"entry-flood", make([]int64, 100001)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			walk := func(_ string, visit fs.WalkDirFunc) error {
				for _, size := range tc.sizes {
					entry := budgetEntry{DirEntry: entries[0], info: func() (fs.FileInfo, error) { return budgetFileInfo{info, size}, nil }}
					if err := visit(path, entry, nil); err != nil {
						return err
					}
				}
				return nil
			}
			if err := checkDiskBudgetWalk(context.Background(), root, walk); err == nil {
				t.Fatal("budget violation accepted")
			}
		})
	}
}
