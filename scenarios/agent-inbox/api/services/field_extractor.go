// Package services provides application services for the Agent Inbox scenario.
//
// This file provides utilities for extracting values from nested map structures
// using dot-notation paths. These utilities are used by the async tracker to
// extract operation IDs, status values, and progress information from tool
// results based on configurable field paths.
//
// EXAMPLE USAGE:
//
//	result := map[string]interface{}{
//	    "data": map[string]interface{}{
//	        "run": map[string]interface{}{
//	            "id": "run_123",
//	            "status": "running",
//	            "progress": 45,
//	        },
//	    },
//	}
//
//	id := ExtractStringField(result, "data.run.id")        // "run_123"
//	status := ExtractStringField(result, "data.run.status") // "running"
//	progress := ExtractIntField(result, "data.run.progress") // 45
package services

import "strings"

// ExtractField extracts a value from a nested map using dot notation.
// Returns nil if the path doesn't exist or any intermediate value is not a map.
//
// Example paths:
//   - "id" -> m["id"]
//   - "data.id" -> m["data"]["id"]
//   - "result.run.status" -> m["result"]["run"]["status"]
func ExtractField(m map[string]interface{}, path string) interface{} {
	if path == "" {
		return nil
	}

	parts := splitDotPath(path)
	current := interface{}(m)

	for _, part := range parts {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = currentMap[part]
		if current == nil {
			return nil
		}
	}

	return current
}

// ExtractStringField extracts a string value from a nested map using dot notation.
// Returns empty string if the path doesn't exist or the value is not a string.
func ExtractStringField(m map[string]interface{}, path string) string {
	val := ExtractField(m, path)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// ExtractIntField extracts an int value from a nested map using dot notation.
// Returns nil if the path doesn't exist or the value cannot be converted to int.
// Handles float64 (JSON default), int, and int64 types.
func ExtractIntField(m map[string]interface{}, path string) *int {
	val := ExtractField(m, path)
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case float64:
		i := int(v)
		return &i
	case int:
		return &v
	case int64:
		i := int(v)
		return &i
	}
	return nil
}

// splitDotPath splits a dot-notation path into parts.
// Handles empty strings and consecutive dots gracefully.
func splitDotPath(path string) []string {
	if path == "" {
		return nil
	}
	// Use strings.Split for efficiency - it handles the common case well
	parts := strings.Split(path, ".")
	// Filter out empty parts (from consecutive dots or leading/trailing dots)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// ContainsString checks if a slice contains a specific string.
// Uses case-sensitive comparison.
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
