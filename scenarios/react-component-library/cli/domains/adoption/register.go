package adoption

import (
	"fmt"
	"os"
	"strings"

	"react-component-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `adoption` subcommand group tracking which scenarios use
// which components. The API exposes list and upsert-create today.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "adoption",
		Description: "Track scenario adoption of components",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List adoption records", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Record a new adoption", Run: func(args []string) error { return runCreate(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("adoption list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/adoptions", nil)
	if err != nil {
		return err
	}
	var adoptions []support.AdoptionRecord
	if err := support.Decode(body, &adoptions); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Adoption records: %d", len(adoptions))},
		ResultsHeading: "Adoptions",
		Results:        adoptionRows(adoptions),
		RetrievalHints: []string{
			fmt.Sprintf("%s component list", support.CLIName),
			fmt.Sprintf("%s adoption create --component-id <id> --scenario <name> --path <path>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("adoption create")
	componentID := fs.String("component-id", "", "Component ID being adopted (required)")
	scenario := fs.String("scenario", "", "Scenario name adopting the component (required)")
	adoptedPath := fs.String("path", "", "Filesystem path where the component is adopted (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*componentID) == "" {
		return fmt.Errorf("--component-id is required")
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	if strings.TrimSpace(*adoptedPath) == "" {
		return fmt.Errorf("--path is required")
	}

	body, err := core.Request("POST", "/adoptions", nil, map[string]interface{}{
		"componentId":  *componentID,
		"scenarioName": *scenario,
		"adoptedPath":  *adoptedPath,
	})
	if err != nil {
		return err
	}
	var a support.AdoptionRecord
	if err := support.Decode(body, &a); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recorded adoption for component %s in %s", a.ComponentID, a.ScenarioName)},
		Changes: []string{
			fmt.Sprintf("ID: %s", a.ID),
			fmt.Sprintf("Path: %s", a.AdoptedPath),
			fmt.Sprintf("Version: %s", a.Version),
			fmt.Sprintf("Status: %s", a.Status),
		},
		NextCommand: []string{fmt.Sprintf("%s adoption list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func adoptionRows(adoptions []support.AdoptionRecord) []string {
	if len(adoptions) == 0 {
		return []string{"(no adoption records)"}
	}
	rows := make([]string, 0, len(adoptions))
	for _, a := range adoptions {
		rows = append(rows, fmt.Sprintf("%s | scenario=%s | component=%s | v%s | %s | %s",
			support.ShortID(a.ID), a.ScenarioName, support.ShortID(a.ComponentID),
			a.Version, a.Status, a.AdoptedPath))
	}
	return rows
}
