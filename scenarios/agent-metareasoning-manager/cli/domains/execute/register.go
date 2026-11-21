package execute

import (
	"encoding/json"
	"fmt"
	"os"

	"agent-metareasoning-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `agent-metareasoning-manager execute` as a flat command
// that proxies to `POST /execute/{platform}/{workflowId}`. Body payloads can
// be arbitrary JSON (forwarded to n8n or windmill), so the CLI takes a
// `--body-file` rather than synthesizing JSON from scalar flags.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Execute",
		Commands: []cliapp.Command{
			{
				Name:        "execute",
				Description: "Proxy a workflow execution to its platform (n8n or windmill)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runExecute(core, args) },
			},
		},
	}
}

func runExecute(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("execute")
	bodyFile := fs.String("body-file", "", "Path to JSON file used as the proxied request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: execute <platform> <workflow-id> [--body-file PATH]")
	}

	platform := fs.Arg(0)
	workflowID := fs.Arg(1)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = json.RawMessage(raw)
	}

	body, err := core.Request("POST", "/execute/"+platform+"/"+workflowID, nil, payload)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if decodeErr := support.Decode(body, &data); decodeErr != nil {
		data = nil
	}

	report := cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Executed %s workflow %s", platform, workflowID)},
		Changes: []string{
			fmt.Sprintf("Proxied POST to /execute/%s/%s", platform, workflowID),
		},
		NextCommand: []string{
			fmt.Sprintf("%s reasoning results --limit 10", support.CLIName),
			fmt.Sprintf("%s workflows list", support.CLIName),
		},
	}
	if data != nil {
		report.Result = append(report.Result, support.MapRows(data)...)
	} else {
		report.Result = append(report.Result, fmt.Sprintf("Raw response: %s", string(body)))
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
