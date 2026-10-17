package visual

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "visual"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"VisualHealthService.AnalyzeArtifacts": h.analyzeArtifacts,
		"VisualHealthService.CompareArtifacts": h.compareArtifacts,
		"VisualHealthService.ListRules":        h.rules,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("visual: load from manifest: %w", err)
	}
	return group, nil
}
