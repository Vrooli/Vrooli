// Package file_preview is the CLI's file-preview command surface. It mirrors
// the API's Connect-RPC FilePreviewService — resolve a path into a preview
// target and read bounded text content — and is built from the embedded
// manifest. Binary/media blob streaming is UI/browser-only (the blob route is
// consumed by native media elements), so the CLI exposes metadata + text only.
package file_preview

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "file-preview"

// Register builds the `file-preview` subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"FilePreviewService.Resolve":        h.resolve,
		"FilePreviewService.GetTextContent": h.text,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("file-preview: load from manifest: %w", err)
	}
	return group, nil
}
