// Package administration owns administrator-facing HTTP and Connect transport.
package administration

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	admin "landing-page-business-suite-api/internal/administration"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

const sessionName = "admin_session"

type (
	LoginRequest struct {
		Email             string `json:"email"`
		Password          string `json:"password"`
		TOTPCode          string `json:"totp_code,omitempty"`
		PasskeyAssertion  []byte `json:"passkey_assertion,omitempty"`
		PasskeyCeremonyID string `json:"passkey_ceremony_id,omitempty"`
	}
	SessionResponse struct {
		Email         string `json:"email,omitempty"`
		Authenticated bool   `json:"authenticated"`
		ResetEnabled  bool   `json:"reset_enabled"`
		SessionID     string `json:"session_id,omitempty"`
		Assurance     string `json:"assurance,omitempty"`
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

type AssuredSessionCreator interface {
	CreateSessionWithAssurance(context.Context, string, string, time.Time, string, string, string) error
}

type SessionStateReader interface {
	SessionState(context.Context, string, string) (admin.AdminSessionState, error)
}

type ReauthenticationMarker interface {
	MarkReauthenticated(context.Context, string, string) error
}
type (
	SessionManager interface {
		GetSession(*http.Request, string) (*sessions.Session, error)
		SaveSession(*http.Request, http.ResponseWriter, *sessions.Session) error
	}
	// AdminThrottle counts failed administrator sign-ins durably.
	AdminThrottle interface {
		Exceeded(context.Context, string, admin.ThrottleRule) (bool, error)
		Record(context.Context, string) error
		Reset(context.Context, string) error
	}
	// AdminSecondFactor verifies authenticator-app or recovery codes.
	AdminSecondFactor interface {
		Enabled(context.Context, string) (bool, error)
		Verify(context.Context, string, string) error
	}
	AdminPasskeyVerifier interface {
		VerifyLogin(context.Context, string, string, []byte, string, string) error
	}
	SecurityEvents interface {
		Record(context.Context, string, string, string, string, string, map[string]any) error
	}
	DeviceDetector interface {
		IsNewDevice(context.Context, string, string, string) (bool, error)
	}
	SecurityNotifier interface {
		Notify(context.Context, string, string, string) error
	}
	Dependencies struct {
		Auth           AuthService
		Throttle       AdminThrottle
		MFA            AdminSecondFactor
		Passkeys       AdminPasskeyVerifier
		Sessions       SessionManager
		GenerateID     func() (string, error)
		Now            func() time.Time
		ClientIP       func(*http.Request) string
		SecureCookies  func() bool
		WriteError     func(http.ResponseWriter, int, string, string)
		Log            func(string, map[string]any)
		LogError       func(string, map[string]any)
		SecurityEvents SecurityEvents
		DeviceDetector DeviceDetector
		Notifier       SecurityNotifier
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

// Admin login failure kinds that clients branch on.
const (
	LoginKindMFARequired = "mfa_required"
	LoginKindMFAInvalid  = "mfa_invalid"
	LoginKindLocked      = "rate_limited"
)

var (
	timingHashOnce sync.Once
	timingHash     []byte
)

// equalizeUnknownAdmin spends the same bcrypt work as a real comparison so
// response time does not reveal whether an administrator email exists.
func equalizeUnknownAdmin(password string) {
	timingHashOnce.Do(func() {
		timingHash, _ = bcrypt.GenerateFromPassword([]byte("lpbs-timing-equalizer"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(timingHash, []byte(password))
}

// LoginSession applies credential validation and writes the session cookie to
// the supplied response writer for the generated Connect transport.
func LoginSession(r *http.Request, w http.ResponseWriter, request LoginRequest, deps Dependencies) (SessionResponse, *SessionError) {
	request.Email = strings.TrimSpace(request.Email)
	ctx := r.Context()
	userBucket := admin.ThrottleBucket("admin-login-email", request.Email)
	ipBucket := ""
	if ip := strings.TrimSpace(deps.ClientIP(r)); ip != "" {
		ipBucket = admin.ThrottleBucket("admin-login-ip", ip)
	}
	locked := &SessionError{Status: http.StatusTooManyRequests, Message: "Too many failed sign-in attempts. Wait 15 minutes and try again.", Kind: LoginKindLocked}
	if deps.Throttle != nil {
		for _, check := range []struct {
			bucket string
			rule   admin.ThrottleRule
		}{{userBucket, admin.AdminFailuresPerUser}, {ipBucket, admin.AdminFailuresPerIP}} {
			if check.bucket == "" {
				continue
			}
			exceeded, err := deps.Throttle.Exceeded(ctx, check.bucket, check.rule)
			if err != nil {
				deps.LogError("admin_login_throttle_degraded", map[string]any{"error": err.Error()})
			}
			if exceeded {
				deps.Log("admin_login_locked", map[string]any{"level": "warn", "email": request.Email})
				return SessionResponse{}, locked
			}
		}
	}
	recordFailure := func() {
		if deps.Throttle == nil {
			return
		}
		for _, bucket := range []string{userBucket, ipBucket} {
			if bucket == "" {
				continue
			}
			if err := deps.Throttle.Record(ctx, bucket); err != nil {
				deps.LogError("admin_login_throttle_record_failed", map[string]any{"error": err.Error()})
			}
		}
	}
	invalid := &SessionError{Status: http.StatusUnauthorized, Message: "Invalid credentials", Kind: "unauthorized"}
	recordEvent := func(event, sessionID string, detail map[string]any) {
		if deps.SecurityEvents != nil {
			if err := deps.SecurityEvents.Record(ctx, event, request.Email, sessionID, deps.ClientIP(r), r.UserAgent(), detail); err != nil && deps.LogError != nil {
				deps.LogError("admin_security_event_record_failed", map[string]any{"event": event, "error": err.Error()})
			}
		}
	}

	hash, err := deps.Auth.PasswordHash(ctx, request.Email)
	if errors.Is(err, sql.ErrNoRows) {
		equalizeUnknownAdmin(request.Password)
		recordFailure()
		deps.Log("login_invalid_email", map[string]any{"level": "warn", "email": request.Email})
		recordEvent("login_failure", "", map[string]any{"reason": "unknown_email"})
		return SessionResponse{}, invalid
	}
	if err != nil {
		deps.LogError("login_db_error", map[string]any{"error": err.Error()})
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Unable to verify credentials. Please try again.", Kind: "server_error"}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(request.Password)); err != nil {
		recordFailure()
		deps.Log("login_invalid_password", map[string]any{"level": "warn", "email": request.Email})
		recordEvent("login_failure", "", map[string]any{"reason": "password"})
		return SessionResponse{}, invalid
	}
	assurance := "full"
	if deps.MFA != nil {
		enabled, err := deps.MFA.Enabled(ctx, request.Email)
		if err != nil {
			deps.LogError("admin_mfa_status_failed", map[string]any{"error": err.Error()})
			return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Unable to verify credentials. Please try again.", Kind: "server_error"}
		}
		if !enabled && AdminMFARequired() {
			assurance = "enrollment_only"
		}
		if enabled {
			if len(request.PasskeyAssertion) > 0 {
				if deps.Passkeys == nil || strings.TrimSpace(request.PasskeyCeremonyID) == "" || deps.Passkeys.VerifyLogin(ctx, request.Email, request.PasskeyCeremonyID, request.PasskeyAssertion, r.Header.Get("X-Lpbs-Browser-Binding"), r.UserAgent()) != nil {
					recordFailure()
					return SessionResponse{}, &SessionError{Status: http.StatusUnauthorized, Message: "That passkey was not accepted.", Kind: LoginKindMFAInvalid}
				}
			} else {
				if strings.TrimSpace(request.TOTPCode) == "" {
					return SessionResponse{}, &SessionError{Status: http.StatusPreconditionRequired, Message: "Enter the code from your authenticator app.", Kind: LoginKindMFARequired}
				}
				if err := deps.MFA.Verify(ctx, request.Email, request.TOTPCode); err != nil {
					if errors.Is(err, admin.ErrMFAUnavailable) {
						deps.LogError("admin_mfa_unavailable", map[string]any{"error": err.Error()})
						return SessionResponse{}, &SessionError{Status: http.StatusServiceUnavailable, Message: "Two-factor verification is unavailable. Check the server's credential authority.", Kind: "server_error"}
					}
					recordFailure()
					deps.Log("login_invalid_mfa", map[string]any{"level": "warn", "email": request.Email})
					recordEvent("login_failure", "", map[string]any{"reason": "mfa"})
					return SessionResponse{}, &SessionError{Status: http.StatusUnauthorized, Message: "That code isn't right. Try the current code from your app.", Kind: LoginKindMFAInvalid}
				}
				if len(strings.TrimSpace(request.TOTPCode)) != 6 {
					recordEvent("recovery_code_used", "", nil)
				}
			}
		}
	}
	if deps.Throttle != nil {
		if err := deps.Throttle.Reset(ctx, userBucket); err != nil {
			deps.LogError("admin_login_throttle_reset_failed", map[string]any{"error": err.Error()})
		}
	}
	if err := deps.Auth.UpdateLastLogin(r.Context(), request.Email); err != nil {
		deps.LogError("last_login_update_failed", map[string]any{"error": err.Error(), "email": request.Email})
	}
	id, err := deps.GenerateID()
	if err != nil {
		deps.LogError("session_id_generation_failed", map[string]any{"error": err.Error()})
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Failed to create session. Please try again.", Kind: "server_error"}
	}
	absoluteTTL := adminSessionAbsoluteTTL()
	var createErr error
	if creator, ok := deps.Auth.(AssuredSessionCreator); ok {
		createErr = creator.CreateSessionWithAssurance(r.Context(), id, request.Email, deps.Now().Add(absoluteTTL), deps.ClientIP(r), r.UserAgent(), assurance)
	} else {
		createErr = deps.Auth.CreateSession(r.Context(), id, request.Email, deps.Now().Add(absoluteTTL), deps.ClientIP(r), r.UserAgent())
	}
	if createErr != nil {
		deps.LogError("admin_session_create_failed", map[string]any{"error": createErr.Error(), "email": request.Email})
		return SessionResponse{}, &SessionError{Status: http.StatusInternalServerError, Message: "Failed to create session. Please try again.", Kind: "server_error"}
	}
	session, _ := deps.Sessions.GetSession(r, sessionName)
	session.Values["email"] = request.Email
	session.Values["session_id"] = id
	session.Options.HttpOnly = true
	session.Options.Secure = deps.SecureCookies()
	session.Options.MaxAge = int(absoluteTTL / time.Second)
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
	if deps.DeviceDetector != nil {
		if newDevice, detectErr := deps.DeviceDetector.IsNewDevice(ctx, request.Email, r.UserAgent(), deps.ClientIP(r)); detectErr == nil && newDevice {
			recordEvent("new_device_login", id, nil)
			if deps.Notifier != nil {
				if notifyErr := deps.Notifier.Notify(ctx, request.Email, "new_device_login", "A new administrator login was detected."); notifyErr != nil {
					recordEvent("new_device_login", id, map[string]any{"notify_failed": notifyErr.Error()})
				}
			}
		}
	}
	recordEvent("login_success", id, nil)
	result := response(request.Email, true, id)
	result.Assurance = assurance
	return result, nil
}

func AdminMFARequired() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_REQUIRE_MFA")))
	if value == "" {
		environment := strings.ToLower(strings.TrimSpace(os.Getenv("LPBS_ENVIRONMENT")))
		return environment == "production" || environment == "prod"
	}
	return value != "0" && value != "false" && value != "no"
}

func adminSessionAbsoluteTTL() time.Duration {
	const fallback = 7 * 24 * time.Hour
	raw := strings.TrimSpace(os.Getenv("LPBS_ADMIN_SESSION_ABSOLUTE_TTL"))
	if raw == "" {
		return fallback
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	if ttl < time.Hour {
		ttl = time.Hour
	}
	if ttl > 14*24*time.Hour {
		ttl = 14 * 24 * time.Hour
	}
	return ttl
}

func adminSessionIdleTTL() time.Duration {
	const fallback = 12 * time.Hour
	raw := strings.TrimSpace(os.Getenv("LPBS_ADMIN_SESSION_IDLE_TTL"))
	if raw == "" {
		return fallback
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	if ttl < time.Hour {
		ttl = time.Hour
	}
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}
	return ttl
}

// AdminSessionIdleTTL exposes the bounded idle policy to the control-plane
// middleware without duplicating environment parsing there.
func AdminSessionIdleTTL() time.Duration { return adminSessionIdleTTL() }

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
		if reader, ok := deps.Auth.(SessionStateReader); ok {
			state, err := reader.SessionState(r.Context(), id, email)
			if errors.Is(err, sql.ErrNoRows) || (err == nil && (deps.Now().After(state.ExpiresAt) || deps.Now().After(state.LastActivity.Add(adminSessionIdleTTL())))) {
				session.Options.MaxAge = -1
				_ = deps.Sessions.SaveSession(r, w, session)
				return response("", false, ""), false
			}
			if err != nil {
				return response("", false, ""), false
			}
			result := response(email, true, id)
			result.Assurance = state.Assurance
			if time.Since(state.LastActivity) >= 60*time.Second {
				_ = deps.Auth.TouchSession(r.Context(), id)
			}
			return result, true
		}
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
