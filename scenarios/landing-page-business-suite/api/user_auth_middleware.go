package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"

	userauthhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/logx"
)

// trustedProxyCIDRs holds the parsed CIDR ranges for trusted proxies.
// These are loaded once at startup from TRUSTED_PROXY_CIDRS environment variable.
var (
	trustedProxyCIDRs  []*net.IPNet
	trustedProxiesOnce sync.Once
)

// initTrustedProxies parses the TRUSTED_PROXY_CIDRS environment variable.
// Format: comma-separated CIDR ranges, e.g., "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
// Common values:
//   - "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16" for private networks
//   - "127.0.0.1/32" for localhost only
//   - Empty string means no proxies are trusted (X-Forwarded-For always ignored)
func initTrustedProxies() {
	trustedProxiesOnce.Do(func() {
		cidrsEnv := strings.TrimSpace(envx.Get("TRUSTED_PROXY_CIDRS"))
		if cidrsEnv == "" {
			// Browsers reach this API through the scenario's own UI server on
			// the loopback interface, which appends the real peer to
			// X-Forwarded-For. Without trusting loopback every visitor would
			// share one address and per-client limits would become global.
			// Remote peers are still never trusted.
			cidrsEnv = "127.0.0.0/8,::1/128"
			logx.Info("trusted_proxies_default_loopback", map[string]interface{}{
				"level":   "info",
				"message": "TRUSTED_PROXY_CIDRS not set; trusting only the local UI proxy on loopback",
			})
		}

		cidrs := strings.Split(cidrsEnv, ",")
		for _, cidr := range cidrs {
			cidr = strings.TrimSpace(cidr)
			if cidr == "" {
				continue
			}
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				logx.Error("trusted_proxy_cidr_parse_error", map[string]interface{}{
					"cidr":  cidr,
					"error": err.Error(),
				})
				continue
			}
			trustedProxyCIDRs = append(trustedProxyCIDRs, network)
		}

		if len(trustedProxyCIDRs) > 0 {
			logx.Info("trusted_proxies_configured", map[string]interface{}{
				"level": "info",
				"count": len(trustedProxyCIDRs),
			})
		}
	})
}

// isIPFromTrustedProxy checks if the given IP address is from a trusted proxy.
func isIPFromTrustedProxy(ipStr string) bool {
	initTrustedProxies()

	if len(trustedProxyCIDRs) == 0 {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, network := range trustedProxyCIDRs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// validateIPFormat checks if the given string is a valid IP address (IPv4 or IPv6).
func validateIPFormat(ip string) bool {
	return net.ParseIP(ip) != nil
}

// extractIPFromRemoteAddr extracts the IP address from r.RemoteAddr, stripping any port.
func extractIPFromRemoteAddr(remoteAddr string) string {
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(remoteAddr, "[") {
		if idx := strings.LastIndex(remoteAddr, "]:"); idx != -1 {
			return remoteAddr[1:idx]
		}
		return strings.Trim(remoteAddr, "[]")
	}
	// Handle IPv4 addresses
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}

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
	accessName, _, _ := userauthhttp.AuthCookieNames(isSecureCookiesEnabled())
	if cookie, err := r.Cookie(accessName); err == nil {
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
			if errors.Is(err, administration.ErrTokenExpired) {
				msg = "Token has expired. Please refresh your session."
			} else {
				msg = "Invalid or expired token"
			}
			writeJSONError(w, http.StatusUnauthorized, msg, ApiErrorTypeUnauthorized)
			return
		}
		if err := s.userAuthService.ValidateSession(r.Context(), claims.SessionID); err != nil {
			message := "Session has been revoked. Please log in again."
			reason := "session_revoked"
			if errors.Is(err, administration.ErrSessionExpired) {
				message = "Session has expired. Please log in again."
				reason = "session_expired"
			}
			writeJSONErrorReason(w, http.StatusUnauthorized, message, ApiErrorTypeUnauthorized, reason)
			return
		}

		// Inject claims into context
		ctx := context.WithValue(r.Context(), userClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// requireRecentUserAuth protects high-impact customer operations with a
// recent sign-in or explicit reauthentication on the current session.
func (s *Server) requireRecentUserAuth(next http.HandlerFunc) http.HandlerFunc {
	return s.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := getUserClaims(r.Context())
		if !ok || claims == nil || s.accountSecurityService == nil {
			w.Header().Set("X-Lpbs-Auth-Reason", "reauthentication_required")
			writeJSONErrorReason(w, http.StatusForbidden, "Recent reauthentication is required", ApiErrorTypeForbidden, "reauthentication_required")
			return
		}
		recent, err := s.accountSecurityService.IsRecent(r.Context(), claims.SessionID)
		if err != nil || !recent {
			w.Header().Set("X-Lpbs-Auth-Reason", "reauthentication_required")
			writeJSONErrorReason(w, http.StatusForbidden, "Recent reauthentication is required", ApiErrorTypeForbidden, "reauthentication_required")
			return
		}
		next(w, r)
	})
}

// optionalUserAuth is middleware that extracts user claims if present but doesn't require them.
// Useful for endpoints that behave differently for authenticated vs anonymous users.
func (s *Server) optionalUserAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractBearerToken(r)

		// If we have a token, try to validate it
		if tokenString != "" {
			if claims, err := s.userAuthService.ValidateAccessToken(tokenString); err == nil {
				if err := s.userAuthService.ValidateSession(r.Context(), claims.SessionID); err != nil {
					next(w, r)
					return
				}
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
func getUserClaims(ctx context.Context) (*administration.UserClaims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(*administration.UserClaims)
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
//
// X-Forwarded-For is walked from the right: each trusted proxy appends the
// peer it saw, so the first untrusted address from the right is the client.
// Entries to its left were supplied by that client and are never trusted.
// Headers are only consulted when the direct connection is a trusted proxy.
func getClientIP(r *http.Request) string {
	initTrustedProxies()

	directIP := extractIPFromRemoteAddr(r.RemoteAddr)
	if !isIPFromTrustedProxy(directIP) {
		if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" {
			logx.Info("xff_untrusted_proxy_ignored", map[string]interface{}{
				"level":     "warn",
				"direct_ip": directIP,
				"message":   "forwarding headers ignored - connection not from trusted proxy",
				"security":  true,
			})
		}
		return directIP
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		hops := strings.Split(xff, ",")
		origin, wellFormed := "", true
		for index := len(hops) - 1; index >= 0; index-- {
			hop := strings.TrimSpace(hops[index])
			if !validateIPFormat(hop) {
				wellFormed = false
				logx.Info("xff_invalid_ip_format", map[string]interface{}{"level": "warn", "direct_ip": directIP, "security": true})
				break
			}
			if !isIPFromTrustedProxy(hop) {
				return hop
			}
			origin = hop
		}
		// Every hop was a trusted proxy, so the leftmost one is the origin.
		if wellFormed && origin != "" {
			return origin
		}
	}

	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		if validateIPFormat(xri) {
			return xri
		}
		logx.Info("xrealip_invalid_ip_format", map[string]interface{}{
			"level":     "warn",
			"direct_ip": directIP,
			"security":  true,
		})
	}

	return directIP
}
