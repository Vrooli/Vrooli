package adapters

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "adapters"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"AdapterService.ListCapabilities":         h.capabilities,
		"AdapterService.ExplainUnsupportedAction": h.explain,
		"AdapterService.GetPlatformSummary":       h.platform,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("adapters: load from manifest: %w", err)
	}
	return group, nil
}
