// Package tts is the CLI's TTS-domain command surface, mirroring
// vrooli.audio_tools.v1.tts.TTSService.
//
// The command surface is declared in cli/manifest.json — the single
// source of truth. Register loads the "tts" group and wires each
// binding to a handler in handlers.go.
package tts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "tts"

// Register builds the tts subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"TTSService.Synthesize":          h.synthesize,
		"TTSService.SynthesizeStream":    h.synthesizeStream,
		"TTSService.ListVoices":          h.voices,
		"TTSService.GetSupportedFormats": h.formats,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("tts: load from manifest: %w", err)
	}
	return group, nil
}
