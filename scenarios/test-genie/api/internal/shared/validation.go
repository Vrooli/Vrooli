package shared

import "errors"

// ValidationError surfaces user-facing issues that should translate into 4xx responses.
type ValidationError struct {
	msg string
}

func (e ValidationError) Error() string {
	return e.msg
}

// NewValidationError wraps the provided message in a structured error.
func NewValidationError(message string) error {
	return ValidationError{msg: message}
}

// IsValidationError reports whether the error can be treated as a validation failure.
func IsValidationError(err error) bool {
	var target ValidationError
	return errors.As(err, &target)
}
