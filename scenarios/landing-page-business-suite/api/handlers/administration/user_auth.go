package administration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
)

// UserAuthService is the application boundary for browser authentication.
type UserAuthService interface {
	RequestSignIn(context.Context, admin.SignInRequest) (*admin.SignInStarted, error)
	PreviewSignIn(context.Context, string, string) (*admin.SignInPreview, error)
	VerifySignIn(context.Context, admin.SignInVerification) (*admin.SignInResult, error)
	RefreshTokens(context.Context, string) (*admin.TokenPair, error)
	Logout(context.Context, string) error
	GetUserByID(context.Context, string) (*admin.User, error)
}

// EmailRateLimiter is intentionally narrow so transport does not depend on a
// particular rate-limiting implementation.
type EmailRateLimiter interface{ Allow(string) bool }

// SignInThrottle bounds sign-in requests and code guesses durably.
type SignInThrottle interface {
	Allow(context.Context, string, admin.ThrottleRule) (bool, error)
}

type UserAuthDependencies struct {
	Service       UserAuthService
	RateLimiter   EmailRateLimiter
	Throttle      SignInThrottle
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
		Email          string          `json:"email"`
		BrowserBinding string          `json:"browser_binding,omitempty"`
		Context        json.RawMessage `json:"context,omitempty"`
	}
	MagicLinkResponse struct {
		Message   string `json:"message"`
		ExpiresAt string `json:"expires_at,omitempty"`
	}
	SignInTokenRequest struct {
		Token          string `json:"token"`
		BrowserBinding string `json:"browser_binding,omitempty"`
	}
	SignInCodeRequest struct {
		Email          string `json:"email"`
		Code           string `json:"code"`
		BrowserBinding string `json:"browser_binding"`
	}
	TokenRefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
)

// Code guesses are bounded per address so the six-digit space cannot be
// searched: five wrong codes per fifteen minutes.
var signInCodeGuesses = admin.ThrottleRule{Limit: 5, Window: 15 * time.Minute}

// Machine-readable reasons carried beside the shared error_type vocabulary.
const (
	reasonTokenExpired        = "token_expired"
	reasonTokenUsed           = "token_used"
	reasonTokenInvalid        = "token_invalid"
	reasonCodeInvalid         = "code_invalid"
	reasonRateLimited         = "rate_limited"
	reasonDeliveryUnavailable = "delivery_unavailable"
)

// RequestMagicLink starts passwordless sign-in. Every well-formed address may
// sign up, so the response truthfully reports delivery failures instead of
// pretending an email is on its way.
func RequestMagicLink(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request MagicLinkRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		email := admin.NormalizeEmail(request.Email)
		if !admin.LooksLikeEmail(email) {
			deps.WriteError(w, http.StatusBadRequest, "Valid email address is required", "validation")
			return
		}
		ip := deps.ClientIP(r)
		if !allowSignIn(r.Context(), deps, email, ip) {
			deps.Log("magic_link_rate_limited", map[string]any{"level": "warn", "email": email})
			writeAuthError(w, http.StatusTooManyRequests, "Too many sign-in requests. Please wait a few minutes and try again.", "rate_limited", reasonRateLimited, 15*time.Minute)
			return
		}
		started, err := deps.Service.RequestSignIn(r.Context(), admin.SignInRequest{
			Email: email, IPAddress: ip, UserAgent: r.Header.Get("User-Agent"),
			BrowserBinding: request.BrowserBinding, Context: request.Context,
		})
		switch {
		case err == nil:
		case errors.Is(err, admin.ErrContextInvalid):
			deps.WriteError(w, http.StatusBadRequest, "This sign-in request came from an unsupported app. Start again from the app.", "validation")
			return
		case errors.Is(err, admin.ErrDeliveryUnavailable):
			deps.LogError("magic_link_delivery_unavailable", map[string]any{"error": err.Error(), "email": email})
			writeAuthError(w, http.StatusServiceUnavailable, "We couldn't send your sign-in email right now. Please try again in a few minutes.", "server_error", reasonDeliveryUnavailable, time.Minute)
			return
		default:
			deps.LogError("magic_link_request_failed", map[string]any{"error": err.Error(), "email": email})
			deps.WriteError(w, http.StatusInternalServerError, "We couldn't start sign-in. Please try again.", "server_error")
			return
		}
		writeJSON(w, MagicLinkResponse{Message: "Check your email for a sign-in code and link", ExpiresAt: started.ExpiresAt.UTC().Format(time.RFC3339)}, deps, "encode_response_failed")
	}
}

// PreviewSignIn describes a link without consuming it so the page can ask the
// person to confirm; automated link scanners never complete a sign-in.
func PreviewSignIn(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request SignInTokenRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&request); err != nil || strings.TrimSpace(request.Token) == "" {
			writeAuthError(w, http.StatusBadRequest, "This sign-in link is incomplete. Request a new one.", "validation", reasonTokenInvalid, 0)
			return
		}
		preview, err := deps.Service.PreviewSignIn(r.Context(), request.Token, request.BrowserBinding)
		if err != nil {
			writeSignInFailure(w, deps, err, "sign_in_preview_failed")
			return
		}
		writeJSON(w, preview, deps, "encode_response_failed")
	}
}

// VerifyMagicLink consumes a link token after the person confirms.
func VerifyMagicLink(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request SignInTokenRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&request); err != nil || strings.TrimSpace(request.Token) == "" {
			writeAuthError(w, http.StatusBadRequest, "This sign-in link is incomplete. Request a new one.", "validation", reasonTokenInvalid, 0)
			return
		}
		result, err := deps.Service.VerifySignIn(r.Context(), admin.SignInVerification{
			Token: request.Token, BrowserBinding: request.BrowserBinding,
			IPAddress: deps.ClientIP(r), UserAgent: r.Header.Get("User-Agent"),
		})
		if err != nil {
			writeSignInFailure(w, deps, err, "magic_link_verify_failed")
			return
		}
		SetAuthCookies(w, result.Tokens, deps.SecureCookies(), deps.Now())
		writeJSON(w, signInResponse(result), deps, "encode_response_failed")
	}
}

// VerifySignInCode completes sign-in with the emailed code from the browser
// that requested it.
func VerifySignInCode(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request SignInCodeRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&request); err != nil {
			deps.WriteError(w, http.StatusBadRequest, "Invalid request body", "validation")
			return
		}
		email := admin.NormalizeEmail(request.Email)
		code := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, request.Code)
		if !admin.LooksLikeEmail(email) || len(code) != 6 || strings.TrimSpace(request.BrowserBinding) == "" {
			writeAuthError(w, http.StatusBadRequest, "Enter the 6-digit code from your email.", "validation", reasonCodeInvalid, 0)
			return
		}
		if !allowThrottle(r.Context(), deps, admin.ThrottleBucket("sign-in-code", email), signInCodeGuesses) {
			writeAuthError(w, http.StatusTooManyRequests, "Too many incorrect codes. Request a new code in a few minutes.", "rate_limited", reasonRateLimited, signInCodeGuesses.Window)
			return
		}
		result, err := deps.Service.VerifySignIn(r.Context(), admin.SignInVerification{
			Email: email, Code: code, BrowserBinding: request.BrowserBinding,
			IPAddress: deps.ClientIP(r), UserAgent: r.Header.Get("User-Agent"),
		})
		if err != nil {
			writeSignInFailure(w, deps, err, "sign_in_code_verify_failed")
			return
		}
		SetAuthCookies(w, result.Tokens, deps.SecureCookies(), deps.Now())
		writeJSON(w, signInResponse(result), deps, "encode_response_failed")
	}
}

func allowSignIn(ctx context.Context, deps UserAuthDependencies, email, ip string) bool {
	if deps.RateLimiter != nil && !deps.RateLimiter.Allow(email) {
		return false
	}
	if !allowThrottle(ctx, deps, admin.ThrottleBucket("sign-in-email", email), admin.SignInPerEmail) {
		return false
	}
	if ip != "" && !allowThrottle(ctx, deps, admin.ThrottleBucket("sign-in-ip", ip), admin.SignInPerIP) {
		return false
	}
	return true
}

func allowThrottle(ctx context.Context, deps UserAuthDependencies, bucket string, rule admin.ThrottleRule) bool {
	if deps.Throttle == nil {
		return true
	}
	allowed, err := deps.Throttle.Allow(ctx, bucket, rule)
	if err != nil && deps.LogError != nil {
		deps.LogError("auth_throttle_degraded", map[string]any{"error": err.Error()})
	}
	return allowed
}

func writeSignInFailure(w http.ResponseWriter, deps UserAuthDependencies, err error, event string) {
	switch {
	case errors.Is(err, admin.ErrTokenExpired):
		writeAuthError(w, http.StatusUnauthorized, "This sign-in link has expired. Request a new one.", "unauthorized", reasonTokenExpired, 0)
	case errors.Is(err, admin.ErrTokenUsed):
		writeAuthError(w, http.StatusUnauthorized, "This sign-in link was already used. Request a new one if you're not signed in.", "unauthorized", reasonTokenUsed, 0)
	case errors.Is(err, admin.ErrTokenInvalid):
		writeAuthError(w, http.StatusUnauthorized, "This sign-in link isn't valid. Request a new one.", "unauthorized", reasonTokenInvalid, 0)
	case errors.Is(err, admin.ErrCodeInvalid):
		writeAuthError(w, http.StatusUnauthorized, "That code isn't right or has expired. Check the latest email and try again.", "unauthorized", reasonCodeInvalid, 0)
	default:
		deps.LogError(event, map[string]any{"error": err.Error()})
		deps.WriteError(w, http.StatusInternalServerError, "We couldn't finish signing you in. Please try again.", "server_error")
	}
}

// writeAuthError keeps the shared error envelope and adds a stable reason the
// sign-in pages use to choose recovery actions.
func writeAuthError(w http.ResponseWriter, status int, message, errorType, reason string, retryAfter time.Duration) {
	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter/time.Second)))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	retryable := errorType == "rate_limited" || errorType == "server_error"
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message, "error_type": errorType, "reason": reason, "retryable": retryable})
}

func signInResponse(result *admin.SignInResult) map[string]any {
	out := tokenResponse(result.Tokens, result.User)
	out["same_browser"] = result.SameBrowser
	if len(result.Context) > 0 {
		out["context"] = result.Context
	}
	return out
}

func RefreshTokens(deps UserAuthDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request TokenRefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil && err != io.EOF {
			deps.WriteError(w, http.StatusBadRequest, "Malformed request body", "validation")
			return
		}
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
