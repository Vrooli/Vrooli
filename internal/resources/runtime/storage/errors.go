package storage

import "fmt"

type ErrorKind string

const (
	ErrInvalidInput ErrorKind = "invalid_input"
	ErrOwnership    ErrorKind = "ownership"
	ErrResolve      ErrorKind = "resolve"
)

type Error struct {
	Kind    ErrorKind
	Message string
	Details string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	switch {
	case e.Err != nil && e.Details != "":
		return fmt.Sprintf("%s: %s: %v", e.Message, e.Details, e.Err)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	case e.Details != "":
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	default:
		return e.Message
	}
}

func (e *Error) Unwrap() error { return e.Err }
