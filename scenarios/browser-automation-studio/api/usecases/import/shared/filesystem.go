package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathValidator provides methods for validating and normalizing paths.
type PathValidator struct {
	// AllowedRoots restricts paths to these root directories (empty = allow all)
	AllowedRoots []string
}

// NewPathValidator creates a new PathValidator.
func NewPathValidator(allowedRoots ...string) *PathValidator {
	return &PathValidator{
		AllowedRoots: allowedRoots,
	}
}

// ValidatePath validates a path and returns an error if invalid.
func (v *PathValidator) ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// If allowed roots specified, ensure path is under one of them
	if len(v.AllowedRoots) > 0 {
		abs, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path")
		}

		allowed := false
		for _, root := range v.AllowedRoots {
			absRoot, err := filepath.Abs(root)
			if err != nil {
				continue
			}
			if strings.HasPrefix(abs, absRoot+string(os.PathSeparator)) || abs == absRoot {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed directories")
		}
	}

	return nil
}

// NormalizePath converts a path to absolute form.
func (v *PathValidator) NormalizePath(path string) (string, error) {
	if err := v.ValidatePath(path); err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

// SafeJoin safely joins paths, preventing traversal outside the base.
func SafeJoin(base string, paths ...string) (string, error) {
	if base == "" {
		return "", fmt.Errorf("base path required")
	}

	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}

	// Join all path components
	joined := absBase
	for _, p := range paths {
		if p == "" {
			continue
		}
		// Clean each component
		cleaned := filepath.Clean(p)
		if filepath.IsAbs(cleaned) {
			return "", fmt.Errorf("absolute path not allowed: %s", p)
		}
		joined = filepath.Join(joined, cleaned)
	}

	// Verify result is still under base
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if !strings.HasPrefix(absJoined, absBase+string(os.PathSeparator)) && absJoined != absBase {
		return "", fmt.Errorf("path traversal detected")
	}

	return absJoined, nil
}

// RelativePath returns the relative path from base to target.
func RelativePath(base, target string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return filepath.Rel(absBase, absTarget)
}

// IsSubPath checks if target is under base.
func IsSubPath(base, target string) (bool, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false, err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false, err
	}

	return strings.HasPrefix(absTarget, absBase+string(os.PathSeparator)) || absTarget == absBase, nil
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// GetParentDir returns the parent directory of a path.
func GetParentDir(path string) string {
	return filepath.Dir(path)
}

// HasExtension checks if a filename has one of the given extensions.
func HasExtension(filename string, exts ...string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range exts {
		if strings.HasSuffix(lower, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// WorkflowFileSuffix is the standard suffix for workflow files.
const WorkflowFileSuffix = ".workflow.json"

// ProjectMetadataPath is the standard path for project metadata.
const ProjectMetadataPath = ".bas/project.json"

// WorkflowsDir is the standard directory for workflows within a project.
const WorkflowsDir = "workflows"

// IsWorkflowFile checks if a filename is a workflow file.
func IsWorkflowFile(filename string) bool {
	return HasExtension(filename, WorkflowFileSuffix)
}

// IsProjectDir checks if a directory contains project metadata.
// This is a convenience function that ignores errors.
func IsProjectDir(path string) bool {
	metaPath := filepath.Join(path, ProjectMetadataPath)
	_, err := os.Stat(metaPath)
	return err == nil
}

// DefaultProjectsRoot returns the default projects directory.
// Returns user's home directory + "Projects" or current working directory.
func DefaultProjectsRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	return filepath.Join(home, "Projects")
}

// NormalizePath converts a path to absolute form.
// Standalone function that doesn't require a PathValidator.
func NormalizePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	return filepath.Abs(path)
}
