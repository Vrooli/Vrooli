package campaigns

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"tidiness-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const cliName = "tidiness-manager"

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "campaigns",
		Description: "Manage auto-tidiness campaigns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List campaigns", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one campaign", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "start", Description: "Start a campaign", Run: func(args []string) error { return runStart(core, args) }},
			{Name: "pause", Description: "Pause a campaign", Run: func(args []string) error { return runAction(core, args, "pause") }},
			{Name: "resume", Description: "Resume a campaign", Run: func(args []string) error { return runAction(core, args, "resume") }},
			{Name: "stop", Aliases: []string{"terminate"}, Description: "Stop a campaign", Run: func(args []string) error { return runAction(core, args, "terminate") }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaigns list")
	status := fs.String("status", "", "Optional campaign status filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if strings.TrimSpace(*status) != "" {
		query.Set("status", *status)
	}

	body, err := core.Get("/campaigns", query)
	if err != nil {
		return err
	}

	var response support.CampaignsResponse
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Campaigns: %d", response.Count),
		},
		ResultsHeading: "Campaigns",
		Results:        campaignRows(response.Campaigns),
		RetrievalHints: []string{fmt.Sprintf("%s campaigns start <scenario> --max-sessions 10 --max-files 5", cliName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaigns get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: campaigns get <campaign-id>")
	}
	campaignID := fs.Arg(0)

	body, err := core.Get("/campaigns/"+campaignID, nil)
	if err != nil {
		return err
	}

	var response support.CampaignEnvelope
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Campaign ID: %d", response.Campaign.ID),
			fmt.Sprintf("Scenario: %s", response.Campaign.Scenario),
			fmt.Sprintf("Status: %s", response.Campaign.Status),
		},
		ResultsHeading: "Details",
		Results: []string{
			fmt.Sprintf("Sessions: %d/%d", response.Campaign.CurrentSession, response.Campaign.MaxSessions),
			fmt.Sprintf("Files visited: %d/%d", response.Campaign.FilesVisited, response.Campaign.FilesTotal),
			fmt.Sprintf("Files per session: %d", response.Campaign.MaxFilesPerSession),
		},
		RetrievalHints: []string{fmt.Sprintf("%s campaigns pause --id %d", cliName, response.Campaign.ID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runStart(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaigns start")
	maxSessions := fs.Int("max-sessions", 10, "Maximum sessions to run")
	maxFiles := fs.Int("max-files", 5, "Maximum files per session")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: campaigns start <scenario> [--max-sessions N] [--max-files N]")
	}
	scenario := fs.Arg(0)

	body, err := core.Request("POST", "/campaigns", nil, map[string]interface{}{
		"scenario":              scenario,
		"max_sessions":          *maxSessions,
		"max_files_per_session": *maxFiles,
	})
	if err != nil {
		return err
	}

	var response support.CampaignEnvelope
	if err := support.Decode(body, &response); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Campaign %d started", response.Campaign.ID),
			fmt.Sprintf("Scenario: %s", response.Campaign.Scenario),
		},
		Changes: []string{
			fmt.Sprintf("Status: %s", response.Campaign.Status),
			fmt.Sprintf("Max sessions: %d", response.Campaign.MaxSessions),
			fmt.Sprintf("Max files per session: %d", response.Campaign.MaxFilesPerSession),
		},
		NextCommand: []string{
			fmt.Sprintf("%s campaigns get %d", cliName, response.Campaign.ID),
			fmt.Sprintf("%s campaigns pause --id %d", cliName, response.Campaign.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runAction(core *cliapp.ScenarioApp, args []string, action string) error {
	fs := support.NewFlagSet("campaigns action")
	campaignID := fs.Int("id", 0, "Campaign ID")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	scenario := ""
	if fs.NArg() > 0 {
		scenario = fs.Arg(0)
	}
	resolvedID := *campaignID
	if resolvedID == 0 {
		if strings.TrimSpace(scenario) == "" {
			return fmt.Errorf("either a scenario name or --id is required")
		}
		allCampaigns, err := listCampaigns(core)
		if err != nil {
			return err
		}
		preferredStatus := ""
		switch action {
		case "pause", "terminate":
			preferredStatus = "active"
		case "resume":
			preferredStatus = "paused"
		}
		resolvedID, err = support.ResolveCampaignID(allCampaigns, scenario, preferredStatus)
		if err != nil {
			return err
		}
	}

	body, err := core.Request("POST", "/campaigns/"+strconv.Itoa(resolvedID)+"/action", nil, map[string]interface{}{
		"action": action,
	})
	if err != nil {
		return err
	}

	var response support.CampaignEnvelope
	if err := support.Decode(body, &response); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Campaign %s complete", action),
			fmt.Sprintf("Campaign ID: %d", response.Campaign.ID),
		},
		Changes: []string{
			fmt.Sprintf("Status: %s", response.Campaign.Status),
			fmt.Sprintf("Scenario: %s", response.Campaign.Scenario),
		},
		NextCommand: []string{
			fmt.Sprintf("%s campaigns get %d", cliName, response.Campaign.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func listCampaigns(core *cliapp.ScenarioApp) ([]support.Campaign, error) {
	body, err := core.Get("/campaigns", nil)
	if err != nil {
		return nil, err
	}
	var response support.CampaignsResponse
	if err := support.Decode(body, &response); err != nil {
		return nil, err
	}
	return response.Campaigns, nil
}

func campaignRows(campaigns []support.Campaign) []string {
	if len(campaigns) == 0 {
		return []string{"No campaigns found"}
	}
	rows := make([]string, 0, len(campaigns))
	for _, campaign := range campaigns {
		rows = append(rows, fmt.Sprintf("#%d %s | %s | sessions %d/%d | files %d/%d",
			campaign.ID,
			campaign.Scenario,
			campaign.Status,
			campaign.CurrentSession,
			campaign.MaxSessions,
			campaign.FilesVisited,
			campaign.FilesTotal,
		))
	}
	return rows
}
