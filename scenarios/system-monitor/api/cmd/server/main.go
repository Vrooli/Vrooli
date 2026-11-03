package main

import (
	"log/slog"
	"os"

	"github.com/vrooli/api-core/preflight"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/server"
)

func main() {
	os.Exit(run(preflight.Run, config.Load, server.Run))
}

// run is main's testable body. preflight.Run performs the staleness/rebuild
// check and the lifecycle guard (direct execution outside
// `vrooli scenario start` is rejected); returning true means it already
// handled the process, so the server must not start.
func run(checkPreflight func(preflight.Config) bool, load func() (*config.Config, error), serve func(*config.Config) error) int {
	if checkPreflight(preflight.Config{ScenarioName: "system-monitor"}) {
		return 0
	}
	cfg, err := load()
	if err != nil {
		slog.Error("System Monitor API configuration failed", "error", err)
		return 1
	}
	if err := serve(cfg); err != nil {
		slog.Error("System Monitor API failed", "error", err)
		return 1
	}
	return 0
}
