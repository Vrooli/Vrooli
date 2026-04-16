package document

import (
	"fmt"
	"os"
	"strings"

	"idea-generator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `document` subcommand group wrapping /api/documents/process.
// The API owns extraction, storage, and vector indexing; this package only shapes
// the request body from CLI flags.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "document",
		Description: "Process uploaded documents for idea context",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "process", Description: "Trigger extraction for a stored document", Run: func(args []string) error { return runProcess(core, args) }},
		},
	}
}

func runProcess(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("document process")
	docID := fs.String("id", "", "Document ID (required)")
	campaign := fs.String("campaign", "", "Campaign ID the document belongs to (required)")
	filePath := fs.String("file", "", "Stored file path the API should read")
	bodyFile := fs.String("body-file", "", "Path to JSON request body, or '-' for stdin. Overrides other flags if set.")
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
		if strings.TrimSpace(*docID) == "" || strings.TrimSpace(*campaign) == "" {
			return fmt.Errorf("--id and --campaign are required (or supply --body-file)")
		}
		built := map[string]interface{}{
			"document_id": strings.TrimSpace(*docID),
			"campaign_id": strings.TrimSpace(*campaign),
		}
		if strings.TrimSpace(*filePath) != "" {
			built["file_path"] = strings.TrimSpace(*filePath)
		}
		payload = built
	}

	body, err := core.Request("POST", "/documents/process", nil, payload)
	if err != nil {
		return err
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		var resp map[string]interface{}
		if err := support.Decode(body, &resp); err == nil {
			if m, ok := resp["message"].(string); ok {
				message = m
			}
		}
	}
	if message == "" {
		message = "Document processing started"
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Document %s queued", support.ShortID(strings.TrimSpace(*docID)))},
		NextCommand: []string{
			fmt.Sprintf("%s idea generate --campaign %s --prompt \"...\"", support.CLIName, strings.TrimSpace(*campaign)),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
