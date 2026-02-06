package stringutil

import (
	"sort"
	"strings"
)

// SortedUnique returns a sorted slice with duplicates and empty strings removed.
// Returns nil for empty input.
func SortedUnique(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(slice))
	out := make([]string, 0, len(slice))
	for _, v := range slice {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

// OrderedUnique returns a deduplicated slice preserving first-occurrence order.
// Trims whitespace and skips empty strings.
func OrderedUnique(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

// Contains reports whether slice contains value.
func Contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
