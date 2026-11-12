//go:build linux || darwin

package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func ensureOpenRGBStageParent(ctx context.Context, parent string) error {
	ancestor := parent
	for {
		if _, err := os.Lstat(ancestor); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return err
		}
		ancestor = filepath.Dir(ancestor)
	}
	if err := verifyOpenRGBAppDirAncestry(ctx, ancestor, uint32(os.Geteuid())); err != nil {
		return err
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if err := prepareOpenRGBNativeArtifactParents(parent); err != nil {
		return err
	}
	if err := verifyOpenRGBAppDirAncestry(ctx, parent, uint32(os.Geteuid())); err != nil {
		return err
	}
	for dir := parent; ; dir = filepath.Dir(dir) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := syncOpenRGBStageDirectory(dir); err != nil {
			return err
		}
		if dir == string(filepath.Separator) {
			return nil
		}
	}
}

func verifyOpenRGBPendingOwner(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
		return errors.New("OpenRGB pending stage has an unexpected owner, type or permissions")
	}
	return nil
}

func syncOpenRGBStageDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func sealOpenRGBStage(ctx context.Context, root string) error {
	native, err := openRGBArtifactNativeACLSealer()
	if err != nil {
		return err
	}
	var directories []string
	entries := 0
	var size int64
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		entries++
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uint32(os.Geteuid()) || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 || entries > 1024 {
			return errors.New("OpenRGB stage contains an unowned or excessive extraction entry")
		}
		if info.Mode()&os.ModeSymlink == 0 {
			if err := native(path, info); err != nil {
				return err
			}
		}
		if info.IsDir() {
			directories = append(directories, path)
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.Mode().IsRegular() || stat.Nlink != 1 || info.Size() > (256<<20)-size {
			return errors.New("OpenRGB stage contains an unsafe or oversized extraction entry")
		}
		size += info.Size()
		mode := os.FileMode(0o444)
		if info.Mode().Perm()&0o111 != 0 {
			mode = 0o555
		}
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
		return syncOpenRGBStageDirectory(path)
	}); err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err := os.Chmod(directories[i], 0o555); err != nil {
			return err
		}
		if err := syncOpenRGBStageDirectory(directories[i]); err != nil {
			return err
		}
	}
	return nil
}
