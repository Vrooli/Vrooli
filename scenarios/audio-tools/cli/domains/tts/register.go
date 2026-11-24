// Package tts is the CLI's TTS-domain command surface, mirroring
// vrooli.audio_tools.v1.tts.TTSService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "tts" group and wires each
// binding to a handler in handlers.go.
package tts

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "tts"

// Register builds the tts subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	synthesize := h.synthesizePrimitive()
	streaming := cliapp.Streaming(h.synthesizeStream)
	voices := h.voicesPrimitive()
	formats := h.formatsPrimitive()
	bindings := map[string]func(cliapp.RunContext) error{
		"TTSService.Synthesize":          synthesize.Run,
		"TTSService.SynthesizeStream":    streaming.Run,
		"TTSService.ListVoices":          voices.Run,
		"TTSService.GetSupportedFormats": formats.Run,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "synthesize":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(synthesize)
		case "synthesize-stream":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(streaming)
		case "voices":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(voices)
		case "formats":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(formats)
		}
	}
	return group, nil
}
