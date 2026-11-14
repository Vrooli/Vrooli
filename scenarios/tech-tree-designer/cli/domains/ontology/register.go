package ontology

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "ontology"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"OntologyService.ListCapabilities":        h.listCapabilities,
		"OntologyService.GetCapability":           h.getCapability,
		"OntologyService.UpsertCapability":        h.upsertCapability,
		"OntologyService.DeleteCapability":        h.removeCapability,
		"OntologyService.UpsertCapabilityEdge":    h.addEdge,
		"OntologyService.DeleteCapabilityEdge":    h.removeEdge,
		"OntologyService.ImportTopology":          h.importTopology,
		"OntologyService.LinkFulfillment":         h.fulfill,
		"OntologyService.UnlinkFulfillment":       h.unfulfill,
		"OntologyService.ListFulfillments":        h.listFulfillments,
		"OntologyService.GetCoverage":             h.coverage,
		"OntologyService.ListFocus":               h.focus,
		"OntologyService.GetCapabilityScenarios":  h.capabilityScenarios,
		"OntologyService.GetScenarioCapabilities": h.scenarioCapabilities,
		"OntologyService.DescribeOverlayGraph":    h.overlay,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ontology: load from manifest: %w", err)
	}
	return group, nil
}
