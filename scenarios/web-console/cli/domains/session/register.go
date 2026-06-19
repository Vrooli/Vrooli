// Package session is the CLI's session-domain command surface. It mirrors the
// API's Connect-RPC SessionsService — session CRUD, expiration policy, and
// persistent-session recovery — and is built from the embedded manifest.
package session

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

// GroupName is the manifest group name this package owns.
const GroupName = "session"

// Register builds the `session` subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers. All RPCs go through the Connect-RPC
// SessionsService — the legacy REST routes have been removed.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SessionsService.List":               h.list,
		"SessionsService.Get":                h.get,
		"SessionsService.Create":             h.create,
		"SessionsService.Delete":             h.delete,
		"SessionsService.GetPolicy":          h.policyGet,
		"SessionsService.UpdatePolicy":       h.policySet,
		"SessionsService.ListRecoverable":    h.listRecoverable,
		"SessionsService.Recover":            h.recover,
		"SessionsService.DismissRecoverable": h.dismiss,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("session: load from manifest: %w", err)
	}
	// Preserve the pre-manifest subcommand aliases (cli-manifest/v1 has no
	// per-command alias field).
	support.ApplyAliases(group.Subcommands, map[string][]string{
		"list":   {"ls"},
		"get":    {"show"},
		"delete": {"rm"},
	})
	return group, nil
}
