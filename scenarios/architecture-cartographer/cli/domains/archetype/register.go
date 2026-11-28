// Package archetype is the CLI surface for domain archetype inference (Q20).
package archetype

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "archetype"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GraphService.InferArchetype": h.infer,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("archetype: load from manifest: %w", err)
	}
	return group, nil
}
