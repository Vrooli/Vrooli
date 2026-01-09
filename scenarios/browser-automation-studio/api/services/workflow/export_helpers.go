package workflow

import (
	"os"
	"strings"
)

// sanitizeFilename removes path separators and null bytes from a filename.
func sanitizeFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "browser-automation-export"
	}
	trimmed = strings.ReplaceAll(trimmed, string(os.PathSeparator), "-")
	trimmed = strings.ReplaceAll(trimmed, "\x00", "")
	return trimmed
}
