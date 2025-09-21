package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Enforce lifecycle management - prevent direct execution
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		log.Fatal("❌ This service must be started through the Vrooli lifecycle system.\n" +
			"   Use: vrooli scenario start scenario-authenticator\n" +
			"   Or:  cd scenarios/scenario-authenticator && make run")
	}

	// Change to project root directory for consistent file operations
	if err := os.Chdir("../.."); err != nil {
		log.Printf("⚠️  Warning: Could not change to project root: %v", err)
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

	// Configure CORS - this properly handles OPTIONS requests
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins for development
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

	// User management endpoints
	router.Get("/api/v1/users", handlers.GetUsersHandler)
	router.Get("/api/v1/users/{id}", handlers.GetUserHandler)
	router.Put("/api/v1/users/{id}", handlers.UpdateUserHandler)
	router.Delete("/api/v1/users/{id}", handlers.DeleteUserHandler)

	// Session management
	router.Get("/api/v1/sessions", handlers.GetSessionsHandler)
	router.Delete("/api/v1/sessions/{id}", handlers.RevokeSessionHandler)

	// Application management
	router.Get("/api/v1/applications", handlers.GetApplicationsHandler)
	router.Post("/api/v1/applications", handlers.RegisterApplicationHandler)
	router.Get("/api/v1/applications/{id}", handlers.GetApplicationHandler)
	router.Put("/api/v1/applications/{id}", handlers.UpdateApplicationHandler)
	router.Delete("/api/v1/applications/{id}", handlers.DeleteApplicationHandler)
	router.Get("/api/v1/applications/{id}/integration-code", handlers.GenerateIntegrationCodeHandler)

	// Start server
	log.Printf("[scenario-authenticator/api] 🚀 Authentication API server starting on port %s", port)
	log.Printf("[scenario-authenticator/api] 📍 Health check: http://localhost:%s/health", port)
	log.Printf("[scenario-authenticator/api] 🔑 JWT keys loaded successfully")
	log.Printf("[scenario-authenticator/api] 🎯 Ready to process authentication requests")
	log.Printf("[scenario-authenticator/api] ✨ CORS enabled with Chi router")

	if err := http.ListenAndServe(":"+port, router); err != nil {
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
