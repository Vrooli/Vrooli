// Package diagnostics is the CLI's diagnostics-domain command surface,
// mirroring vrooli.audio_tools.v1.diagnostics.DiagnosticsService.
package diagnostics

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the diagnostics SubcommandGroup.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "diagnostics",
		Description: "Operator-facing capability suite (STT/TTS/Summarize/Transcode) against bundled fixtures",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "run",
				Description: "Run the diagnostics suite and print the per-capability result table (or --json envelope)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "capability", Description: "Comma-separated subset (stt,tts,summarize,transcode); empty runs all"},
					},
				},
				RunCtx: h.run,
			},
			{
				Name:        "last",
				Description: "Print the most recent suite result (or --json envelope)",
				RunCtx:      h.last,
			},
		},
	}
}
