package campaigns

import (
	"fmt"
	"os"

	"social-media-scheduler/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register wraps /api/v1/campaigns/*. Create/update accept nested brand_guidelines
// maps, so they expose --body-file instead of a flag-per-field interface.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "campaign",
		Description: "Manage marketing campaigns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List campaigns", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show a campaign", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a campaign (--body-file PATH)", Run: func(args []string) error {
				return runBody(core, args, "campaign create", "POST", "", "Created campaign")
			}},
			{Name: "update", Description: "Update a campaign (--body-file PATH)", Run: func(args []string) error {
				return runBody(core, args, "campaign update", "PUT", "/{id}", "Updated campaign")
			}},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a campaign", Run: func(args []string) error { return runDelete(core, args) }},
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
		Results:        rows(campaigns),
		RetrievalHints: []string{fmt.Sprintf("%s campaign get <campaign-id>", support.CLIName)},
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
		fmt.Sprintf("Status: %s", c.Status),
	}
	if c.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", c.Description))
	}
	if c.StartDate != nil && *c.StartDate != "" {
		results = append(results, fmt.Sprintf("Start: %s", *c.StartDate))
	}
	if c.EndDate != nil && *c.EndDate != "" {
		results = append(results, fmt.Sprintf("End: %s", *c.EndDate))
	}
	if len(c.BrandGuidelines) > 0 {
		results = append(results, "Brand guidelines:")
		results = append(results, support.MapRows(c.BrandGuidelines)...)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Campaign: %s", c.Name)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runBody(core *cliapp.ScenarioApp, args []string, cmdName, method, pathSuffix, fallback string) error {
	fs := support.NewFlagSet(cmdName)
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the request payload")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	id := ""
	if pathSuffix == "/{id}" {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: %s <campaign-id> --body-file PATH", cmdName)
		}
		id = fs.Arg(0)
	}

	payload, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	path := "/campaigns"
	if id != "" {
		path = path + "/" + id
	}
	body, err := core.Request(method, path, nil, payload)
	if err != nil {
		return err
	}
	var c support.Campaign
	_ = support.Decode(body, &c)

	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fallback
		if c.ID != "" {
			msg = fmt.Sprintf("%s (%s)", fallback, c.ID)
		}
	}

	report := cliapp.MutationReport{
		Result:  []string{msg},
		Changes: []string{fmt.Sprintf("%s %s", method, path)},
	}
	if c.ID != "" {
		report.NextCommand = []string{fmt.Sprintf("%s campaign get %s", support.CLIName, c.ID)}
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

	body, err := core.Request("DELETE", "/campaigns/"+id, nil, nil)
	if err != nil {
		return err
	}
	msg := support.EnvelopeMessage(body)
	if msg == "" {
		msg = fmt.Sprintf("Deleted campaign %s", id)
	}
	report := cliapp.MutationReport{
		Result:      []string{msg},
		Changes:     []string{fmt.Sprintf("Campaign %s deleted", id)},
		NextCommand: []string{fmt.Sprintf("%s campaign list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func rows(campaigns []support.Campaign) []string {
	if len(campaigns) == 0 {
		return []string{"No campaigns found"}
	}
	out := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		out = append(out, fmt.Sprintf("%s | %s | status=%s", support.ShortID(c.ID), c.Name, c.Status))
	}
	return out
}
