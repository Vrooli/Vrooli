// Package validation provides input validation utilities.
package validation

import "strings"

// IsSafeScenarioName checks if the scenario name is safe (no path traversal).
// Returns true if the name is non-empty and does not contain path traversal characters.
func IsSafeScenarioName(name string) bool {
	if name == "" {
		return false
	}
	return !strings.Contains(name, "..") && !strings.Contains(name, "/") && !strings.Contains(name, "\\")
}
