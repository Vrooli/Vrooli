package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ecosystem-manager/api/pkg/server"
)

// LOGGING STRATEGY:
//
// This codebase uses TWO intentionally separate logging systems:
//
// 1. Standard library log.* (stdout/stderr) - Real-time Operational Logs
//    - Purpose: Immediate feedback for operators/developers watching the process
//    - Destination: stdout/stderr (captured by process managers, systemd, Docker, kubectl)
//    - Use for: Startup messages, progress indicators, debugging info, operational observability
//    - Volume: Verbose, ephemeral
//    - Examples: "🚀 Starting API...", "Processing task...", "Warning: retrying..."
//
// 2. Custom systemlog.* (file-based) - Historical Audit Trail
//    - Purpose: Persistent, structured logs for UI consumers and post-mortem analysis
//    - Destination: Date-stamped files in ../logs/ directory
//    - Served via: /api/logs HTTP endpoint (see pkg/handlers/logs.go)
//    - Use for: Significant business events, errors, state changes worth persisting
//    - Volume: Selective, permanent (only ~28% of log.* call volume)
//    - Severity levels: Debug, Info, Warn, Error
//    - Examples: Task state transitions, configuration changes, critical errors
//
// GUIDELINES:
// - Use log.* for operational observability (what's happening right now)
// - Use systemlog.* for audit trail (what happened that matters historically)
// - Avoid logging the same message to both systems (choose the right destination)
// - systemlog.* powers the UI log viewer; log.* powers real-time monitoring
//

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start ecosystem-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting Ecosystem Manager API...")

	port := os.Getenv("API_PORT")
	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	app, err := server.New(server.Config{
		Port:           port,
		AllowedOrigins: allowedOrigins,
	})
	if err != nil {
		log.Fatalf("Server failed to initialize: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func parseAllowedOrigins(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{"*"}
	}

	parts := strings.Split(trimmed, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			origins = append(origins, v)
		}
	}

	if len(origins) == 0 {
		return []string{"*"}
	}

	return origins
}
