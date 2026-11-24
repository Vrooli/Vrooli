// Package settings hosts the `audio-tools settings ...` subtree.
//
// The proto-bound command surface (provider / byok-list / byok-upsert /
// byok-delete) is declared in cli/manifest.json — the single source of
// truth. Register loads the "settings" group and wires each binding to a
// handler in handlers.go.
//
// `settings providers` is a local manifest command because it composes two
// reads into one availability matrix.
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
	providers := h.providersPrimitive()
	provider := h.providerPrimitive()
	byokList := h.byokListPrimitive()
	byokUpsert := h.byokUpsertPrimitive()
	byokDelete := h.byokDeletePrimitive()
	bindings := map[string]func(cliapp.RunContext) error{
		"SettingsService.GetProviderConfig":    provider.Run,
		"SettingsService.ListBYOKCredentials":  byokList.Run,
		"SettingsService.UpsertBYOKCredential": h.byokUpsert,
		"SettingsService.DeleteBYOKCredential": byokDelete.Run,
		"providers":                            providers.Run,
	}
	group, err := climanifest.LoadGroup(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	for i := range group.Subcommands {
		switch group.Subcommands[i].Name {
		case "providers":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(providers)
		case "provider":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(provider)
		case "byok-list":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(byokList)
		case "byok-upsert":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(byokUpsert)
		case "byok-delete":
			group.Subcommands[i] = group.Subcommands[i].WithPrimitive(byokDelete)
		}
	}
	return group, nil
}
