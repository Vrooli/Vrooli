package fixconfig

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this domain binds to.
const GroupName = "fix-config"

// Register wires the `fix-config` group's manifest commands to their handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ValidationService.PreviewFixConfig": h.run,
		"ValidationService.ApplyFixConfig":   h.apply,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("fix-config: load from manifest: %w", err)
	}
	return group, nil
}
