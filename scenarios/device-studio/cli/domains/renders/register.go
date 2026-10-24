package renders

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "renders"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"RenderJobsService.SubmitRender":          cliapp.ProtoMutation(h.submitCall, h.submitReport),
		"RenderJobsService.GetRender":             cliapp.ProtoList(h.getCall, h.getReport),
		"RenderJobsService.ListRenders":           cliapp.ProtoList(h.listCall, h.listReport),
		"RenderJobsService.CancelRender":          cliapp.ProtoMutation(h.cancelCall, h.cancelReport),
		"RenderJobsService.GetVideoSampleBinding": cliapp.ProtoList(h.sampleBindingCall, h.sampleBindingReport),
		"export": cliapp.ProtoMutation(h.exportCall, h.exportReport),
	}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("renders: load manifest: %w", err)
	}
	return g, nil
}
