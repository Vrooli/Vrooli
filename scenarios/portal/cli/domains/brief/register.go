package brief

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "brief"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"BriefService.Build":     h.build,
		"BriefService.Get":       h.get,
		"BriefService.List":      h.list,
		"BriefService.RecordUse": h.recordUse,
		"BriefService.Stats":     h.stats,
		"hook":                   h.hook,
		"hooks":                  h.hooks,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("brief: load from manifest: %w", err)
	}
	return group, nil
}
