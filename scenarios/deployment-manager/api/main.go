// Package main provides the Deployment Manager API server entry point.
//
// The server exposes HTTP endpoints for:
//   - Dependency analysis and fitness scoring
//   - Profile management (CRUD operations with versioning)
//   - Deployment orchestration
//   - Swap analysis and cascading impact detection
//   - Secret management and validation
//   - Bundle validation and assembly
//   - Telemetry ingestion and reporting
//
// Architecture (Screaming Architecture):
//
//	main.go              - Entry point
//	server/              - HTTP server, routes, middleware
//	bundles/             - Bundle validation and assembly
//	profiles/            - Deployment profile management
//	fitness/             - Tier fitness scoring
//	secrets/             - Secret identification and templating
//	telemetry/           - Deployment telemetry ingestion
//	deployments/         - Deployment execution
//	dependencies/        - Dependency analysis
//	swaps/               - Resource swap analysis
//	health/              - Health checks
//	shared/              - Cross-cutting utilities
package main

import (
	"fmt"
	"log"
	"os"

	"deployment-manager/server"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start deployment-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	srv, err := server.New()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
