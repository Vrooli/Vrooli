package binaryfetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// treeDigestWorkers bounds simultaneous file opens and 64 KiB digest buffers.
const treeDigestWorkers = 4

// TreeDigest returns the deterministic digest used for executable trees. It
// hashes the sorted list `relative/path NUL entry-digest newline`; regular
// files use their content SHA-256 and safe relative symlinks use the SHA-256
// of `symlink NUL link-target`. Directory metadata is intentionally excluded
// because it is recreated by extraction. No repository metadata or git command
// is consulted.
func TreeDigest(root string) (string, error) { return TreeDigestContext(context.Background(), root) }

// TreeDigestContext preserves TreeDigest's exact ordering and symlink format.
func TreeDigestContext(ctx context.Context, root string) (string, error) {
	return treeDigestContext(ctx, root, treeDigestWorkers, FileDigestContext)
}

func treeDigestContext(ctx context.Context, root string, workerCount int, digestFile func(context.Context, string) (string, error)) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	type entry struct {
		path string
		sum  string
		seq  int
		err  error
	}
	var entries []*entry
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > treeDigestWorkers {
		workerCount = treeDigestWorkers
	}
	root = filepath.Clean(root)
	if info, err := os.Lstat(root); err != nil {
		return "", fmt.Errorf("binaryfetch: stat tree: %w", err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("binaryfetch: tree root %q is not a directory", root)
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("binaryfetch: resolve tree root: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	jobs := make(chan *entry, workerCount)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer workers.Done()
			for item := range jobs {
				item.sum, item.err = digestFile(ctx, filepath.Join(root, filepath.FromSlash(item.path)))
			}
		}()
	}
	sequence := 0
	err = filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, walkErr error) error {
		sequence++
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if dirEntry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("binaryfetch: read tree symlink %q: %w", path, err)
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("binaryfetch: resolve tree path: %w", err)
			}
			_, err = resolveSafeSymlink(root, relative, link)
			if err != nil {
				return fmt.Errorf("binaryfetch: tree symlink %q escapes root: %w", path, err)
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("binaryfetch: tree symlink %q is dangling: %w", path, err)
			}
			actual, err := filepath.Rel(root, resolved)
			if err != nil || actual == ".." || strings.HasPrefix(actual, ".."+string(filepath.Separator)) || filepath.IsAbs(actual) {
				return fmt.Errorf("binaryfetch: tree symlink %q resolves outside root", path)
			}
			linkSum := sha256.Sum256(append([]byte("symlink\x00"), []byte(link)...))
			entries = append(entries, &entry{path: filepath.ToSlash(relative), sum: hex.EncodeToString(linkSum[:]), seq: sequence})
			return nil
		}
		if dirEntry.IsDir() {
			return nil
		}
		if !dirEntry.Type().IsRegular() {
			return fmt.Errorf("binaryfetch: tree contains non-regular file %q", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("binaryfetch: resolve tree path: %w", err)
		}
		item := &entry{path: filepath.ToSlash(relative), seq: sequence}
		entries = append(entries, item)
		select {
		case jobs <- item:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	close(jobs)
	workers.Wait()
	if err != nil {
		// Keep WalkDir's first-error behavior relative to any earlier file digest
		// failures discovered by the workers.
		for _, item := range entries {
			if item.err != nil && item.seq < sequence {
				return "", fmt.Errorf("binaryfetch: walk tree: %w", item.err)
			}
		}
		return "", fmt.Errorf("binaryfetch: walk tree: %w", err)
	}
	for _, item := range entries {
		if item.err != nil {
			return "", fmt.Errorf("binaryfetch: walk tree: %w", item.err)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	hash := sha256.New()
	for _, item := range entries {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		_, _ = io.WriteString(hash, item.path)
		_, _ = hash.Write([]byte{0})
		_, _ = io.WriteString(hash, item.sum)
		_, _ = io.WriteString(hash, "\n")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateDigest(value string) error {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("digest must be a %d-character SHA-256 value", sha256.Size*2)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("digest is not hexadecimal: %w", err)
	}
	return nil
}
