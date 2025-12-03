// Package main provides string and JSON utility functions for the secrets-manager API.
//
// These utilities support common operations such as deduplication, set intersection,
// membership testing, and safe JSON decoding.
package main

import (
	"encoding/json"
	"sort"
	"strings"
)

// dedupeStrings returns a sorted slice of unique, non-empty strings.
// Empty strings and duplicates are removed. Returns nil for empty input.
func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	if len(set) == 0 {
		return nil
	}

	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// intersectStrings returns the sorted intersection of two string slices.
// Only elements present in both slices are included. Returns nil if
// either input is empty or if no common elements exist.
func intersectStrings(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(b))
	for _, value := range b {
		set[value] = struct{}{}
	}

	result := []string{}
	seen := make(map[string]struct{})
	for _, candidate := range a {
		if _, ok := set[candidate]; ok {
			if _, dup := seen[candidate]; dup {
				continue
			}
			seen[candidate] = struct{}{}
			result = append(result, candidate)
		}
	}

	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

// containsString checks whether target exists in list.
func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// mergeResourceLists combines two string slices, removing duplicates and empty strings.
// Returns a new slice containing all unique, non-empty elements from both inputs.
func mergeResourceLists(base, extras []string) []string {
	set := make(map[string]struct{}, len(base)+len(extras))
	for _, item := range base {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	for _, item := range extras {
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}

	merged := make([]string, 0, len(set))
	for key := range set {
		merged = append(merged, key)
	}
	return merged
}

// decodeJSONMap safely decodes a JSON payload into a map[string]interface{}.
// Returns nil if the payload is empty, null, or cannot be decoded.
func decodeJSONMap(payload []byte) map[string]interface{} {
	if len(payload) == 0 || string(payload) == "null" {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}
	return result
}

// decodeStringMap safely decodes a JSON payload into a map[string]string.
// Returns an empty map if the payload is empty, null, or cannot be decoded.
func decodeStringMap(payload []byte) map[string]string {
	if len(payload) == 0 || string(payload) == "null" {
		return map[string]string{}
	}

	var result map[string]string
	if err := json.Unmarshal(payload, &result); err != nil {
		return map[string]string{}
	}
	return result
}
