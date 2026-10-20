package media

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "media"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ProjectsService.ListSourceMedia": cliapp.ProtoList(h.listCall, h.listReport),
		"ProjectsService.GetSourceMedia":  cliapp.ProtoList(h.getCall, h.getReport),
	}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("media: load manifest: %w", err)
	}
	g.Subcommands = append(g.Subcommands, cliapp.Command{Name: "import", Description: "Import a local PNG or JPEG into a project", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "project-id", Required: true, Description: "Project id"}}, Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "Path to PNG or JPEG"}}}, Architecture: cliapp.CommandArchitecture{Exception: cliapp.ExceptionUpload, ExceptionReason: "REST multipart source image upload"}}.WithPrimitive(cliapp.Upload(h.importCall, h.importReport)))
	return g, nil
}
