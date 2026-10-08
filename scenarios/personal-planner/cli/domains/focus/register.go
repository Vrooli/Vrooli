package focus

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "focus"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"FocusService.GetCurrentSession":     cliapp.ProtoList(h.currentCall, h.currentReport),
		"FocusService.StartFocus":            cliapp.ProtoMutation(h.startCall, h.startReport),
		"FocusService.PauseFocus":            cliapp.ProtoMutation(h.pauseCall, h.transitionReport),
		"FocusService.ResumeFocus":           cliapp.ProtoMutation(h.resumeCall, h.transitionReport),
		"FocusService.EndFocus":              cliapp.ProtoMutation(h.endCall, h.endReport),
		"FocusService.RecordManualActual":    cliapp.ProtoMutation(h.recordActualCall, h.recordActualReport),
		"FocusService.ListActuals":           cliapp.ProtoList(h.listActualsCall, h.listActualsReport),
		"FocusService.ListActualCorrections": cliapp.ProtoList(h.listCorrectionsCall, h.listCorrectionsReport),
		"FocusService.CorrectActual":         cliapp.ProtoMutation(h.correctActualCall, h.correctActualReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("focus: load from manifest: %w", err)
	}
	return group, nil
}
