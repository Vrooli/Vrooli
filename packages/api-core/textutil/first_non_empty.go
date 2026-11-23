// Package textutil contains small, pure helpers for working with text.
package textutil

import "strings"

// FirstNonEmpty returns the first value containing non-whitespace text.
// Whitespace is used only to decide whether a value is empty; the selected
// value is returned unchanged.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
