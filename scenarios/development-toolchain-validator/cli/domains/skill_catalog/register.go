// Package skill_catalog is the CLI's skill-catalog command surface.
// Mirrors the API's Connect-RPC SkillCatalogService. Command surface
// loads from cli/manifest.json via cliapp.LoadFromManifest.
package skill_catalog

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "skill-catalog"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SkillCatalogService.Sync":       h.sync,
		"SkillCatalogService.ListSkills": h.list,
		"SkillCatalogService.GetSkill":   h.get,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("skill-catalog: load from manifest: %w", err)
	}
	return group, nil
}
