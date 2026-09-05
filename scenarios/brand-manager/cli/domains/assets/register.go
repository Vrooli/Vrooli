// Package assets is the CLI's assets-domain command surface. Mirrors the API's
// Connect-RPC AssetsService and the UI's api/assets.ts client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, positionals, RPC bindings) and is the SINGLE source of truth for the
// command-line shape; handlers in handlers.go are wired via the bindings map.
package assets

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "assets"

// Register builds the assets subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AssetsService.ListAssets":    h.list,
		"AssetsService.UploadAsset":   h.upload,
		"AssetsService.GetAsset":      h.get,
		"AssetsService.DownloadAsset": h.download,
		"AssetsService.DeleteAsset":   h.delete,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("assets: load from manifest: %w", err)
	}
	return group, nil
}
