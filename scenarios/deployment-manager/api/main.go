// Package main provides the Deployment Manager API server.
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
// Architecture:
//
//	main.go              - Entry point
//	server.go            - Server struct, config, middleware, logging
//	routes.go            - Route setup
//	handlers_*.go        - HTTP request handlers
//	services_*.go        - Business logic and external service clients
//	bundle_validation.go - Bundle manifest validation
//	bundle_mapper.go     - Secret mapping for bundles
//	handlers_telemetry.go- Telemetry handlers
package main

import (
	"fmt"
	"log"
	"os"
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

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
