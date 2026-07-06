package conformance

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "conformance"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ConformanceService.ScanScenario": h.scan,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("conformance: load from manifest: %w", err)
	}
	return group, nil
}
