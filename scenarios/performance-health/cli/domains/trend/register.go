// Package trend is the CLI's trend-domain command surface. It mirrors the API's
// TrendService: read a scenario's persisted performance trend.
package trend

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "trend"

// Register builds the trend subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"TrendService.GetTrend": h.get,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("trend: load from manifest: %w", err)
	}
	return group, nil
}
