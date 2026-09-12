package search

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "search"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"SearchService.Suggest": h.suggest,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("search: load from manifest: %w", err)
	}
	return group, nil
}
