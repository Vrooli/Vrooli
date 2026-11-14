package recoverypoint

import (
	"errors"
	"fmt"
)

// Stable reason codes. The cloud side maps them onto its typed error model
// and the certification matrix names them; do not rename.
const (
	CodeInvalidArgument       = "invalid_argument"
	CodeRecoveryPointCorrupt  = "recovery_point_corrupt"
	CodeRecoveryKeyMissing    = "recovery_key_unavailable"
	CodeRestoreTargetNotClean = "restore_target_not_clean"
	CodeProviderUnavailable   = "backup_provider_unavailable"
	CodeCaptureFailed         = "capture_failed"
	CodeRestoreFailed         = "restore_failed"
	CodeVerifyFailed          = "verify_failed"
	CodeRecoveryPointExists   = "recovery_point_exists"
)

// Error is the typed failure every operation returns.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	// Blocker names what an operator must supply before a retry can succeed
	// (for example the key reference of an unavailable recovery key).
	Blocker string `json:"blocker,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func newError(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

func (e *Error) withDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	e.Details[key] = value
	return e
}

// AsError extracts the typed error, wrapping an untyped one as a failure with
// the given fallback code.
func AsError(err error, fallback string) *Error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}
	return &Error{Code: fallback, Message: err.Error()}
}

// CodeOf returns the stable code of err, or "" for nil.
func CodeOf(err error) string {
	if err == nil {
		return ""
	}
	return AsError(err, CodeCaptureFailed).Code
}
