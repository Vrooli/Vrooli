package shared

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// FilesystemScanner implements DirectoryScanner using the real filesystem.
type FilesystemScanner struct {
	log       *logrus.Logger
	validator *PathValidator
}

// NewFilesystemScanner creates a new FilesystemScanner.
func NewFilesystemScanner(log *logrus.Logger, allowedRoots ...string) *FilesystemScanner {
	return &FilesystemScanner{
		log:       log,
		validator: NewPathValidator(allowedRoots...),
	}
}

// ScanDirectory lists entries in a directory.
func (s *FilesystemScanner) ScanDirectory(ctx context.Context, path string) ([]FileEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absPath, err := s.validator.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	result := make([]FileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			s.log.WithError(err).WithField("path", entry.Name()).Debug("Failed to get file info")
			continue
		}

		result = append(result, FileEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(absPath, entry.Name()),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		})
	}

	return result, nil
}

// ScanForPattern recursively finds files matching a pattern.
func (s *FilesystemScanner) ScanForPattern(ctx context.Context, root string, pattern string, maxDepth int) ([]FileEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absRoot, err := s.validator.NormalizePath(root)
	if err != nil {
		return nil, err
	}

	var result []FileEntry

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Check context for cancellation
		if err := ctx.Err(); err != nil {
			return err
		}

		// Calculate depth
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip if directory
		if d.IsDir() {
			return nil
		}

		// Check pattern match
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if !matched {
			// Also check for suffix match (e.g., *.workflow.json)
			if !strings.HasSuffix(strings.ToLower(d.Name()), strings.ToLower(strings.TrimPrefix(pattern, "*"))) {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			s.log.WithError(err).WithField("path", path).Debug("Failed to get file info")
			return nil
		}

		result = append(result, FileEntry{
			Name:  d.Name(),
			Path:  path,
			IsDir: false,
			Size:  info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return result, nil
}

// ReadFile reads a file's contents.
func (s *FilesystemScanner) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absPath, err := s.validator.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(absPath)
}

// WriteFile writes content to a file.
func (s *FilesystemScanner) WriteFile(ctx context.Context, path string, content []byte, perm os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	absPath, err := s.validator.NormalizePath(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := EnsureDir(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	return os.WriteFile(absPath, content, perm)
}

// CopyFile copies a file from src to dst.
func (s *FilesystemScanner) CopyFile(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	absSrc, err := s.validator.NormalizePath(src)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	absDst, err := s.validator.NormalizePath(dst)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Open source
	srcFile, err := os.Open(absSrc)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Get source info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Ensure parent directory exists
	if err := EnsureDir(filepath.Dir(absDst), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Create destination
	dstFile, err := os.OpenFile(absDst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	return nil
}

// Exists checks if a path exists.
func (s *FilesystemScanner) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	absPath, err := s.validator.NormalizePath(path)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(absPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsDir checks if a path is a directory.
func (s *FilesystemScanner) IsDir(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	absPath, err := s.validator.NormalizePath(path)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// Stat returns file info for a path.
func (s *FilesystemScanner) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	absPath, err := s.validator.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	return os.Stat(absPath)
}

// Ensure FilesystemScanner implements DirectoryScanner
var _ DirectoryScanner = (*FilesystemScanner)(nil)
