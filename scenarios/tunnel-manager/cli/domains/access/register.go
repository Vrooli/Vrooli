// Package access is the CLI's access-domain command surface. It binds the
// ConfigService.GetAccessStatus RPC that reports the global /public
// Access-bypass switch and the per-host bypass state, under a dedicated
// `access` command group so operators reason about edge public-exemption
// separately from base config and ingress drift.
//
// Follows the canonical domain shape: a Register(core, manifest) returning a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The global on/off toggle lives on `config public-exposure`
// (ConfigService.SetPublicExposure); this group is read-only.
package access

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "access"

// Register builds the access subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go. status, dry-run, and
// list all read GetAccessStatus and render distinct views of one response.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ConfigService.GetAccessStatus": h.status,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("access: load from manifest: %w", err)
	}
	return group, nil
}
