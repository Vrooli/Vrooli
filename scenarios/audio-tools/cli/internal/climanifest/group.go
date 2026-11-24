// Package climanifest centralizes the manifest-backed CLI group contract.
package climanifest

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// LoadGroup loads one manifest group and consistently annotates failures with
// the owning domain. Domain packages retain only their handler construction
// and Service.Method binding map.
func LoadGroup(manifest []byte, group string, bindings map[string]func(cliapp.RunContext) error) (cliapp.SubcommandGroup, error) {
	loaded, err := cliapp.LoadFromManifest(manifest, group, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", group, err)
	}
	return loaded, nil
}

// LoadPrimitiveGroup is LoadGroup for renderer-separated primitive handlers.
func LoadPrimitiveGroup(manifest []byte, group string, bindings map[string]cliapp.PrimitiveHandler) (cliapp.SubcommandGroup, error) {
	loaded, err := cliapp.LoadFromManifestPrimitives(manifest, group, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", group, err)
	}
	return loaded, nil
}
