package administration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	admin "landing-page-business-suite-api/internal/administration"
)

// AdminMFAService is the two-factor management boundary for the signed-in
// administrator.
type AdminMFAService interface {
	Status(context.Context, string) (*admin.AdminMFAStatus, error)
	BeginEnrollment(context.Context, string) (*admin.AdminMFAEnrollment, error)
	ConfirmEnrollment(context.Context, string, string) ([]string, error)
	Disable(context.Context, string, string) error
	RegenerateRecoveryCodes(context.Context, string, string) ([]string, error)
	Reset(context.Context, string) error
}

type AdminMFADependencies struct {
	MFA        AdminMFAService
	AdminEmail func(*http.Request) (string, bool)
	IsService  func(*http.Request) bool
	WriteError func(http.ResponseWriter, int, string, string)
	Log        func(string, map[string]any)
	LogError   func(string, map[string]any)
}

type mfaCodeRequest struct {
	Code string `json:"code"`
}

// AdminMFAStatus reports the signed-in administrator's two-factor state.
func AdminMFAStatus(deps AdminMFADependencies) http.HandlerFunc {
	return withAdminEmail(deps, func(w http.ResponseWriter, r *http.Request, email string) {
		status, err := deps.MFA.Status(r.Context(), email)
		if err != nil {
			deps.LogError("admin_mfa_status_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Unable to read two-factor settings.", "server_error")
			return
		}
		writeMFAJSON(w, status)
	})
}

// BeginAdminMFAEnrollment issues a pending authenticator secret.
func BeginAdminMFAEnrollment(deps AdminMFADependencies) http.HandlerFunc {
	return withAdminEmail(deps, func(w http.ResponseWriter, r *http.Request, email string) {
		enrollment, err := deps.MFA.BeginEnrollment(r.Context(), email)
		if err != nil {
			writeMFAError(w, deps, err, "admin_mfa_enroll_failed")
			return
		}
		deps.Log("admin_mfa_enrollment_started", map[string]any{"email": email})
		writeMFAJSON(w, enrollment)
	})
}

// ConfirmAdminMFAEnrollment enables two-factor after a valid code.
func ConfirmAdminMFAEnrollment(deps AdminMFADependencies) http.HandlerFunc {
	return withAdminCode(deps, func(w http.ResponseWriter, r *http.Request, email, code string) {
		codes, err := deps.MFA.ConfirmEnrollment(r.Context(), email, code)
		if err != nil {
			writeMFAError(w, deps, err, "admin_mfa_confirm_failed")
			return
		}
		deps.Log("admin_mfa_enabled", map[string]any{"email": email})
		writeMFAJSON(w, map[string]any{"enabled": true, "recovery_codes": codes})
	})
}

// DisableAdminMFA turns two-factor off after a valid code.
func DisableAdminMFA(deps AdminMFADependencies) http.HandlerFunc {
	return withAdminCode(deps, func(w http.ResponseWriter, r *http.Request, email, code string) {
		if err := deps.MFA.Disable(r.Context(), email, code); err != nil {
			writeMFAError(w, deps, err, "admin_mfa_disable_failed")
			return
		}
		deps.Log("admin_mfa_disabled", map[string]any{"level": "warn", "email": email})
		writeMFAJSON(w, map[string]any{"enabled": false})
	})
}

// RegenerateAdminRecoveryCodes replaces recovery codes after a valid code.
func RegenerateAdminRecoveryCodes(deps AdminMFADependencies) http.HandlerFunc {
	return withAdminCode(deps, func(w http.ResponseWriter, r *http.Request, email, code string) {
		codes, err := deps.MFA.RegenerateRecoveryCodes(r.Context(), email, code)
		if err != nil {
			writeMFAError(w, deps, err, "admin_mfa_recovery_regenerate_failed")
			return
		}
		deps.Log("admin_mfa_recovery_codes_regenerated", map[string]any{"email": email})
		writeMFAJSON(w, map[string]any{"recovery_codes": codes})
	})
}

// ResetAdminMFA is the operator recovery path for a lost authenticator and
// lost recovery codes. Only the service principal (the LPBS CLI running with
// the credential authority) may call it; a browser admin session may not.
func ResetAdminMFA(deps AdminMFADependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.IsService == nil || !deps.IsService(r) {
			deps.WriteError(w, http.StatusForbidden, "Two-factor reset requires the operator CLI.", "forbidden")
			return
		}
		var request struct {
			Email string `json:"email"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&request) != nil || strings.TrimSpace(request.Email) == "" {
			deps.WriteError(w, http.StatusBadRequest, "email is required", "validation")
			return
		}
		if err := deps.MFA.Reset(r.Context(), strings.TrimSpace(request.Email)); err != nil {
			deps.LogError("admin_mfa_reset_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Unable to reset two-factor authentication.", "server_error")
			return
		}
		deps.Log("admin_mfa_reset_by_operator", map[string]any{"level": "warn", "email": request.Email})
		writeMFAJSON(w, map[string]any{"enabled": false})
	}
}

func withAdminEmail(deps AdminMFADependencies, next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, ok := deps.AdminEmail(r)
		if !ok {
			deps.WriteError(w, http.StatusUnauthorized, "Session expired. Please log in again.", "unauthorized")
			return
		}
		next(w, r, email)
	}
}

func withAdminCode(deps AdminMFADependencies, next func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return withAdminEmail(deps, func(w http.ResponseWriter, r *http.Request, email string) {
		var request mfaCodeRequest
		if json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&request) != nil || strings.TrimSpace(request.Code) == "" {
			deps.WriteError(w, http.StatusBadRequest, "Enter a code from your authenticator app.", "validation")
			return
		}
		next(w, r, email, request.Code)
	})
}

func writeMFAError(w http.ResponseWriter, deps AdminMFADependencies, err error, event string) {
	switch {
	case errors.Is(err, admin.ErrMFAInvalid), errors.Is(err, admin.ErrMFARequired):
		deps.WriteError(w, http.StatusBadRequest, "That code isn't right. Use the current code from your app.", "validation")
	case errors.Is(err, admin.ErrMFANotEnrolling):
		deps.WriteError(w, http.StatusConflict, "Start setup again to get a new secret.", "validation")
	case errors.Is(err, admin.ErrMFAAlreadyEnabled):
		deps.WriteError(w, http.StatusConflict, "Two-factor authentication is already on.", "validation")
	default:
		deps.LogError(event, map[string]any{"error": err.Error()})
		deps.WriteError(w, http.StatusInternalServerError, "Two-factor settings are unavailable right now.", "server_error")
	}
}

func writeMFAJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(value)
}
