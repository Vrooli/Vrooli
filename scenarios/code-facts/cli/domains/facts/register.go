package facts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "facts"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CodeFactsService.DescribeCodeFacts":  h.describe,
		"CodeFactsService.ListSurfaces":       h.surfaces,
		"CodeFactsService.CheckProtoAdoption": h.protoAdoption,
		"CodeFactsService.CheckEndpointProof": h.endpointProof,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("facts: load from manifest: %w", err)
	}
	return group, nil
}
