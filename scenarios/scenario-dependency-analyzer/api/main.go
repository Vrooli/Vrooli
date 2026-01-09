package main

import (
	"github.com/vrooli/api-core/preflight"
	"log"

	"scenario-dependency-analyzer/internal/app"
	appconfig "scenario-dependency-analyzer/internal/config"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "scenario-dependency-analyzer",
	}) {
		return // Process was re-exec'd after rebuild
	}

	cfg := appconfig.Load()

	db, err := appconfig.InitDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := app.Run(cfg, db); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
