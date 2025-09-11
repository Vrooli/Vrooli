package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"scenario-authenticator/auth"
	"scenario-authenticator/db"
	"scenario-authenticator/handlers"
	"scenario-authenticator/middleware"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration from environment variables
	port := getRequiredEnv("AUTH_API_PORT", "API_PORT")
	dbURL := getDBURL()
	redisURL := getRequiredEnv("REDIS_URL", "")

	// Initialize database
	if err := db.InitDB(dbURL); err != nil {
		log.Fatal("❌ Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis
	if err := db.InitRedis(redisURL); err != nil {
		log.Fatal("❌ Failed to initialize Redis:", err)
	}

	// Load JWT keys
	if err := auth.LoadJWTKeys(); err != nil {
		log.Fatal("❌ Failed to load JWT keys:", err)
	}

	// Setup routes
	router := mux.NewRouter()
	
	// Health check
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")
	
	// Authentication endpoints
	router.HandleFunc("/api/v1/auth/register", handlers.RegisterHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/login", handlers.LoginHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/validate", handlers.ValidateHandler).Methods("GET", "POST")
	router.HandleFunc("/api/v1/auth/refresh", handlers.RefreshHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/logout", handlers.LogoutHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/reset-password", handlers.ResetPasswordHandler).Methods("POST")
	router.HandleFunc("/api/v1/auth/complete-reset", handlers.CompleteResetHandler).Methods("POST")
	
	// User management endpoints
	router.HandleFunc("/api/v1/users", handlers.GetUsersHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{id}", handlers.GetUserHandler).Methods("GET")
	router.HandleFunc("/api/v1/users/{id}", handlers.UpdateUserHandler).Methods("PUT")
	router.HandleFunc("/api/v1/users/{id}", handlers.DeleteUserHandler).Methods("DELETE")
	
	// Session management
	router.HandleFunc("/api/v1/sessions", handlers.GetSessionsHandler).Methods("GET")
	router.HandleFunc("/api/v1/sessions/{id}", handlers.RevokeSessionHandler).Methods("DELETE")

	// Apply CORS middleware
	router.Use(middleware.CORSMiddleware)
	
	// Start server
	log.Printf("🚀 Authentication API server starting on port %s", port)
	log.Printf("📍 Health check: http://localhost:%s/health", port)
	log.Printf("🔑 JWT keys loaded successfully")
	
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("❌ Server failed to start:", err)
	}
}

// getRequiredEnv gets an environment variable with optional fallback
func getRequiredEnv(primary, fallback string) string {
	value := os.Getenv(primary)
	if value == "" && fallback != "" {
		value = os.Getenv(fallback)
	}
	if value == "" {
		log.Fatalf("❌ %s environment variable is required", primary)
	}
	return value
}

// getDBURL constructs database URL from environment variables
func getDBURL() string {
	// Try POSTGRES_URL first
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL != "" {
		return dbURL
	}
	
	// Build from individual components
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}
	
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)
}