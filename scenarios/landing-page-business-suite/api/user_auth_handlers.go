package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MagicLinkRequest is the request body for requesting a magic link.
type MagicLinkRequest struct {
	Email string `json:"email"`
}

// MagicLinkResponse is the response after requesting a magic link.
type MagicLinkResponse struct {
	Message string `json:"message"`
}

// TokenRefreshRequest is the request body for refreshing tokens.
type TokenRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthMeResponse is the response for the /auth/me endpoint.
type AuthMeResponse struct {
	User *User `json:"user"`
}

// handleMagicLinkRequest handles POST /api/v1/auth/magic-link
// Generates and sends a magic link to the provided email.
func handleMagicLinkRequest(authService *UserAuthService, rateLimiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MagicLinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		email := strings.TrimSpace(strings.ToLower(req.Email))
		if email == "" || !strings.Contains(email, "@") {
			writeJSONError(w, http.StatusBadRequest, "Valid email address is required", ApiErrorTypeValidation)
			return
		}

		// Apply rate limiting by email
		if rateLimiter != nil && !rateLimiter.Allow(email) {
			logStructured("magic_link_rate_limited", map[string]interface{}{
				"level": "warn",
				"email": email,
			})
			writeJSONError(w, http.StatusTooManyRequests,
				"Too many login attempts. Please try again later.",
				ApiErrorTypeRateLimited)
			return
		}

		// Get client info for logging/security
		ipAddress := getClientIP(r)
		userAgent := r.Header.Get("User-Agent")

		// Request magic link (always returns success to not reveal email existence)
		if err := authService.RequestMagicLink(r.Context(), email, ipAddress, userAgent); err != nil {
			logStructuredError("magic_link_request_failed", map[string]interface{}{
				"error": err.Error(),
				"email": email,
			})
			// Don't reveal the error to the user
		}

		// Always return success to prevent email enumeration
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(MagicLinkResponse{
			Message: "Check your email for a login link",
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleMagicLinkVerify handles GET /api/v1/auth/verify?token=xxx
// Verifies a magic link token and returns authentication tokens.
func handleMagicLinkVerify(authService *UserAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeJSONError(w, http.StatusBadRequest, "Token is required", ApiErrorTypeValidation)
			return
		}

		// Get client info
		ipAddress := getClientIP(r)
		userAgent := r.Header.Get("User-Agent")

		tokenPair, user, err := authService.VerifyMagicLink(r.Context(), token, ipAddress, userAgent)
		if err != nil {
			var msg string
			var status int

			switch {
			case errors.Is(err, ErrTokenExpired):
				msg = "This login link has expired. Please request a new one."
				status = http.StatusUnauthorized
			case errors.Is(err, ErrTokenUsed):
				msg = "This login link has already been used. Please request a new one."
				status = http.StatusUnauthorized
			case errors.Is(err, ErrTokenInvalid):
				msg = "Invalid login link. Please request a new one."
				status = http.StatusUnauthorized
			default:
				logStructuredError("magic_link_verify_failed", map[string]interface{}{
					"error": err.Error(),
				})
				msg = "Failed to verify login link. Please try again."
				status = http.StatusInternalServerError
			}

			writeJSONError(w, status, msg, ApiErrorTypeUnauthorized)
			return
		}

		// Set cookies for web clients
		setAuthCookies(w, tokenPair)

		// Check for redirect URL
		redirectURL := resolveSecret("AUTH_SUCCESS_REDIRECT_URL")
		if redirectURL != "" {
			// Redirect with tokens in URL fragment (for SPAs)
			redirectWithTokens(w, r, redirectURL, tokenPair)
			return
		}

		// Return token pair as JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt.Format(time.RFC3339),
			"token_type":    tokenPair.TokenType,
			"user": map[string]interface{}{
				"id":             user.ID,
				"email":          user.Email,
				"email_verified": user.EmailVerified,
			},
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleTokenRefresh handles POST /api/v1/auth/refresh
// Refreshes access and refresh tokens.
func handleTokenRefresh(authService *UserAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var refreshToken string

		// Try to get refresh token from request body
		var req TokenRefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}

		// Fall back to cookie
		if refreshToken == "" {
			if cookie, err := r.Cookie("refresh_token"); err == nil {
				refreshToken = cookie.Value
			}
		}

		if refreshToken == "" {
			writeJSONError(w, http.StatusBadRequest, "Refresh token is required", ApiErrorTypeValidation)
			return
		}

		tokenPair, err := authService.RefreshTokens(r.Context(), refreshToken)
		if err != nil {
			var msg string
			var status int

			switch {
			case errors.Is(err, ErrTokenExpired):
				msg = "Session has expired. Please log in again."
				status = http.StatusUnauthorized
			case errors.Is(err, ErrSessionRevoked):
				msg = "Session has been revoked. Please log in again."
				status = http.StatusUnauthorized
			case errors.Is(err, ErrTokenInvalid):
				msg = "Invalid refresh token. Please log in again."
				status = http.StatusUnauthorized
			default:
				logStructuredError("token_refresh_failed", map[string]interface{}{
					"error": err.Error(),
				})
				msg = "Failed to refresh session. Please log in again."
				status = http.StatusInternalServerError
			}

			// Clear cookies on refresh failure
			clearAuthCookies(w)
			writeJSONError(w, status, msg, ApiErrorTypeUnauthorized)
			return
		}

		// Update cookies
		setAuthCookies(w, tokenPair)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt.Format(time.RFC3339),
			"token_type":    tokenPair.TokenType,
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleUserLogout handles POST /api/v1/auth/logout
// Revokes the current session. Requires authentication.
func handleUserLogout(authService *UserAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionID(r.Context())
		if sessionID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Not authenticated", ApiErrorTypeUnauthorized)
			return
		}

		if err := authService.Logout(r.Context(), sessionID); err != nil {
			logStructuredError("user_logout_failed", map[string]interface{}{
				"error":      err.Error(),
				"session_id": sessionID,
			})
			// Don't reveal error - just log it
		}

		logStructured("user_logout", map[string]interface{}{
			"session_id": sessionID,
		})

		// Clear cookies
		clearAuthCookies(w)

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAuthMe handles GET /api/v1/auth/me
// Returns information about the currently authenticated user. Requires authentication.
func handleAuthMe(authService *UserAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r.Context())
		if userID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Not authenticated", ApiErrorTypeUnauthorized)
			return
		}

		user, err := authService.GetUserByID(r.Context(), userID)
		if err != nil {
			logStructuredError("auth_me_get_user_failed", map[string]interface{}{
				"error":   err.Error(),
				"user_id": userID,
			})
			writeJSONError(w, http.StatusInternalServerError,
				"Failed to retrieve user information", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"user": map[string]interface{}{
				"id":                 user.ID,
				"email":              user.Email,
				"email_verified":     user.EmailVerified,
				"stripe_customer_id": user.StripeCustomerID,
				"created_at":         user.CreatedAt.Format(time.RFC3339),
				"last_login_at":      formatNullableTime(user.LastLoginAt),
			},
		}); err != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// setAuthCookies sets HTTP-only cookies for access and refresh tokens.
func setAuthCookies(w http.ResponseWriter, tokenPair *TokenPair) {
	// Access token cookie (short-lived)
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		Expires:  tokenPair.ExpiresAt,
		HttpOnly: true,
		Secure:   isSecureContext(),
		SameSite: http.SameSiteLaxMode,
	})

	// Refresh token cookie (longer-lived)
	// Expires in 7 days from now
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenPair.RefreshToken,
		Path:     "/api/v1/auth", // Only sent to auth endpoints
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   isSecureContext(),
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookies removes authentication cookies.
func clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureContext(),
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureContext(),
		SameSite: http.SameSiteLaxMode,
	})
}

// redirectWithTokens redirects to the given URL with tokens in the fragment.
func redirectWithTokens(w http.ResponseWriter, r *http.Request, redirectURL string, tokenPair *TokenPair) {
	// Build redirect URL with tokens in fragment (client-side only, not sent to server)
	u, err := url.Parse(redirectURL)
	if err != nil {
		logStructuredError("invalid_redirect_url", map[string]interface{}{
			"error":        err.Error(),
			"redirect_url": redirectURL,
		})
		// Fall back to JSON response
		w.Header().Set("Content-Type", "application/json")
		if encErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  tokenPair.AccessToken,
			"refresh_token": tokenPair.RefreshToken,
			"expires_at":    tokenPair.ExpiresAt.Format(time.RFC3339),
			"token_type":    tokenPair.TokenType,
		}); encErr != nil {
			logStructuredError("encode_response_failed", map[string]interface{}{"error": encErr.Error()})
		}
		return
	}

	// Add tokens to fragment
	fragment := url.Values{}
	fragment.Set("access_token", tokenPair.AccessToken)
	fragment.Set("refresh_token", tokenPair.RefreshToken)
	fragment.Set("expires_at", tokenPair.ExpiresAt.Format(time.RFC3339))
	fragment.Set("token_type", tokenPair.TokenType)
	u.Fragment = fragment.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

// isSecureContext returns true if the app should use secure cookies.
func isSecureContext() bool {
	// Check if we're in production or have HTTPS configured
	baseURL := resolveSecret("AUTH_MAGIC_LINK_BASE_URL")
	return strings.HasPrefix(baseURL, "https://")
}

// formatNullableTime formats a nullable time pointer for JSON output.
func formatNullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
