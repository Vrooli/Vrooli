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
	bindings := map[string]func(cliapp.RunContext) error{
		"PreviewService.GetPreviewBundle": h.bundle,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("preview: load from manifest: %w", err)
	}
	return group, nil
}
