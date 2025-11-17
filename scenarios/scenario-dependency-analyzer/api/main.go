package main

import (
	"fmt"
	"log"
	"os"

	"scenario-dependency-analyzer/internal/app"
	appconfig "scenario-dependency-analyzer/internal/config"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-dependency-analyzer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
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
