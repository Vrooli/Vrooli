package inference

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "inference"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings, err := cliapp.ProtoBindings(core, "vrooli.ai_gateway.v1.inference.InferenceService", cliapp.ProtoBindingOptions{
		Render: map[string]cliapp.Renderer{
			"InferenceService.Run":         renderRun,
			"InferenceService.RunBatch":    renderRunBatch,
			"InferenceService.Embed":       renderEmbed,
			"InferenceService.Decide":      renderDecide,
			"InferenceService.DecideBatch": renderDecideBatch,
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("inference: build proto bindings: %w", err)
	}
	// These commands carry nested sampling controls or read a schema file; the
	// generic resolver cannot map those shapes, so keep their explicit handlers.
	bindings["InferenceService.Run"] = h.run
	bindings["InferenceService.RunBatch"] = h.runBatch
	bindings["InferenceService.Embed"] = h.embed
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("inference: load from manifest: %w", err)
	}
	return group, nil
}
