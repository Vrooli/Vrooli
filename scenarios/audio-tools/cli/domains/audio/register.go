// Package audio is the CLI's audio-domain command surface, mirroring
// vrooli.audio_tools.v1.audio.AudioProcessingService.
//
// The command surface (name/flags/governance and the Service.Method
// binding) is declared in cli/manifest.json — the single source of
// truth. Register loads the "audio" group from the manifest and wires
// each binding to a handler in handlers.go.
package audio

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "audio"

// Register builds the audio subcommand group from the embedded manifest
// and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"AudioProcessingService.Transcode": h.transcode,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("audio: load from manifest: %w", err)
	}
	return group, nil
}
