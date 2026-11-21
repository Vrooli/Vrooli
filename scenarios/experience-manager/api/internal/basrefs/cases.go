// Package basrefs contains shared helpers for reading experience-manager's
// BAS metadata references without taking ownership of workflow-health assets.
package basrefs

import (
	"path/filepath"
	"sort"
)

// CaseFiles returns BAS case JSON files at the depths experience-manager
// scaffolds and validates today.
func CaseFiles(root string) []string {
	var out []string
	for _, pattern := range []string{
		filepath.Join(root, "bas", "cases", "*.json"),
		filepath.Join(root, "bas", "cases", "*", "*.json"),
		filepath.Join(root, "bas", "cases", "*", "*", "*.json"),
	} {
		matches, _ := filepath.Glob(pattern)
		out = append(out, matches...)
	}
	sort.Strings(out)
	return out
}
