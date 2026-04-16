package customize

import (
	"encoding/json"
	"fmt"
	"os"

	"landing-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `customize` as a flat command. The API is `POST /api/v1/customize`
// which delegates to the issue tracker and agent pipeline.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Customization",
		Commands: []cliapp.Command{
			{
				Name:        "customize",
				Description: "Trigger agent customization of a generated landing page",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("customize")
	briefFile := fs.String("brief-file", "", "Path to a plain-text brief file (required unless --body-file is used)")
	previewMode := fs.Bool("preview", false, "Run in preview mode")
	persona := fs.String("persona", "", "Optional persona ID to guide the agent")
	bodyFile := fs.String("body-file", "", "Path to a full JSON request body; overrides other flags if provided")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var raw json.RawMessage

	if *bodyFile != "" {
		var err error
		raw, err = support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
	} else {
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: customize <scenario-id> --brief-file <path> [--preview] [--persona <id>]")
		}
		scenarioID := fs.Arg(0)
		if *briefFile == "" {
			return fmt.Errorf("--brief-file is required (or use --body-file for a full JSON payload)")
		}
		brief, err := os.ReadFile(*briefFile)
		if err != nil {
			return fmt.Errorf("read brief file %s: %w", *briefFile, err)
		}

		payload := map[string]interface{}{
			"scenario_id": scenarioID,
			"brief":       string(brief),
			"preview":     *previewMode,
		}
		if *persona != "" {
			payload["persona_id"] = *persona
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal customize payload: %w", err)
		}
		raw = data
	}

	body, err := core.Request("POST", "/customize", nil, raw)
	if err != nil {
		return err
	}

	var decoded map[string]interface{}
	if err := support.Decode(body, &decoded); err != nil {
		decoded = map[string]interface{}{}
	}

	message := support.EnvelopeMessage(body)
	if message == "" {
		if v, ok := decoded["message"].(string); ok && v != "" {
			message = v
		} else {
			message = "Customization request queued"
		}
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: support.MapRows(decoded),
		NextCommand: []string{
			fmt.Sprintf("%s preview <scenario-id>", support.CLIName),
			fmt.Sprintf("%s analytics events", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
