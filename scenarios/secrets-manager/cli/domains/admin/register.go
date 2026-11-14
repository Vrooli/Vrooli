package admin

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "admin",
		Description: "Administrative override cleanup operations",
		Subcommands: []cliapp.Command{
			{Name: "orphans", NeedsAPI: true, Description: "List orphaned overrides", Run: func(args []string) error { return runOrphans(core, args) }},
			{Name: "cleanup-orphans", NeedsAPI: true, Description: "Delete orphaned overrides or preview the cleanup", Run: func(args []string) error { return runCleanupOrphans(core, args) }},
		},
	}
}

func runOrphans(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("admin orphans")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var resp support.OrphansResponse
	if err := support.GetJSON(core, "/admin/overrides/orphans", nil, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Orphans))
	for _, item := range resp.Orphans {
		results = append(results, fmt.Sprintf("%s/%s | scenario=%s | tier=%s | %s",
			item.Override.ResourceName, item.Override.SecretKey, item.Override.ScenarioName, item.Override.Tier, item.Reason))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Orphan overrides: %d", resp.Count)},
		ResultsHeading: "Orphans",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " admin cleanup-orphans --dry-run", support.CLIName + " admin cleanup-orphans"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}

func runCleanupOrphans(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("admin cleanup-orphans")
	dryRun := fs.Bool("dry-run", false, "Preview what would be deleted")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload := map[string]any{"dry_run": *dryRun}
	var resp support.OverrideMutationResponse
	if err := support.RequestJSON(core, "POST", "/admin/overrides/cleanup", nil, payload, &resp); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			"Orphan cleanup completed",
			fmt.Sprintf("Dry run: %t", resp.Success && *dryRun),
		},
		Changes: []string{
			fmt.Sprintf("Deleted: %d", resp.Deleted),
			fmt.Sprintf("Would delete: %d", resp.WouldDelete),
		},
		NextCommand: []string{support.CLIName + " admin orphans"},
	}
	return support.PrintMutation(*jsonOutput, resp, report)
}
