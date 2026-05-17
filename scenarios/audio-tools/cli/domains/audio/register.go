// Package audio is the CLI's audio-domain command surface, mirroring
// vrooli.audio_tools.v1.audio.AudioProcessingService.
package audio

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "audio",
		Description: "Audio processing (transcode, trim, merge, ...)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "transcode",
				Description: "Transcode an audio file to WAV (16 kHz mono)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "input", Required: true, Description: "Input audio path"},
						{Name: "output", Required: true, Description: "Output path"},
					},
				},
				RunCtx: h.transcode,
			},
		},
	}
}
