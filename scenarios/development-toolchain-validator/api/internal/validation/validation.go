// DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
// Package validation provides shared validation utilities for the API.
//
// This package consolidates common validation patterns used across domain
// services to ensure consistency and reduce duplication.
//
// # Design Principles
//
//   - Validators are pure functions (no side effects)
//   - All regex patterns are compiled at package init time
//   - Validators return bool; error construction is caller's responsibility
//   - Domain-specific validation (slug length limits) uses config injection
//
// # When to Add Here
//
// Add a validator here when:
//   - The same validation logic is needed in 2+ domains
//   - The validation is format-based (regex, enum, range)
//
// Keep validation in domain when:
//   - It requires domain-specific context (e.g., checking repo for duplicates)
//   - It's a one-off business rule
package validation

import (
	"regexp"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Slug and Identifier Validation
// ─────────────────────────────────────────────────────────────────────────────

// slugRegex validates URL-safe slug identifiers.
//
// Format rules:
//   - Lowercase letters (a-z): URL-friendly, case-insensitive matching
//   - Numbers (0-9): Allow version suffixes like "react-app-v2"
//   - Hyphens (-): Human-readable word separation
//   - Must start and end with alphanumeric: Prevents edge cases like "--slug"
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// IsValidSlugFormat checks if a string matches the slug format requirements.
// Does NOT check length - that's handled by SlugLengthValidator.
// Single character slugs fail the regex (need start AND end alphanumeric).
func IsValidSlugFormat(slug string) bool {
	if len(slug) == 1 {
		// Single char passes if alphanumeric
		return regexp.MustCompile(`^[a-z0-9]$`).MatchString(slug)
	}
	return slugRegex.MatchString(slug)
}

// skillIDRegex validates skill IDs with slightly relaxed rules.
//
// Format rules:
//   - Lowercase letters (a-z): URL-friendly
//   - Numbers (0-9): Allow version suffixes
//   - Hyphens (-): Word separation
//   - Must start with a letter: Skills are named concepts
var skillIDRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// IsValidSkillIDFormat checks if a string matches the skill ID format.
func IsValidSkillIDFormat(skillID string) bool {
	return skillIDRegex.MatchString(skillID)
}

// ─────────────────────────────────────────────────────────────────────────────
// JSONPath Validation
// ─────────────────────────────────────────────────────────────────────────────

// jsonPathRegex validates JSONPath expressions.
// Supports: $.field, $[0], $.field[0].nested, $[*], etc.
var jsonPathRegex = regexp.MustCompile(`^\$(\.[a-zA-Z_][a-zA-Z0-9_]*|\[\d+\]|\[\*\])*$`)

// IsValidJSONPath checks if an expression is a valid JSONPath.
func IsValidJSONPath(path string) bool {
	if path == "$" {
		return true
	}
	return jsonPathRegex.MatchString(path)
}

// ─────────────────────────────────────────────────────────────────────────────
// Length Validation
// ─────────────────────────────────────────────────────────────────────────────

// IsLengthInRange checks if a string length is within bounds (inclusive).
func IsLengthInRange(s string, min, max int) bool {
	length := len(s)
	return length >= min && length <= max
}

// ─────────────────────────────────────────────────────────────────────────────
// Command Safety Validation
// ─────────────────────────────────────────────────────────────────────────────

// dangerousPatterns contains shell patterns that could be used for destructive operations.
var dangerousPatterns = []string{
	"rm ", "rm\t", "rmdir", "dd ", "mkfs", "format",
	"sudo", "> /", ">> /", "2> /", "2>> /",
	"mv ", "cp /", "chmod", "chown",
	"kill", "pkill", "shutdown", "reboot", "halt",
	"curl | ", "wget | ", "| bash", "| sh", "| zsh",
	"eval ", "exec ", "$(", "`",
}

// allowedCommandPrefixes contains commands that are known-safe and explicitly allowed.
var allowedCommandPrefixes = []string{
	"scenario-auditor", "test-genie", "scenario-completeness-scoring",
	"vrooli scenario", "ast-grep", "sg ", "jq ", "yq ",
}

// CommandSafetyResult contains the result of command safety validation.
type CommandSafetyResult struct {
	IsSafe             bool
	DangerousPattern   string
	UsesAllowedCommand bool
}

// ValidateCommandSafety checks if a command is safe to execute.
// Returns detailed result about what was found.
func ValidateCommandSafety(cmd string) CommandSafetyResult {
	cmdLower := strings.ToLower(cmd)

	// Check for dangerous patterns
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return CommandSafetyResult{
				IsSafe:           false,
				DangerousPattern: pattern,
			}
		}
	}

	// Check if command starts with an allowed prefix
	hasAllowed := false
	for _, allowed := range allowedCommandPrefixes {
		if strings.HasPrefix(cmdLower, allowed) {
			hasAllowed = true
			break
		}
	}

	return CommandSafetyResult{
		IsSafe:             true,
		UsesAllowedCommand: hasAllowed,
	}
}

// IsCommandSafe returns true if the command doesn't contain dangerous patterns.
// Convenience wrapper around ValidateCommandSafety.
func IsCommandSafe(cmd string) bool {
	return ValidateCommandSafety(cmd).IsSafe
}

// ─────────────────────────────────────────────────────────────────────────────
// String Utilities
// ─────────────────────────────────────────────────────────────────────────────

// Truncate shortens a string to maxLen characters, adding "..." if truncated.
// If maxLen < 4, returns empty string to prevent invalid output.
func Truncate(s string, maxLen int) string {
	if maxLen < 4 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
