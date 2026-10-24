// Package design is the CLI's design-domain command surface. Mirrors the API's
// Connect-RPC DesignService and the UI's api/design.ts client.
//
// The manifest (cli/manifest.json) carries the declarative surface (governance,
// flags, RPC bindings) and is the SINGLE source of truth for the command-line
// shape; handlers in handlers.go are wired via the bindings map.
package design

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "design"

// Register builds the design subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"DesignService.GenerateDesignLanguage": h.generate,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("design: load from manifest: %w", err)
	}
	return group, nil
}
