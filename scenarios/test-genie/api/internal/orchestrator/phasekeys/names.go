package phasekeys

import "strings"

// NormalizeKey trims and lowercases a caller-provided descriptor key. Alias
// tables do not belong here: descriptor identity is provider-owned and must be
// preserved exactly apart from whitespace and case normalization.
func NormalizeKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
