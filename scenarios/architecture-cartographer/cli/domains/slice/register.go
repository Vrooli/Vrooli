// Package slice is the CLI surface for domain implementation slices.
package slice

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "slice"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"GraphService.GetSlice": h.show,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("slice: load from manifest: %w", err)
	}
	return group, nil
}
