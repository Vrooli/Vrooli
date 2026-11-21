// Package stringsx holds small, behaviorally shared string primitives.
package stringsx

import "strings"

// FirstNonEmpty returns the first non-blank value, trimmed.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
