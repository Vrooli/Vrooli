// Package brands is the CLI's brands-domain command surface. Mirrors the API's
// Connect-RPC BrandsService and the UI's api/brands.ts client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, positionals, RPC bindings) and is the SINGLE source of truth for the
// command-line shape; handlers in handlers.go are wired via the bindings map.
package brands

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "brands"

// Register builds the brands subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"BrandsService.ListBrands":        h.list,
		"BrandsService.CreateBrand":       h.create,
		"BrandsService.GetBrand":          h.get,
		"BrandsService.UpdateBrand":       h.update,
		"BrandsService.DeleteBrand":       h.delete,
		"BrandsService.ListBrandVersions": h.versions,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("brands: load from manifest: %w", err)
	}
	return group, nil
}
