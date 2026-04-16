package index

import (
	"fmt"
	"os"
	"strings"

	"document-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `document-manager index` as a flat command wrapping
// POST /api/index. The API expects a nested {application_id, documents[]}
// payload, so the command takes `--body-file` rather than hand-building JSON.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Index",
		Commands: []cliapp.Command{
			{
				Name:        "index",
				Description: "Index documents for similarity search (body from --body-file)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("index")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/index", nil, raw)
	if err != nil {
		return err
	}
	var resp support.IndexResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Indexed: %d", resp.Indexed),
		fmt.Sprintf("Failed: %d", resp.Failed),
	}
	changes := []string{
		fmt.Sprintf("Documents indexed: %d", resp.Indexed),
		fmt.Sprintf("Documents failed: %d", resp.Failed),
	}
	if len(resp.Errors) > 0 {
		changes = append(changes, "Errors:")
		for _, e := range resp.Errors {
			changes = append(changes, "  - "+strings.TrimSpace(e))
		}
	}

	report := cliapp.MutationReport{
		Result:      summary,
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("%s search --query <text>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
