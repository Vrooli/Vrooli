package main

import (
	"log"
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
func run(checkPreflight func(preflight.Config) bool, load func() *config.Config, serve func(*config.Config) error) int {
	if checkPreflight(preflight.Config{ScenarioName: "system-monitor"}) {
		return 0
	}
	if err := serve(load()); err != nil {
		log.Printf("System Monitor API failed: %v", err)
		return 1
	}
	return 0
}
