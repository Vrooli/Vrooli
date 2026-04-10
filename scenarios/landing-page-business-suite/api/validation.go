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

// Common validation errors for URL and batch validation
var (
	ErrURLRequired   = errors.New("URL is required")
	ErrURLInvalid    = errors.New("invalid URL format")
	ErrBatchTooLarge = errors.New("batch size exceeds maximum")
)

// ValidateURL validates that a string is a valid URL.
// It checks for proper scheme (http/https) and basic URL structure.
// Returns the normalized URL on success, or an error if invalid.
func ValidateURL(urlStr string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "", ErrURLRequired
	}

	// Must start with http:// or https://
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return "", ErrURLInvalid
	}

	// Basic structure check - must have at least scheme + host
	parts := strings.SplitN(urlStr, "://", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", ErrURLInvalid
	}

	// Check for valid host part (at least one character before path/query)
	hostPart := parts[1]
	slashIdx := strings.Index(hostPart, "/")
	if slashIdx == 0 {
		return "", ErrURLInvalid
	}
	if slashIdx == -1 {
		// No path - host is the whole thing
		if len(hostPart) == 0 {
			return "", ErrURLInvalid
		}
	} else {
		// Has path - check host is non-empty
		if slashIdx == 0 {
			return "", ErrURLInvalid
		}
	}

	return urlStr, nil
}

// ValidateURLOptional validates a URL if provided, returns empty string if not provided.
// This is useful for optional URL fields.
func ValidateURLOptional(urlStr string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return "", nil // Optional - empty is valid
	}
	return ValidateURL(urlStr)
}

// NormalizeRedirectURL validates an optional redirect URL.
// Allows relative paths ("/success") or absolute http(s) URLs.
// Returns empty string when input is empty.
func NormalizeRedirectURL(urlStr string) (string, error) {
	trimmed := strings.TrimSpace(urlStr)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed, nil
	}
	return ValidateURL(trimmed)
}

// NormalizeRedirectURLForHandler validates a redirect URL and writes an error response if invalid.
// Returns normalized URL and true on success, or empty string and false on failure.
func NormalizeRedirectURLForHandler(w http.ResponseWriter, urlStr, fieldName string) (string, bool) {
	normalized, err := NormalizeRedirectURL(urlStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid "+fieldName+" format", ApiErrorTypeValidation)
		return "", false
	}
	return normalized, true
}

// ValidateURLForHandler validates a URL and writes an error response if invalid.
// Returns the normalized URL and true on success, or empty string and false on failure.
func ValidateURLForHandler(w http.ResponseWriter, urlStr, fieldName string) (string, bool) {
	normalized, err := ValidateURL(urlStr)
	if err != nil {
		if errors.Is(err, ErrURLRequired) {
			writeJSONError(w, http.StatusBadRequest, fieldName+" is required", ApiErrorTypeValidation)
		} else {
			writeJSONError(w, http.StatusBadRequest, "Invalid "+fieldName+" format", ApiErrorTypeValidation)
		}
		return "", false
	}
	return normalized, true
}

// ValidateBatchSize validates that a batch operation size is within limits.
// maxSize is the maximum allowed batch size.
// Returns nil if valid, or an error if the batch is too large.
func ValidateBatchSize(size, maxSize int) error {
	if size > maxSize {
		return ErrBatchTooLarge
	}
	return nil
}

// ValidateBatchSizeForHandler validates batch size and writes an error response if too large.
// Returns true if valid, or false and writes error response if too large.
func ValidateBatchSizeForHandler(w http.ResponseWriter, size, maxSize int, fieldName string) bool {
	if err := ValidateBatchSize(size, maxSize); err != nil {
		writeJSONError(w, http.StatusBadRequest,
			fieldName+" exceeds maximum allowed batch size",
			ApiErrorTypeValidation)
		return false
	}
	return true
}
