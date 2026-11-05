// Package readiness is the CLI's readiness-domain command surface. It mirrors
// the API's ReadinessService: validate a scenario's reachable capture tier and
// preview/apply the format-preserving fixes that move it toward Tier 1.
package readiness

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "readiness"

// Register builds the readiness subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ReadinessService.ValidateReadiness":   h.validate,
		"ReadinessService.PreviewReadinessFix": h.fix,
		"ReadinessService.ApplyReadinessFix":   h.apply,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("readiness: load from manifest: %w", err)
	}
	return group, nil
}
