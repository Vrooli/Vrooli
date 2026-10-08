package source

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "source"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"SourceRepositoryService.AnalyzeClosure":     cliapp.ProtoList(h.analyzeCall, h.analyzeReport),
		"SourceRepositoryService.AssembleExport":     cliapp.ProtoMutation(h.assembleCall, h.assembleReport),
		"SourceRepositoryService.VerifyExport":       cliapp.ProtoList(h.verifyCall, h.verifyReport),
		"SourceRepositoryService.PreparePublication": cliapp.ProtoMutation(h.preparePublicationCall, h.preparePublicationReport),
		"SourceRepositoryService.GetDistribution":    cliapp.ProtoList(h.getDistributionCall, h.getDistributionReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("source: load manifest: %w", err)
	}
	return group, nil
}
