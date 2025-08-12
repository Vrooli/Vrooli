package middleware

import (
	"context"
	"net/http"
	"strings"
)

// AuthMiddleware checks for valid authentication token
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check for Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}
		
		token := parts[1]
		
		// Simple token validation (TODO: replace with proper JWT/OAuth)
		if !isValidToken(token) {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		
		// Add user context
		ctx := context.WithValue(r.Context(), "user", getUserFromToken(token))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValidToken performs simple token validation
func isValidToken(token string) bool {
	// TODO: Replace with proper authentication
	validTokens := []string{
		"metareasoning_cli_default_2024",
		"test_token_2024",
		"admin_token_2024",
	}
	
	for _, valid := range validTokens {
		if token == valid {
			return true
		}
	}
	
	return false
}

// getUserFromToken extracts user information from token
func getUserFromToken(token string) string {
	// TODO: Replace with proper user extraction
	switch token {
	case "admin_token_2024":
		return "admin"
	case "metareasoning_cli_default_2024":
		return "cli"
	default:
		return "user"
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// TODO: Use proper JSON encoding
	w.Write([]byte(`{"error":"` + message + `"}`))
}