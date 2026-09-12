// Package generation is the CLI's generation-domain command surface. Mirrors
// the API's Connect-RPC GenerationService and the UI's api/generation.ts client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, RPC bindings) and is the SINGLE source of truth for the command-line
// shape; handlers in handlers.go are wired via the bindings map.
package generation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "generation"

// Register builds the generation subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GenerationService.GetProviderStatus":          h.status,
		"GenerationService.GetImageBackendStatus":      h.imageStatus,
		"GenerationService.GenerateBrandElements":      h.elements,
		"GenerationService.GenerateBrandImage":         h.image,
		"GenerationService.EditBrandImage":             h.editImage,
		"GenerationService.RemoveBrandImageBackground": h.removeBackground,
		"GenerationService.DeriveBrandIcons":           h.deriveIcons,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("generation: load from manifest: %w", err)
	}
	return group, nil
}
