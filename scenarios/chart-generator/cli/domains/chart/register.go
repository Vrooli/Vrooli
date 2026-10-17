package chart

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"chart-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `chart` subcommand group wrapping /api/v1/charts/*
// endpoints. The API owns all rendering; this package is a thin wrapper that
// shapes request bodies from CLI flags and formats responses through the
// standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "chart",
		Description: "Generate and inspect charts",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "generate", Aliases: []string{"create"}, Description: "Generate a chart from a data file (or stdin)", Run: func(args []string) error { return runGenerate(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Get details about a generated chart", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "composite", Description: "Generate a composite chart from a JSON request body", Run: func(args []string) error { return runComposite(core, args) }},
			{Name: "interactive", Description: "Generate an interactive chart with animations and tooltips", Run: func(args []string) error { return runInteractive(core, args) }},
		},
	}
}

var validChartTypes = map[string]struct{}{
	"bar":         {},
	"line":        {},
	"pie":         {},
	"scatter":     {},
	"area":        {},
	"gantt":       {},
	"heatmap":     {},
	"treemap":     {},
	"candlestick": {},
}

func runGenerate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chart generate")
	data := fs.String("data", "", "Path to JSON data file, or '-' for stdin (required)")
	style := fs.String("style", "professional", "Chart style (e.g. professional, minimal, vibrant)")
	formats := fs.String("format", "png", "Comma-separated export formats (png,svg,pdf,html)")
	width := fs.Int("width", 800, "Chart width in pixels")
	height := fs.Int("height", 600, "Chart height in pixels")
	title := fs.String("title", "", "Chart title")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chart generate <type> --data <file|-> [--style S] [--format png,svg] [--width N] [--height N] [--title T]")
	}
	chartType := strings.TrimSpace(fs.Arg(0))
	if _, ok := validChartTypes[chartType]; !ok {
		return fmt.Errorf("invalid chart type %q (supported: bar, line, pie, scatter, area, gantt, heatmap, treemap, candlestick)", chartType)
	}
	if strings.TrimSpace(*data) == "" {
		return fmt.Errorf("--data is required (file path or '-' for stdin)")
	}

	raw, err := support.ReadJSONFile(*data, true)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"chart_type":     chartType,
		"data":           raw,
		"style":          *style,
		"export_formats": support.SplitCSV(*formats),
		"width":          *width,
		"height":         *height,
	}
	if strings.TrimSpace(*title) != "" {
		payload["title"] = *title
	}

	body, err := core.Request("POST", "/charts/generate", nil, payload)
	if err != nil {
		return err
	}
	return renderChartResponse(body, *jsonOutput, "Generated chart", chartType, *style)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chart get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: chart get <chart-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/charts/"+id, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chart: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s chart generate <type> --data data.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runComposite(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chart composite")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("--body-file is required (path to JSON with chart_type, data, config.composition, etc.)")
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/charts/composite", nil, raw)
	if err != nil {
		return err
	}
	return renderChartResponse(body, *jsonOutput, "Generated composite chart", "composite", "")
}

func runInteractive(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("chart interactive")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("--body-file is required (path to JSON with chart_type, data, etc.)")
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/charts/interactive", nil, raw)
	if err != nil {
		return err
	}
	return renderChartResponse(body, *jsonOutput, "Generated interactive chart", "interactive", "")
}

func renderChartResponse(body []byte, jsonOutput bool, title, chartType, style string) error {
	var resp support.ChartGenerationResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if !resp.Success {
		errMsg := "chart generation failed"
		if resp.Error != nil && strings.TrimSpace(resp.Error.Message) != "" {
			errMsg = resp.Error.Message
		}
		return fmt.Errorf("%s", errMsg)
	}

	result := []string{
		fmt.Sprintf("Chart ID: %s", resp.ChartID),
	}
	if chartType != "" {
		result = append(result, fmt.Sprintf("Type: %s", chartType))
	}
	if style != "" {
		result = append(result, fmt.Sprintf("Style: %s", style))
	}

	metaKeys := []string{"generation_time_ms", "data_point_count", "style_applied", "formats_generated", "dimensions", "chart_count", "composition_type", "animation_enabled"}
	for _, k := range metaKeys {
		if v, ok := resp.Metadata[k]; ok {
			result = append(result, fmt.Sprintf("%s: %s", k, support.RenderValue(v)))
		}
	}
	if files := sortedFileEntries(resp.Files); len(files) > 0 {
		result = append(result, "", "--- Files ---")
		result = append(result, files...)
	}

	changes := []string{fmt.Sprintf("Created chart %s", support.ShortID(resp.ChartID))}
	report := cliapp.MutationReport{
		Result:  append([]string{title}, result...),
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s chart get %s", support.CLIName, resp.ChartID),
		},
	}
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func sortedFileEntries(files map[string]string) []string {
	if len(files) == 0 {
		return nil
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%s: %s", strings.ToUpper(k), files[k]))
	}
	return rows
}
