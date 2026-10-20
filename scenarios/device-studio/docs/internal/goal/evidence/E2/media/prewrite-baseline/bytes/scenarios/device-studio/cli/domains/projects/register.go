package projects

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "projects"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ProjectsService.ListProjects":  cliapp.ProtoList(h.listCall, h.listReport),
		"ProjectsService.CreateProject": cliapp.ProtoMutation(h.createCall, h.createReport),
		"ProjectsService.GetProject":    cliapp.ProtoList(h.getCall, h.getReport),
		"ProjectsService.RenameProject": cliapp.ProtoMutation(h.renameCall, h.renameReport),
	}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("projects: load manifest: %w", err)
	}
	return g, nil
}
