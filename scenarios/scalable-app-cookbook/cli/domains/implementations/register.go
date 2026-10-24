package implementations

import (
	"fmt"
	"os"
	"strings"

	"scalable-app-cookbook/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `scalable-app-cookbook implementations` as a flat command.
// The API's /api/v1/implementations endpoint requires a recipe_id query
// parameter; this wrapper passes through scenario filters without reinterpreting
// them.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Implementations",
		Commands: []cliapp.Command{
			{
				Name:        "implementations",
				Description: "List recipe implementations by language",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("implementations")
	recipeID := fs.String("recipe-id", "", "Recipe id to filter by (required by the API)")
	language := fs.String("language", "", "Optional language filter")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*recipeID) == "" {
		return fmt.Errorf("usage: implementations --recipe-id <id> [--language <lang>]")
	}

	query := support.BuildQuery(map[string]string{
		"recipe_id": *recipeID,
		"language":  *language,
	})
	body, err := core.Get("/implementations", query)
	if err != nil {
		return err
	}
	var impls []support.Implementation
	if err := support.Decode(body, &impls); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Implementations for %s: %d", *recipeID, len(impls))},
		ResultsHeading: "Implementations",
		Results:        implRows(impls),
		RetrievalHints: []string{
			fmt.Sprintf("%s recipes get %s", support.CLIName, *recipeID),
			fmt.Sprintf("%s recipes generate %s <language>", support.CLIName, *recipeID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func implRows(impls []support.Implementation) []string {
	if len(impls) == 0 {
		return []string{"(no implementations found)"}
	}
	rows := make([]string, 0, len(impls))
	for _, impl := range impls {
		line := fmt.Sprintf("%s (%s) | language=%s", support.ShortID(impl.ID), impl.RecipeID, impl.Language)
		if impl.FilePath != "" {
			line += fmt.Sprintf(" | file=%s", impl.FilePath)
		}
		if len(impl.Dependencies) > 0 {
			line += fmt.Sprintf(" | deps=%s", strings.Join(impl.Dependencies, ","))
		}
		if strings.TrimSpace(impl.Description) != "" {
			line += fmt.Sprintf(" | %s", impl.Description)
		}
		rows = append(rows, line)
	}
	return rows
}
