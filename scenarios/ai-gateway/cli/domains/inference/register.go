package inference

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "inference"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"InferenceService.Run":      h.run,
		"InferenceService.RunBatch": h.runBatch,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("inference: load from manifest: %w", err)
	}
	return group, nil
}
