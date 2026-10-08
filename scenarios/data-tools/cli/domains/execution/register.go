package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"data-tools/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `execution` subcommand group covering workflow execution
// against /api/v1/execute and /api/v1/executions. The bash CLI split these
// across three top-level verbs (`execute`, `executions`, `status`); combining
// them under `execution` matches the REST path shape.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "execution",
		Description: "Execute and inspect workflows",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List workflow executions", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"status", "show"}, Description: "Show execution status and details", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "run", Aliases: []string{"execute", "exec"}, Description: "Execute a workflow", Run: func(args []string) error { return runExecute(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("execution list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/executions", nil)
	if err != nil {
		return err
	}
	var executions []support.Execution
	if err := support.Decode(body, &executions); err != nil {
		return err
	}

	rows := make([]string, 0, len(executions))
	for _, e := range executions {
		workflow := e.WorkflowID
		if workflow == "" {
			workflow = "(unknown)"
		}
		status := e.Status
		if status == "" {
			status = "(unknown)"
		}
		started := e.StartedAt
		if started == "" {
			started = "unknown"
		}
		rows = append(rows, fmt.Sprintf("%s | workflow=%s | status=%s | started=%s",
			support.ShortID(e.ID), workflow, status, support.FormatTime(started)))
	}
	if len(rows) == 0 {
		rows = []string{"No executions found"}
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Executions: %d", len(executions))},
		ResultsHeading: "Executions",
		Results:        rows,
		RetrievalHints: []string{
			fmt.Sprintf("%s execution get <execution-id>", support.CLIName),
			fmt.Sprintf("%s execution run <workflow-id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("execution get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: execution get <execution-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/executions/"+id, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Execution: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{fmt.Sprintf("%s execution list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runExecute(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("execution run")
	inputLiteral := fs.String("input", "", "Inline JSON input for the workflow")
	inputFile := fs.String("input-file", "", "Path to JSON input file")
	bodyFile := fs.String("body-file", "", "Path to JSON file containing the full request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 && strings.TrimSpace(*bodyFile) == "" {
		return fmt.Errorf("usage: execution run <workflow-id> [--input '{...}' | --input-file PATH | --body-file PATH]")
	}

	var payload map[string]interface{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("--body-file must contain a JSON object: %w", err)
		}
	} else {
		workflowID := fs.Arg(0)
		payload = map[string]interface{}{"workflow_id": workflowID}

		switch {
		case strings.TrimSpace(*inputLiteral) != "":
			var inputValue json.RawMessage
			if err := json.Unmarshal([]byte(*inputLiteral), &inputValue); err != nil {
				return fmt.Errorf("parse --input JSON: %w", err)
			}
			payload["input"] = inputValue
		case strings.TrimSpace(*inputFile) != "":
			inputValue, err := support.ReadJSONFile(*inputFile, true)
			if err != nil {
				return err
			}
			payload["input"] = inputValue
		}
	}

	body, err := core.Request("POST", "/execute", nil, payload)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	executionID, _ := result["execution_id"].(string)
	if executionID == "" {
		if id, ok := result["id"].(string); ok {
			executionID = id
		}
	}
	if executionID == "" {
		executionID = "<unknown>"
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Execution started: %s", executionID)},
		Changes:     support.MapRows(result),
		NextCommand: []string{fmt.Sprintf("%s execution get %s", support.CLIName, executionID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
