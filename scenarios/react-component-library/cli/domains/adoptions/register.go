// Package adoptions is the CLI's adoption-registry surface. Mirrors
// the API's Connect-RPC AdoptionsService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package adoptions

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "adoptions"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AdoptionsService.ListAdoptions":       h.list,
		"AdoptionsService.ApplyAdoption":       h.apply,
		"AdoptionsService.ReapplyAdoption":     h.reapply,
		"AdoptionsService.DeleteAdoption":      h.delete,
		"AdoptionsService.RefreshAdoptions":    h.refresh,
		"AdoptionsService.ResolveAdoptionPath": h.resolvePath,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("adoptions: load from manifest: %w", err)
	}
	return group, nil
}
