package middleware

import (
	"context"
	"net/http"
	"scenario-authenticator/auth"
	"scenario-authenticator/models"
	"scenario-authenticator/utils"
	"strings"
)

// AuthMiddleware validates JWT tokens for protected routes
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.SendError(w, "No authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.SendError(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Check if token is blacklisted
		if auth.IsTokenBlacklisted(token) {
			utils.SendError(w, "Token has been revoked", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			utils.SendError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireRole checks if the user has the required role
func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value("claims").(*models.Claims)

		// Check if user has the required role
		hasRole := false
		for _, userRole := range claims.Roles {
			if userRole == role || userRole == "admin" {
				hasRole = true
				break
			}
		}

		if !hasRole {
			utils.SendError(w, "Insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
