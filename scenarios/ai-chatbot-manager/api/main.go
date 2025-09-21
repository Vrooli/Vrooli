package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start ai-chatbot-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
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
	defer db.Close()

	// Create and start server
	server := NewServer(config, db, logger)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Println("📛 Shutdown signal received")
		if err := server.Shutdown(); err != nil {
			logger.Printf("⚠️  Shutdown error: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	if err := server.Start(); err != nil {
		logger.Fatalf("❌ Server failed to start: %v", err)
	}
}