// Package provider is the CLI's provider_lifecycle-domain command
// surface, mirroring
// vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService.
//
// `--dry-run` is provided globally by cli-core (cliapp.GlobalFlags);
// when set, every mutating subcommand emits the X-Dry-Run: true header
// automatically through cliapp.NewConnectHTTPClient's request pipeline.
package provider

import (
	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the provider SubcommandGroup. Default output is the
// human-friendly table (feedback_cli_default_human_output); --json is
// provided globally by cli-core.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "provider",
		Description: "Lifecycle actions on local-tier providers (start/stop/restart, pull-model on ollama, stream logs)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List local providers, their current process_state, and supported actions",
				RunCtx:      h.list,
			},
			{
				Name:        "start",
				Description: "Start a local provider (use --dry-run to preview)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "provider-id", Description: "Provider id (e.g. whisper-stt, kokoro-tts, ollama, speaker-verification)", Required: true}},
				},
				RunCtx: h.start,
			},
			{
				Name:        "stop",
				Description: "Stop a local provider (use --dry-run to preview)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}},
				},
				RunCtx: h.stop,
			},
			{
				Name:        "restart",
				Description: "Restart a local provider (use --dry-run to preview)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}},
				},
				RunCtx: h.restart,
			},
			{
				Name:        "pull-model",
				Description: "Pull a model on the ollama provider (ollama-only; use --dry-run to preview)",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "model-name", Description: "Model identifier accepted by `ollama pull` (e.g. phi3:mini)", Required: true}},
				},
				RunCtx: h.pullModel,
			},
			{
				Name:        "logs",
				Description: "Stream stdout/stderr lines from a provider's backing resource",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "provider-id", Required: true}},
					Flags: []cliapp.Flag{
						{Name: "follow", Bool: true, Description: "Keep the stream open (tail -f behavior)"},
						{Name: "tail", Description: "Number of historical lines to emit before tailing"},
					},
				},
				RunCtx: h.logs,
			},
		},
	}
}
