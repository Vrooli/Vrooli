// Package validation provides shared validation utilities for prompt-manager handlers.
//
// This package consolidates common validation functions used across agents, teams,
// and other handlers to prevent duplication and ensure consistent behavior.
package validation

import (
	"regexp"
	"strings"
)

// HexColorRegex matches valid hex color strings like #FF00FF.
var HexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// IsValidHexColor checks if a string is a valid hex color (e.g., #FF00FF).
func IsValidHexColor(color string) bool {
	return HexColorRegex.MatchString(color)
}

// Slugify converts a string to a URL-friendly slug.
// It lowercases the string, replaces spaces with hyphens, and removes
// all characters except lowercase letters, numbers, and hyphens.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var result []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	return string(result)
}
