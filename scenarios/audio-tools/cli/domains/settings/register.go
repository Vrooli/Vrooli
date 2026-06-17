// Package settings hosts the `audio-tools settings ...` subtree.
//
// The proto-bound command surface (provider / byok-list / byok-upsert /
// byok-delete) is declared in cli/manifest.json — the single source of
// truth. Register loads the "settings" group and wires each binding to a
// handler in handlers.go.
//
// `settings providers` is a client-side convenience that composes two
// reads (SettingsService.GetProviderConfig — already bound to `settings
// provider` — and TTSService.GetStatus) into one availability matrix. It
// has no unique RPC of its own, so (like image-tools' `models search`)
// it is hand-appended here rather than declared as a manifest
// connect-rpc binding.
package settings

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
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
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("settings: load from manifest: %w", err)
	}
	// `providers` is a client-side composite over GetProviderConfig +
	// TTSService.GetStatus; it reuses those RPCs rather than introducing
	// a new one, so it can't be a manifest connect-rpc binding (those are
	// keyed by RPC method and `settings provider` already owns
	// GetProviderConfig). It is appended directly.
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "providers",
		Description: "Print the per-tier provider-availability matrix (routing config + TTS reachability)",
		NeedsAPI:    true,
		RunCtx:      h.providers,
	})
	return group, nil
}
