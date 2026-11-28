package components

import (
	"path/filepath"
	"strings"
)

// IsAuthoredReleaseFile identifies bytes frozen by the release immutability
// contract. Derived artifacts are regenerated and stay outside the hash ledger.
func IsAuthoredReleaseFile(path string) bool {
	name := filepath.Base(filepath.ToSlash(path))
	// story.json and dependencies.json are generator outputs. Only source
	// modules, including the authored story harness, belong in the immutable
	// authored-byte ledger.
	return strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")
}
