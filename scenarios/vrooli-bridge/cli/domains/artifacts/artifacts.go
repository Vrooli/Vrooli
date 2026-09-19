// Package artifacts is the CLI's artifacts-domain command surface. Mirrors the
// API's Connect-RPC ArtifactsService: ship non-git artifacts to fleet nodes via
// device-sync-hub directed delivery, inspect distributions, and retrieve
// bounded owner-scoped run artifacts. The manifest (cli/manifest.json) is the
// single source of truth for the command shape; handlers.go binds the RPCs.
package artifacts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "artifacts"

// Register builds the artifacts subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ArtifactsService.DistributeArtifact": h.distribute,
		"ArtifactsService.GetDistribution":    h.get,
		"ArtifactsService.ListDistributions":  h.list,
		"ArtifactsService.GetRunArtifact":     h.getRunArtifact,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("artifacts: load from manifest: %w", err)
	}
	return group, nil
}
