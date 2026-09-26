package recipes

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"scalable-app-cookbook/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `recipes` subcommand group covering get/generate.
// Recipes are lookups and code-generation calls that the API owns.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "recipes",
		Description: "Inspect recipes and generate code",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get", Aliases: []string{"show"}, Description: "Get a recipe by id or title", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "generate", Description: "Generate code from a recipe for a target language", Run: func(args []string) error { return runGenerate(core, args) }},
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
		return fmt.Errorf("usage: recipes get <recipe-id-or-title>")
	}
	id := strings.Join(fs.Args(), " ")

	body, err := core.Get("/recipes/"+id, nil)
	if err != nil {
		return err
	}
	var recipe support.Recipe
	if err := support.Decode(body, &recipe); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Recipe: %s (%s)", recipe.Title, recipe.Type)},
		ResultsHeading: "Details",
		Results:        recipeDetailRows(recipe),
		RetrievalHints: []string{
			fmt.Sprintf("%s recipes generate %s <language>", support.CLIName, recipe.ID),
			fmt.Sprintf("%s implementations --recipe-id %s", support.CLIName, recipe.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("recipes generate")
	platform := fs.String("platform", "", "Target platform for code generation")
	outputDir := fs.String("output-dir", "", "Write generated code to this directory instead of stdout")
	bodyFile := fs.String("body-file", "", "Optional JSON file with extra parameters to merge into the request")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: recipes generate <recipe-id> <language> [--platform name] [--output-dir path] [--body-file path]")
	}
	recipeID := fs.Arg(0)
	language := fs.Arg(1)

	// Base request; --body-file can contribute extra "parameters".
	request := map[string]interface{}{
		"recipe_id":       recipeID,
		"language":        language,
		"target_platform": *platform,
		"parameters":      map[string]interface{}{},
	}

	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		var extras map[string]interface{}
		if err := json.Unmarshal(raw, &extras); err != nil {
			return fmt.Errorf("parse %s as JSON object: %w", *bodyFile, err)
		}
		if params, ok := extras["parameters"].(map[string]interface{}); ok {
			request["parameters"] = params
		} else {
			request["parameters"] = extras
		}
	}

	body, err := core.Request("POST", "/recipes/generate", nil, request)
	if err != nil {
		return err
	}
	var result support.GenerationResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	// Optionally persist generated code to an output directory.
	persisted := ""
	if *outputDir != "" {
		target := *outputDir
		if fileName := mainFileFromStructure(result.FileStructure); fileName != "" {
			target = joinPath(*outputDir, fileName)
		}
		if err := support.WriteOutput(target, []byte(result.GeneratedCode)); err != nil {
			return err
		}
		persisted = target
	}

	changes := []string{
		fmt.Sprintf("Generated %s code from recipe %s", language, recipeID),
	}
	if persisted != "" {
		changes = append(changes, fmt.Sprintf("Wrote: %s", persisted))
	}

	resultLines := []string{
		fmt.Sprintf("Recipe: %s", recipeID),
		fmt.Sprintf("Language: %s", language),
	}
	if len(result.Dependencies) > 0 {
		resultLines = append(resultLines, fmt.Sprintf("Dependencies: %s", strings.Join(result.Dependencies, ", ")))
	}
	if len(result.SetupInstructions) > 0 {
		resultLines = append(resultLines, "--- Setup instructions ---")
		resultLines = append(resultLines, result.SetupInstructions...)
	}
	if persisted == "" && strings.TrimSpace(result.GeneratedCode) != "" {
		resultLines = append(resultLines, "--- Generated code ---")
		resultLines = append(resultLines, result.GeneratedCode)
	}

	report := cliapp.MutationReport{
		Result:  resultLines,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s implementations --recipe-id %s --language %s", support.CLIName, recipeID, language),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func recipeDetailRows(r support.Recipe) []string {
	rows := []string{
		fmt.Sprintf("ID: %s", r.ID),
		fmt.Sprintf("Pattern ID: %s", r.PatternID),
		fmt.Sprintf("Title: %s", r.Title),
		fmt.Sprintf("Type: %s", r.Type),
		fmt.Sprintf("Steps: %d", len(r.Steps)),
		fmt.Sprintf("Timeout: %ds", r.TimeoutSec),
	}
	if len(r.Prerequisites) > 0 {
		rows = append(rows, fmt.Sprintf("Prerequisites: %s", strings.Join(r.Prerequisites, ", ")))
	}
	if len(r.ValidationChecks) > 0 {
		rows = append(rows, fmt.Sprintf("Validation checks: %s", strings.Join(r.ValidationChecks, ", ")))
	}
	if len(r.Artifacts) > 0 {
		rows = append(rows, fmt.Sprintf("Artifacts: %s", strings.Join(r.Artifacts, ", ")))
	}
	if len(r.Metrics) > 0 {
		rows = append(rows, fmt.Sprintf("Metrics: %s", strings.Join(r.Metrics, ", ")))
	}
	if len(r.Rollbacks) > 0 {
		rows = append(rows, fmt.Sprintf("Rollbacks: %s", strings.Join(r.Rollbacks, ", ")))
	}
	if len(r.Prompts) > 0 {
		rows = append(rows, fmt.Sprintf("Prompts: %s", strings.Join(r.Prompts, ", ")))
	}
	return rows
}

func mainFileFromStructure(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ""
	}
	if name, ok := decoded["main_file"].(string); ok {
		return strings.TrimSpace(name)
	}
	return ""
}

func joinPath(dir, name string) string {
	dir = strings.TrimRight(dir, "/")
	name = strings.TrimLeft(name, "/")
	if dir == "" {
		return name
	}
	if name == "" {
		return dir
	}
	return dir + "/" + name
}
