package artifacts

import (
	"path/filepath"
	"strings"
)

// SanitizeFilename converts a string to a safe filename.
// It replaces path separators and spaces with dashes, removes
// invalid characters, collapses multiple dashes, and trims
// leading/trailing dashes.
func SanitizeFilename(name string) string {
	// Replace path separators and spaces with dashes
	name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Keep only alphanumeric, dash, and underscore
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)

	// Collapse multiple dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	return strings.Trim(name, "-")
}

// SanitizeFilenameWithoutExtension sanitizes a filename and removes
// its extension. Useful for generating artifact names from source files.
func SanitizeFilenameWithoutExtension(name string) string {
	// Remove extension first
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return SanitizeFilename(name)
}
