package components

import (
	"path/filepath"
	"strings"
)

// IsAuthoredReleaseFile identifies bytes frozen by the release immutability
// contract. Derived artifacts are regenerated and stay outside the hash ledger.
func IsAuthoredReleaseFile(path string) bool {
	name := filepath.Base(filepath.ToSlash(path))
	return name == "experience-contract.json" || name == "story.json" || strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")
}
