package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PathValidator validates file paths to ensure they stay within allowed boundaries.
// SECURITY: This prevents path traversal attacks and ensures agents only access
// files within their designated scenario directory.
type PathValidator struct {
	allowedRoot string
}

// NewPathValidator creates a path validator for the given root directory.
func NewPathValidator(allowedRoot string) (*PathValidator, error) {
	if allowedRoot == "" {
		return nil, fmt.Errorf("allowed root path cannot be empty")
	}

	absRoot, err := NormalizeToAbsolute(allowedRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid allowed root: %w", err)
	}

	return &PathValidator{allowedRoot: absRoot}, nil
}

// ValidatePath checks if a path is within the allowed root directory.
func (v *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	absPath, err := NormalizeToAbsolute(path)
	if err != nil {
		return fmt.Errorf("invalid path '%s': %w", path, err)
	}

	if !IsPathWithinRoot(absPath, v.allowedRoot) {
		return fmt.Errorf("path '%s' is outside allowed root '%s'", path, v.allowedRoot)
	}

	return nil
}

// ValidatePaths checks multiple paths and returns all validation errors.
func (v *PathValidator) ValidatePaths(paths []string) []error {
	var errors []error
	for _, path := range paths {
		if err := v.ValidatePath(path); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetAllowedRoot returns the allowed root directory.
func (v *PathValidator) GetAllowedRoot() string {
	return v.allowedRoot
}

// NormalizeToAbsolute converts a path to its absolute, cleaned form.
func NormalizeToAbsolute(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = home + path[1:]
	}

	cleaned := filepath.Clean(path)

	if !filepath.IsAbs(cleaned) {
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return "", err
		}
		cleaned = abs
	}

	return cleaned, nil
}

// IsPathWithinRoot checks if absPath is within or equal to absRoot.
func IsPathWithinRoot(absPath, absRoot string) bool {
	rootWithSep := absRoot
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}
	return absPath == absRoot || strings.HasPrefix(absPath, rootWithSep)
}

// ValidateScopePaths validates that all scope paths are within the scenario directory.
func ValidateScopePaths(scenario string, scopePaths []string, repoRoot string) error {
	if scenario == "" {
		return nil
	}

	if repoRoot == "" {
		repoRoot = os.Getenv("VROOLI_ROOT")
	}
	if repoRoot == "" {
		return fmt.Errorf("VROOLI_ROOT not set; cannot validate paths")
	}

	scenarioRoot := filepath.Join(repoRoot, "scenarios", scenario)

	validator, err := NewPathValidator(scenarioRoot)
	if err != nil {
		return fmt.Errorf("failed to create path validator: %w", err)
	}

	for _, scopePath := range scopePaths {
		// Reject paths that start with ~ (home directory reference)
		if strings.HasPrefix(scopePath, "~") {
			return fmt.Errorf("invalid scope path '%s': home directory references are not allowed", scopePath)
		}

		fullPath := scopePath
		if !filepath.IsAbs(scopePath) {
			fullPath = filepath.Join(scenarioRoot, scopePath)
		}

		if err := validator.ValidatePath(fullPath); err != nil {
			return fmt.Errorf("invalid scope path '%s': %w", scopePath, err)
		}
	}

	return nil
}

// ScanPromptForPaths extracts file paths from a prompt and validates them.
func ScanPromptForPaths(prompt string, validator *PathValidator) []string {
	var suspiciousPaths []string

	pathPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(/[a-zA-Z0-9._/-]+)`),
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(~/[a-zA-Z0-9._/-]+)`),
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(\.\./[a-zA-Z0-9._/-]+)`),
		regexp.MustCompile(`(?m)(?:^|\s|["'\(])(\./\.\./[a-zA-Z0-9._/-]+)`),
	}

	for _, pattern := range pathPatterns {
		matches := pattern.FindAllStringSubmatch(prompt, -1)
		for _, match := range matches {
			if len(match) > 1 {
				path := match[1]
				if strings.HasPrefix(path, "/api/") ||
					strings.HasPrefix(path, "/v1/") ||
					strings.HasPrefix(path, "/home/") && strings.Contains(path, "scenarios/") {
					if validator != nil {
						if err := validator.ValidatePath(path); err != nil {
							suspiciousPaths = append(suspiciousPaths, path)
						}
					}
				} else if strings.Contains(path, "..") {
					suspiciousPaths = append(suspiciousPaths, path)
				}
			}
		}
	}

	return suspiciousPaths
}
