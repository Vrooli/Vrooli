package roadmap

import (
	"fmt"
	"os"

	"product-manager-agent/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `roadmap` subcommand group: get the current roadmap,
// post a new one, or generate one from feature IDs.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "roadmap",
		Description: "Manage product roadmaps",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Aliases: []string{"show"}, Description: "Show the current roadmap", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Aliases: []string{"post"}, Description: "Create a roadmap (--body-file PATH)", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "generate", Description: "Generate a roadmap from feature IDs (--body-file PATH)", Run: func(args []string) error { return runGenerate(core, args) }},
		},
	}
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("roadmap get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/roadmap", nil)
	if err != nil {
		return err
	}
	var r support.Roadmap
	if err := support.Decode(body, &r); err != nil {
		return err
	}
	return renderRoadmap(r, "Roadmap", *jsonOutput)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("roadmap create")
	bodyFile := fs.String("body-file", "", "Path to a JSON Roadmap body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/roadmap", nil, payload)
	if err != nil {
		return err
	}
	var r support.Roadmap
	if err := support.Decode(body, &r); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Roadmap created: %s", r.Name)},
		Changes: []string{
			fmt.Sprintf("ID: %s", r.ID),
			fmt.Sprintf("Version: %d", r.Version),
			fmt.Sprintf("Features: %d", len(r.Features)),
		},
		NextCommand: []string{fmt.Sprintf("%s roadmap get", support.CLIName)},
	}
	if jsonOutput != nil && *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("roadmap generate")
	bodyFile := fs.String("body-file", "", `Path to JSON body: {"feature_ids":[...],"start_date":"...","duration_months":N,"team_capacity":N}`)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/roadmap/generate", nil, payload)
	if err != nil {
		return err
	}
	var r support.Roadmap
	if err := support.Decode(body, &r); err != nil {
		return err
	}
	return renderRoadmap(r, "Generated roadmap", *jsonOutput)
}

func renderRoadmap(r support.Roadmap, heading string, asJSON bool) error {
	results := []string{
		fmt.Sprintf("ID: %s", r.ID),
		fmt.Sprintf("Name: %s", r.Name),
		fmt.Sprintf("Version: %d", r.Version),
		fmt.Sprintf("Start: %s", support.FormatTimeValue(r.StartDate)),
		fmt.Sprintf("End: %s", support.FormatTimeValue(r.EndDate)),
		fmt.Sprintf("Features: %d", len(r.Features)),
		fmt.Sprintf("Milestones: %d", len(r.Milestones)),
	}
	for _, m := range r.Milestones {
		results = append(results, fmt.Sprintf("  - %s @ %s (%d features)",
			m.Name, support.FormatTimeValue(m.Date), len(m.Features)))
	}

	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s roadmap generate --body-file request.json", support.CLIName),
			fmt.Sprintf("%s features list", support.CLIName),
		},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
