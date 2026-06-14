package main

import (
	"context"
	"log"

	"scenario-dependency-analyzer/internal/app"
	"scenario-dependency-analyzer/internal/modules"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	_ "modernc.org/sqlite"

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

	db, err := appconfig.InitDatabase(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("Failed to initialize schemas: %v", err)
	}

	if err := app.Run(cfg, db.Primary()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
