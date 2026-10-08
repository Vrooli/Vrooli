package validation

import (
	"path/filepath"
	"strings"
)

// nonEmpty returns s when it has non-whitespace content, else fallback.
func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// truncate clamps b to at most n bytes for safe inclusion in error messages.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return strings.TrimSpace(string(b))
	}
	return strings.TrimSpace(string(b[:n])) + "…"
}

// relPath renders p relative to scenarioDir using forward slashes. Scanners
// emit absolute or module-relative paths; this normalizes them to the
// scenario-root-relative form the Finding contract specifies. Falls back to
// the cleaned input when a relative path can't be computed.
func relPath(scenarioDir, p string) string {
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(filepath.Clean(p))
	}
	rel, err := filepath.Rel(scenarioDir, p)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(p))
	}
	return filepath.ToSlash(rel)
}
