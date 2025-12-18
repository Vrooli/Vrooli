package main

import (
	"github.com/vrooli/api-core/preflight"
	"log"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "knowledge-observatory",
	}) {
		return // Process was re-exec'd after rebuild
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

