package secrets

import "fmt"

// ErrorKind identifies structured secrets failures.
type ErrorKind string

const (
	ErrInvalidInput        ErrorKind = "invalid_input"
	ErrResolve             ErrorKind = "resolve_error"
	ErrIO                  ErrorKind = "io_error"
	ErrInvalidData         ErrorKind = "invalid_data"
	ErrInsecurePermissions ErrorKind = "insecure_permissions"
	ErrSymlinkPath         ErrorKind = "symlink_path"
)

// Error is a structured secrets error.
type Error struct {
	Kind    ErrorKind
	Message string
	Details string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := fmt.Sprintf("api-core secrets %s: %s", e.Kind, e.Message)
	if e.Details != "" {
		msg += " (" + e.Details + ")"
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
