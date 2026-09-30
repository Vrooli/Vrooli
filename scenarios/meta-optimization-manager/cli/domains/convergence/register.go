// Package convergence is the CLI's convergence-domain command surface. It mirrors
// the API's ConvergenceService. The manifest (cli/manifest.json) carries the
// declarative command shape; handlers.go builds each typed request and renders
// the response.
package convergence

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "convergence"

// Register builds the convergence subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ConvergenceService.GetConvergenceStatus": h.status,
		"ConvergenceService.GetTemplateFitness":   h.fitness,
		"ConvergenceService.ListReferences":       h.references,
		"ConvergenceService.GetConvergenceTrend":  h.trend,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("convergence: load from manifest: %w", err)
	}
	return group, nil
}
