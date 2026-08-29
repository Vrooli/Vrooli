// Package values contains small, domain-neutral value selection helpers.
package values

import (
	"slices"
	"strings"
)

// FirstNonEmpty returns the first value containing non-whitespace text.
func FirstNonEmpty(candidates ...string) string {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

// UniqueStrings trims, removes empty values and returns deterministic output.
func UniqueStrings(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}
	slices.Sort(result)
	return result
}
