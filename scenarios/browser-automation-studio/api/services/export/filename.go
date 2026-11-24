package export

import (
	"os"
	"strings"
)

// SanitizeFilename removes path separators and NUL bytes from an exported
// name; fallback preserves the caller's public default for a blank name.
func SanitizeFilename(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	trimmed = strings.ReplaceAll(trimmed, string(os.PathSeparator), "-")
	return strings.ReplaceAll(trimmed, "\x00", "")
}
