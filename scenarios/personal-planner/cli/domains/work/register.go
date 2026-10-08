package work

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "work"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"WorkService.ListWorkItems":  cliapp.ProtoList(h.listCall, h.listReport),
		"WorkService.CreateWorkItem": cliapp.ProtoMutation(h.createCall, h.createReport),
		"WorkService.GetWorkItem":    cliapp.ProtoList(h.getCall, h.getReport),
		"WorkService.GetTodayPlan":   cliapp.ProtoList(h.planCall, h.planReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("work: load from manifest: %w", err)
	}
	return group, nil
}
