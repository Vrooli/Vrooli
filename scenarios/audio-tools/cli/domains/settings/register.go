// Package settings hosts the `audio-tools settings ...` subtree.
package settings

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Provider routing, BYOK credentials, voice overrides",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "provider",
				Description: "Show the current provider routing config",
				RunCtx:      h.provider,
			},
			{
				Name:        "byok-list",
				Description: "List stored BYOK credentials (redacted)",
				RunCtx:      h.byokList,
			},
			{
				Name:        "byok-upsert",
				Description: "Add or replace a BYOK credential",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "provider", Required: true, Description: "Provider id (e.g. openai-tts)"},
						{Name: "capability", Required: true, Description: "stt | tts | summarize"},
						{Name: "key", Required: true, Description: "API key value"},
					},
				},
				RunCtx: h.byokUpsert,
			},
			{
				Name:        "byok-delete",
				Description: "Delete a BYOK credential",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "provider", Required: true, Description: "Provider id"},
						{Name: "capability", Required: true, Description: "stt | tts | summarize"},
					},
				},
				RunCtx: h.byokDelete,
			},
		},
	}
}
