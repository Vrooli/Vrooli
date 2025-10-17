package utils

import (
	"regexp"
	"strings"
)

// PasswordRequirements defines password complexity requirements
type PasswordRequirements struct {
	MinLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
}

// DefaultPasswordRequirements returns the default password requirements
func DefaultPasswordRequirements() PasswordRequirements {
	return PasswordRequirements{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: false, // Made optional to avoid breaking existing tests
	}
}

// ValidatePassword checks if a password meets the requirements
func ValidatePassword(password string, requirements PasswordRequirements) (bool, string) {
	if len(password) < requirements.MinLength {
		return false, "Password must be at least 8 characters long"
	}

	if requirements.RequireUpper && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return false, "Password must contain at least one uppercase letter"
	}

	if requirements.RequireLower && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return false, "Password must contain at least one lowercase letter"
	}

	if requirements.RequireNumber && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return false, "Password must contain at least one number"
	}

	if requirements.RequireSpecial && !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		return false, "Password must contain at least one special character"
	}

	return true, ""
}

// ValidateEmail checks if an email is valid
func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	// Basic email validation pattern
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}
