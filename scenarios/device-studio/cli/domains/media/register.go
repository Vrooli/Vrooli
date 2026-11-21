package media

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "media"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ProjectsService.ListSourceMedia":             cliapp.ProtoList(h.listCall, h.listReport),
		"ProjectsService.GetSourceMedia":              cliapp.ProtoList(h.getCall, h.getReport),
		"ProjectsService.EnsureSourceMediaInspection": cliapp.ProtoMutation(h.inspectCall, h.inspectReport),
		"ProjectsService.RetrySourceMediaInspection":  cliapp.ProtoMutation(h.retryInspectionCall, h.retryInspectionReport),
		"ProjectsService.CancelSourceMediaInspection": cliapp.ProtoMutation(h.cancelInspectionCall, h.cancelInspectionReport),
		"ProjectsService.GetSourceMediaClockPage":     cliapp.ProtoList(h.clockCall, h.clockReport),
	}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("media: load manifest: %w", err)
	}
	g.Subcommands = append(g.Subcommands, cliapp.Command{Name: "import", Description: "Import an image or recording; recordings start local inspection", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "project-id", Required: true, Description: "Project id"}}, Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "Path to PNG, JPEG, MP4, MOV, WebM or Matroska media"}}}, Architecture: cliapp.CommandArchitecture{Exception: cliapp.ExceptionUpload, ExceptionReason: "REST multipart source media upload"}}.WithPrimitive(cliapp.Upload(h.importCall, h.importReport)))
	return g, nil
}
