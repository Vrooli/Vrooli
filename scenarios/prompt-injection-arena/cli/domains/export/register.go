package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"prompt-injection-arena/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `export` subcommand group covering /api/v1/export/*.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "export",
		Description: "Export research data in supported formats",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "formats", Description: "List supported export formats", Run: func(args []string) error { return runFormats(core, args) }},
			{Name: "run", Description: "Run a research export", Run: func(args []string) error { return runExport(core, args) }},
		},
	}
}

func runFormats(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("export formats")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/export/formats", nil)
	if err != nil {
		return err
	}

	var resp support.ExportFormatsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	results := make([]string, 0, len(resp.Formats))
	for _, f := range resp.Formats {
		results = append(results, fmt.Sprintf("%s (%s) — %s [%s]",
			f.Name, f.Format, f.Description, f.ContentType))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Supported formats: %d", len(resp.Formats)),
			fmt.Sprintf("Default: %s", resp.Default),
		},
		ResultsHeading: "Formats",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s export run --format json --output export.json", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("export run")
	format := fs.String("format", "json", "Export format: json|csv|markdown")
	output := fs.String("output", "", "Output file path (default: stdout)")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full request body (overrides --format and supplies filters)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		payload = map[string]interface{}{
			"format":  *format,
			"filters": map[string]interface{}{},
		}
	}

	body, err := core.Request("POST", "/export/research", nil, payload)
	if err != nil {
		return err
	}

	dest := strings.TrimSpace(*output)
	if dest == "" {
		if _, err := os.Stdout.Write(body); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(dest, body, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Export complete (format=%s, bytes=%d)", *format, len(body)),
		},
		Changes: []string{
			func() string {
				if dest == "" {
					return "Output: stdout"
				}
				return fmt.Sprintf("Output: %s", dest)
			}(),
		},
		NextCommand: []string{
			fmt.Sprintf("%s export formats", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
