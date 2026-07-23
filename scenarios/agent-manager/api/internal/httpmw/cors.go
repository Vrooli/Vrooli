// Package httpmw contains transport-only HTTP middleware used at the server
// composition root. It intentionally has no orchestration dependency.
package httpmw

import (
	"net/http"
	"os"
	"strings"
)

// CORS applies the Agent Manager browser-origin policy. Origins may be set
// with CORS_ALLOWED_ORIGINS as a comma-separated list; localhost wildcard
// ports are retained as the development-safe default.
func CORS(next http.Handler) http.Handler {
	allowedOrigins := AllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && OriginAllowed(origin, allowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AllowedOrigins parses the CORS setting and returns development-safe defaults
// when it is unset.
func AllowedOrigins(raw string) []string {
	if origins := strings.TrimSpace(raw); origins != "" {
		var result []string
		for _, origin := range strings.Split(origins, ",") {
			if origin = strings.TrimSpace(origin); origin != "" {
				result = append(result, origin)
			}
		}
		return result
	}
	return []string{"http://localhost:*", "http://127.0.0.1:*"}
}

// OriginAllowed reports whether an origin exactly matches a configured value
// or a supported wildcard-port pattern.
func OriginAllowed(origin string, allowed []string) bool {
	for _, pattern := range allowed {
		if strings.HasSuffix(pattern, ":*") {
			if strings.HasPrefix(origin, strings.TrimSuffix(pattern, "*")) {
				return true
			}
			continue
		}
		if origin == pattern {
			return true
		}
	}
	return false
}
