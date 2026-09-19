package sprint

import (
	"fmt"
	"os"

	"product-manager-agent/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `sprint` subcommand group: inspect the current sprint
// and plan a new sprint from team size/velocity/duration.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "sprint",
		Description: "Sprint planning commands",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "current", Aliases: []string{"show"}, Description: "Show the current sprint", Run: func(args []string) error { return runCurrent(core, args) }},
			{Name: "plan", Description: "Plan a new sprint (--body-file PATH)", Run: func(args []string) error { return runPlan(core, args) }},
		},
	}
}

func runCurrent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sprint current")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/sprint/current", nil)
	if err != nil {
		return err
	}
	var sp support.SprintPlan
	if err := support.Decode(body, &sp); err != nil {
		return err
	}
	return renderSprint(sp, "Current sprint", *jsonOutput)
}

func runPlan(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("sprint plan")
	bodyFile := fs.String("body-file", "", `Path to JSON body: {"team_size":N,"velocity":N,"duration_weeks":N}`)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/sprint/plan", nil, payload)
	if err != nil {
		return err
	}
	var sp support.SprintPlan
	if err := support.Decode(body, &sp); err != nil {
		return err
	}
	return renderSprint(sp, "Planned sprint", *jsonOutput)
}

func renderSprint(sp support.SprintPlan, heading string, asJSON bool) error {
	results := []string{
		fmt.Sprintf("ID: %s", sp.ID),
		fmt.Sprintf("Sprint number: %d", sp.SprintNumber),
		fmt.Sprintf("Capacity: %d story points", sp.Capacity),
		fmt.Sprintf("Total effort: %d", sp.TotalEffort),
		fmt.Sprintf("Estimated value: %.2f", sp.EstimatedValue),
		fmt.Sprintf("Velocity: %.2f", sp.Velocity),
		fmt.Sprintf("Risk: %s", sp.RiskLevel),
		fmt.Sprintf("Planned at: %s", support.FormatTimeValue(sp.PlannedAt)),
		fmt.Sprintf("Features: %d", len(sp.Features)),
	}
	for _, f := range sp.Features {
		results = append(results, fmt.Sprintf("  - %s (%s) score=%.2f effort=%d",
			f.Name, support.ShortID(f.ID), f.Score, f.Effort))
	}

	report := cliapp.ListReport{
		Summary:        []string{heading},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s sprint plan --body-file request.json", support.CLIName),
			fmt.Sprintf("%s features list", support.CLIName),
		},
	}
	if asJSON {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
