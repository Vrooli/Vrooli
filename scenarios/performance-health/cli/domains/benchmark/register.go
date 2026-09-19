// Package benchmark is the CLI's benchmark-domain command surface (axis ①). It
// mirrors the API's BenchmarkService: time a scenario's build surfaces.
package benchmark

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "benchmark"

// Register builds the benchmark subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"BenchmarkService.RunBenchmark": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("benchmark: load from manifest: %w", err)
	}
	return group, nil
}
