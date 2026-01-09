package main

import (
	"log"

	"github.com/vrooli/api-core/preflight"

	"test-genie/internal/app"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "test-genie",
	}) {
		return // Process was re-exec'd after rebuild
	}

	server, err := app.NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
