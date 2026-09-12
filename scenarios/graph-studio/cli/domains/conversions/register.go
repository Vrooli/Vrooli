package conversions

import (
	"fmt"
	"os"
	"sort"

	"graph-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `conversions` subcommand group. It wraps the two read
// endpoints the API exposes:
//   - GET /api/v1/conversions            -> list all supported conversion paths
//   - GET /api/v1/conversions/:from/:to  -> metadata for a single pair
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "conversions",
		Description: "Inspect supported graph format conversions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List supported conversion paths", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show metadata for a conversion pair", Run: func(args []string) error { return runGet(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversions list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/conversions", nil)
	if err != nil {
		return err
	}
	var payload struct {
		Conversions map[string][]map[string]interface{} `json:"conversions"`
		TotalPaths  int                                 `json:"total_paths"`
	}
	if err := support.Decode(body, &payload); err != nil {
		return err
	}

	rows := conversionRows(payload.Conversions)
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Conversion source formats: %d", payload.TotalPaths)},
		ResultsHeading: "Conversions",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s conversions get <from> <to>", support.CLIName),
			fmt.Sprintf("%s graphs convert <graph-id> <target-format>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("conversions get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: conversions get <from> <to>")
	}
	from := fs.Arg(0)
	to := fs.Arg(1)

	body, err := core.Get("/conversions/"+from+"/"+to, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Conversion: %s -> %s", from, to)},
		ResultsHeading: "Metadata",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s graphs convert <graph-id> %s", support.CLIName, to)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func conversionRows(conversions map[string][]map[string]interface{}) []string {
	if len(conversions) == 0 {
		return []string{"(no conversions registered)"}
	}
	rows := make([]string, 0)
	// Stable ordering by source format.
	keys := make([]string, 0, len(conversions))
	for k := range conversions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, from := range keys {
		rows = append(rows, fmt.Sprintf("=== from %s ===", from))
		for _, entry := range conversions[from] {
			target := stringField(entry, "target")
			name := stringField(entry, "name")
			quality := stringField(entry, "quality")
			line := fmt.Sprintf("-> %s", target)
			if name != "" {
				line += fmt.Sprintf(" (%s)", name)
			}
			if quality != "" {
				line += fmt.Sprintf(" | quality=%s", quality)
			}
			if dataLoss, ok := entry["data_loss"].(bool); ok && dataLoss {
				line += " | data_loss=true"
			}
			rows = append(rows, line)
		}
	}
	return rows
}

func stringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
