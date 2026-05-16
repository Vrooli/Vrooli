package voice

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "voice",
		Description: "Speech-to-text operations",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "transcribe",
				Description: "Transcribe an audio file",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "file", Required: true, Description: "Audio file path"},
						{Name: "language", Description: "Language hint (empty = auto-detect)"},
						{Name: "format", Description: "Audio format hint (default wav)"},
					},
				},
				RunCtx: h.transcribe,
			},
		},
	}
}
