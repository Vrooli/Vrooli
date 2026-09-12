package analyze

import (
	"fmt"
	"os"

	"file-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `analyze` subcommand group. All four endpoints accept
// nested option payloads, so we standardize on `--body-file PATH` rather than
// hand-rolling flags for every option. Response shapes vary and are surfaced
// via MapRows for human output.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "analyze",
		Description: "File relationship, storage, access pattern, and integrity analysis",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "relationships", Description: "Map file relationships and dependencies (--body-file PATH)", Run: func(args []string) error {
				return runJSONBody(core, args, "analyze relationships", "/files/relationships/map", "Relationship mapping complete")
			}},
			{Name: "storage", Description: "Storage optimization recommendations (--body-file PATH)", Run: func(args []string) error {
				return runJSONBody(core, args, "analyze storage", "/files/storage/optimize", "Storage optimization complete")
			}},
			{Name: "access", Description: "File access pattern analysis (--body-file PATH)", Run: func(args []string) error {
				return runJSONBody(core, args, "analyze access", "/files/access/analyze", "Access pattern analysis complete")
			}},
			{Name: "integrity", Description: "File integrity monitoring (--body-file PATH)", Run: func(args []string) error {
				return runJSONBody(core, args, "analyze integrity", "/files/integrity/monitor", "Integrity monitoring complete")
			}},
		},
	}
}

func runJSONBody(core *cliapp.ScenarioApp, args []string, label, path, summary string) error {
	fs := support.NewFlagSet(label)
	bodyFile := fs.String("body-file", "", "JSON body file (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	payload, err := support.ReadJSONBody(*bodyFile, true)
	if err != nil {
		return err
	}
	body, err := core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summaryLines := []string{summary}
	if op, ok := resp["operation_id"].(string); ok && op != "" {
		summaryLines = append(summaryLines, fmt.Sprintf("Operation: %s", op))
	}
	report := cliapp.ListReport{
		Summary:        summaryLines,
		ResultsHeading: "Fields",
		Results:        support.MapRows(resp),
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}
