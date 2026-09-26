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

// SetEnv returns an environment with key set to value. It always copies the
// input slice before changing or appending an entry, so callers retain
// ownership of their backing array even when it has spare capacity.
func SetEnv(env []string, key, value string) []string {
	updated := append([]string(nil), env...)
	prefix := key + "="
	for i, entry := range updated {
		if strings.HasPrefix(entry, prefix) {
			updated[i] = prefix + value
			return updated
		}
	}
	return append(updated, prefix+value)
}

// EnvValue returns the value for key from an environment slice, or an empty
// string when the key is absent.
func EnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

// UniqueStrings trims, removes empty values and returns deterministic output.
func UniqueStrings(candidates []string) []string {
	result := UniqueStringsOrdered(candidates)
	slices.Sort(result)
	return result
}

// UniqueStringsOrdered trims, removes empty values, and preserves first-seen
// order for callers whose output is user-visible or semantically ordered.
func UniqueStringsOrdered(candidates []string) []string {
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
	return result
}
