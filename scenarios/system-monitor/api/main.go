package main

// DOC: docs/QUICKSTART.md
// DOC: docs/concepts/ARCHITECTURE.md

import (
	"log/slog"
	"os"

	"github.com/vrooli/api-core/preflight"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/server"
)

func main() {
	// preflight.Run performs the staleness/rebuild check and the lifecycle
	// guard (direct execution outside `vrooli scenario start` is rejected).
	if preflight.Run(preflight.Config{
		ScenarioName: "system-monitor",
		// The lifecycle performs freshness checks before launching this binary.
		// Repeating the repository walk here can starve the health listener and
		// create a second compiler workload during managed startup.
		DisableStaleness: true,
	}) {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("System Monitor API configuration failed", "error", err)
		os.Exit(1)
	}
	if err := server.Run(cfg); err != nil {
		slog.Error("System Monitor API failed", "error", err)
		os.Exit(1)
	}
}
