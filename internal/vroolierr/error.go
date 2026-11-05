package vroolierr

import (
	"errors"
	"fmt"
	"strings"
)

type Error struct {
	Code        string
	Category    string
	Hint        string
	Suggestions []string
	Exit        int
	HTTPStatus  int
	Operation   string
	Resource    string
	Message     string
	Err         error
}

// Option adds presentation metadata without changing the error's identity.
type Option func(*Error)

// New constructs a typed control-plane error.
func New(code, category, message string, options ...Option) *Error {
	err := &Error{Code: code, Category: category, Message: message}
	applyOptions(err, options)
	return err
}

// Wrap constructs a typed control-plane error that retains its cause.
func Wrap(cause error, code, category, message string, options ...Option) *Error {
	err := New(code, category, message, options...)
	err.Err = cause
	return err
}

// Ensure returns an existing typed error or wraps an untyped boundary error.
// The boolean is true only when wrapping was required, so debug surfaces can
// report producers that have not adopted typed errors yet.
func Ensure(err error, code, category, message string, options ...Option) (*Error, bool) {
	var typed *Error
	if errors.As(err, &typed) {
		return typed, false
	}
	return Wrap(err, code, category, message, options...), true
}

func WithHint(hint string) Option { return func(err *Error) { err.Hint = hint } }

func WithSuggestions(suggestions ...string) Option {
	return func(err *Error) { err.Suggestions = append([]string(nil), suggestions...) }
}

func WithHTTPStatus(status int) Option { return func(err *Error) { err.HTTPStatus = status } }

func WithExitCode(code int) Option { return func(err *Error) { err.Exit = code } }

func applyOptions(err *Error, options []Option) {
	for _, option := range options {
		if option != nil {
			option(err)
		}
	}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		if e.Err != nil {
			return e.Message + ": " + e.Err.Error()
		}
		return e.Message
	}
	if e.Err != nil && strings.TrimSpace(e.Resource) == "" && strings.TrimSpace(e.Operation) == "" {
		return e.Err.Error()
	}

	target := strings.TrimSpace(e.Resource)
	if target == "" {
		target = "resource"
	}
	action := strings.TrimSpace(e.Operation)
	switch {
	case e.Err == nil && action == "":
		return target
	case e.Err == nil:
		return fmt.Sprintf("%s %s", action, target)
	case action == "":
		return fmt.Sprintf("%s: %v", target, e.Err)
	default:
		return fmt.Sprintf("%s %s: %v", action, target, e.Err)
	}
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) ErrorCategory() string {
	if e == nil {
		return ""
	}
	return e.Category
}

func (e *Error) ErrorHint() string {
	if e == nil {
		return ""
	}
	return e.Hint
}

func (e *Error) ErrorSuggestions() []string {
	if e == nil {
		return nil
	}
	return append([]string(nil), e.Suggestions...)
}

func (e *Error) ExitCode() int {
	if e == nil {
		return 1
	}
	if e.Exit > 0 {
		return e.Exit
	}
	var withCode interface{ ExitCode() int }
	if errors.As(e.Err, &withCode) {
		return withCode.ExitCode()
	}
	return 1
}

func HTTPStatus(err error, fallback int) int {
	if err == nil {
		return fallback
	}
	var typed *Error
	if errors.As(err, &typed) && typed.HTTPStatus > 0 {
		return typed.HTTPStatus
	}
	return fallback
}

func Code(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	var typed *Error
	if errors.As(err, &typed) && strings.TrimSpace(typed.Code) != "" {
		return typed.Code
	}
	return fallback
}

func Category(err error) string {
	if err == nil {
		return ""
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Category
	}
	return ""
}

func Hint(err error) string {
	if err == nil {
		return ""
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Hint
	}
	return ""
}

func Suggestions(err error) []string {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return append([]string(nil), typed.Suggestions...)
	}
	return nil
}

func ExitCode(err error, fallback int) int {
	if err == nil {
		return 0
	}
	var withCode interface{ ExitCode() int }
	if errors.As(err, &withCode) {
		return withCode.ExitCode()
	}
	return fallback
}
