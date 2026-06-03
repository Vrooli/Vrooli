// Package providers is the CLI's providers-domain command surface. It mirrors
// the API's RegistryService Connect-RPC (register / list / remove) and is the
// first real domain replacing the notes reference.
//
// New domain packages copy this shape: a Register(core, manifest) returning a
// cliapp.SubcommandGroup built from cli/manifest.json via
// cliapp.LoadFromManifest, plus one handler per Connect-RPC subcommand in
// handlers.go. The manifest carries the declarative surface (governance, flags,
// positionals, RPC bindings) and is the SINGLE source of truth for the
// command-line shape; do not hand-author SubcommandGroup literals for
// Connect-RPC commands.
package providers

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns. Exported so the
// package's tests can call RequireProtoServiceCoverage against the same
// manifest the runtime loads.
const GroupName = "providers"

// Register builds the providers subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"RegistryService.RegisterProvider":   h.register,
		"RegistryService.ListProviders":      h.list,
		"RegistryService.DeregisterProvider": h.remove,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("providers: load from manifest: %w", err)
	}
	return group, nil
}
