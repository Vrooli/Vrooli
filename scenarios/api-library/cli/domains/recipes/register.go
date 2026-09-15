package recipes

import (
	"fmt"
	"os"
	"strings"

	"api-library/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `recipes` subcommands against the global recipe endpoints
// (get, update, delete, successful). Per-API recipe listing/creation lives in
// the `apis` group.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "recipes",
		Description: "Browse and manage integration recipes",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Aliases: []string{"show"}, Description: "Show a single recipe", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Description: "Update a recipe (requires --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Description: "Delete a recipe", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "successful", Description: "List high-success-rate recipes", Run: func(args []string) error { return runSuccessful(core, args) }},
		},
	}
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recipes get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: recipes get <recipe-id>")
	}
	id := fs.Arg(0)
	raw, err := core.Get("/recipes/"+id, nil)
	if err != nil {
		return err
	}
	var recipe support.Recipe
	if err := support.Decode(raw, &recipe); err != nil {
		return err
	}
	results := []string{
		fmt.Sprintf("ID: %s", recipe.ID),
		fmt.Sprintf("Name: %s", recipe.Name),
		fmt.Sprintf("API: %s", orDash(recipe.APIID)),
		fmt.Sprintf("Use case: %s", orDash(recipe.UseCase)),
		fmt.Sprintf("Difficulty: %s", orDash(recipe.DifficultyLevel)),
		fmt.Sprintf("Estimated time: %d minutes", recipe.EstimatedTimeMinutes),
		fmt.Sprintf("Success rate: %.1f%%", recipe.SuccessRate*100),
		fmt.Sprintf("Rating: %.2f (%d ratings)", recipe.Rating, recipe.RatingCount),
		fmt.Sprintf("Times used: %d", recipe.TimesUsed),
	}
	if recipe.Description != "" {
		results = append(results, "", "Description:", recipe.Description)
	}
	if recipe.ExpectedOutcome != "" {
		results = append(results, "", "Expected outcome:", recipe.ExpectedOutcome)
	}
	if len(recipe.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %s", strings.Join(recipe.Tags, ", ")))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recipe: %s", recipe.Name)},
		ResultsHeading: "Details",
		Results:        results,
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recipes update")
	bodyFile := fs.String("body-file", "", "Path to JSON file with the updated recipe fields")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: recipes update <recipe-id> --body-file PATH")
	}
	id := fs.Arg(0)
	body, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	if _, err := core.Request("PUT", "/recipes/"+id, nil, body); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated recipe %s", id)},
		Changes: []string{fmt.Sprintf("PUT /recipes/%s", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recipes delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: recipes delete <recipe-id>")
	}
	id := fs.Arg(0)
	if _, err := core.Request("DELETE", "/recipes/"+id, nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Deleted recipe %s", id)},
		Changes: []string{fmt.Sprintf("DELETE /recipes/%s", id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSuccessful(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recipes successful")
	minRate := fs.String("min-success-rate", "", "Minimum success rate (0.0 - 1.0)")
	limit := fs.String("limit", "", "Maximum recipes to return")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := support.BuildQuery(map[string]string{
		"min_success_rate": *minRate,
		"limit":            *limit,
	})
	raw, err := core.Get("/recipes/successful", query)
	if err != nil {
		return err
	}
	var env support.RecipesEnvelope
	if err := support.Decode(raw, &env); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Successful recipes: %d", env.Count)},
		ResultsHeading: "Recipes",
		Results:        recipeRows(env.Recipes),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func recipeRows(list []support.Recipe) []string {
	if len(list) == 0 {
		return []string{"(no recipes)"}
	}
	rows := make([]string, 0, len(list))
	for _, r := range list {
		rows = append(rows, fmt.Sprintf("%s (%s) | difficulty=%s | success=%.1f%% | rating=%.2f | used=%d",
			r.Name, support.ShortID(r.ID), orDash(r.DifficultyLevel), r.SuccessRate*100, r.Rating, r.TimesUsed))
	}
	return rows
}

func orDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
