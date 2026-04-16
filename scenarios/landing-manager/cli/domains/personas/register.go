package personas

import (
	"fmt"
	"os"
	"strings"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `personas` subcommand group backed by `GET /api/v1/personas`
// and `GET /api/v1/personas/{id}`.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "personas",
		Description: "List agent personas available for customization",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available personas", Run: func(args []string) error { return runList(core, args) }},
			{Name: "show", Description: "Show persona prompt and guidance", Run: func(args []string) error { return runShow(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("personas list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/personas", nil)
	if err != nil {
		return err
	}
	var personas []support.Persona
	if err := support.Decode(body, &personas); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Personas: %d", len(personas))},
		ResultsHeading: "Personas",
		Results:        personaRows(personas),
		RetrievalHints: []string{
			fmt.Sprintf("%s personas show <id>", support.CLIName),
			fmt.Sprintf("%s customize <scenario-id> --brief-file <path> --persona <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("personas show")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: personas show <id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/personas/"+id, nil)
	if err != nil {
		return err
	}
	var persona support.Persona
	if err := support.Decode(body, &persona); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", persona.ID),
		fmt.Sprintf("Name: %s", persona.Name),
	}
	if persona.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", persona.Description))
	}
	if len(persona.UseCases) > 0 {
		results = append(results, "Use cases: "+strings.Join(persona.UseCases, ", "))
	}
	if len(persona.Keywords) > 0 {
		results = append(results, "Keywords: "+strings.Join(persona.Keywords, ", "))
	}
	if persona.Prompt != "" {
		results = append(results, "Prompt:", persona.Prompt)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Persona: %s (%s)", persona.Name, persona.ID)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s customize <scenario-id> --brief-file <path> --persona %s", support.CLIName, persona.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func personaRows(personas []support.Persona) []string {
	if len(personas) == 0 {
		return []string{"No personas available"}
	}
	rows := make([]string, 0, len(personas))
	for _, p := range personas {
		line := fmt.Sprintf("%s | %s", p.ID, p.Name)
		if p.Description != "" {
			line += " | " + p.Description
		}
		if len(p.UseCases) > 0 {
			line += " | use=" + strings.Join(p.UseCases, ",")
		}
		rows = append(rows, line)
	}
	return rows
}
