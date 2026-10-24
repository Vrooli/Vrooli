package catalog

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
)

const GroupName = "catalog"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"CatalogService.RunGate": cliapp.ProtoListOutcome(h.gateCall, h.gateReport, func(resp *catalogv1.RunGateResponse) error {
			if resp.NonDiscriminating {
				return fmt.Errorf("catalog gate %s is non-discriminating", resp.Gate)
			}
			return nil
		}),
		"CatalogService.GetAssetRelationships": cliapp.ProtoList(h.graphCall, h.graphReport),
		"CatalogService.GetReadiness":          cliapp.ProtoOperational(h.readinessCall, h.readinessReport),
		"corpus-report":                        {Run: h.corpusReport},
		"build":                                {Run: h.build},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("catalog: load from manifest: %w", err)
	}
	return group, nil
}
