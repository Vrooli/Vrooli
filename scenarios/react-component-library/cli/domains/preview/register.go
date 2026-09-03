// Package preview is the CLI's live-preview-bundler surface. Mirrors
// the API's Connect-RPC PreviewService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package preview

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "preview"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"PreviewService.GetPreviewBundle": cliapp.ProtoList(h.bundleCall, h.bundleReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("preview: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, (cliapp.Command{
		Name:        "populate-store",
		Description: "Populate the governed preview runtime store for one asset version through Scenario Dependency Analyzer",
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "component-id", Required: true, Description: "Library id, for example react-component-library:SidebarShell"},
			},
			Flags: []cliapp.Flag{{Name: "version", Required: true, Description: "Asset version whose @deps declarations should be installed"}},
		},
	}).WithPrimitive(cliapp.ExternalDelegation(h.populateStore)))
	return group, nil
}
