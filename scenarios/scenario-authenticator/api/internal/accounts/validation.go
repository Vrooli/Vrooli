package accounts

import (
	"regexp"
	"strings"
)

// Password complexity requirements. Ported verbatim from the old
// utils.DefaultPasswordRequirements: min 8 chars, upper + lower + number;
// special char optional.
const passwordMinLength = 8

var (
	reUpper  = regexp.MustCompile(`[A-Z]`)
	reLower  = regexp.MustCompile(`[a-z]`)
	reNumber = regexp.MustCompile(`[0-9]`)
	reEmail  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidatePassword reports whether the password meets complexity requirements,
// returning a human message on failure. Ported verbatim.
func ValidatePassword(password string) (bool, string) {
	if len(password) < passwordMinLength {
		return false, "Password must be at least 8 characters long"
	}
	if !reUpper.MatchString(password) {
		return false, "Password must contain at least one uppercase letter"
	}
	if !reLower.MatchString(password) {
		return false, "Password must contain at least one lowercase letter"
	}
	if !reNumber.MatchString(password) {
		return false, "Password must contain at least one number"
	}
	return true, ""
}

// ValidateEmail reports whether the email is syntactically valid. Ported
// verbatim.
func ValidateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	return reEmail.MatchString(email)
}
