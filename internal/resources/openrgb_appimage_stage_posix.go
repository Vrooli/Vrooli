//go:build linux || darwin

package resources

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

func verifyOpenRGBAppDirSeal(ctx context.Context, root string) error {
	uid, err := openRGBArtifactOwnerUID()
	if err != nil {
		return err
	}
	if err := verifyOpenRGBAppDirAncestry(ctx, filepath.Dir(root), uid); err != nil {
		return err
	}
	return verifyOpenRGBAppDirEntries(ctx, root, uid)
}

// The canonical artifact owner controls publication. Other accounts cannot
// replace its path or dependencies. The same operator can deliberately alter
// their own files, so every launch revalidates the complete closure.
func verifyOpenRGBAppDirAncestry(ctx context.Context, root string, uid uint32) error {
	native, err := openRGBArtifactNativeACLVerifier()
	if err != nil {
		return err
	}
	for dir := root; ; dir = filepath.Dir(dir) {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Uid != 0 && stat.Uid != uid || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("untrusted OpenRGB AppDir ancestor %s", dir)
		}
		if err := native(dir, info, false); err != nil {
			return err
		}
		if dir == string(filepath.Separator) {
			break
		}
	}
	return nil
}

func verifyOpenRGBAppDirEntries(ctx context.Context, root string, uid uint32) error {
	native, err := openRGBArtifactNativeACLVerifier()
	if err != nil {
		return err
	}
	const maxEntries = 512
	const maxBytes = 128 << 20
	var entries int
	var total int64
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		entries++
		if entries > maxEntries {
			return errors.New("OpenRGB AppDir exceeds the entry limit")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		mode := info.Mode()
		if !ok || stat.Uid != uid || mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || path == root && !mode.IsDir() {
			return fmt.Errorf("untrusted OpenRGB AppDir entry %s", path)
		}
		if mode&os.ModeSymlink != 0 {
			// TreeDigest validates and binds each symlink's in-tree target.
			return nil
		}
		if mode.Perm()&0o222 != 0 {
			return fmt.Errorf("OpenRGB AppDir entry is writable: %s", path)
		}
		if err := native(path, info, true); err != nil {
			return err
		}
		if mode.IsDir() {
			return nil
		}
		if !mode.IsRegular() || stat.Nlink != 1 {
			return fmt.Errorf("OpenRGB AppDir entry is not a single-link regular file: %s", path)
		}
		if info.Size() > maxBytes-total {
			return errors.New("OpenRGB AppDir exceeds the size limit")
		}
		total += info.Size()
		if err := verifyOpenRGBFileCapabilities(path); err != nil {
			return err
		}
		return nil
	})
}
