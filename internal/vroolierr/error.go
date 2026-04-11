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
