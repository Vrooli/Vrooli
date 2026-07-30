package administration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
)

// UserAuthService is the application boundary for browser authentication.
type UserAuthService interface {
	RequestMagicLink(context.Context, string, string, string) error
	VerifyMagicLink(context.Context, string, string, string) (*admin.TokenPair, *admin.User, error)
	RefreshTokens(context.Context, string) (*admin.TokenPair, error)
	Logout(context.Context, string) error
	GetUserByID(context.Context, string) (*admin.User, error)
}

// EmailRateLimiter is intentionally narrow so transport does not depend on a
// particular rate-limiting implementation.
type EmailRateLimiter interface{ Allow(string) bool }

type UserAuthDependencies struct {
	Service       UserAuthService
	RateLimiter   EmailRateLimiter
	ClientIP      func(*http.Request) string
	SessionID     func(context.Context) string
	UserID        func(context.Context) string
	ResolveSecret func(string) string
	SecureCookies func() bool
	Now           func() time.Time
	WriteError    func(http.ResponseWriter, int, string, string)
	Log           func(string, map[string]any)
	LogError      func(string, map[string]any)
}

type (
	MagicLinkRequest struct {
		Email string `json:"email"`
	}
	MagicLinkResponse struct {
		Message string `json:"message"`
	}
	TokenRefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
)

func RequestMagicLink(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request MagicLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		email := strings.TrimSpace(strings.ToLower(request.Email))
		if email == "" || !strings.Contains(email, "@") {
			deps.WriteError(w, http.StatusBadRequest, "Valid email address is required", "validation")
			return
		}
		if deps.RateLimiter != nil && !deps.RateLimiter.Allow(email) {
			deps.Log("magic_link_rate_limited", map[string]any{"level": "warn", "email": email})
			deps.WriteError(w, http.StatusTooManyRequests, "Too many login attempts. Please try again later.", "rate_limited")
			return
		}
		if err := deps.Service.RequestMagicLink(r.Context(), email, deps.ClientIP(r), r.Header.Get("User-Agent")); err != nil {
			// Deliberately return success below: this endpoint must not enumerate users.
			deps.LogError("magic_link_request_failed", map[string]any{"error": err.Error(), "email": email})
		}
		writeJSON(w, MagicLinkResponse{Message: "Check your email for a login link"}, deps, "encode_response_failed")
	}
}

func VerifyMagicLink(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			deps.WriteError(w, http.StatusBadRequest, "Token is required", "validation")
			return
		}
		pair, user, err := deps.Service.VerifyMagicLink(r.Context(), token, deps.ClientIP(r), r.Header.Get("User-Agent"))
		if err != nil {
			message, status := authError(err, "This login link has expired. Please request a new one.", "This login link has already been used. Please request a new one.", "Invalid login link. Please request a new one.", "Failed to verify login link. Please try again.")
			if status == http.StatusInternalServerError {
				deps.LogError("magic_link_verify_failed", map[string]any{"error": err.Error()})
			}
			deps.WriteError(w, status, message, "unauthorized")
			return
		}
		SetAuthCookies(w, pair, deps.SecureCookies(), deps.Now())
		if redirectURL := deps.ResolveSecret("AUTH_SUCCESS_REDIRECT_URL"); redirectURL != "" {
			RedirectWithTokens(w, r, redirectURL, pair, deps)
			return
		}
		writeJSON(w, tokenResponse(pair, user), deps, "encode_response_failed")
	}
}

func RefreshTokens(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request TokenRefreshRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		refreshToken := request.RefreshToken
		if refreshToken == "" {
			if cookie, err := r.Cookie("refresh_token"); err == nil {
				refreshToken = cookie.Value
			}
		}
		if refreshToken == "" {
			deps.WriteError(w, http.StatusBadRequest, "Refresh token is required", "validation")
			return
		}
		pair, err := deps.Service.RefreshTokens(r.Context(), refreshToken)
		if err != nil {
			message, status := authError(err, "Session has expired. Please log in again.", "Session has been revoked. Please log in again.", "Invalid refresh token. Please log in again.", "Failed to refresh session. Please log in again.")
			if status == http.StatusInternalServerError {
				deps.LogError("token_refresh_failed", map[string]any{"error": err.Error()})
			}
			ClearAuthCookies(w, deps.SecureCookies())
			deps.WriteError(w, status, message, "unauthorized")
			return
		}
		SetAuthCookies(w, pair, deps.SecureCookies(), deps.Now())
		writeJSON(w, tokenResponse(pair, nil), deps, "encode_response_failed")
	}
}

// LogoutUser terminates an end-user JWT session. Its explicit domain name keeps
// it distinct from the administrator-session Logout handler in this package.
func LogoutUser(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := deps.SessionID(r.Context())
		if sessionID == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Not authenticated", "unauthorized")
			return
		}
		if err := deps.Service.Logout(r.Context(), sessionID); err != nil {
			deps.LogError("user_logout_failed", map[string]any{"error": err.Error(), "session_id": sessionID})
		}
		deps.Log("user_logout", map[string]any{"session_id": sessionID})
		ClearAuthCookies(w, deps.SecureCookies())
		w.WriteHeader(http.StatusNoContent)
	}
}

func Me(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := deps.UserID(r.Context())
		if userID == "" {
			deps.WriteError(w, http.StatusUnauthorized, "Not authenticated", "unauthorized")
			return
		}
		user, err := deps.Service.GetUserByID(r.Context(), userID)
		if err != nil {
			deps.LogError("auth_me_get_user_failed", map[string]any{"error": err.Error(), "user_id": userID})
			deps.WriteError(w, http.StatusInternalServerError, "Failed to retrieve user information", "server_error")
			return
		}
		writeJSON(w, map[string]any{"user": map[string]any{"id": user.ID, "email": user.Email, "email_verified": user.EmailVerified, "stripe_customer_id": user.StripeCustomerID, "created_at": user.CreatedAt.Format(time.RFC3339), "last_login_at": FormatNullableTime(user.LastLoginAt)}}, deps, "encode_response_failed")
	}
}

func SetAuthCookies(w http.ResponseWriter, pair *admin.TokenPair, secure bool, now time.Time) {
	// #nosec G124 -- secure is explicitly derived from the deployment's HTTPS policy;
	// forcing it in local HTTP development would make the authentication flow unusable.
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: pair.AccessToken, Path: "/", Expires: pair.ExpiresAt, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	// #nosec G124 -- see the deployment-policy rationale above.
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: pair.RefreshToken, Path: "/api/v1/auth", Expires: now.Add(7 * 24 * time.Hour), HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func ClearAuthCookies(w http.ResponseWriter, secure bool) {
	// #nosec G124 -- deletion must exactly match the deployment-selected Secure attribute.
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	// #nosec G124 -- deletion must exactly match the deployment-selected Secure attribute.
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/api/v1/auth", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}

func RedirectWithTokens(w http.ResponseWriter, r *http.Request, redirectURL string, pair *admin.TokenPair, deps UserAuthDependencies) {
	u, err := url.Parse(redirectURL)
	if err != nil {
		deps.LogError("invalid_redirect_url", map[string]any{"error": err.Error(), "redirect_url": redirectURL})
		writeJSON(w, tokenResponse(pair, nil), deps, "encode_response_failed")
		return
	}
	fragment := url.Values{}
	fragment.Set("access_token", pair.AccessToken)
	fragment.Set("refresh_token", pair.RefreshToken)
	fragment.Set("expires_at", pair.ExpiresAt.Format(time.RFC3339))
	fragment.Set("token_type", pair.TokenType)
	u.Fragment = fragment.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func FormatNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(time.RFC3339)
}

func authError(err error, expired, used, invalid, internal string) (string, int) {
	switch {
	case errors.Is(err, admin.ErrTokenExpired):
		return expired, http.StatusUnauthorized
	case errors.Is(err, admin.ErrTokenUsed), errors.Is(err, admin.ErrSessionRevoked):
		return used, http.StatusUnauthorized
	case errors.Is(err, admin.ErrTokenInvalid):
		return invalid, http.StatusUnauthorized
	default:
		return internal, http.StatusInternalServerError
	}
}

func tokenResponse(pair *admin.TokenPair, user *admin.User) map[string]any {
	out := map[string]any{"access_token": pair.AccessToken, "refresh_token": pair.RefreshToken, "expires_at": pair.ExpiresAt.Format(time.RFC3339), "token_type": pair.TokenType}
	if user != nil {
		out["user"] = map[string]any{"id": user.ID, "email": user.Email, "email_verified": user.EmailVerified}
	}
	return out
}

func writeJSON(w http.ResponseWriter, value any, deps UserAuthDependencies, event string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		deps.LogError(event, map[string]any{"error": err.Error()})
	}
}
