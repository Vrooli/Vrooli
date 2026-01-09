package main

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// withinBase checks if target path is within base directory.
func withinBase(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// normalizeBundlePath strips leading parent directory traversals from a path.
// For example: "../../../bin/linux-x64/api" becomes "bin/linux-x64/api"
// This ensures files are always staged inside the bundle directory.
func normalizeBundlePath(rel string) string {
	// Convert to forward slashes for consistent handling
	clean := filepath.ToSlash(rel)
	// Remove leading "../" segments
	for strings.HasPrefix(clean, "../") {
		clean = strings.TrimPrefix(clean, "../")
	}
	// Also handle edge case of just ".."
	if clean == ".." {
		clean = ""
	}
	return clean
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	// Resolve absolute paths to detect if src and dst are the same file
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if absSrc == absDst {
		// Source and destination are the same file, nothing to copy
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// copyPath copies a file or directory from src to dst.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst, info.Mode())
	}
	return copyFile(src, dst)
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(dst, mode.Perm()); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target)
	})
}
