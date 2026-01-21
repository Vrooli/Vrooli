package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// contextKey is used for context value keys to avoid collisions.
type contextKey string

const (
	// userClaimsKey is the context key for authenticated user claims.
	userClaimsKey contextKey = "user_claims"
)

// extractBearerToken extracts the JWT access token from the request.
// It checks the Authorization header first (Bearer token), then falls back to the access_token cookie.
// Returns empty string if no token is found.
func extractBearerToken(r *http.Request) string {
	// Try Authorization header first (preferred)
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Fall back to cookie
	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}

	return ""
}

// requireUserAuth is middleware that validates JWT tokens and injects claims into context.
// It checks for tokens in the Authorization header (Bearer token) or access_token cookie.
func (s *Server) requireUserAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractBearerToken(r)

		if tokenString == "" {
			writeJSONError(w, http.StatusUnauthorized,
				"Authentication required", ApiErrorTypeUnauthorized)
			return
		}

		claims, err := s.userAuthService.ValidateAccessToken(tokenString)
		if err != nil {
			var msg string
			if errors.Is(err, ErrTokenExpired) {
				msg = "Token has expired. Please refresh your session."
			} else {
				msg = "Invalid or expired token"
			}
			writeJSONError(w, http.StatusUnauthorized, msg, ApiErrorTypeUnauthorized)
			return
		}

		// Inject claims into context
		ctx := context.WithValue(r.Context(), userClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// optionalUserAuth is middleware that extracts user claims if present but doesn't require them.
// Useful for endpoints that behave differently for authenticated vs anonymous users.
func (s *Server) optionalUserAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractBearerToken(r)

		// If we have a token, try to validate it
		if tokenString != "" {
			if claims, err := s.userAuthService.ValidateAccessToken(tokenString); err == nil {
				ctx := context.WithValue(r.Context(), userClaimsKey, claims)
				next(w, r.WithContext(ctx))
				return
			}
		}

		// Continue without auth
		next(w, r)
	}
}

// getUserClaims retrieves user claims from the request context.
// Returns nil, false if not authenticated.
func getUserClaims(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*UserClaims)
	return claims, ok
}

// getUserEmail is a convenience function to get the authenticated user's email.
// Returns empty string if not authenticated.
func getUserEmail(ctx context.Context) string {
	claims, ok := getUserClaims(ctx)
	if !ok || claims == nil {
		return ""
	}
	return claims.Email
}

// getUserID is a convenience function to get the authenticated user's ID.
// Returns empty string if not authenticated.
func getUserID(ctx context.Context) string {
	claims, ok := getUserClaims(ctx)
	if !ok || claims == nil {
		return ""
	}
	return claims.UserID
}

// getSessionID is a convenience function to get the current session ID.
// Returns empty string if not authenticated.
func getSessionID(ctx context.Context) string {
	claims, ok := getUserClaims(ctx)
	if !ok || claims == nil {
		return ""
	}
	return claims.SessionID
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (strip port if present)
	addr := r.RemoteAddr
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]:"); idx != -1 {
			return addr[1:idx]
		}
		return strings.Trim(addr, "[]")
	}
	// Handle IPv4 addresses
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
