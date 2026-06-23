package resolver

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "resolver"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ResolverService.GetResolverStatus":    h.status,
		"ResolverService.ConfigureAdGuardHome": h.configureAdGuard,
		"ResolverService.UpdateUpstreams":      h.upstreams,
		"ResolverService.CheckResolverHealth":  h.health,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("resolver: load from manifest: %w", err)
	}
	return group, nil
}
