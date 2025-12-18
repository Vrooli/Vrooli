package main

import (
	"github.com/vrooli/api-core/preflight"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
)

// Package-level logger
var logger *Logger

// Database connection
var db *sql.DB

func initDB() *sql.DB {
	var err error

	// Database configuration - support both POSTGRES_URL and individual components
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all required database connection parameters (HOST, PORT, USER, PASSWORD, DB)")
		}

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	logger.Info("🔄 Attempting database connection with exponential backoff...")
	logger.Info("📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			logger.Info("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
		actualDelay := delay + jitter

		logger.Warning("Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		logger.Info("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			logger.Info("📈 Retry progress:")
			logger.Info("   - Attempts made: %d/%d", attempt+1, maxRetries)
			logger.Info("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			logger.Info("   - Current delay: %v", actualDelay)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	logger.Info("🎉 Database connection pool established successfully!")

	return db
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "secrets-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize structured logger
	logger = NewLogger("secrets-manager")

	skipDB := strings.EqualFold(os.Getenv("SECRETS_MANAGER_SKIP_DB"), "true")
	if skipDB {
		logger.Info("⚠️ Skipping database initialization (SECRETS_MANAGER_SKIP_DB=true)")
	} else {
		db = initDB()
		defer db.Close()
		warmSecurityScanCache()
	}
	logger.Info("🚀 Starting Secrets Manager API (database optional)")

	apiServer := newAPIServer(db, logger)
	r := apiServer.routes()

	// CORS headers
	corsHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT
		if port == "" {
			log.Fatal("❌ API_PORT or PORT environment variable is required")
		}
	}

	logger.Info("🔐 Secrets Manager API starting on port %s", port)
	logger.Info("   📊 Health check: http://localhost:%s/health", port)
	logger.Info("   🔍 Scan endpoint: http://localhost:%s/api/v1/secrets/scan", port)
	logger.Info("   ✅ Validate endpoint: http://localhost:%s/api/v1/secrets/validate", port)

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handlers.CORS(corsHeaders, corsMethods, corsOrigins)(r),
	}

	log.Fatal(server.ListenAndServe())
}
