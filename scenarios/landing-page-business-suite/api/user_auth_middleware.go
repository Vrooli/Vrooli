package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"

	"landing-page-business-suite-api/internal/envx"
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
			// No trusted proxies configured - this is the secure default
			logStructured("trusted_proxies_not_configured", map[string]interface{}{
				"level":   "info",
				"message": "TRUSTED_PROXY_CIDRS not set; X-Forwarded-For headers will be ignored",
			})
			return
		}

		cidrs := strings.Split(cidrsEnv, ",")
		for _, cidr := range cidrs {
			cidr = strings.TrimSpace(cidr)
			if cidr == "" {
				continue
			}
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				logStructuredError("trusted_proxy_cidr_parse_error", map[string]interface{}{
					"cidr":  cidr,
					"error": err.Error(),
				})
				continue
			}
			trustedProxyCIDRs = append(trustedProxyCIDRs, network)
		}

		if len(trustedProxyCIDRs) > 0 {
			logStructured("trusted_proxies_configured", map[string]interface{}{
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
// It validates X-Forwarded-For headers only when the direct connection is from a trusted proxy.
// This prevents IP spoofing attacks where malicious clients send fake X-Forwarded-For headers.
func getClientIP(r *http.Request) string {
	initTrustedProxies()

	// Extract the direct connection IP (RemoteAddr, without port)
	directIP := extractIPFromRemoteAddr(r.RemoteAddr)

	// Only trust X-Forwarded-For and X-Real-IP if the direct connection is from a trusted proxy
	if isIPFromTrustedProxy(directIP) {
		// Check X-Forwarded-For header (for proxied requests)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP in the chain (original client)
			var clientIP string
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					clientIP = strings.TrimSpace(xff[:i])
					break
				}
			}
			if clientIP == "" {
				clientIP = strings.TrimSpace(xff)
			}

			// Validate the extracted IP format
			if validateIPFormat(clientIP) {
				return clientIP
			}
			// Invalid IP format in X-Forwarded-For, log and fall through
			logStructured("xff_invalid_ip_format", map[string]interface{}{
				"level":        "warn",
				"xff_header":   xff,
				"extracted_ip": clientIP,
				"direct_ip":    directIP,
				"security":     true,
			})
		}

		// Check X-Real-IP header
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			xri = strings.TrimSpace(xri)
			if validateIPFormat(xri) {
				return xri
			}
			// Invalid IP format in X-Real-IP, log and fall through
			logStructured("xrealip_invalid_ip_format", map[string]interface{}{
				"level":     "warn",
				"xrealip":   xri,
				"direct_ip": directIP,
				"security":  true,
			})
		}
	} else {
		// Direct connection is NOT from a trusted proxy - log if they're trying to spoof
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			logStructured("xff_untrusted_proxy_ignored", map[string]interface{}{
				"level":      "warn",
				"xff_header": xff,
				"direct_ip":  directIP,
				"message":    "X-Forwarded-For header ignored - connection not from trusted proxy",
				"security":   true,
			})
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			logStructured("xrealip_untrusted_proxy_ignored", map[string]interface{}{
				"level":     "warn",
				"xrealip":   xri,
				"direct_ip": directIP,
				"message":   "X-Real-IP header ignored - connection not from trusted proxy",
				"security":  true,
			})
		}
	}

	// Fall back to RemoteAddr
	return directIP
}
