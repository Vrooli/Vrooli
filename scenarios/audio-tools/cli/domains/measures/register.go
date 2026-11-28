package measures

import (
	"audio-tools/cli/internal/climanifest"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "measure"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"MeasuresService.CountBYOKCredentials":            h.count("CountBYOKCredentials", "credential capabilities"),
		"MeasuresService.CountCorpusClips":                h.count("CountCorpusClips", "corpus clips"),
		"MeasuresService.CountExperimentRuns":             h.count("CountExperimentRuns", "experiment runs"),
		"MeasuresService.CountExperiments":                h.count("CountExperiments", "experiments"),
		"MeasuresService.CountSpeakerProfiles":            h.count("CountSpeakerProfiles", "speaker profiles"),
		"MeasuresService.CountWakewordTemplates":          h.count("CountWakewordTemplates", "wakeword templates"),
		"MeasuresService.CountDuplexAccountingOperations": h.count("CountDuplexAccountingOperations", "duplex accounting operations"),
	}
	return climanifest.LoadPrimitiveGroup(manifest, GroupName, bindings)
}
