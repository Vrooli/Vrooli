package main

import (
	"context"
	"log"
	"os"

	"github.com/vrooli/api-core/preflight"
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
	config.RedisURL = os.Getenv("REDIS_URL")
	config.OllamaURL = os.Getenv("OLLAMA_URL")
	config.DatabaseURL = os.Getenv("DATABASE_URL")

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