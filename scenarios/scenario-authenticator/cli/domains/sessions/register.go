// Package sessions is the CLI's session-management command surface, mirroring
// the API's SessionsService Connect-RPC service. The cli/manifest.json
// "sessions" group is the single source of truth for the command shape.
package sessions

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "sessions"

// Register builds the sessions subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SessionsService.ListSessions":      h.list,
		"SessionsService.RevokeSession":     h.revoke,
		"SessionsService.RevokeAllSessions": h.revokeAll,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("sessions: load from manifest: %w", err)
	}
	return group, nil
}
