package main

import (
	"context"
	"log"
	"os"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	redisconfig "github.com/vrooli/api-core/redis"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "text-tools",
	}) {
		return // Process was re-exec'd after rebuild
	}

	log.Println("Text Tools API starting...")

	// Initialize configuration
	config := &Config{
		Port: os.Getenv("API_PORT"),
	}

	// Load optional resource configurations
	config.MinIOURL = os.Getenv("MINIO_URL")
	if redisEnv, err := redisconfig.Resolve(os.Getenv); err == nil {
		config.RedisURL = "redis://" + redisEnv.Addr
	}
	config.DatabaseURL, _ = database.ResolvePostgresDSN(os.Getenv)

	// Create and initialize server
	srv := NewServer(config)
	if err := srv.Initialize(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
