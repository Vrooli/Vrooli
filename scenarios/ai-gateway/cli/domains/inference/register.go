package inference

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "inference"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings, err := cliapp.ProtoBindings(core, "vrooli.ai_gateway.v1.inference.InferenceService", cliapp.ProtoBindingOptions{
		Render: map[string]cliapp.Renderer{
			"InferenceService.Run":      renderRun,
			"InferenceService.RunBatch": renderRunBatch,
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("inference: build proto bindings: %w", err)
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("inference: load from manifest: %w", err)
	}
	return group, nil
}
