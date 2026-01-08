package vps

import (
	"sort"
)

// stableUniqueStrings returns a sorted, deduplicated copy of the input slice.
func stableUniqueStrings(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(slice))
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

// contains checks if a slice contains a value.
func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}
