package binaryfetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeDigest returns the deterministic digest used for executable trees. It
// hashes the sorted list `relative/path NUL entry-digest newline`; regular
// files use their content SHA-256 and safe relative symlinks use the SHA-256
// of `symlink NUL link-target`. Directory metadata is intentionally excluded
// because it is recreated by extraction. No repository metadata or git command
// is consulted.
func TreeDigest(root string) (string, error) {
	type entry struct {
		path string
		sum  string
	}
	var entries []entry
	root = filepath.Clean(root)
	if info, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("binaryfetch: stat tree: %w", err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("binaryfetch: tree root %q is not a directory", root)
	}
	err := filepath.WalkDir(root, func(path string, dirEntry os.DirEntry, walkErr error) error {
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
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(link)))
			rel, err := filepath.Rel(root, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				return fmt.Errorf("binaryfetch: tree symlink %q escapes root", path)
			}
			if _, err := os.Stat(resolved); err != nil {
				return fmt.Errorf("binaryfetch: tree symlink %q is dangling: %w", path, err)
			}
			linkSum := sha256.Sum256(append([]byte("symlink\x00"), []byte(link)...))
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("binaryfetch: resolve tree path: %w", err)
			}
			entries = append(entries, entry{path: filepath.ToSlash(relative), sum: hex.EncodeToString(linkSum[:])})
			return nil
		}
		if dirEntry.IsDir() {
			return nil
		}
		if !dirEntry.Type().IsRegular() {
			return fmt.Errorf("binaryfetch: tree contains non-regular file %q", path)
		}
		sum, err := fileDigest(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("binaryfetch: resolve tree path: %w", err)
		}
		entries = append(entries, entry{path: filepath.ToSlash(relative), sum: sum})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("binaryfetch: walk tree: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	hash := sha256.New()
	for _, item := range entries {
		_, _ = io.WriteString(hash, item.path)
		_, _ = hash.Write([]byte{0})
		_, _ = io.WriteString(hash, item.sum)
		_, _ = io.WriteString(hash, "\n")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path) //nolint:gosec // path comes from a walked tree root
	if err != nil {
		return "", fmt.Errorf("binaryfetch: open tree file %q: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("binaryfetch: hash tree file %q: %w", path, err)
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
