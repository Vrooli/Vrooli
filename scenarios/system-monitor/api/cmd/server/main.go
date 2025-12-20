package main

import (
	"fmt"
	"log"
	"os"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/server"
)

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start system-monitor

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	cfg := config.Load()
	if err := server.Run(cfg); err != nil {
		log.Fatalf("System Monitor API failed: %v", err)
	}
}
