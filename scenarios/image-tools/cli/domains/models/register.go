// Package models is the CLI's models-domain command surface. Mirrors the API's
// Connect-RPC ModelsService: the read + enable/disable verbs over the declarative
// model registry (list / get / operations / select / enable / blocklist).
//
// `models select` previews which model would run for an operation on this host
// without executing; `models enable` toggles the persisted overlay over the
// read-only seed catalog.
package models

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "models"

// Register builds the models subcommand group from the embedded manifest and
// wires Connect-RPC bindings to handlers in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ModelsService.ListModels":         h.list,
		"ModelsService.GetModel":           h.get,
		"ModelsService.ListOperations":     h.operations,
		"ModelsService.SelectModel":        h.selectModel,
		"ModelsService.ExplainResolution":  h.explain,
		"ModelsService.SetModelEnabled":    h.setEnabled,
		"ModelsService.ClearModelOverride": h.reset,
		"ModelsService.ListBlocklist":      h.blocklist,
		"ModelsService.DoctorCatalog":      h.doctor,
		"ModelsService.InstallModel":       h.install,
		"ModelsService.ProvisionModels":    h.provision,
		"ModelsService.RemoveModel":        h.remove,
		"ModelsService.AddCustomModel":     h.addCustom,
		"ModelsService.InspectModelSource": h.inspect,
		"ModelsService.ImportModel":        h.importModel,
		"ModelsService.GetModelStorage":    h.storage,
		"ModelsService.OffloadModels":      h.offload,
		"ModelsService.SetModelKeep":       h.keep,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("models: load from manifest: %w", err)
	}
	// `search` is a client-side convenience over ListModels. It reuses that RPC
	// rather than introducing a new one, so it can't be a manifest connect-rpc
	// binding (those are keyed by RPC method and `models list` already owns
	// ListModels). It is appended directly, mirroring the ai submit commands.
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "search",
		Description: "Find models by case-insensitive substring over id/name/operations",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{
				{Name: "query", Required: true, Description: "Substring to match against id, name, or operations"},
			},
		},
		RunCtx: h.search,
	})
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "inventory",
		Description: "Reconcile installed model and adapter receipts with physical storage and recorded successful use",
		NeedsAPI:    true,
		RunCtx:      h.inventory,
	})
	group.Subcommands = append(group.Subcommands,
		cliapp.Command{
			Name:        "strays",
			Description: "Find model bytes outside image-tools' receipts (library default caches, untracked managed files); changes nothing",
			NeedsAPI:    true,
			RunCtx:      h.strays,
		},
		cliapp.Command{
			Name:        "adopt-stray",
			Description: "Move a stray finding into managed storage and record a receipt",
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Finding id from `models strays`"}}},
			RunCtx:      h.strayAction("adopt"),
		},
		cliapp.Command{
			Name:        "reclaim-stray",
			Description: "Delete a stray finding that is safe to delete (a duplicate of a managed copy, or old untracked managed bytes)",
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Finding id from `models strays`"}}},
			RunCtx:      h.strayAction("reclaim"),
		},
		cliapp.Command{
			Name:        "containment",
			Description: "Show the sidecar environment pins, offline flags and the disk floor with its reasons",
			NeedsAPI:    true,
			RunCtx:      h.containment,
		},
	)
	// `unkeep` is the explicit spelling of `keep --unkeep`; it reuses the
	// SetModelKeep binding that `keep` owns in the manifest.
	group.Subcommands = append(group.Subcommands, cliapp.Command{
		Name:        "unkeep",
		Description: "Remove a model's keep pin so the adaptive offload policy may remove it when idle",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "id", Required: true, Description: "Model id"}},
		},
		RunCtx: h.unkeep,
	})
	return group, nil
}

// RootCommands reuses the manifest declarations for the two preparation entry
// points, so image-tools doctor/provision and models doctor/provision agree.
func RootCommands(core *cliapp.ScenarioApp, manifest []byte) cliapp.CommandGroup {
	group, err := Register(core, manifest)
	if err != nil {
		panic(err)
	}
	out := cliapp.CommandGroup{Title: "Preparation"}
	for _, command := range group.Subcommands {
		if command.Name == "doctor" || command.Name == "provision" {
			out.Commands = append(out.Commands, command)
		}
	}
	return out
}
