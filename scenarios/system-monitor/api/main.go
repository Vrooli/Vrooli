package main

// DOC: docs/QUICKSTART.md
// DOC: docs/concepts/ARCHITECTURE.md

import (
	"log"

	"github.com/vrooli/api-core/preflight"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/server"
)

func main() {
	// preflight.Run performs the staleness/rebuild check and the lifecycle
	// guard (direct execution outside `vrooli scenario start` is rejected).
	if preflight.Run(preflight.Config{ScenarioName: "system-monitor"}) {
		return
	}

	cfg := config.Load()
	if err := server.Run(cfg); err != nil {
		log.Fatalf("System Monitor API failed: %v", err)
	}
}
