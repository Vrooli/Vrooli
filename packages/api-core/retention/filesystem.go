package retention

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MeasureDirectory is the shared filesystem measurement primitive used by
// domain retention callers. Missing targets are reported as empty.
func MeasureDirectory(ctx context.Context, path string) (bytes int64, exists bool, err error) {
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if !info.IsDir() {
		return info.Size(), true, nil
	}
	err = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		fileInfo, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		bytes += fileInfo.Size()
		return nil
	})
	return bytes, true, err
}

// WalkDirectory is the shared traversal primitive. The callback receives the
// path, directory entry, and any walk error for that entry; callers retain
// ownership and policy decisions while traversal remains centralized.
func WalkDirectory(ctx context.Context, root string, visit func(string, fs.DirEntry, error) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return visit(path, entry, walkErr)
	})
}

// RemovePath is the shared deletion primitive. Callers accepting an external
// path must use DeleteContained instead.
func RemovePath(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" || filepath.Clean(path) == "." {
		return fmt.Errorf("retention path is not configured")
	}
	return os.RemoveAll(filepath.Clean(path))
}

// DeleteContained removes target only when it is a strict child of root and
// resolves within that root. This is the safe deletion boundary for domain
// callers whose selection is not performed by DirectoryPruner.
func DeleteContained(ctx context.Context, root, target string, protectedRoots []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root = filepath.Clean(strings.TrimSpace(root))
	target = filepath.Clean(strings.TrimSpace(target))
	if root == "." || target == "." || !filepath.IsAbs(root) || !filepath.IsAbs(target) {
		return fmt.Errorf("retention root and target must be configured as absolute paths")
	}
	if target == root || !PathContains(root, target) || ProtectedPathOverlap(target, protectedRoots) {
		return fmt.Errorf("retention target is outside the permitted root")
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err == nil && (resolved == root || !PathContains(root, resolved)) {
		return fmt.Errorf("retention target resolves outside the permitted root")
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("resolve retention target: %w", err)
	}
	return RemovePath(ctx, target)
}
