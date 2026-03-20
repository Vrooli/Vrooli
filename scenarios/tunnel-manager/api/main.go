package main

import (
	"context"
	"log"
	"log/slog"

	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"tunnel-manager/store"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "tunnel-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize structured logging [REQ:OBS-003]
	InitStructuredLogging(slog.LevelInfo)

	// Connect to database with automatic retry and backoff
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Ensure schema tables exist
	if err := store.EnsureSchema(db); err != nil {
		log.Fatalf("Schema migration failed: %v", err)
	}

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
