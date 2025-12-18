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
	"github.com/vrooli/api-core/preflight"
	"log"

	"deployment-manager/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "deployment-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := server.New()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
