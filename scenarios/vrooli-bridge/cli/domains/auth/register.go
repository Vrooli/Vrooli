// Package auth owns the owner-session commands. It lets terminal operators
// obtain the same Bridge owner JWT as the browser without copying it from
// browser storage.
package auth

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "auth"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"IdentityService.Login":   h.login,
		"IdentityService.Refresh": h.refresh,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("auth: load from manifest: %w", err)
	}
	return group, nil
}
