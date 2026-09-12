package repocontract

import "fmt"

// ErrorKind identifies the class of repo-contract adapter failure.
type ErrorKind string

const (
	ErrInvalidInput       ErrorKind = "invalid_input"
	ErrInvalidContract    ErrorKind = "invalid_contract"
	ErrUnsupportedVersion ErrorKind = "unsupported_version"
	ErrNotFound           ErrorKind = "not_found"
)

// Error carries structured adapter failures.
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
	if e.Details != "" {
		return fmt.Sprintf("repo-contract %s: %s (%s)", e.Kind, e.Message, e.Details)
	}
	return fmt.Sprintf("repo-contract %s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
