// Package summarize is the CLI's summarize-domain command surface,
// mirroring vrooli.audio_tools.v1.summarize.SummarizeService.
package summarize

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "summarize",
		Description: "Text summarization via the audio-tools summarize chain",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "text",
				Description: "Summarize text from --text or stdin",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "text", Required: true, Description: "Text to summarize"},
						{Name: "level", Description: "light|moderate|heavy (default moderate)"},
					},
				},
				RunCtx: h.text,
			},
		},
	}
}
