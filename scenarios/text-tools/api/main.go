package main

import (
	"github.com/vrooli/api-core/preflight"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// Validate required configuration
	if config.Port == "" {
		log.Fatal("API_PORT environment variable is required")
	}

	// Load optional resource configurations
	config.MinIOURL = os.Getenv("MINIO_URL")
	config.RedisURL = os.Getenv("REDIS_URL")
	config.OllamaURL = os.Getenv("OLLAMA_URL")
	config.DatabaseURL = os.Getenv("DATABASE_URL")

	// Create and initialize server
	server := NewServer(config)
	if err := server.Initialize(); err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		log.Fatalf("Server failed to start: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Text Tools API stopped")
}