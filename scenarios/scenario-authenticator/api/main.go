package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/handlers"
	apimiddleware "scenario-authenticator/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Enforce lifecycle management - prevent direct execution
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-authenticator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Change to project root directory for consistent file operations
	if rootDir := resolveRepoRoot(); rootDir != "" {
		if err := os.Chdir(rootDir); err != nil {
			log.Printf("⚠️  Warning: Could not change to project root (%s): %v", rootDir, err)
		}
	} else {
		log.Printf("⚠️  Warning: Could not determine repository root; continuing with current working directory")
	}

	// Load configuration from environment variables
	port := getRequiredEnv("API_PORT", "")
	dbURL := getDBURL()
	redisURL := getRequiredEnv("REDIS_URL", "")

	// Initialize database
	if err := db.InitDB(dbURL); err != nil {
		log.Fatalf("[scenario-authenticator/api] ❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	if err := db.InitRedis(redisURL); err != nil {
		log.Fatalf("[scenario-authenticator/api] ❌ Failed to initialize Redis: %v", err)
	}

	// Load JWT keys
	if err := auth.LoadJWTKeys(); err != nil {
		log.Fatalf("[scenario-authenticator/api] ❌ Failed to load JWT keys: %v", err)
	}

	// Setup Chi router
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Configure CORS with environment-based origins for security
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080"}
	if corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		allowedOrigins = strings.Split(corsOrigins, ",")
	}

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Health check
	router.Get("/health", handlers.HealthHandler)

	// Authentication endpoints
	router.Post("/api/v1/auth/register", handlers.RegisterHandler)
	router.Post("/api/v1/auth/login", handlers.LoginHandler)
	router.Get("/api/v1/auth/validate", handlers.ValidateHandler)
	router.Post("/api/v1/auth/validate", handlers.ValidateHandler)
	router.Post("/api/v1/auth/refresh", handlers.RefreshHandler)
	router.Post("/api/v1/auth/logout", handlers.LogoutHandler)
	router.Post("/api/v1/auth/reset-password", handlers.ResetPasswordHandler)
	router.Post("/api/v1/auth/complete-reset", handlers.CompleteResetHandler)

	// Two-Factor Authentication endpoints
	router.Post("/api/v1/auth/2fa/setup", apimiddleware.AuthMiddleware(handlers.Setup2FAHandler))
	router.Post("/api/v1/auth/2fa/enable", apimiddleware.AuthMiddleware(handlers.Enable2FAHandler))
	router.Post("/api/v1/auth/2fa/disable", apimiddleware.AuthMiddleware(handlers.Disable2FAHandler))
	router.Post("/api/v1/auth/2fa/verify", handlers.Verify2FAHandler)

	// User management endpoints
	router.Get("/api/v1/users", apimiddleware.RequireRole("admin", handlers.GetUsersHandler))
	router.Get("/api/v1/users/{id}", apimiddleware.AuthMiddleware(handlers.GetUserHandler))
	router.Put("/api/v1/users/{id}", apimiddleware.RequireRole("admin", handlers.UpdateUserHandler))
	router.Delete("/api/v1/users/{id}", apimiddleware.RequireRole("admin", handlers.DeleteUserHandler))

	// Session management
	router.Get("/api/v1/sessions", apimiddleware.AuthMiddleware(handlers.GetSessionsHandler))
	router.Delete("/api/v1/sessions/{id}", apimiddleware.AuthMiddleware(handlers.RevokeSessionHandler))

	// OAuth endpoints
	handlers.InitOAuth() // Initialize OAuth configuration
	router.Get("/api/v1/auth/oauth/providers", handlers.GetOAuthProvidersHandler)
	router.Get("/api/v1/auth/oauth/login", handlers.OAuthLoginHandler)
	router.Get("/api/v1/auth/oauth/google/callback", handlers.OAuthCallbackHandler)
	router.Get("/api/v1/auth/oauth/github/callback", handlers.OAuthCallbackHandler)

	// API Key management
	router.Post("/api/v1/apikeys", apimiddleware.AuthMiddleware(handlers.CreateAPIKeyHandler))
	router.Get("/api/v1/apikeys", apimiddleware.AuthMiddleware(handlers.ListAPIKeysHandler))
	router.Delete("/api/v1/apikeys/{id}", apimiddleware.AuthMiddleware(handlers.RevokeAPIKeyHandler))
	router.Post("/api/v1/apikeys/validate", handlers.ValidateAPIKeyHandler)

	// Application management
	router.Get("/api/v1/applications", apimiddleware.RequireRole("admin", handlers.GetApplicationsHandler))
	router.Post("/api/v1/applications", apimiddleware.RequireRole("admin", handlers.RegisterApplicationHandler))
	router.Get("/api/v1/applications/{id}", apimiddleware.RequireRole("admin", handlers.GetApplicationHandler))
	router.Put("/api/v1/applications/{id}", apimiddleware.RequireRole("admin", handlers.UpdateApplicationHandler))
	router.Delete("/api/v1/applications/{id}", apimiddleware.RequireRole("admin", handlers.DeleteApplicationHandler))
	router.Get("/api/v1/applications/{id}/integration-code", apimiddleware.RequireRole("admin", handlers.GenerateIntegrationCodeHandler))

	// Start server
	log.Printf("[scenario-authenticator/api] 🚀 Authentication API server starting on port %s", port)
	log.Printf("[scenario-authenticator/api] 📍 Health check: http://localhost:%s/health", port)
	log.Printf("[scenario-authenticator/api] 🔑 JWT keys loaded successfully")
	log.Printf("[scenario-authenticator/api] 🎯 Ready to process authentication requests")
	log.Printf("[scenario-authenticator/api] ✨ CORS enabled with Chi router")
	log.Printf("[scenario-authenticator/api] 🔒 Security headers enabled for all responses")

	// Wrap router with security middleware (request size limit, then security headers)
	handler := apimiddleware.RequestSizeLimitMiddleware(router)
	handler = apimiddleware.SecurityHeadersMiddleware(handler)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("[scenario-authenticator/api] ❌ Server failed to start: %v", err)
	}
}

// getRequiredEnv gets an environment variable with optional fallback
func getRequiredEnv(primary, fallback string) string {
	value := os.Getenv(primary)
	if value == "" && fallback != "" {
		value = os.Getenv(fallback)
	}
	if value == "" {
		log.Fatalf("[scenario-authenticator/api] ❌ %s environment variable is required", primary)
	}
	return value
}

// getDBURL constructs database URL from environment variables
func getDBURL() string {
	// Try POSTGRES_URL first, but override the database name
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL != "" {
		// Replace the database name with scenario_authenticator
		// Parse the URL and replace the database part
		if strings.Contains(dbURL, "vrooli?") {
			dbURL = strings.Replace(dbURL, "vrooli?", "scenario_authenticator?", 1)
		} else if strings.Contains(dbURL, "vrooli") && strings.Contains(dbURL, "sslmode") {
			dbURL = strings.Replace(dbURL, "vrooli", "scenario_authenticator", 1)
		}
		return dbURL
	}

	// Build from individual components
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	// Force scenario_authenticator database
	dbName := "scenario_authenticator"

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" {
		log.Fatal("[scenario-authenticator/api] ❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)
}

// resolveRepoRoot attempts to locate the repository root so relative assets (like JWT keys) persist between runs.
func resolveRepoRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		if candidate := verifyRepoRoot(filepath.Join(dir, "..", "..", "..")); candidate != "" {
			return candidate
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if candidate := verifyRepoRoot(filepath.Join(cwd, "..", "..", "..")); candidate != "" {
			return candidate
		}
	}

	return ""
}

// verifyRepoRoot ensures the candidate path looks like the repository root.
func verifyRepoRoot(candidate string) string {
	candidate = filepath.Clean(candidate)
	if candidate == "" {
		return ""
	}

	if info, err := os.Stat(filepath.Join(candidate, "scenarios")); err == nil && info.IsDir() {
		return candidate
	}

	return ""
}
