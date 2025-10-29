package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type corsConfig struct {
	allowedOrigins []string
	allowAll       bool
}

var wildcardWarningOnce sync.Once

func resolveAllowedOrigins() corsConfig {
	envSources := []string{
		os.Getenv("CORS_ALLOWED_ORIGINS"),
		os.Getenv("ALLOWED_ORIGINS"),
		os.Getenv("CORS_ALLOWED_ORIGIN"),
	}

	raw := ""
	for _, val := range envSources {
		if strings.TrimSpace(val) != "" {
			raw = val
			break
		}
	}

	if strings.TrimSpace(raw) == "*" {
		return corsConfig{allowAll: true}
	}

	if raw != "" {
		parts := strings.Split(raw, ",")
		origins := make([]string, 0, len(parts))
		allowAll := false
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if trimmed == "*" {
				allowAll = true
				continue
			}
			origins = append(origins, trimmed)
		}
		return corsConfig{allowedOrigins: origins, allowAll: allowAll}
	}

	// Default safe origins include lifecycle-managed UI and App Monitor proxy
	uiPort := strings.TrimSpace(os.Getenv("UI_PORT"))
	defaults := []string{"https://app-monitor.itsagitime.com", "null"}
	if uiPort != "" {
		defaults = append(defaults,
			fmt.Sprintf("http://localhost:%s", uiPort),
			fmt.Sprintf("http://127.0.0.1:%s", uiPort),
		)
	} else {
		defaults = append(defaults, "http://localhost:3000", "http://127.0.0.1:3000")
	}

	return corsConfig{allowedOrigins: defaults}
}

func corsMiddleware(log *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := resolveAllowedOrigins()
			origin := strings.TrimSpace(r.Header.Get("Origin"))

			// Always vary on origin unless wildcard is enabled for transparency
			if !cfg.allowAll {
				w.Header().Add("Vary", "Origin")
			}

			if cfg.allowAll {
				wildcardWarningOnce.Do(func() {
					log.Warn("Using wildcard CORS (CORS_ALLOWED_ORIGINS=*) - not recommended for production")
				})
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin == "" {
				// Requests without Origin header (non-browser clients) are allowed without CORS headers
			} else if isOriginAllowed(origin, cfg.allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				log.WithFields(logrus.Fields{
					"origin":          origin,
					"allowed_origins": strings.Join(cfg.allowedOrigins, ","),
				}).Warn("Rejected CORS request from unauthorized origin")
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}

			// Set common headers when CORS is active
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			if acrh := r.Header.Get("Access-Control-Request-Headers"); acrh != "" {
				w.Header().Set("Access-Control-Allow-Headers", acrh)
			} else {
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
			}
			w.Header().Set("Access-Control-Expose-Headers", "Link")
			w.Header().Set("Access-Control-Max-Age", "300")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, allowedOrigin := range allowed {
		if strings.EqualFold(allowedOrigin, origin) {
			return true
		}
	}
	return false
}
