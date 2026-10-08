// Package companion exposes the owner-facing Device Control companion
// lifecycle through the same authenticated CLI transport as other Bridge
// domains.
package companion

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "companion"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CompanionService.Install": h.install,
		"CompanionService.Inspect": h.inspect,
		"CompanionService.Upgrade": h.upgrade,
		"CompanionService.Revoke":  h.revoke,
		"CompanionService.Remove":  h.remove,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("companion: load from manifest: %w", err)
	}
	return group, nil
}
