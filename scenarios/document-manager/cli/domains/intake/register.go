package intake

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "intake"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"IntakeService.Ingest":         cliapp.ProtoMutation(h.ingestCall, h.ingestReport),
		"IntakeService.GetDocument":    cliapp.ProtoList(h.getCall, h.getReport),
		"IntakeService.ListDocuments":  cliapp.ProtoList(h.listCall, h.listReport),
		"IntakeService.ListSources":    cliapp.ProtoList(h.sourcesCall, h.sourcesReport),
		"IntakeService.ConfigureWatch": cliapp.ProtoMutation(h.watchCall, h.watchReport),
		"IntakeService.GetTypeVerdict": cliapp.ProtoList(h.verdictCall, h.verdictReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("intake: load from manifest: %w", err)
	}
	return group, nil
}
