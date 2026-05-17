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
			{
				Name:        "transcribe-stream",
				Description: "Stream-transcribe an audio file (emits one line per stream event)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "file", Required: true, Description: "Audio file path"},
						{Name: "language", Description: "Language hint (empty = auto-detect)"},
						{Name: "chunk-bytes", Description: "Chunk size to feed the stream in bytes (default 32768)"},
					},
				},
				RunCtx: h.transcribeStream,
			},
			{
				Name:        "stream-config",
				Description: "Show the resolved streaming STT configuration (5 operator levers + legacy partial-window fields).",
				RunCtx:      h.streamConfigGet,
			},
			{
				Name:        "stream-config-set",
				Description: "Update one or more streaming STT levers. Only provided flags are mutated.",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "streaming-mode", Description: "auto|off — master switch (default auto)"},
						{Name: "strategy-preference", Description: "auto|vad|overlap|passthrough — strategy hint (default auto)"},
						{Name: "vad-silence-ms", Description: "VAD silence window in ms; range 200–3000 (default 700)"},
						{Name: "overlap-window-ms", Description: "Sliding-window size for overlap-and-agree; range 1000–5000 (default 2000)"},
						{Name: "overlap-commit-runs", Description: "Consecutive agreeing runs to commit a prefix; range 2–4 (default 2)"},
					},
				},
				RunCtx: h.streamConfigSet,
			},
		},
	}
}
