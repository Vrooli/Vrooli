// Package experiment hosts the `audio-tools experiment ...` subtree for
// persisted async STT experiment runs. The command surface is declared in
// cli/manifest.json; Register binds ExperimentService RPCs to handlers.go.
package experiment

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "experiment"

// Register builds the experiment subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ExperimentService.StartExperiment":             h.start,
		"ExperimentService.GetExperiment":               h.get,
		"ExperimentService.WaitExperiment":              h.wait,
		"ExperimentService.ListExperiments":             h.list,
		"ExperimentService.CancelExperiment":            h.cancel,
		"ExperimentService.DeleteExperiment":            h.delete,
		"ExperimentService.StreamExperimentEvents":      h.watch,
		"ExperimentService.GetExperimentReport":         h.report,
		"ExperimentService.CompareExperiments":          h.compare,
		"ExperimentService.RecordQualificationEvidence": h.recordEvidence,
		"ExperimentService.ListQualificationEvidence":   h.listEvidence,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	primitives := map[string]cliapp.PrimitiveHandler{
		"get": h.getPrimitive(), "report": h.reportPrimitive(), "list": h.listPrimitive(),
		"compare": h.comparePrimitive(), "list-evidence": h.listEvidencePrimitive(),
		"start": h.startPrimitive(), "wait": h.waitPrimitive(), "cancel": h.cancelPrimitive(),
		"delete": h.deletePrimitive(), "record-evidence": h.recordEvidencePrimitive(), "watch": h.watchPrimitive(),
	}
	for i := range group.Subcommands {
		if primitive, ok := primitives[group.Subcommands[i].Name]; ok {
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(primitive)
		}
	}
	return group, nil
}
