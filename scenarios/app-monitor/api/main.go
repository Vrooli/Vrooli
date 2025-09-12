package main

import (
	"log"
	"os"

	"app-monitor-api/config"
)

const (
	serviceName = "App Monitor"
	apiVersion  = "1.0.0"
)

func main() {
	// Ensure required environment variables
	if os.Getenv("API_PORT") == "" {
		log.Fatal("Error: API_PORT environment variable is required")
	}

	// Log startup
	log.Printf("Starting %s API v%s", serviceName, apiVersion)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize and run server
	server, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Start server (blocks until shutdown)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}