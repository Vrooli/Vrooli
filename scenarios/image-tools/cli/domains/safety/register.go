package safety

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the safety subcommand group. `policy` is bound from the
// manifest to the SafetyService.GetPolicy Connect RPC.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SafetyService.GetPolicy": h.policy,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("safety: load from manifest: %w", err)
	}
	return group, nil
}
