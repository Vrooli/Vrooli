package main

import (
	"fmt"
	"log"
	"os"

	"app-monitor-api/config"
)

const (
	serviceName = "App Monitor"
	apiVersion  = "1.0.0"
)

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start app-monitor

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

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
