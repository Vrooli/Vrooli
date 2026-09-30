package analyze

import (
	"fmt"
	"os"
	"strings"

	"agent-metareasoning-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes `agent-metareasoning-manager analyze` as a flat command
// since it is a single verb that routes analysis types to backend workflows
// via `POST /analyze`.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Analyze",
		Commands: []cliapp.Command{
			{
				Name:        "analyze",
				Description: "Route an analysis request to the matching workflow",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runAnalyze(core, args) },
			},
		},
	}
}

func runAnalyze(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("analyze")
	analysisType := fs.String("type", "", "Analysis type (pros-cons, swot, risk, decision, self-review, reasoning-chain)")
	input := fs.String("input", "", "Input text to analyze")
	context := fs.String("context", "", "Optional context string")
	model := fs.String("model", "", "Optional model override")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	// Allow positional usage: analyze <type> <input> [context]
	positional := fs.Args()
	if strings.TrimSpace(*analysisType) == "" && len(positional) > 0 {
		*analysisType = positional[0]
		positional = positional[1:]
	}
	if strings.TrimSpace(*input) == "" && len(positional) > 0 {
		*input = positional[0]
		positional = positional[1:]
	}
	if strings.TrimSpace(*context) == "" && len(positional) > 0 {
		*context = positional[0]
	}

	if strings.TrimSpace(*analysisType) == "" || strings.TrimSpace(*input) == "" {
		return fmt.Errorf("usage: analyze --type <type> --input <text> [--context <text>] [--model <name>]")
	}

	payload := map[string]interface{}{
		"type":  *analysisType,
		"input": *input,
	}
	if strings.TrimSpace(*context) != "" {
		payload["context"] = *context
	}
	if strings.TrimSpace(*model) != "" {
		payload["model"] = *model
	}

	body, err := core.Request("POST", "/analyze", nil, payload)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	// The API returns {analysis_type, platform, workflow_id, result, metadata}.
	status := []string{}
	if v, ok := data["analysis_type"].(string); ok && v != "" {
		status = append(status, fmt.Sprintf("Type: %s", v))
	}
	if v, ok := data["platform"].(string); ok && v != "" {
		status = append(status, fmt.Sprintf("Platform: %s", v))
	}
	if v, ok := data["workflow_id"].(string); ok && v != "" {
		status = append(status, fmt.Sprintf("Workflow: %s", v))
	}
	if len(status) == 0 {
		status = []string{fmt.Sprintf("Analysis dispatched for type %q", *analysisType)}
	}

	triage := []cliapp.TriageGroup{}
	if result, ok := data["result"].(map[string]interface{}); ok {
		triage = append(triage, cliapp.TriageGroup{Heading: "Result", Items: support.MapRows(result)})
	}
	if meta, ok := data["metadata"].(map[string]interface{}); ok {
		triage = append(triage, cliapp.TriageGroup{Heading: "Metadata", Items: support.MapRows(meta)})
	}

	report := cliapp.OperationalReport{
		Status: status,
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("%s workflows list", support.CLIName),
			fmt.Sprintf("%s reasoning results", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}
