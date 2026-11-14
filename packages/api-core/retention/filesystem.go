package retention

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirectoryWalkCursor retains a bounded directory stream between observations.
// It performs no deletion, follows no symlink entries, and refuses an unreadable
// or replaced directory rather than treating an incomplete subtree as complete.
type DirectoryWalkCursor struct {
	path        string
	root        *os.Root
	initial     fs.FileInfo
	started     bool
	pending     string
	pendingInfo fs.FileInfo
	frames      []directoryWalkFrame
	closed      bool
}

type directoryWalkFrame struct {
	rel     string
	file    *os.File
	info    fs.FileInfo
	entries []fs.DirEntry
	index   int
	eof     bool
}

func OpenDirectoryWalk(path string) (*DirectoryWalkCursor, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	c := &DirectoryWalkCursor{path: filepath.Clean(path), initial: info}
	if info.IsDir() {
		c.root, err = os.OpenRoot(path)
		if err != nil {
			return nil, err
		}
		bound, statErr := c.root.Lstat(".")
		if statErr != nil || !os.SameFile(info, bound) {
			_ = c.Close()
			return nil, fmt.Errorf("directory changed while opening walk: %s", path)
		}
	}
	return c, nil
}

func (c *DirectoryWalkCursor) Next(ctx context.Context) (string, fs.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if c.closed {
		return "", nil, io.EOF
	}
	if !c.started {
		c.started = true
		if c.initial.IsDir() {
			c.pending, c.pendingInfo = ".", c.initial
		}
		return c.path, c.initial, nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}
		if c.pendingInfo != nil {
			if len(c.frames) >= 256 {
				return "", nil, fmt.Errorf("walk depth exceeded at %s", c.path)
			}
			f, err := c.root.Open(c.pending)
			if err != nil {
				return "", nil, err
			}
			bound, err := f.Stat()
			if err != nil || !os.SameFile(bound, c.pendingInfo) {
				_ = f.Close()
				return "", nil, fmt.Errorf("directory changed during walk: %s", c.pending)
			}
			c.frames = append(c.frames, directoryWalkFrame{rel: c.pending, file: f, info: bound})
			c.pendingInfo = nil
		}
		if len(c.frames) == 0 {
			_ = c.Close()
			return "", nil, io.EOF
		}
		frame := &c.frames[len(c.frames)-1]
		if frame.index == len(frame.entries) {
			if frame.eof {
				current, err := c.root.Lstat(frame.rel)
				if err != nil || !os.SameFile(current, frame.info) || !current.ModTime().Equal(frame.info.ModTime()) {
					return "", nil, fmt.Errorf("directory changed during walk: %s", frame.rel)
				}
				_ = frame.file.Close()
				c.frames = c.frames[:len(c.frames)-1]
				continue
			}
			entries, err := frame.file.ReadDir(128)
			if err != nil && err != io.EOF {
				return "", nil, err
			}
			frame.entries, frame.index, frame.eof = entries, 0, err == io.EOF
			continue
		}
		entry := frame.entries[frame.index]
		frame.index++
		rel := filepath.Join(frame.rel, entry.Name())
		info, err := c.root.Lstat(rel)
		if err != nil {
			return "", nil, err
		}
		if info.IsDir() {
			c.pending, c.pendingInfo = rel, info
		}
		return filepath.Join(c.path, rel), info, nil
	}
}

func (c *DirectoryWalkCursor) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	for _, frame := range c.frames {
		_ = frame.file.Close()
	}
	c.frames = nil
	if c.root != nil {
		return c.root.Close()
	}
	return nil
}

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
	if err == nil {
		if resolved == root || !PathContains(root, resolved) {
			return fmt.Errorf("retention target resolves outside the permitted root")
		}
		if ProtectedPathOverlap(resolved, protectedRoots) {
			return fmt.Errorf("retention target resolves to a protected path")
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("resolve retention target: %w", err)
	}
	return RemovePath(ctx, target)
}
