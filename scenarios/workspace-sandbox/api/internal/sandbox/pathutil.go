package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"workspace-sandbox/internal/types"
)

// PathValidator provides path normalization and validation for sandboxes.
type PathValidator struct {
	projectRoot string
}

// NewPathValidator creates a validator with the given project root.
func NewPathValidator(projectRoot string) (*PathValidator, error) {
	// Normalize the project root
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid project root: %w", err)
	}

	// Resolve symlinks
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		// If path doesn't exist, use the absolute path
		if os.IsNotExist(err) {
			realRoot = absRoot
		} else {
			return nil, fmt.Errorf("failed to resolve project root: %w", err)
		}
	}

	return &PathValidator{projectRoot: realRoot}, nil
}

// ProjectRoot returns the normalized project root.
func (v *PathValidator) ProjectRoot() string {
	return v.projectRoot
}

// NormalizePath normalizes a scope path relative to the project root.
// It returns an absolute, normalized path and validates it's within the project.
func (v *PathValidator) NormalizePath(scopePath string) (string, error) {
	// Handle empty path as project root
	if scopePath == "" {
		return v.projectRoot, nil
	}

	// Make absolute if relative
	var absPath string
	if filepath.IsAbs(scopePath) {
		absPath = scopePath
	} else {
		absPath = filepath.Join(v.projectRoot, scopePath)
	}

	// Clean the path (removes .., ., trailing slashes, etc.)
	cleanPath := filepath.Clean(absPath)

	// Try to resolve symlinks
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// If path doesn't exist, use cleaned path
		if os.IsNotExist(err) {
			realPath = cleanPath
		} else {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Validate path is within project root
	if !v.IsWithinProject(realPath) {
		return "", fmt.Errorf("scope path %q is outside project root %q", realPath, v.projectRoot)
	}

	return realPath, nil
}

// IsWithinProject checks if a path is within the project root.
func (v *PathValidator) IsWithinProject(path string) bool {
	// Normalize both paths
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(v.projectRoot)

	// Check if path is the root or starts with root/
	if cleanPath == cleanRoot {
		return true
	}

	return strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator))
}

// RelativePath returns the path relative to the project root.
func (v *PathValidator) RelativePath(absPath string) (string, error) {
	return filepath.Rel(v.projectRoot, absPath)
}

// FindConflicts checks a new scope path against existing sandboxes.
func FindConflicts(newScope string, existingSandboxes []*types.Sandbox) []types.PathConflict {
	var conflicts []types.PathConflict

	for _, s := range existingSandboxes {
		// Only check active sandboxes
		if !s.Status.IsActive() {
			continue
		}

		existing := make([]string, 0, 1)
		if len(s.ReservedPaths) > 0 {
			existing = append(existing, s.ReservedPaths...)
		} else if s.ReservedPath != "" {
			existing = append(existing, s.ReservedPath)
		} else if s.ScopePath != "" {
			existing = append(existing, s.ScopePath)
		}

		for _, existingPrefix := range existing {
			conflictType := types.CheckPathOverlap(existingPrefix, newScope)
			if conflictType == "" {
				continue
			}
			conflicts = append(conflicts, types.PathConflict{
				ExistingID:    s.ID.String(),
				ExistingScope: existingPrefix,
				NewScope:      newScope,
				ConflictType:  conflictType,
			})
		}
	}

	return conflicts
}

// ValidateScopePath performs full validation on a scope path.
// Returns the normalized path and any validation error.
func ValidateScopePath(scopePath, projectRoot string) (string, error) {
	validator, err := NewPathValidator(projectRoot)
	if err != nil {
		return "", err
	}

	return validator.NormalizePath(scopePath)
}
