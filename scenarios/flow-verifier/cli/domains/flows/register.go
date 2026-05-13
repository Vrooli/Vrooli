// Package flows is the CLI's flow-discovery/lifecycle command surface,
// a thin wrapper over the Connect-RPC FlowsService.
package flows

import "github.com/vrooli/cli-core/cliapp"

// Register returns the `flows` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	flowFlag := cliapp.Flag{Name: "flow", Required: true, Description: "Flow id to target"}
	kindFlag := cliapp.Flag{Name: "kind", Description: "Filter by flow kind (e.g. temporal, navigation)"}
	return cliapp.SubcommandGroup{
		Name:        "flows",
		Description: "Discover, validate, scaffold, and explain flow.json contracts",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List every discovered flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, kindFlag}},
				RunCtx:      h.list,
			},
			{
				Name:        "validate",
				Description: "Validate every flow against the embedded schema",
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					rootFlag,
					{Name: "flow", Description: "Validate only this flow id"},
					kindFlag,
				}},
				RunCtx: h.validate,
			},
			{
				Name:        "new",
				Description: "Scaffold a new flow under <feature-dir>/flow/",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{{Name: "feature-dir", Required: true, Description: "Parent directory for the new flow/"}},
					Flags: []cliapp.Flag{
						{Name: "flow-id", Required: true, Description: "Flow identifier"},
						{Name: "kind", Description: "Flow kind to scaffold (temporal|navigation); defaults to temporal"},
						{Name: "lang", Description: "Target language for temporal kinds (ts or go); defaults to ts"},
						rootFlag,
					},
				},
				RunCtx: h.create,
			},
			{
				Name:        "explain",
				Description: "Print the human-readable explain report for one flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{flowFlag, rootFlag}},
				RunCtx:      h.explain,
			},
			{
				Name:        "show",
				Description: "Print the typed flow.json projection consumed by the UI",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{flowFlag, rootFlag}},
				RunCtx:      h.show,
			},
			{
				Name:        "codegen",
				Description: "Emit codegen artifacts for a navigation flow (routes.generated.ts)",
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					flowFlag,
					rootFlag,
					{Name: "lang", Description: "Codegen target language (typescript)", Default: "typescript"},
					{Name: "write", Description: "Write artifacts to disk under the scenario root", Default: "false"},
				}},
				RunCtx: h.codegen,
			},
			{
				Name:        "reconcile",
				Description: "Reconcile a navigation flow's spec against the scenario's ui/src tree",
				Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
					flowFlag,
					rootFlag,
					{Name: "scenario", Description: "Scenario root containing ui/src (defaults to derived from contract path)"},
				}},
				RunCtx: h.reconcile,
			},
			{
				Name:        "studio",
				Description: "Print the Flow Studio descriptor (routes, affordances, containers, context toggles, invariant pass/fail) for one navigation flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{flowFlag, rootFlag}},
				RunCtx:      h.studio,
			},
		},
	}
}
