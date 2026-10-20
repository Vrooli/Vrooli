package shots

import (
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "shots"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ShotsService.CreateShot":        cliapp.ProtoMutation(h.createCall, h.createReport),
		"ShotsService.SaveShot":          cliapp.ProtoMutation(h.saveCall, h.saveReport),
		"ShotsService.ListShots":         cliapp.ProtoList(h.listCall, h.listReport),
		"ShotsService.GetShot":           cliapp.ProtoList(h.getCall, h.getReport),
		"ShotsService.GetShotRevision":   cliapp.ProtoList(h.revisionCall, h.revisionReport),
		"ShotsService.ListShotRevisions": cliapp.ProtoList(h.revisionsCall, h.revisionsReport),
		"ShotsService.CompileShot":       cliapp.ProtoList(h.compileCall, h.compileReport),
		"ShotsService.ValidateShot":      cliapp.ProtoList(h.validateCall, h.validateReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("shots: load manifest: %w", err)
	}
	return group, nil
}
