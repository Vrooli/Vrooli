package main

import (
	"context"
	"fmt"
	"os"

	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "ai-chatbot-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize logger
	logger := NewLogger()
	logger.Println("🚀 Starting AI Chatbot Manager API...")

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := NewDatabase(config, logger)
	if err != nil {
		logger.Fatalf("❌ Database connection failed: %v", err)
	}

	// Create server
	srv := NewServer(config, db, logger)

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		logger.Fatalf("❌ Server error: %v", err)
	}
}
