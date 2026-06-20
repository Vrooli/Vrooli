package perf

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this domain binds to.
const GroupName = "perf"

// Register wires the `perf` group's manifest commands to their handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"PerfService.BenchmarkStartup": h.measure,
		"PerfService.GetPerfTrend":     h.trend,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("perf: load from manifest: %w", err)
	}
	return group, nil
}
