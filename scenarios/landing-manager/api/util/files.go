package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyDir copies a directory tree, skipping heavy build artifacts.
func CopyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		// handle single-file sources (e.g., Makefile, PRD.md)
		return CopyFile(src, dst, info.Mode())
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip unwanted directories
		skipDirs := map[string]bool{
			"node_modules": true,
			"dist":         true,
			"coverage":     true,
			".git":         true,
			"generated":    true,
		}
		if info.IsDir() && skipDirs[info.Name()] {
			return filepath.SkipDir
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return CopyFile(path, target, info.Mode())
	})
}

// CopyFile copies a single file preserving mode.
func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

// IsPathWithinDirectory checks if a path is safely contained within a base directory
func IsPathWithinDirectory(path, baseDir string) bool {
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(baseDir)
	return strings.HasPrefix(cleanPath, cleanBase)
}

// ResolvePackagePath validates and resolves a package path securely
func ResolvePackagePath(packageName, packagesDir string) (string, bool) {
	// Security: Clean the path to prevent traversal attacks
	cleanedName := filepath.Clean(packageName)

	// Reject paths containing traversal attempts
	if strings.Contains(cleanedName, "..") {
		return "", false
	}

	absolutePath := filepath.Join(packagesDir, cleanedName)

	// Verify the resolved path stays within the packages directory
	if !IsPathWithinDirectory(absolutePath, packagesDir) {
		return "", false
	}

	return absolutePath, true
}
