package workflow

import (
	"fmt"
	"os"

	"idea-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `idea-generator workflows` as a flat command since workflows
// is a single, read-only capability surface in the API (`GET /api/workflows`).
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Workflows",
		Commands: []cliapp.Command{
			{
				Name:        "workflows",
				Description: "List available processing capabilities",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("workflows")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/workflows", nil)
	if err != nil {
		return err
	}
	var workflows []support.Workflow
	if err := support.Decode(body, &workflows); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Capabilities: %d", len(workflows))},
		ResultsHeading: "Workflows",
		Results:        workflowRows(workflows),
		RetrievalHints: []string{
			fmt.Sprintf("%s idea list", support.CLIName),
			fmt.Sprintf("%s campaign list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func workflowRows(workflows []support.Workflow) []string {
	if len(workflows) == 0 {
		return []string{"(no workflows exposed)"}
	}
	rows := make([]string, 0, len(workflows))
	for _, w := range workflows {
		endpoint := w.Endpoint
		if endpoint == "" {
			endpoint = w.URL
		}
		line := fmt.Sprintf("%s (%s)", w.Name, w.ID)
		if w.Status != "" {
			line += fmt.Sprintf(" | %s", w.Status)
		}
		if endpoint != "" {
			line += fmt.Sprintf(" | %s", endpoint)
		}
		if w.Description != "" {
			line += fmt.Sprintf(" — %s", w.Description)
		}
		rows = append(rows, line)
	}
	return rows
}
