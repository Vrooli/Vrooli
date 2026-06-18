// Package auth is the CLI's account-lifecycle command surface, mirroring the
// API's AccountsService Connect-RPC service. The cli/manifest.json "auth" group
// is the single source of truth for the command shape; handlers in handlers.go
// call the generated Connect client.
package auth

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "auth"

// Register builds the auth subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AccountsService.Register": h.register,
		"AccountsService.Login":    h.login,
		"AccountsService.Refresh":  h.refresh,
		"AccountsService.Logout":   h.logout,
		"AccountsService.Validate": h.validate,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("auth: load from manifest: %w", err)
	}
	return group, nil
}
