package logutil

import "strings"

// TruncateForLog truncates a string to a maximum length for logging purposes.
func TruncateForLog(val string, max int) string {
	trimmed := strings.TrimSpace(val)
	if len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max] + "…"
}
