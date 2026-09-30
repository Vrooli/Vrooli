package fixtures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepositoryRoot resolves the Vrooli checkout once for fixture consumers.
// It returns os.ErrNotExist when no repository is available; tests should
// Skip on that error. Any other error means a repository was found but its
// layout is malformed and should fail the test rather than being hidden.
func RepositoryRoot(start string) (string, error) {
	if configured := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); configured != "" {
		return validateRepositoryRoot(configured)
	}
	if strings.TrimSpace(start) == "" {
		return "", os.ErrNotExist
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve fixture start directory: %w", err)
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if root, err := validateRepositoryRoot(dir); err == nil {
			return root, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
	}
}

func validateRepositoryRoot(root string) (string, error) {
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		if os.IsNotExist(err) {
			return "", os.ErrNotExist
		}
		return "", fmt.Errorf("inspect repository marker: %w", err)
	}
	if _, err := os.Stat(filepath.Join(root, "scenarios", "prompt-manager")); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("repository root %q lacks scenarios/prompt-manager", root)
		}
		return "", fmt.Errorf("inspect prompt-manager fixture root: %w", err)
	}
	return root, nil
}
