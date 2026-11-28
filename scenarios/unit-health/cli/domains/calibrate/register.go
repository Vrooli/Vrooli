package calibrate

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "calibrate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ValidationService.RunCalibration": h.run,
		"calibrate.corpus":                 h.corpus,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("calibrate: load from manifest: %w", err)
	}
	return group, nil
}
