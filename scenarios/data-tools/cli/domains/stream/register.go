package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"data-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `stream` subcommand group for streaming data source
// management. The API currently only exposes create (POST /api/v1/data/stream/
// create); the bash CLI advertised list/start/stop/status subcommands but none
// were wired to an endpoint and they are intentionally not implemented here.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "stream",
		Description: "Manage streaming data sources",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "create", Description: "Create a streaming data source", Run: func(args []string) error { return runCreate(core, args) }},
		},
	}
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("stream create")
	source := fs.String("source", "", "Source type (kafka, webhook, file_watch)")
	configFile := fs.String("config-file", "", "Path to connection config JSON file")
	rulesFile := fs.String("rules-file", "", "Path to processing rules JSON file")
	destination := fs.String("destination", "dataset", "Output destination type")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
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
		if strings.TrimSpace(*source) == "" {
			return fmt.Errorf("--source is required (or pass --body-file PATH)")
		}
		if strings.TrimSpace(*configFile) == "" {
			return fmt.Errorf("--config-file is required (or pass --body-file PATH)")
		}
		config, err := support.ReadJSONFile(*configFile, true)
		if err != nil {
			return err
		}
		rules, err := support.ReadJSONFile(*rulesFile, false)
		if err != nil {
			return err
		}
		if rules == nil {
			rules = json.RawMessage("[]")
		}
		payload = map[string]interface{}{
			"source_config": map[string]interface{}{
				"type":       *source,
				"connection": config,
			},
			"processing_rules": rules,
			"output_config": map[string]interface{}{
				"destination": *destination,
				"config":      map[string]interface{}{},
			},
		}
	}

	body, err := core.Request("POST", "/data/stream/create", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	streamID, _ := result["stream_id"].(string)
	if streamID == "" {
		if id, ok := result["id"].(string); ok {
			streamID = id
		}
	}
	if streamID == "" {
		streamID = "<unknown>"
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Stream created: %s", streamID)},
		Changes:     support.MapRows(result),
		NextCommand: []string{fmt.Sprintf("%s stream create --source <type> --config-file <path>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
