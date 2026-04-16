package generate

import (
	"encoding/json"
	"fmt"
	"os"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `generate` as a flat command. The API is `POST /api/v1/generate`
// and a companion `GET /api/v1/generated` for listing.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Generation",
		Commands: []cliapp.Command{
			{
				Name:        "generate",
				Description: "Generate a new landing-page scenario from a template",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runGenerate(core, args) },
			},
			{
				Name:        "generated",
				Description: "List previously generated landing-page scenarios",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("generate")
	name := fs.String("name", "", "Display name for the generated scenario (required)")
	slug := fs.String("slug", "", "URL-safe slug for the generated scenario (required)")
	dryRun := fs.Bool("dry-run", false, "Plan only; do not write files")
	bodyFile := fs.String("body-file", "", "Optional JSON file providing extra options")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: generate <template-id> --name <name> --slug <slug> [--dry-run] [--body-file PATH]")
	}
	templateID := fs.Arg(0)
	if *name == "" || *slug == "" {
		return fmt.Errorf("--name and --slug are required")
	}

	options := map[string]interface{}{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &options); err != nil {
			return fmt.Errorf("parse --body-file as JSON object: %w", err)
		}
	}
	options["dry_run"] = *dryRun

	payload := map[string]interface{}{
		"template_id": templateID,
		"name":        *name,
		"slug":        *slug,
		"options":     options,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal generate payload: %w", err)
	}

	body, err := core.Request("POST", "/generate", nil, raw)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}
	if err := support.Decode(body, &decoded); err != nil {
		decoded = map[string]interface{}{}
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		if v, ok := decoded["message"].(string); ok && v != "" {
			message = v
		} else if *dryRun {
			message = fmt.Sprintf("Dry-run plan for %s (%s)", *slug, templateID)
		} else {
			message = fmt.Sprintf("Generated %s from template %s", *slug, templateID)
		}
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: support.MapRows(decoded),
		NextCommand: []string{
			fmt.Sprintf("%s preview %s", support.CLIName, *slug),
			fmt.Sprintf("%s generated", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("generated")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/generated", nil)
	if err != nil {
		return err
	}
	var scenarios []support.GeneratedScenario
	if err := support.Decode(body, &scenarios); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Generated scenarios: %d", len(scenarios))},
		ResultsHeading: "Generated",
		Results:        scenarioRows(scenarios),
		RetrievalHints: []string{
			fmt.Sprintf("%s preview <scenario-id>", support.CLIName),
			fmt.Sprintf("%s lifecycle status <scenario-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func scenarioRows(scenarios []support.GeneratedScenario) []string {
	if len(scenarios) == 0 {
		return []string{"No generated scenarios yet"}
	}
	rows := make([]string, 0, len(scenarios))
	for _, s := range scenarios {
		id := s.ScenarioID
		if id == "" {
			id = s.ID
		}
		if id == "" {
			id = s.Slug
		}
		line := id
		if s.Name != "" {
			line += " | " + s.Name
		}
		if s.TemplateID != "" {
			line += " | template=" + s.TemplateID
		}
		if s.Status != "" {
			line += " | status=" + s.Status
		}
		if s.CreatedAt != nil {
			line += " | created=" + support.FormatTimeValue(*s.CreatedAt)
		}
		rows = append(rows, line)
	}
	return rows
}
