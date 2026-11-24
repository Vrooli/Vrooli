// Package audio is the CLI's audio-domain command surface, mirroring
// vrooli.audio_tools.v1.audio.AudioProcessingService.
//
// The command surface (name/flags/governance and the Service.Method
// binding) is declared in cli/manifest.json — the single source of
// truth. Register loads the "audio" group from the manifest and wires
// each binding to handlers.go.
package audio

import (
	"audio-tools/cli/internal/climanifest"
	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "audio"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	primitives := map[string]cliapp.PrimitiveHandler{
		"AudioProcessingService.Transcode": h.transcodePrimitive(), "AudioProcessingService.Trim": h.trimPrimitive(), "AudioProcessingService.Merge": h.mergePrimitive(), "AudioProcessingService.Split": h.splitPrimitive(), "AudioProcessingService.Fade": h.fadePrimitive(), "AudioProcessingService.Volume": h.volumePrimitive(), "AudioProcessingService.Normalize": h.normalizePrimitive(), "AudioProcessingService.ExtractMetadata": h.extractMetadataPrimitive(),
	}
	bindings := map[string]func(cliapp.RunContext) error{}
	for binding, primitive := range primitives {
		bindings[binding] = primitive.Run
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		method := map[string]string{"transcode": "Transcode", "trim": "Trim", "merge": "Merge", "split": "Split", "fade": "Fade", "volume": "Volume", "normalize": "Normalize", "extract-metadata": "ExtractMetadata"}[group.Subcommands[i].Name]
		if primitive, ok := primitives["AudioProcessingService."+method]; ok {
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(primitive)
		}
	}
	return group, nil
}
