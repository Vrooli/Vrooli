package phasekeys

import "strings"

// NormalizeKey trims and lowercases an input used for case-insensitive key
// comparison. Alias resolution remains with the owner of each key domain.
func NormalizeKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
