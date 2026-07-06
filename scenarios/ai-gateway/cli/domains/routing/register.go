package routing

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "routing"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"RoutingService.PreviewRoute":      h.preview,
		"RoutingService.ExecuteRoute":      h.execute,
		"RoutingService.ListRouteEvidence": h.evidenceList,
		"RoutingService.GetRouteEvidence":  h.evidenceShow,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("routing: load from manifest: %w", err)
	}
	return group, nil
}
