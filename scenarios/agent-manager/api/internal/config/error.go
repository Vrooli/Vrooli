package config

import "fmt"

// Error reports invalid or absent configuration. It deliberately exposes the
// setting and missing state so callers can retain stable recovery behavior
// without coupling configuration loading to business-domain errors.
type Error struct {
	Setting string
	Message string
	Cause   error
	Missing bool
}

func (e *Error) Error() string {
	if e.Setting != "" {
		return fmt.Sprintf("config error for %s: %s", e.Setting, e.Message)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

func NewMissing(setting, message string, cause error) *Error {
	return &Error{Setting: setting, Message: message, Cause: cause, Missing: true}
}

func NewInvalid(setting, message string, cause error) *Error {
	return &Error{Setting: setting, Message: message, Cause: cause}
}
