// Package artifacts is the CLI's codegen-lifecycle command surface,
// a thin wrapper over the Connect-RPC ArtifactsService (per-flow) and
// ScenariosService (scenario-wide + streaming). Command surface loads
// from cli/manifest.json via cliapp.LoadFromManifest. Each `artifacts`
// command primarily binds to its per-flow RPC; the scenario-wide
// dispatch (under --scenario) is captured in the manifest's omitted[]
// list because cli-manifest/v1 enforces 1:1 command<->method bindings.
package artifacts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "artifacts"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ArtifactsService.GetArtifactStatus": h.status,
		"ArtifactsService.GenerateArtifacts": h.generate,
		"ArtifactsService.ClearArtifacts":    h.clear,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("artifacts: load from manifest: %w", err)
	}
	return group, nil
}
