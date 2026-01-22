package main

import "strings"

// NormalizeEmail normalizes an email address for consistent storage and lookup.
// It lowercases and trims whitespace to prevent case-sensitivity issues
// that could lead to duplicate credit balances or subscription lookups failing.
func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
