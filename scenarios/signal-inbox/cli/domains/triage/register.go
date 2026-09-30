package triage

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "triage"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"TriageService.GetTriage":      cliapp.ProtoList(h.getCall, h.getReport),
		"TriageService.SetDisposition": cliapp.ProtoMutation(h.setCall, h.setReport),
		"TriageService.AddAnnotation":  cliapp.ProtoMutation(h.annotateCall, h.annotateReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("triage: load manifest: %w", err)
	}
	return group, nil
}
