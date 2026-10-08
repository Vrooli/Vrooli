// Package settings hosts the `audio-tools settings ...` subtree.
//
// The proto-bound command surface (provider / byok-list / byok-upsert /
// byok-delete) is declared in cli/manifest.json — the single source of
// truth. Register loads the "settings" group and wires each binding to a
// handler in handlers.go.
//
// `settings providers` is a declared manifest exception because it composes
// two reads into one availability matrix and has no single RPC binding.
package settings

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "settings"

// Register builds the settings subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SettingsService.GetProviderConfig":    h.provider,
		"SettingsService.ListBYOKCredentials":  h.byokList,
		"SettingsService.UpsertBYOKCredential": h.byokUpsert,
		"SettingsService.DeleteBYOKCredential": h.byokDelete,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	// `providers` is the manifest-declared client-side composite over
	// GetProviderConfig and TTSService.GetStatus. It is appended directly
	// because `settings provider` already owns the former RPC binding.
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "providers",
		Description: "Print the per-tier provider-availability matrix (routing config + TTS reachability)",
		NeedsAPI:    true,
		RunCtx:      h.providers,
	})
	return group, nil
}
