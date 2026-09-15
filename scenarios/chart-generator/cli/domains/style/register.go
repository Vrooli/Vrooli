package style

import (
	"fmt"
	"os"
	"strings"

	"chart-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `style` subcommand group wrapping /api/v1/styles and
// /api/v1/styles/builder/*.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "style",
		Description: "List, create, and preview chart styles",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available chart styles", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one style", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a custom style from a JSON body file", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "palettes", Description: "List predefined color palettes", Run: func(args []string) error { return runPalettes(core, args) }},
			{Name: "preview", Description: "Preview a chart rendered with a custom style", Run: func(args []string) error { return runPreview(core, args) }},
			{Name: "save", Description: "Save a custom style built via the style builder", Run: func(args []string) error { return runSave(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("style list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/styles", nil)
	if err != nil {
		return err
	}
	var resp support.StyleListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Styles: %d", resp.Count)},
		ResultsHeading: "Available styles",
		Results:        styleRows(resp.Styles),
		RetrievalHints: []string{
			fmt.Sprintf("%s style get <style-id>", support.CLIName),
			fmt.Sprintf("%s style palettes", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("style get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: style get <style-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/styles/"+id, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Style: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s style list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("style create")
	bodyFile := fs.String("body-file", "", "Path to JSON body describing the style, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("--body-file is required (path to JSON body, or '-' for stdin)")
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/styles", nil, raw)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	id := ""
	if v, ok := data["id"].(string); ok {
		id = v
	}

	changes := support.MapRows(data)
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created style %s", id)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s style get %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runPalettes(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("style palettes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/styles/builder/palettes", nil)
	if err != nil {
		return err
	}
	var resp support.PaletteResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	rows := make([]string, 0, len(resp.Palettes))
	for _, p := range resp.Palettes {
		rows = append(rows, fmt.Sprintf("%s (%s): %s", p.Name, p.ID, strings.Join(p.Colors, ", ")))
	}
	if len(rows) == 0 {
		rows = []string{"(no palettes configured)"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Palettes: %d", len(resp.Palettes))},
		ResultsHeading: "Color palettes",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s style preview --body-file preview.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runPreview(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("style preview")
	bodyFile := fs.String("body-file", "", "Path to JSON body with chart_type and style (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("--body-file is required (path to JSON with chart_type + style object, or '-' for stdin)")
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/styles/builder/preview", nil, raw)
	if err != nil {
		return err
	}
	var resp support.StyleBuilderPreviewResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	preview := resp.Preview
	if preview == "" {
		preview = resp.PreviewURL
	}

	result := []string{fmt.Sprintf("Success: %t", resp.Success)}
	if preview != "" {
		result = append(result, fmt.Sprintf("Preview: %s", preview))
	}

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{"Rendered style preview"},
		NextCommand: []string{fmt.Sprintf("%s style save --body-file style.json", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runSave(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("style save")
	bodyFile := fs.String("body-file", "", "Path to JSON body describing the custom style (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("--body-file is required (path to JSON body, or '-' for stdin)")
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/styles/builder/save", nil, raw)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	id := ""
	if v, ok := data["id"].(string); ok {
		id = v
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Saved style %s", id)},
		Changes:     support.MapRows(data),
		NextCommand: []string{fmt.Sprintf("%s style get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func styleRows(styles []support.Style) []string {
	if len(styles) == 0 {
		return []string{"(no styles configured)"}
	}
	rows := make([]string, 0, len(styles))
	for _, s := range styles {
		marker := " "
		if s.IsDefault {
			marker = "*"
		}
		row := fmt.Sprintf("%s %s (%s)", marker, s.Name, s.ID)
		if s.Category != "" {
			row += fmt.Sprintf(" | category=%s", s.Category)
		}
		if s.Description != "" {
			row += fmt.Sprintf(" | %s", s.Description)
		}
		rows = append(rows, row)
	}
	return rows
}
