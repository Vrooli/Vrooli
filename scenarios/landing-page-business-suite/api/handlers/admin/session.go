// Package admin owns HTTP transport for administrator authentication and reset actions.
package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

const sessionName = "admin_session"

type (
	LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	SessionResponse struct {
		Email         string `json:"email,omitempty"`
		Authenticated bool   `json:"authenticated"`
		ResetEnabled  bool   `json:"reset_enabled"`
		SessionID     string `json:"session_id,omitempty"`
	}
)

type AuthService interface {
	PasswordHash(context.Context, string) (string, error)
	UpdateLastLogin(context.Context, string) error
	CreateSession(context.Context, string, string, time.Time, string, string) error
	DeleteSession(context.Context, string) error
	SessionExpiry(context.Context, string, string) (time.Time, error)
	TouchSession(context.Context, string) error
}
type (
	SessionManager interface {
		GetSession(*http.Request, string) (*sessions.Session, error)
		SaveSession(*http.Request, http.ResponseWriter, *sessions.Session) error
	}
	Dependencies struct {
		Auth          AuthService
		Sessions      SessionManager
		GenerateID    func() (string, error)
		Now           func() time.Time
		ClientIP      func(*http.Request) string
		SecureCookies func() bool
		WriteError    func(http.ResponseWriter, int, string, string)
		Log           func(string, map[string]any)
		LogError      func(string, map[string]any)
	}
)

func response(email string, authenticated bool, sessionID string) SessionResponse {
	out := SessionResponse{Authenticated: authenticated, ResetEnabled: true}
	if authenticated && email != "" {
		out.Email = email
		out.SessionID = strings.TrimSpace(sessionID)
	}
	return out
}

func writeResponse(w http.ResponseWriter, status int, value SessionResponse, deps Dependencies, event string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		deps.LogError(event, map[string]any{"error": err.Error()})
	}
}

func Login(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		hash, err := deps.Auth.PasswordHash(r.Context(), request.Email)
		if err == sql.ErrNoRows {
			deps.Log("login_invalid_email", map[string]any{"level": "warn", "email": request.Email})
			deps.WriteError(w, http.StatusUnauthorized, "Invalid credentials", "unauthorized")
			return
		}
		if err != nil {
			deps.LogError("login_db_error", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Unable to verify credentials. Please try again.", "server_error")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(request.Password)); err != nil {
			deps.Log("login_invalid_password", map[string]any{"level": "warn", "email": request.Email})
			deps.WriteError(w, http.StatusUnauthorized, "Invalid credentials", "unauthorized")
			return
		}
		if err := deps.Auth.UpdateLastLogin(r.Context(), request.Email); err != nil {
			deps.LogError("last_login_update_failed", map[string]any{"error": err.Error(), "email": request.Email})
		}
		id, err := deps.GenerateID()
		if err != nil {
			deps.LogError("session_id_generation_failed", map[string]any{"error": err.Error()})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to create session. Please try again.", "server_error")
			return
		}
		if err := deps.Auth.CreateSession(r.Context(), id, request.Email, deps.Now().Add(7*24*time.Hour), deps.ClientIP(r), r.UserAgent()); err != nil {
			deps.LogError("admin_session_create_failed", map[string]any{"error": err.Error(), "email": request.Email})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to create session. Please try again.", "server_error")
			return
		}
		session, _ := deps.Sessions.GetSession(r, sessionName)
		session.Values["email"] = request.Email
		session.Values["session_id"] = id
		session.Options.HttpOnly = true
		session.Options.Secure = deps.SecureCookies()
		session.Options.MaxAge = 86400 * 7
		session.Options.Path = "/"
		session.Options.SameSite = http.SameSiteLaxMode
		if err := deps.Sessions.SaveSession(r, w, session); err != nil {
			deps.LogError("session_save_error", map[string]any{"error": err.Error()})
			if cleanupErr := deps.Auth.DeleteSession(r.Context(), id); cleanupErr != nil {
				deps.LogError("session_cleanup_after_save_failure", map[string]any{"session_id": id, "error": cleanupErr.Error()})
			}
			deps.WriteError(w, http.StatusInternalServerError, "Failed to create session. Please try again.", "server_error")
			return
		}
		deps.Log("admin_login_success", map[string]any{"level": "info", "email": request.Email})
		writeResponse(w, http.StatusOK, response(request.Email, true, id), deps, "login_response_encode_failed")
	}
}

func Logout(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := deps.Sessions.GetSession(r, sessionName)
		email := session.Values["email"]
		id, _ := session.Values["session_id"].(string)
		if id != "" {
			if err := deps.Auth.DeleteSession(r.Context(), id); err != nil {
				deps.LogError("admin_session_delete_failed", map[string]any{"error": err.Error(), "session_id": id})
			}
		}
		session.Options.MaxAge = -1
		if err := deps.Sessions.SaveSession(r, w, session); err != nil {
			deps.LogError("admin_session_save_failed", map[string]any{"error": err.Error()})
		}
		deps.Log("admin_logout", map[string]any{"level": "info", "email": email})
		w.WriteHeader(http.StatusNoContent)
	}
}

func Session(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := deps.Sessions.GetSession(r, sessionName)
		email, ok := session.Values["email"].(string)
		if !ok || email == "" {
			writeResponse(w, http.StatusUnauthorized, response("", false, ""), deps, "session_response_encode_failed")
			return
		}
		id, _ := session.Values["session_id"].(string)
		if id != "" {
			expiry, err := deps.Auth.SessionExpiry(r.Context(), id, email)
			if err == sql.ErrNoRows || (err == nil && deps.Now().After(expiry)) {
				session.Options.MaxAge = -1
				if saveErr := deps.Sessions.SaveSession(r, w, session); saveErr != nil {
					deps.LogError("session_save_failed_on_expiry", map[string]any{"error": saveErr.Error()})
				}
				writeResponse(w, http.StatusUnauthorized, response("", false, ""), deps, "session_response_encode_failed")
				return
			} else if err != nil {
				deps.LogError("session_lookup_failed", map[string]any{"error": err.Error()})
			} else if err := deps.Auth.TouchSession(r.Context(), id); err != nil {
				deps.LogError("session_activity_update_failed", map[string]any{"error": err.Error(), "session_id": id})
			}
		}
		writeResponse(w, http.StatusOK, response(email, true, id), deps, "session_response_encode_failed")
	}
}
