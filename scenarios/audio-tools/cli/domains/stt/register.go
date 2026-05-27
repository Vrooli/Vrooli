// Package stt hosts the `audio-tools stt ...` subtree (speech-to-text).
package stt

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "stt",
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
				Name:        "formats",
				Description: "Show the STT ingress audio-format capability matrix (accepted input codecs + ffmpeg/canonical-PCM status).",
				RunCtx:      h.formats,
			},
			{
				Name:        "engines",
				Description: "List selectable STT engines (manifest-derived) with availability and the active selection.",
				RunCtx:      h.engines,
			},
			{
				Name:        "engine-impact",
				Description: "Show the shared-resource impact of switching away from an engine (which other scenarios still use its resource + the stop command). Never stops anything.",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "engine", Required: true, Description: "Engine id to assess (see `stt engines`)"},
					},
				},
				RunCtx: h.engineImpact,
			},
			{
				Name:        "stream-config",
				Description: "Show the resolved streaming STT configuration (engine + operator levers + egress gate + legacy partial-window fields).",
				RunCtx:      h.streamConfigGet,
			},
			{
				Name:        "stream-config-set",
				Description: "Update one or more streaming STT levers. Only provided flags are mutated.",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "engine", Description: "Active STT engine id (see `stt engines`); only the Local tier honors it"},
						{Name: "streaming-mode", Description: "auto|off — master switch (default auto)"},
						{Name: "strategy-preference", Description: "auto|vad|overlap|passthrough — strategy hint (default auto)"},
						{Name: "vad-silence-ms", Description: "VAD silence window in ms; range 200–3000 (default 1200)"},
						{Name: "overlap-window-ms", Description: "Sliding-window size for overlap-and-agree; range 1000–5000 (default 2000)"},
						{Name: "overlap-commit-runs", Description: "Consecutive agreeing runs to commit a prefix; range 2–4 (default 2)"},
						{Name: "hallucination-filter", Description: "true|false — drop known Whisper silence-hallucination phrases (default true)"},
						{Name: "vad-filter", Description: "true|false — enable faster-whisper's built-in silence filter on /asr (default true)"},
						{Name: "no-speech-threshold", Description: "Egress drop when mean no_speech_prob exceeds this AND avg_logprob is below logprob-threshold; range (0,1] (default 0.6)"},
						{Name: "logprob-threshold", Description: "Paired with no-speech-threshold; range [-10,0) (default -1.0)"},
						{Name: "denoise", Description: "true|false — pre-recognition ffmpeg afftdn denoise on the canonical-PCM stream (Whisper vad/overlap only; needs ffmpeg). Default off"},
					},
				},
				RunCtx: h.streamConfigSet,
			},
			{
				Name:        "speaker-status",
				Description: "Show speaker-verification config, resource capability, and enrolled profiles. Notes the Whisper-only protection caveat.",
				RunCtx:      h.speakerStatus,
			},
			{
				Name:        "speaker-config",
				Description: "Update speaker-verification config. Only provided flags are mutated.",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "mode", Description: "off|filter|advisory — verification mode (off disables gating)"},
						{Name: "threshold", Description: "Cosine-similarity accept threshold; 0..1 (default 0.7; same-speaker ~0.9, different ~0.1)"},
						{Name: "enabled", Description: "true|false — master enable for speaker verification"},
						{Name: "profiles", Description: "Comma-separated profile ids to set as the active binding (replaces the list)"},
						{Name: "bind-profile", Description: "Append a single profile id to the active binding (reads current list first)"},
						{Name: "reject-behavior", Description: "drop|show-muted — what to do with a rejected segment (default drop)"},
						{Name: "fallback", Description: "true|false — let audio through when no profile/resource is available"},
						{Name: "extraction-enabled", Description: "true|false — isolate the enrolled speaker's voice BEFORE recognition (ingress target extraction; needs an enrolled profile + the extraction-capable resource; default false)"},
					},
				},
				RunCtx: h.speakerConfig,
			},
			{
				Name:        "speaker-enroll",
				Description: "Enroll a voice profile from an audio file. Use --activate to bind+enable in one step.",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "file", Required: true, Description: "Audio file path (wav recommended)"},
						{Name: "profile", Description: "Profile id (empty = server-generated)"},
						{Name: "label", Description: "Human display name for the profile"},
						{Name: "notes", Description: "Optional notes stored with the profile"},
						{Name: "format", Description: "Audio format hint (default wav)"},
						{Name: "activate", Description: "true|false — bind as active profile and enable verification (default off)"},
					},
				},
				RunCtx: h.speakerEnroll,
			},
		},
	}
}
