// Package experiment hosts the `audio-tools experiment ...` subtree for
// persisted async STT experiment runs. The command surface is declared in
// cli/manifest.json; Register binds ExperimentService RPCs to handlers.go.
package experiment

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "experiment"

// Register builds the experiment subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ExperimentService.StartExperiment":        h.start,
		"ExperimentService.GetExperiment":          h.get,
		"ExperimentService.WaitExperiment":         h.wait,
		"ExperimentService.ListExperiments":        h.list,
		"ExperimentService.CancelExperiment":       h.cancel,
		"ExperimentService.StreamExperimentEvents": h.watch,
		"ExperimentService.GetExperimentReport":    h.report,
		"ExperimentService.CompareExperiments":     h.compare,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("experiment: load from manifest: %w", err)
	}
	return group, nil
}
