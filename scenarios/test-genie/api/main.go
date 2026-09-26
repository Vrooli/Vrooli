package main

import (
	"log"

	"test-genie/internal/app"

	"github.com/vrooli/api-core/preflight"
)

func main() {
	if preflight.Run(preflight.Config{
		ScenarioName: "test-genie",
	}) {
		return
	}

	server, err := app.NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
