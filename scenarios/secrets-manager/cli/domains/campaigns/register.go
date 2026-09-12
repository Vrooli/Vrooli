package campaigns

import (
	"fmt"

	"secrets-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "campaigns",
		Description: "Deployment-readiness campaign inventory",
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List deployment campaigns", Run: func(args []string) error { return runList(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaigns list")
	scenario := fs.String("scenario", "", "Filter by scenario")
	includeReadiness := fs.Bool("include-readiness", false, "Include readiness summaries")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.Query("scenario", *scenario)
	if query == nil {
		query = make(map[string][]string)
	}
	if *includeReadiness {
		query.Set("include_readiness", "true")
	}

	var resp support.CampaignListResponse
	if err := support.GetJSON(core, "/campaigns", query, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Campaigns))
	for _, campaign := range resp.Campaigns {
		line := fmt.Sprintf("%s | %s | %s | progress=%d | blockers=%d | updated=%s",
			campaign.Scenario, campaign.Tier, campaign.Status, campaign.Progress, campaign.Blockers, support.FormatTime(campaign.UpdatedAt))
		if campaign.NextAction != "" {
			line += " | next: " + campaign.NextAction
		}
		results = append(results, line)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Campaigns: %d", resp.Count)},
		ResultsHeading: "Campaigns",
		Results:        results,
		RetrievalHints: []string{support.CLIName + " deployment readiness --scenario <scenario>", support.CLIName + " deployment plan --scenario <scenario>"},
	}
	return support.PrintList(*jsonOutput, resp, report)
}
