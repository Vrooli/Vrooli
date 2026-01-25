// Package resolver handles resolution of import paths to actual file paths
package resolver

import (
	"os"
	"path/filepath"
	"strings"
)

// PathResolver resolves import specifiers to actual file paths
type PathResolver interface {
	// Resolve attempts to resolve an import specifier to a file path
	// Returns the resolved path relative to repoRoot, or empty string if not found
	Resolve(importSource string, fromFile string, repoRoot string) string
}

// RelativeResolver resolves relative imports (./foo, ../bar)
type RelativeResolver struct{}

// NewRelativeResolver creates a new relative import resolver
func NewRelativeResolver() *RelativeResolver {
	return &RelativeResolver{}
}

// Resolve resolves a relative import to a file path
func (r *RelativeResolver) Resolve(importSource string, fromFile string, repoRoot string) string {
	if !strings.HasPrefix(importSource, ".") {
		return "" // Not a relative import
	}

	// Get the directory of the importing file
	fromDir := filepath.Dir(fromFile)

	// Resolve the relative path
	resolved := filepath.Join(fromDir, importSource)
	resolved = filepath.Clean(resolved)

	// Try different extensions for the resolved path
	extensions := []string{
		"",     // Exact match
		".ts",  // TypeScript
		".tsx", // TypeScript React
		".js",  // JavaScript
		".jsx", // JavaScript React
		".mjs", // ES modules
		".go",  // Go
		".py",  // Python
		"/index.ts",
		"/index.tsx",
		"/index.js",
		"/index.jsx",
		"/__init__.py",
	}

	for _, ext := range extensions {
		candidate := resolved + ext
		absPath := filepath.Join(repoRoot, candidate)
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			return candidate
		}
	}

	// Check if it's a directory with an index file
	absResolved := filepath.Join(repoRoot, resolved)
	if info, err := os.Stat(absResolved); err == nil && info.IsDir() {
		indexFiles := []string{
			"index.ts",
			"index.tsx",
			"index.js",
			"index.jsx",
			"__init__.py",
			"mod.rs",
		}
		for _, indexFile := range indexFiles {
			candidate := filepath.Join(resolved, indexFile)
			absPath := filepath.Join(repoRoot, candidate)
			if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	return ""
}

// IsRelativeImport returns true if the import is a relative path
func IsRelativeImport(source string) bool {
	return strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../")
}
