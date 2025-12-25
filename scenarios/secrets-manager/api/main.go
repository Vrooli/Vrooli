package main

import (
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"fmt"
	"log"
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
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
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
