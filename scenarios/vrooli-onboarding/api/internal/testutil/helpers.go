// Package testutil contains API-only fixtures shared by onboarding tests.
// Production packages must not import this package.
package testutil

import (
	"os"
	"path/filepath"
)

// WriteFile creates a private fixture file and its parent directories.
func WriteFile(root, relativePath, contents string) error {
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o600)
}
