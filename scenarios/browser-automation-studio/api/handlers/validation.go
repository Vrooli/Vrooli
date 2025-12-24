package handlers

import (
	"fmt"
	"unicode/utf8"
)

// Field length limits matching database schema constraints.
// These prevent database errors by validating before write.
const (
	MaxNameLength           = 255  // VARCHAR(255) for name fields
	MaxCronExpressionLength = 100  // VARCHAR(100) for cron_expression
	MaxTimezoneLength       = 50   // VARCHAR(50) for timezone
	MaxFolderPathLength     = 500  // VARCHAR(500) for folder_path
	MaxFilePathLength       = 1000 // VARCHAR(1000) for file_path
	MaxBulkOperationSize    = 100  // Maximum items in bulk operations
)

// validateStringLength checks if a string exceeds the maximum allowed length.
// Returns nil if valid, or an error with a user-friendly message.
func validateStringLength(field, value string, maxLen int) error {
	// Use rune count for proper Unicode handling
	if utf8.RuneCountInString(value) > maxLen {
		return fmt.Errorf("%s exceeds maximum length of %d characters", field, maxLen)
	}
	return nil
}

// validateName validates a name field against the standard VARCHAR(255) limit.
func validateName(value string) error {
	return validateStringLength("name", value, MaxNameLength)
}

// validateFolderPath validates a folder_path field against the VARCHAR(500) limit.
func validateFolderPath(value string) error {
	return validateStringLength("folder_path", value, MaxFolderPathLength)
}

// validateFilePath validates a file_path field against the VARCHAR(1000) limit.
func validateFilePath(value string) error {
	return validateStringLength("file_path", value, MaxFilePathLength)
}

// validateCronExpression validates a cron_expression field length (not syntax).
func validateCronExpressionLength(value string) error {
	return validateStringLength("cron_expression", value, MaxCronExpressionLength)
}

// validateTimezone validates a timezone field against the VARCHAR(50) limit.
func validateTimezoneLength(value string) error {
	return validateStringLength("timezone", value, MaxTimezoneLength)
}

// validateBulkOperationSize checks if a slice exceeds the maximum bulk operation size.
func validateBulkOperationSize(field string, count int) error {
	if count > MaxBulkOperationSize {
		return fmt.Errorf("%s exceeds maximum of %d items per request", field, MaxBulkOperationSize)
	}
	return nil
}
