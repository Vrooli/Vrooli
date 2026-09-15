// Package rewrite is the CLI's rewrite-domain command surface. Mirrors
// the API's Connect-RPC GoCodeGraphService.RewritePlan and
// GoCodeGraphService.RewriteApply methods.
package rewrite

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns. Exported so
// the package's tests can call cliapp.RequireProtoServiceCoverage against
// the same manifest the runtime loads.
const GroupName = "rewrite"

// Register builds the rewrite subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GoCodeGraphService.RewritePlan":  h.plan,
		"GoCodeGraphService.RewriteApply": h.apply,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("rewrite: load from manifest: %w", err)
	}
	return group, nil
}
