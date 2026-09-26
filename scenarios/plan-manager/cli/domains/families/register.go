// Package families owns the manifest-driven plan-family CLI surface.
package families

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "families"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"FamiliesService.CreateFamily": h.create, "FamiliesService.GetFamily": h.get, "FamiliesService.ListFamilies": h.list,
		"FamiliesService.UpdateFamily": h.update, "FamiliesService.PutMember": h.putMember, "FamiliesService.RemoveMember": h.removeMember,
		"FamiliesService.PutClaim": h.putClaim, "FamiliesService.RemoveClaim": h.removeClaim, "FamiliesService.ProposeGraph": h.propose,
		"FamiliesService.ReviewGraph": h.review, "FamiliesService.GetFrontier": h.frontier,
		"FamiliesService.RenderFamily": h.render,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("families: load group: %w", err)
	}
	return group, nil
}
