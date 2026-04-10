package apierrors
// DOC: docs/internal/ERROR-SEMANTICS.md

import "fmt"

// Category represents the type of API error.
type Category string

const (
	CategoryValidation   Category = "validation"
	CategoryUnauthorized Category = "unauthorized"
	CategoryForbidden    Category = "forbidden"
	CategoryNotFound     Category = "not_found"
	CategoryConflict     Category = "conflict"
	CategoryCooldown     Category = "cooldown"
	CategoryUnavailable  Category = "unavailable"
	CategoryInternal     Category = "internal"
)

// Recovery hints tell the UI what kind of corrective action the user can take.
const (
	RecoveryFixInput     = "fix_input"
	RecoveryAuthenticate = "authenticate"
	RecoveryWait         = "wait"
	RecoveryCheckConfig  = "check_config"
	RecoveryContactAdmin = "contact_admin"
	RecoveryNone         = "none"
)

// APIError is a structured error with category, user-safe message, and internal detail.
type APIError struct {
	Category       Category
	UserMessage    string
	InternalDetail string
	Underlying     error
	RetryAfterSecs int // only used for cooldown
	Field          string
	Recovery       string
}

func (e *APIError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("%s: %s: %v", e.Category, e.UserMessage, e.Underlying)
	}
	return fmt.Sprintf("%s: %s", e.Category, e.UserMessage)
}

func (e *APIError) Unwrap() error {
	return e.Underlying
}

// Constructors

func Validation(field, detail string) *APIError {
	return &APIError{
		Category:    CategoryValidation,
		UserMessage: fmt.Sprintf("%s: %s", field, detail),
		Field:       field,
		Recovery:    RecoveryFixInput,
	}
}

func Unauthorized(detail string) *APIError {
	return &APIError{
		Category:    CategoryUnauthorized,
		UserMessage: detail,
		Recovery:    RecoveryAuthenticate,
	}
}

func Forbidden(detail string) *APIError {
	return &APIError{
		Category:    CategoryForbidden,
		UserMessage: detail,
		Recovery:    RecoveryContactAdmin,
	}
}

func NotFound(resource, id string) *APIError {
	return &APIError{
		Category:    CategoryNotFound,
		UserMessage: fmt.Sprintf("%s not found: %s", resource, id),
		Recovery:    RecoveryNone,
	}
}

func Conflict(userMsg string) *APIError {
	return &APIError{
		Category:    CategoryConflict,
		UserMessage: userMsg,
		Recovery:    RecoveryWait,
	}
}

func Cooldown(seconds int) *APIError {
	return &APIError{
		Category:       CategoryCooldown,
		UserMessage:    fmt.Sprintf("Investigation is in cooldown. Please wait %d seconds", seconds),
		RetryAfterSecs: seconds,
		Recovery:       RecoveryWait,
	}
}

func Unavailable(service string) *APIError {
	return &APIError{
		Category:    CategoryUnavailable,
		UserMessage: fmt.Sprintf("%s is not available", service),
		Recovery:    RecoveryWait,
	}
}

func Internal(userMsg string, cause error) *APIError {
	return &APIError{
		Category:       CategoryInternal,
		UserMessage:    userMsg,
		InternalDetail: cause.Error(),
		Underlying:     cause,
		Recovery:       RecoveryNone,
	}
}
