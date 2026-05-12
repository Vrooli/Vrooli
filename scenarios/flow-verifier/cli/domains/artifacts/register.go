// Package artifacts is the CLI's codegen-lifecycle command surface,
// a thin wrapper over the Connect-RPC ArtifactsService (per-flow) and
// ScenariosService (scenario-wide + streaming).
package artifacts

import "github.com/vrooli/cli-core/cliapp"

// Register returns the `artifacts` subcommand group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	rootFlag := cliapp.Flag{Name: "root", Description: "Repository root to scan (default: cwd)", Default: "."}
	flowFlag := cliapp.Flag{Name: "flow", Description: "Flow id to target"}
	scenarioFlag := cliapp.Flag{Name: "scenario", Description: "Scenario id to target every flow inside"}
	allFlag := cliapp.Flag{Name: "all", Description: "Apply to every discovered flow under --scenario", Default: "false"}
	yesFlag := cliapp.Flag{Name: "yes", Description: "Skip the confirmation prompt for bulk clears", Default: "false"}

	return cliapp.SubcommandGroup{
		Name:        "artifacts",
		Description: "Inspect, generate, or clear a flow's generated/ tree",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "status",
				Description: "Inspect the on-disk generated/ tree for one flow",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag, scenarioFlag, allFlag}},
				RunCtx:      h.status,
			},
			{
				Name:        "generate",
				Description: "Generate or regenerate one flow's artifacts (or every flow with --scenario)",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag, scenarioFlag, allFlag}},
				RunCtx:      h.generate,
			},
			{
				Name:        "clear",
				Description: "Remove one flow's generated/ tree (--scenario requires --yes)",
				Args:        cliapp.ArgSchema{Flags: []cliapp.Flag{rootFlag, flowFlag, scenarioFlag, allFlag, yesFlag}},
				RunCtx:      h.clear,
			},
		},
	}
}
