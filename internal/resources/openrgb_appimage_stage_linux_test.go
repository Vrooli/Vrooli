//go:build linux

package resources

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBAppDirSealRejectsWritableOrUnownedClosures(t *testing.T) {
	ctx := context.Background()
	root, _, _ := testOpenRGBAppDir(t)
	uid := uint32(os.Geteuid())
	if err := verifyOpenRGBAppDirEntries(ctx, root, uid); err == nil {
		t.Fatal("writable stage directories were accepted")
	}
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return err
		})
	})
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			return os.Chmod(path, 0o555)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := verifyOpenRGBAppDirEntries(ctx, root, uid); err != nil {
		t.Fatal(err)
	}
	if err := verifyOpenRGBAppDirEntries(ctx, root, uid+1); err == nil {
		t.Fatal("a different stage owner was accepted")
	}
	if err := verifyOpenRGBAppDirSeal(ctx, root); err == nil {
		t.Fatal("operator-writable scratch ancestry was accepted as a production seal")
	}
	if err := os.Chmod(filepath.Join(root, "usr/lib/library.so"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyOpenRGBAppDirEntries(ctx, root, uid); err == nil {
		t.Fatal("a writable library passed the staged execution check")
	}
}
