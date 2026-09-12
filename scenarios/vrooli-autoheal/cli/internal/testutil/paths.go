package testutil

import "path/filepath"

// FixturePath centralizes portable fixture-path construction for CLI tests.
func FixturePath(root, name string) string {
	return filepath.Join(root, name)
}
