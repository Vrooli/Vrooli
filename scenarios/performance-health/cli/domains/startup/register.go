// Package startup is the CLI's startup-domain command surface (axis ②). It
// mirrors the API's StartupService: benchmark a scenario's startup performance
// and read the persisted trend.
package startup

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "startup"

// Register builds the startup subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"StartupService.BenchmarkStartup": h.measure,
		"StartupService.GetStartupTrend":  h.trend,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("startup: load from manifest: %w", err)
	}
	return group, nil
}
