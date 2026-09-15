// Package administration owns administrator-facing HTTP and Connect transport.
package administration

import (
	"context"
	"database/sql"
	"errors"
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

// SessionError is a transport-neutral authentication failure. HTTP and Connect
// adapters map it to their native status/error formats without reimplementing
// credential or session policy.
type SessionError struct {
	Status  int
	Message string
	Kind    string
}

func (e *SessionError) Error() string { return e.Message }

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

// LoginSession applies credential validation and writes the session cookie to
// the supplied response writer for the generated Connect transport.
func LoginSession(r *http.Request, w http.ResponseWriter, request LoginRequest, deps Dependencies) (SessionResponse, *SessionError) {
	hash, err := deps.Auth.PasswordHash(r.Context(), request.Email)
	if errors.Is(err, sql.ErrNoRows) {
		deps.Log("login_invalid_email", map[string]any{"level": "warn", "email": request.Email})
		return SessionResponse{}, &SessionError{Status: http.StatusUnauthorized, Message: "Invalid credentials", Kind: "unauthorized"}
	}
	if err != nil {
		deps.LogError("login_db_error", map[string]any{"error": err.Error()})
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Unable to verify credentials. Please try again.", Kind: "server_error"}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(request.Password)); err != nil {
		deps.Log("login_invalid_password", map[string]any{"level": "warn", "email": request.Email})
		return SessionResponse{}, &SessionError{Status: http.StatusUnauthorized, Message: "Invalid credentials", Kind: "unauthorized"}
	}
	if err := deps.Auth.UpdateLastLogin(r.Context(), request.Email); err != nil {
		deps.LogError("last_login_update_failed", map[string]any{"error": err.Error(), "email": request.Email})
	}
	id, err := deps.GenerateID()
	if err != nil {
		deps.LogError("session_id_generation_failed", map[string]any{"error": err.Error()})
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Failed to create session. Please try again.", Kind: "server_error"}
	}
	if err := deps.Auth.CreateSession(r.Context(), id, request.Email, deps.Now().Add(7*24*time.Hour), deps.ClientIP(r), r.UserAgent()); err != nil {
		deps.LogError("admin_session_create_failed", map[string]any{"error": err.Error(), "email": request.Email})
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Failed to create session. Please try again.", Kind: "server_error"}
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
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Failed to create session. Please try again.", Kind: "server_error"}
	}
	deps.Log("admin_login_success", map[string]any{"level": "info", "email": request.Email})
	return response(request.Email, true, id), nil
}

// LogoutSession revokes server state and expires the browser cookie.
func LogoutSession(r *http.Request, w http.ResponseWriter, deps Dependencies) {
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
}

// ReadSession validates the request cookie against persisted session state.
func ReadSession(r *http.Request, w http.ResponseWriter, deps Dependencies) (SessionResponse, bool) {
	session, _ := deps.Sessions.GetSession(r, sessionName)
	email, ok := session.Values["email"].(string)
	if !ok || email == "" {
		return response("", false, ""), false
	}
	id, _ := session.Values["session_id"].(string)
	if id != "" {
		expiry, err := deps.Auth.SessionExpiry(r.Context(), id, email)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && deps.Now().After(expiry)) {
			session.Options.MaxAge = -1
			if saveErr := deps.Sessions.SaveSession(r, w, session); saveErr != nil {
				deps.LogError("session_save_failed_on_expiry", map[string]any{"error": saveErr.Error()})
			}
			return response("", false, ""), false
		} else if err != nil {
			deps.LogError("session_lookup_failed", map[string]any{"error": err.Error()})
		} else if err := deps.Auth.TouchSession(r.Context(), id); err != nil {
			deps.LogError("session_activity_update_failed", map[string]any{"error": err.Error(), "session_id": id})
		}
	}
	return response(email, true, id), true
}
