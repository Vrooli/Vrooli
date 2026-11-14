package textutil

import "strings"

// FirstNonBlank returns the first value containing non-whitespace characters,
// preserving the selected value exactly as supplied.
func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
