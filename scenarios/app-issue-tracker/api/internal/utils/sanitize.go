package utils

import (
	"strings"
	"unicode"
)

// ForFilename sanitizes a string to be safe for use in filenames.
// Keeps only alphanumeric characters, hyphens, and underscores.
// All other characters are replaced with underscores.
// Returns "default" if the result would be empty.
func ForFilename(input string) string {
	var builder strings.Builder
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}

	if builder.Len() == 0 {
		return "default"
	}

	return builder.String()
}

// ForArtifact sanitizes a string to be safe for artifact file components.
// Converts to lowercase, keeps only alphanumeric characters and [-_.] punctuation.
// Replaces backslashes, forward slashes, and spaces with hyphens.
// Collapses consecutive whitespace/hyphens into single hyphens.
// Returns empty string if input is empty after trimming.
func ForArtifact(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	// Replace path separators and spaces with hyphens
	replaced := strings.ReplaceAll(trimmed, "\\", "-")
	replaced = strings.ReplaceAll(replaced, "/", "-")
	replaced = strings.ReplaceAll(replaced, " ", "-")

	var builder strings.Builder
	for _, r := range replaced {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		default:
			// Replace other characters with hyphens
			builder.WriteRune('-')
		}
	}

	sanitized := builder.String()

	// Collapse consecutive hyphens/whitespace into single hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Trim leading/trailing hyphens and underscores
	sanitized = strings.Trim(sanitized, "-_")

	// Trim leading dots (security: prevent directory traversal)
	sanitized = strings.TrimLeft(sanitized, ".")

	return sanitized
}

// ValueOrFallback returns the trimmed value if non-empty, otherwise returns the fallback.
// This is useful for template rendering where placeholders need default values.
func ValueOrFallback(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
