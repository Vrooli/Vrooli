package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `config` subcommand group for generating and validating
// service.json snippets from selected resources. `generate` accepts a flat
// --resources comma-separated list; `validate` takes a full JSON config via
// --body-file because the shape is a nested map.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "config",
		Description: "Generate and validate service.json config snippets",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Aliases: []string{"gen"}, Description: "Generate a service.json snippet from selected resources", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "validate", Description: "Validate a resources config object (see --body-file)", Run: func(args []string) error { return runValidate(core, args) }},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config generate")
	resourcesFlag := fs.String("resources", "", "Comma-separated resource names to enable")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full generate request body")
	outputPath := fs.String("output", "", "Write the generated snippet JSON to this path (defaults to stdout)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload, err := buildGenerateBody(*resourcesFlag, *bodyFile)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/config/generate", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ConfigGenerateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	snippetBytes, err := json.MarshalIndent(resp.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snippet: %w", err)
	}
	snippetBytes = append(snippetBytes, '\n')
	if strings.TrimSpace(*outputPath) != "" {
		if err := support.WriteOutput(*outputPath, snippetBytes); err != nil {
			return err
		}
	}

	names := make([]string, 0, len(resp.Config.Resources))
	for name := range resp.Config.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	changes := make([]string, 0, len(names))
	for _, n := range names {
		changes = append(changes, fmt.Sprintf("enabled resource -> %s", n))
	}
	if len(changes) == 0 {
		changes = append(changes, "no resources enabled")
	}

	result := []string{fmt.Sprintf("Generated snippet with %d resource(s)", len(names))}
	if strings.TrimSpace(*outputPath) != "" {
		result = append(result, fmt.Sprintf("Snippet written to: %s", *outputPath))
	}
	for _, w := range resp.Warnings {
		result = append(result, "Warning: "+w)
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s config validate --body-file <snippet.json>", support.CLIName),
			fmt.Sprintf("%s setup-order", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	if strings.TrimSpace(*outputPath) == "" {
		if _, err := os.Stdout.Write(snippetBytes); err != nil {
			return err
		}
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("config validate")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with a resources config object (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("--body-file is required (JSON object with a top-level \"resources\" map)")
	}
	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", "/config/validate", nil, raw)
	if err != nil {
		return err
	}
	var resp support.ConfigValidateResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Valid: %t", resp.Valid), fmt.Sprintf("Results: %d", len(resp.Results))}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Validation results",
		Results:        validationRows(resp.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s config generate --resources postgres,redis", support.CLIName),
			fmt.Sprintf("%s setup-order", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// buildGenerateBody favors --body-file; otherwise builds {"resources": [...]}
// from the comma-separated --resources flag.
func buildGenerateBody(resourcesFlag, bodyFile string) (interface{}, error) {
	if strings.TrimSpace(bodyFile) != "" {
		return support.ReadJSONFile(bodyFile, true)
	}
	list := splitCSV(resourcesFlag)
	if len(list) == 0 {
		return nil, fmt.Errorf("supply --resources <name,...> or --body-file <path>")
	}
	return map[string]interface{}{"resources": list}, nil
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func validationRows(results []support.ValidationResult) []string {
	if len(results) == 0 {
		return []string{"(no validation results)"}
	}
	rows := make([]string, 0, len(results))
	for _, r := range results {
		status := "ok"
		if !r.Valid {
			status = "invalid"
		}
		line := fmt.Sprintf("%s [%s]", r.Resource, status)
		if len(r.Errors) > 0 {
			line += " errors=" + strings.Join(r.Errors, ";")
		}
		if len(r.Warnings) > 0 {
			line += " warnings=" + strings.Join(r.Warnings, ";")
		}
		rows = append(rows, line)
	}
	return rows
}
