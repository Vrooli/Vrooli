package main

import (
	"log"
	"os"

	"test-genie/internal/app"
	"test-genie/internal/orchestrator/phases/seeds"

	"github.com/vrooli/api-core/preflight"
)

func main() {
	if handled, code := seeds.RunWorker(os.Args[1:], os.Stderr); handled {
		os.Exit(code)
	}
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
