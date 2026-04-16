package campaigns

import (
	"fmt"
	"os"

	"campaign-content-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `campaign` subcommand group covering list/create.
// The API is the source of truth; this package is a thin wrapper.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "campaign",
		Description: "List and create content campaigns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List all campaigns", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a new campaign", Run: func(args []string) error { return runCreate(core, args) }},
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
			fmt.Sprintf("%s document list <campaign-id>", support.CLIName),
			fmt.Sprintf("%s generate <campaign-id> <content-type> <prompt>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("campaign create")
	description := fs.String("description", "", "Campaign description")
	bodyFile := fs.String("body-file", "", "Path to JSON file with full request body (overrides positional args)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if raw, err := support.ReadJSONFile(*bodyFile, false); err != nil {
		return err
	} else if raw != nil {
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: campaign create <name> [--description <desc>] | --body-file <path>")
		}
		name := fs.Arg(0)
		desc := *description
		if desc == "" && fs.NArg() >= 2 {
			desc = fs.Arg(1)
		}
		payload = map[string]interface{}{
			"name":        name,
			"description": desc,
			"settings":    map[string]interface{}{},
		}
	}

	body, err := core.Request("POST", "/campaigns", nil, payload)
	if err != nil {
		return err
	}

	var created support.Campaign
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	changes := []string{fmt.Sprintf("Created campaign %s", created.Name)}
	result := []string{
		fmt.Sprintf("ID: %s", created.ID),
		fmt.Sprintf("Name: %s", created.Name),
	}
	if created.Description != "" {
		result = append(result, fmt.Sprintf("Description: %s", created.Description))
	}
	if !created.CreatedAt.IsZero() {
		result = append(result, fmt.Sprintf("Created: %s", support.FormatTimeValue(created.CreatedAt)))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s document list %s", support.CLIName, created.ID),
			fmt.Sprintf("%s generate %s <content-type> <prompt>", support.CLIName, created.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func campaignRows(campaigns []support.Campaign) []string {
	if len(campaigns) == 0 {
		return []string{"No campaigns found"}
	}
	rows := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		line := fmt.Sprintf("%s (%s)", c.Name, support.ShortID(c.ID))
		if c.Description != "" {
			line += " | " + c.Description
		}
		if !c.CreatedAt.IsZero() {
			line += " | created=" + support.FormatTimeValue(c.CreatedAt)
		}
		rows = append(rows, line)
	}
	return rows
}
