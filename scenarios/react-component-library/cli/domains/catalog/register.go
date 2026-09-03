package catalog

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "catalog"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"CatalogService.RunGate":               h.gate,
		"CatalogService.GetAssetRelationships": h.graph,
		"CatalogService.GetReadiness":          h.readiness,
		"corpus-report":                        h.corpusReport,
		"build":                                h.build,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("catalog: load from manifest: %w", err)
	}
	return group, nil
}
