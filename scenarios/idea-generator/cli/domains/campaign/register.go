package campaign

import (
	"fmt"
	"os"
	"strings"

	"idea-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `campaign` subcommand group covering list/get/create/delete
// against /api/campaigns. The API owns persistence; this package is a thin
// wrapper that shapes requests and formats responses via the standard reports.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "campaign",
		Description: "Manage idea campaigns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List active campaigns", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one campaign", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a new campaign", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Soft-delete a campaign", Run: func(args []string) error { return runDelete(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaign list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/campaigns", nil)
	if err != nil {
		return err
	}
	var campaigns []support.Campaign
	if err := support.Decode(body, &campaigns); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Campaigns: %d", len(campaigns))},
		ResultsHeading: "Campaigns",
		Results:        campaignRows(campaigns),
		RetrievalHints: []string{
			fmt.Sprintf("%s campaign get <id>", support.CLIName),
			fmt.Sprintf("%s idea list --campaign <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaign get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: campaign get <campaign-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/campaigns/"+id, nil)
	if err != nil {
		return err
	}
	var c support.Campaign
	if err := support.Decode(body, &c); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", c.ID),
		fmt.Sprintf("Name: %s", c.Name),
	}
	if c.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", c.Description))
	}
	if c.Color != "" {
		results = append(results, fmt.Sprintf("Color: %s", c.Color))
	}
	results = append(results, fmt.Sprintf("Created: %s", support.FormatTimeValue(c.CreatedAt)))

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Campaign: %s", c.Name)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s idea list --campaign %s", support.CLIName, c.ID),
			fmt.Sprintf("%s idea generate --campaign %s --prompt \"...\"", support.CLIName, c.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaign create")
	name := fs.String("name", "", "Campaign name (required, max 100 chars)")
	description := fs.String("description", "", "Campaign description (max 500 chars)")
	color := fs.String("color", "", "Display color hint (free-form)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}

	payload := map[string]interface{}{
		"name":        strings.TrimSpace(*name),
		"description": strings.TrimSpace(*description),
		"color":       strings.TrimSpace(*color),
	}

	body, err := core.Request("POST", "/campaigns", nil, payload)
	if err != nil {
		return err
	}
	var c support.Campaign
	if err := support.Decode(body, &c); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Created campaign %s", c.Name),
			fmt.Sprintf("ID: %s", c.ID),
		},
		Changes: []string{fmt.Sprintf("Campaign %s created", support.ShortID(c.ID))},
		NextCommand: []string{
			fmt.Sprintf("%s campaign get %s", support.CLIName, c.ID),
			fmt.Sprintf("%s idea generate --campaign %s --prompt \"...\"", support.CLIName, c.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaign delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: campaign delete <campaign-id>")
	}
	id := fs.Arg(0)

	if _, err := core.Request("DELETE", "/campaigns/"+id, nil, nil); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Campaign %s deleted", id)},
		Changes:     []string{fmt.Sprintf("Campaign %s soft-deleted", support.ShortID(id))},
		NextCommand: []string{fmt.Sprintf("%s campaign list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func campaignRows(campaigns []support.Campaign) []string {
	if len(campaigns) == 0 {
		return []string{"No campaigns"}
	}
	rows := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		line := fmt.Sprintf("%s (%s)", c.Name, support.ShortID(c.ID))
		if c.Color != "" {
			line += fmt.Sprintf(" | color=%s", c.Color)
		}
		if c.Description != "" {
			line += fmt.Sprintf(" — %s", c.Description)
		}
		rows = append(rows, line)
	}
	return rows
}
