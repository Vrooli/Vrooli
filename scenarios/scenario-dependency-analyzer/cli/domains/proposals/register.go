package proposals

import (
	"encoding/json"
	"fmt"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Planning",
		Commands: []cliapp.Command{
			{
				Name:        "propose",
				Description: "Analyze dependencies for a proposed scenario",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("propose")
	var name string
	var description string
	var requirements string
	var similar string
	var jsonOutput bool
	fs.StringVar(&name, "name", "", "Scenario name")
	fs.StringVar(&description, "description", "", "Scenario description")
	fs.StringVar(&requirements, "requirements", "", "Comma-separated requirements")
	fs.StringVar(&similar, "similar", "", "Comma-separated similar scenarios")
	fs.BoolVar(&jsonOutput, "json", false, "Output JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(requirements) == "" {
		return fmt.Errorf("usage: %s propose --name <name> --requirements <csv> [--description text] [--similar csv] [--json]", support.AppName)
	}

	payload := map[string]interface{}{
		"name":              name,
		"description":       description,
		"requirements":      support.JoinCSV(requirements),
		"similar_scenarios": support.JoinCSV(similar),
	}
	body, err := core.Request("POST", "/analyze/proposed", nil, payload)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Proposal: %s", name),
			fmt.Sprintf("Requirements: %d", len(support.JoinCSV(requirements))),
		},
		Triage: []cliapp.TriageGroup{
			{Heading: "Recommended resources", Items: support.Strings(resp["recommended_resources"])},
			{Heading: "Related scenarios", Items: support.Strings(resp["recommended_scenarios"])},
		},
		NextSteps: []string{
			fmt.Sprintf("%s analyze %s", support.AppName, name),
			fmt.Sprintf("%s optimize %s", support.AppName, name),
		},
	}
	return support.PrintOperational(false, report, nil)
}
