package roadmap

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "roadmap"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"RoadmapService.ListSectors":     h.listSectors,
		"RoadmapService.UpsertSector":    h.upsertSector,
		"RoadmapService.ListMilestones":  h.listMilestones,
		"RoadmapService.UpsertMilestone": h.upsertMilestone,
		"RoadmapService.GetProgress":     h.progress,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("roadmap: load from manifest: %w", err)
	}
	return group, nil
}
