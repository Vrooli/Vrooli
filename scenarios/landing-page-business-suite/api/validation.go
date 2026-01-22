package main

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
)

// Common validation errors
var (
	ErrEmailRequired = errors.New("email is required")
	ErrEmailInvalid  = errors.New("invalid email format")
)

// ValidateEmail validates and normalizes an email address.
// It trims whitespace, lowercases the email, and validates the format using mail.ParseAddress.
// Returns the normalized email on success, or an error if the email is empty or invalid.
func ValidateEmail(email string) (string, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return "", ErrEmailRequired
	}

	// Use mail.ParseAddress for RFC 5322 compliant email validation
	_, err := mail.ParseAddress(email)
	if err != nil {
		return "", ErrEmailInvalid
	}

	return email, nil
}

// ValidateEmailForHandler validates an email and writes an error response if invalid.
// Returns the normalized email and true on success, or empty string and false on failure.
func ValidateEmailForHandler(w http.ResponseWriter, email string) (string, bool) {
	normalized, err := ValidateEmail(email)
	if err != nil {
		if errors.Is(err, ErrEmailRequired) {
			writeJSONError(w, http.StatusBadRequest, "Email is required", ApiErrorTypeValidation)
		} else {
			writeJSONError(w, http.StatusBadRequest, "Invalid email format", ApiErrorTypeValidation)
		}
		return "", false
	}
	return normalized, true
}

// RequireNonEmpty validates that a string field is non-empty after trimming.
// Returns the trimmed value and true on success, or empty string and false on failure.
func RequireNonEmpty(w http.ResponseWriter, value, fieldName string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		writeJSONError(w, http.StatusBadRequest, fieldName+" is required", ApiErrorTypeValidation)
		return "", false
	}
	return trimmed, true
}

// ValidateNonEmpty validates that a string field is non-empty after trimming.
// Returns an error if the value is empty.
func ValidateNonEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(fieldName + " is required")
	}
	return nil
}

// ValidateStringLength validates that a string's length is within bounds.
// Either min or max can be 0 to skip that check.
func ValidateStringLength(value, fieldName string, minLen, maxLen int) error {
	length := len(value)
	if minLen > 0 && length < minLen {
		return errors.New(fieldName + " must be at least " + string(rune('0'+minLen)) + " characters")
	}
	if maxLen > 0 && length > maxLen {
		return errors.New(fieldName + " must be at most " + string(rune('0'+maxLen)) + " characters")
	}
	return nil
}

// ValidateInList validates that a value is in the allowed list.
// Returns an error if the value is not in the list.
func ValidateInList(value, fieldName string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return errors.New(fieldName + " must be one of: " + strings.Join(allowed, ", "))
}
