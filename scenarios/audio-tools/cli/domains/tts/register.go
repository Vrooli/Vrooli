// Package tts is the CLI's TTS-domain command surface, mirroring
// vrooli.audio_tools.v1.tts.TTSService.
package tts

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "tts",
		Description: "Text-to-speech operations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "synthesize",
				Description: "Synthesize speech audio from text",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "text", Required: true, Description: "Text to synthesize"},
						{Name: "voice", Description: "Canonical voice id (default voice.neutral.default)"},
						{Name: "speed", Description: "Playback speed (default 1.0)"},
						{Name: "format", Description: "Output format mp3|wav|opus|flac (default mp3)"},
						{Name: "out", Description: "Output file path (default stdout)"},
					},
				},
				RunCtx: h.synthesize,
			},
			{
				Name:        "synthesize-stream",
				Description: "Stream-synthesize speech audio (writes frames to --out as they arrive)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "text", Required: true, Description: "Text to synthesize"},
						{Name: "voice", Description: "Canonical voice id (default voice.neutral.default)"},
						{Name: "speed", Description: "Playback speed (default 1.0)"},
						{Name: "format", Description: "Output format mp3|wav|opus|flac (default mp3)"},
						{Name: "out", Description: "Output file path (default stdout)"},
					},
				},
				RunCtx: h.synthesizeStream,
			},
			{
				Name:        "voices",
				Description: "List canonical voices",
				RunCtx:      h.voices,
			},
		},
	}
}
