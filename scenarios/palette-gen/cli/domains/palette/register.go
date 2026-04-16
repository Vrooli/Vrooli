package palette

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"palette-gen/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `palette` subcommand group covering generate/suggest/export/history.
// Each subcommand is a thin wrapper over a single API endpoint.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "palette",
		Description: "Generate, suggest, export, and review palette history",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Description: "Generate a palette for a theme", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "suggest", Description: "Get palette suggestions for a use case", Run: func(args []string) error { return runSuggest(core, args) }},
			{Name: "export", Description: "Export a palette as css, json, or scss", Run: func(args []string) error { return runExport(core, args) }},
			{Name: "history", Description: "List recently generated palettes", Run: func(args []string) error { return runHistory(core, args) }},
		},
	}
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("palette generate")
	style := fs.String("style", "", "Palette style (vibrant, pastel, dark, minimal, earthy, auto)")
	numColors := fs.Int("colors", 5, "Number of colors (3-10)")
	baseColor := fs.String("base", "", "Base color to build palette from (hex)")
	includeDebug := fs.Bool("debug", false, "Include AI debug metadata in response")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: palette generate <theme> [--style X] [--colors N] [--base #hex] [--debug]")
		}
		theme := strings.TrimSpace(strings.Join(fs.Args(), " "))
		payload = map[string]interface{}{
			"theme":            theme,
			"style":            *style,
			"num_colors":       *numColors,
			"base_color":       *baseColor,
			"include_ai_debug": *includeDebug,
		}
	}

	body, err := core.Request("POST", "/generate", nil, payload)
	if err != nil {
		return err
	}
	var resp support.PaletteResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if resp.Name != "" {
		summary = append(summary, fmt.Sprintf("Name: %s", resp.Name))
	}
	if resp.Theme != "" {
		summary = append(summary, fmt.Sprintf("Theme: %s", resp.Theme))
	}
	if resp.Style != "" {
		summary = append(summary, fmt.Sprintf("Style: %s", resp.Style))
	}
	if resp.Description != "" {
		summary = append(summary, resp.Description)
	}
	if resp.Error != "" {
		summary = append(summary, fmt.Sprintf("Error: %s", resp.Error))
	}
	if len(summary) == 0 {
		summary = []string{"Palette generated"}
	}

	results := make([]string, 0, len(resp.Palette))
	for i, color := range resp.Palette {
		results = append(results, fmt.Sprintf("%d. %s", i+1, color))
	}
	if len(resp.Debug) > 0 {
		results = append(results, "--- debug ---")
		results = append(results, support.MapRows(resp.Debug)...)
	}
	if len(results) == 0 {
		results = []string{"(no palette returned)"}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Palette",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s palette export css --palette %s", support.CLIName, joinColors(resp.Palette)),
			fmt.Sprintf("%s analyze harmony %s", support.CLIName, joinColors(resp.Palette)),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSuggest(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("palette suggest")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request body (overrides positional argument)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: palette suggest <use-case>")
		}
		payload = map[string]interface{}{
			"use_case": strings.TrimSpace(strings.Join(fs.Args(), " ")),
		}
	}

	body, err := core.Request("POST", "/suggest", nil, payload)
	if err != nil {
		return err
	}
	var resp support.SuggestResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Suggestions)*4)
	for i, s := range resp.Suggestions {
		name := stringField(s, "name")
		desc := stringField(s, "description")
		colors := colorListField(s, "colors")
		header := fmt.Sprintf("%d. %s", i+1, name)
		if header == fmt.Sprintf("%d. ", i+1) {
			header = fmt.Sprintf("%d. (unnamed)", i+1)
		}
		results = append(results, header)
		if desc != "" {
			results = append(results, fmt.Sprintf("   %s", desc))
		}
		if len(colors) > 0 {
			results = append(results, fmt.Sprintf("   colors: %s", strings.Join(colors, ", ")))
		}
	}
	if len(results) == 0 {
		results = []string{"(no suggestions returned)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Suggestions: %d", len(resp.Suggestions))},
		ResultsHeading: "Palette suggestions",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s palette generate \"<theme>\" --style vibrant", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("palette export")
	paletteFlag := fs.String("palette", "", "Comma-separated list of hex colors")
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the full request body (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	var format string
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
		// Try to peek at the format for the summary line; ignore errors.
		var peek struct {
			Format string `json:"format"`
		}
		_ = json.Unmarshal(raw, &peek)
		format = peek.Format
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: palette export <format> --palette <hex,hex,...>")
		}
		format = strings.TrimSpace(fs.Arg(0))
		colors := support.SplitColors(*paletteFlag)
		if len(colors) == 0 {
			return fmt.Errorf("palette export requires --palette <hex,hex,...>")
		}
		payload = map[string]interface{}{
			"format":  format,
			"palette": colors,
		}
	}

	body, err := core.Request("POST", "/export", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ExportResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{}
	if format != "" {
		summary = append(summary, fmt.Sprintf("Format: %s", format))
	}
	if resp.Export == "" {
		summary = append(summary, "(empty export body)")
	}

	results := strings.Split(resp.Export, "\n")
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Export",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s palette export json --palette <hex,hex,...>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runHistory(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("palette history")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/history", nil)
	if err != nil {
		return err
	}
	var resp support.HistoryResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.History)*2)
	for i, entry := range resp.History {
		header := fmt.Sprintf("%d. %s", i+1, entry.Name)
		if entry.Name == "" {
			header = fmt.Sprintf("%d. (unnamed)", i+1)
		}
		if entry.Style != "" || entry.Theme != "" {
			header += fmt.Sprintf(" [%s/%s]", entry.Style, entry.Theme)
		}
		results = append(results, header)
		if len(entry.Palette) > 0 {
			results = append(results, fmt.Sprintf("   %s", strings.Join(entry.Palette, ", ")))
		}
	}
	if len(results) == 0 {
		results = []string{"(no history entries)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("History entries: %d", len(resp.History))},
		ResultsHeading: "Recent palettes",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s palette generate \"<theme>\"", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func joinColors(colors []string) string {
	if len(colors) == 0 {
		return "<hex,hex,...>"
	}
	return strings.Join(colors, ",")
}

func stringField(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func colorListField(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
