// Package apply is the CLI's apply-domain command surface. It mirrors
// the API's Connect-RPC ApplyService: deterministic plan derivation,
// apply-run history, and the build baseline.
//
// v0.1 wires `apply run` as a command, but the API returns
// CodeUnimplemented (execution is a separate, later plan). The run
// handler surfaces that honestly as a capability-not-available message,
// not a crash.
//
// Follows the graph-domain shape: Register(core, manifest) builds a
// cliapp.SubcommandGroup from cli/manifest.json via LoadFromManifest,
// with one handler per Connect-RPC subcommand in handlers.go.
package apply

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "apply"

// Register builds the apply subcommand group from the embedded CLI
// manifest and wires every ApplyService Connect-RPC binding to a handler
// in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ApplyService.PlanApply":        h.plan,
		"ApplyService.RunApply":         h.run,
		"ApplyService.ListApplyHistory": h.history,
		"ApplyService.GetBuildBaseline": h.baseline,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("apply: load from manifest: %w", err)
	}
	return group, nil
}
