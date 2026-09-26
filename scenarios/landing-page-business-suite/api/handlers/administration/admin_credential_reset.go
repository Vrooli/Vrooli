package administration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
	admin "landing-page-business-suite-api/internal/administration"
)

// AdminCredentialResetService is the narrow store boundary for operator
// credential recovery. It exposes only the reads and the single write needed
// to restore a locked-out administrator; it deliberately has no list or
// create operation, so recovery can never become account provisioning.
type AdminCredentialResetService interface {
	Profile(context.Context, string) (admin.AdminProfile, error)
	EmailInUse(context.Context, string, int64) (bool, error)
	UpdateProfile(context.Context, int64, string, string) error
}

// AdminCredentialResetDependencies carries the collaborators for operator
// recovery. The declared authority credential is written alongside the stored
// hash so a restart on the bootstrap account cannot reintroduce the previous
// password.
type AdminCredentialResetDependencies struct {
	Auth                AdminCredentialResetService
	IsService           func(*http.Request) bool
	DefaultPassword     func() string
	ValidateEmail       func(string) error
	PutAuthority        func(key, value string) error
	RevokeOtherSessions func(context.Context, string, string) (int64, error)
	SecurityEvents      SecurityEvents
	Notifier            SecurityNotifier
	WriteError          func(http.ResponseWriter, int, string, string)
	Log                 func(string, map[string]any)
	LogError            func(string, map[string]any)
}

type adminCredentialResetRequest struct {
	Email       string `json:"email"`
	NewEmail    string `json:"new_email"`
	NewPassword string `json:"new_password"`
}

// ResetAdminCredential is the operator recovery path for a locked-out
// administrator. Only the service principal (the LPBS CLI or a local
// control-plane recovery run holding the credential authority) may call it; a
// browser admin session may not. It updates the stored hash and, when a new
// password is supplied, the declared `admin-default-password` authority value
// so a later restart on the bootstrap account does not revert the reset.
//
// The new password is never returned, logged, or reflected in the response.
func ResetAdminCredential(deps AdminCredentialResetDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.IsService == nil || !deps.IsService(r) {
			deps.WriteError(w, http.StatusForbidden, "Administrator credential reset requires the operator CLI.", "forbidden")
			return
		}
		var request adminCredentialResetRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "A JSON body is required.", "validation")
			return
		}
		email := strings.TrimSpace(request.Email)
		newEmail := strings.TrimSpace(request.NewEmail)
		newPassword := strings.TrimSpace(request.NewPassword)
		if email == "" {
			deps.WriteError(w, http.StatusBadRequest, "email is required", "validation")
			return
		}
		if newEmail == "" && newPassword == "" {
			deps.WriteError(w, http.StatusBadRequest, "new_password or new_email is required", "validation")
			return
		}

		profile, err := deps.Auth.Profile(r.Context(), email)
		if errors.Is(err, sql.ErrNoRows) {
			deps.WriteError(w, http.StatusNotFound, "No administrator account matches that email.", "not_found")
			return
		}
		if err != nil {
			deps.LogError("admin_credential_reset_lookup_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Unable to load the administrator account.", "server_error")
			return
		}

		targetEmail := profile.Email
		targetHash := profile.PasswordHash
		if newEmail != "" && !strings.EqualFold(newEmail, profile.Email) {
			if deps.ValidateEmail != nil {
				if err := deps.ValidateEmail(newEmail); err != nil {
					deps.WriteError(w, http.StatusBadRequest, "Enter a valid email address.", "validation")
					return
				}
			}
			exists, err := deps.Auth.EmailInUse(r.Context(), newEmail, profile.ID)
			if err != nil {
				deps.LogError("admin_credential_reset_email_check_failed", map[string]any{"error": err.Error()})
				deps.WriteError(w, http.StatusInternalServerError, "Unable to validate the administrator email.", "server_error")
				return
			}
			if exists {
				deps.WriteError(w, http.StatusConflict, "That email is already in use.", "conflict")
				return
			}
			targetEmail = newEmail
		}
		if newPassword != "" {
			defaultPassword := ""
			if deps.DefaultPassword != nil {
				defaultPassword = deps.DefaultPassword()
			}
			if err := ValidateProfilePassword(newPassword, profile.PasswordHash, defaultPassword); err != nil {
				deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				deps.LogError("admin_credential_reset_hash_failed", map[string]any{"error": err.Error()})
				deps.WriteError(w, http.StatusInternalServerError, "Unable to process the administrator password.", "server_error")
				return
			}
			targetHash = string(hash)
		}

		if err := deps.Auth.UpdateProfile(r.Context(), profile.ID, targetEmail, targetHash); err != nil {
			deps.LogError("admin_credential_reset_update_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Unable to reset the administrator credential.", "server_error")
			return
		}

		// Align the declared credential so restart reconciliation cannot return
		// the account to the previous password. A failed authority write still
		// leaves the database reset in place and is reported to the operator.
		authorityUpdated := false
		if newPassword != "" && deps.PutAuthority != nil {
			if err := deps.PutAuthority("ADMIN_DEFAULT_PASSWORD", newPassword); err != nil {
				deps.LogError("admin_credential_reset_authority_failed", map[string]any{"error": err.Error()})
			} else {
				authorityUpdated = true
			}
		}

		if deps.RevokeOtherSessions != nil {
			if _, err := deps.RevokeOtherSessions(r.Context(), profile.Email, ""); err != nil {
				deps.LogError("admin_credential_reset_session_revoke_failed", map[string]any{"error": err.Error()})
			}
		}

		emailChanged := !strings.EqualFold(targetEmail, profile.Email)
		detail := map[string]any{
			"email_changed":     emailChanged,
			"password_changed":  newPassword != "",
			"authority_updated": authorityUpdated,
		}
		deps.Log("admin_credential_reset_by_operator", map[string]any{"level": "warn", "email": email, "email_changed": emailChanged, "password_changed": newPassword != "", "authority_updated": authorityUpdated})
		deps.recordSecurityEvent(r, "admin_credential_reset", email, detail)
		deps.notify(r, profile.Email, "admin_credential_reset", "Your administrator credentials were reset by an operator.")
		if emailChanged {
			deps.notify(r, targetEmail, "admin_credential_reset", "Your administrator sign-in email was set by an operator.")
		}

		writeMFAJSON(w, map[string]any{
			"email":             targetEmail,
			"email_changed":     emailChanged,
			"password_changed":  newPassword != "",
			"authority_updated": authorityUpdated,
		})
	}
}

func (deps AdminCredentialResetDependencies) recordSecurityEvent(r *http.Request, event, email string, detail map[string]any) {
	if deps.SecurityEvents == nil {
		return
	}
	_ = deps.SecurityEvents.Record(r.Context(), event, email, "", r.RemoteAddr, r.UserAgent(), detail)
}

func (deps AdminCredentialResetDependencies) notify(r *http.Request, email, event, detail string) {
	if deps.Notifier == nil {
		return
	}
	if err := deps.Notifier.Notify(r.Context(), email, event, detail); err != nil {
		deps.recordSecurityEvent(r, event, email, map[string]any{"notify_failed": err.Error()})
	}
}
