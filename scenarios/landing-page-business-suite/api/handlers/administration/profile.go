package administration

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
	"landing-page-business-suite-api/internal/administration"
)

type (
	ProfileResponse struct {
		Email             string `json:"email"`
		IsDefaultEmail    bool   `json:"is_default_email"`
		IsDefaultPassword bool   `json:"is_default_password"`
	}
	ProfileUpdateRequest struct {
		CurrentPassword string `json:"current_password"`
		NewEmail        string `json:"new_email"`
		NewPassword     string `json:"new_password"`
	}
)

type ProfileService interface {
	Profile(context.Context, string) (administration.AdminProfile, error)
	EmailInUse(context.Context, string, int64) (bool, error)
	UpdateProfile(context.Context, int64, string, string) error
	RevokeOtherSessions(context.Context, string, string) (int64, error)
}

type ProfileDependencies struct {
	Auth            ProfileService
	Sessions        SessionManager
	DefaultEmail    func() string
	DefaultPassword func() string
	ValidateEmail   func(string) error
	WriteError      func(http.ResponseWriter, int, string, string)
	Log             func(string, map[string]any)
	LogError        func(string, map[string]any)
}

func Profile(deps ProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email, ok := profileSessionEmail(deps, r)
		if !ok {
			deps.WriteError(w, http.StatusUnauthorized, "Session expired. Please log in again.", "unauthorized")
			return
		}
		profile, err := deps.Auth.Profile(r.Context(), email)
		if err == sql.ErrNoRows {
			deps.WriteError(w, http.StatusUnauthorized, "Session expired. Please log in again.", "unauthorized")
			return
		}
		if err != nil {
			deps.LogError("admin_profile_lookup_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load profile. Please try again.", "server_error")
			return
		}
		writeProfile(w, profileResponse(deps, profile.Email, profile.PasswordHash), deps, "admin_profile_encode_failed")
	}
}

func UpdateProfile(deps ProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currentEmail, ok := profileSessionEmail(deps, r)
		if !ok {
			deps.WriteError(w, http.StatusUnauthorized, "Session expired. Please log in again.", "unauthorized")
			return
		}
		var request ProfileUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		request.CurrentPassword, request.NewEmail, request.NewPassword = strings.TrimSpace(request.CurrentPassword), strings.TrimSpace(request.NewEmail), strings.TrimSpace(request.NewPassword)
		if request.CurrentPassword == "" {
			deps.WriteError(w, http.StatusBadRequest, "Current password is required", "validation")
			return
		}
		if request.NewEmail == "" && request.NewPassword == "" {
			deps.WriteError(w, http.StatusBadRequest, "Provide a new email or password to update", "validation")
			return
		}
		profile, err := deps.Auth.Profile(r.Context(), currentEmail)
		if err == sql.ErrNoRows {
			deps.WriteError(w, http.StatusUnauthorized, "Session expired. Please log in again.", "unauthorized")
			return
		}
		if err != nil {
			deps.LogError("admin_profile_load_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load admin profile. Please try again.", "server_error")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(request.CurrentPassword)) != nil {
			deps.WriteError(w, http.StatusUnauthorized, "Invalid credentials", "unauthorized")
			return
		}
		targetEmail, targetHash := profile.Email, profile.PasswordHash
		if request.NewEmail != "" && !strings.EqualFold(request.NewEmail, profile.Email) {
			if err := deps.ValidateEmail(request.NewEmail); err != nil {
				deps.WriteError(w, http.StatusBadRequest, "Invalid email address", "validation")
				return
			}
			exists, err := deps.Auth.EmailInUse(r.Context(), request.NewEmail, profile.ID)
			if err != nil {
				deps.LogError("admin_email_validation_failed", map[string]any{"error": err.Error()})
				deps.WriteError(w, http.StatusInternalServerError, "Failed to validate email. Please try again.", "server_error")
				return
			}
			if exists {
				deps.WriteError(w, http.StatusConflict, "Email already in use", "validation")
				return
			}
			targetEmail = request.NewEmail
		}
		if request.NewPassword != "" {
			if err := ValidateProfilePassword(request.NewPassword, profile.PasswordHash, deps.DefaultPassword()); err != nil {
				deps.WriteError(w, http.StatusBadRequest, err.Error(), "validation")
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				deps.LogError("admin_password_hash_failed", map[string]any{"error": err.Error()})
				deps.WriteError(w, http.StatusInternalServerError, "Failed to process password. Please try again.", "server_error")
				return
			}
			targetHash = string(hash)
		}
		if targetEmail == profile.Email && targetHash == profile.PasswordHash {
			deps.WriteError(w, http.StatusBadRequest, "No changes detected", "validation")
			return
		}
		if err := deps.Auth.UpdateProfile(r.Context(), profile.ID, targetEmail, targetHash); err != nil {
			deps.LogError("admin_profile_update_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to update profile. Please try again.", "server_error")
			return
		}
		session, _ := deps.Sessions.GetSession(r, sessionName)
		currentSessionID, _ := session.Values["session_id"].(string)
		if targetHash != profile.PasswordHash {
			if affected, err := deps.Auth.RevokeOtherSessions(r.Context(), currentEmail, currentSessionID); err != nil {
				deps.LogError("admin_sessions_invalidation_failed", map[string]any{"error": err.Error(), "email": currentEmail})
			} else if affected > 0 {
				deps.Log("admin_sessions_invalidated_on_password_change", map[string]any{"level": "info", "email": currentEmail, "sessions_revoked": affected, "security": true})
			}
		}
		session.Values["email"] = targetEmail
		if err := deps.Sessions.SaveSession(r, w, session); err != nil {
			deps.LogError("session_save_after_profile_update_failed", map[string]any{"error": err.Error()})
		}
		deps.Log("admin_profile_updated", map[string]any{"level": "info", "changed_email": targetEmail != profile.Email, "changed_secret": targetHash != profile.PasswordHash})
		writeProfile(w, profileResponse(deps, targetEmail, targetHash), deps, "admin_profile_update_encode_failed")
	}
}

func profileSessionEmail(deps ProfileDependencies, r *http.Request) (string, bool) {
	session, _ := deps.Sessions.GetSession(r, sessionName)
	email, ok := session.Values["email"].(string)
	return email, ok && strings.TrimSpace(email) != ""
}

func profileResponse(deps ProfileDependencies, email, passwordHash string) ProfileResponse {
	configured := deps.DefaultPassword()
	return ProfileResponse{Email: email, IsDefaultEmail: strings.EqualFold(email, deps.DefaultEmail()), IsDefaultPassword: configured != "" && bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(configured)) == nil}
}

func writeProfile(w http.ResponseWriter, value ProfileResponse, deps ProfileDependencies, event string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		deps.LogError(event, map[string]any{"error": err.Error()})
	}
}

func ValidateProfilePassword(candidate, currentHash, defaultPassword string) error {
	if len(candidate) < 12 {
		return fmt.Errorf("Password must be at least 12 characters")
	}
	letter, digit := false, false
	for _, c := range candidate {
		letter = letter || unicode.IsLetter(c)
		digit = digit || unicode.IsDigit(c)
	}
	if !letter || !digit {
		return fmt.Errorf("Password must include letters and numbers")
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(candidate)) == nil {
		return fmt.Errorf("New password must be different from the current password")
	}
	if defaultPassword != "" && subtle.ConstantTimeCompare([]byte(defaultPassword), []byte(candidate)) == 1 {
		return fmt.Errorf("New password cannot use the default credential")
	}
	return nil
}
