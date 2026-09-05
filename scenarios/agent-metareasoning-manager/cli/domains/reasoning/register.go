package reasoning

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"agent-metareasoning-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the `reasoning` subcommand group covering the native
// metareasoning endpoints and the results inspection surface. The
// ReasoningRequest body shape is non-trivial (input, type, model,
// chain_type, custom_chain, context, constraints, metadata) so mutating
// commands take `--body-file` rather than synthesizing JSON from flags.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "reasoning",
		Description: "Invoke native metareasoning analyses and inspect results",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "run", Description: "Run a generic reasoning request (POST /reasoning)", Run: func(args []string) error { return runReasoning(core, args, "/reasoning", "run") }},
			{Name: "pros-cons", Description: "Run a pros/cons analysis", Run: func(args []string) error { return runReasoning(core, args, "/reasoning/pros-cons", "pros-cons") }},
			{Name: "swot", Description: "Run a SWOT analysis", Run: func(args []string) error { return runReasoning(core, args, "/reasoning/swot", "swot") }},
			{Name: "risk", Description: "Run a risk assessment", Run: func(args []string) error { return runReasoning(core, args, "/reasoning/risk-assessment", "risk") }},
			{Name: "self-review", Description: "Run a self-review analysis", Run: func(args []string) error { return runReasoning(core, args, "/reasoning/self-review", "self-review") }},
			{Name: "chain", Description: "Start a reasoning chain (POST /reasoning/chain)", Run: func(args []string) error { return runReasoning(core, args, "/reasoning/chain", "chain") }},
			{Name: "chain-status", Description: "Get status of a running reasoning chain", Run: func(args []string) error { return runChainStatus(core, args) }},
			{Name: "results", Description: "List reasoning results", Run: func(args []string) error { return runResultsList(core, args) }},
			{Name: "result", Description: "Get a single reasoning result by id", Run: func(args []string) error { return runResultGet(core, args) }},
		},
	}
}

func runReasoning(core *cliapp.ScenarioApp, args []string, path, verb string) error {
	fs := support.NewFlagSet("reasoning " + verb)
	bodyFile := fs.String("body-file", "", "Path to a JSON file containing the reasoning request body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return fmt.Errorf("%s requires --body-file PATH containing the request body: %w", verb, err)
	}

	body, err := core.Request("POST", path, nil, json.RawMessage(raw))
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if decodeErr := support.Decode(body, &data); decodeErr != nil {
		data = nil
	}

	status := []string{fmt.Sprintf("Reasoning: %s", verb)}
	if data != nil {
		if id, ok := data["id"].(string); ok && id != "" {
			status = append(status, fmt.Sprintf("ID: %s", id))
		}
		if t, ok := data["type"].(string); ok && t != "" {
			status = append(status, fmt.Sprintf("Type: %s", t))
		}
		if success, ok := data["success"].(bool); ok {
			status = append(status, fmt.Sprintf("Success: %t", success))
		}
		if conf, ok := data["confidence"].(float64); ok {
			status = append(status, fmt.Sprintf("Confidence: %.3f", conf))
		}
	}

	triage := []cliapp.TriageGroup{}
	if data != nil {
		triage = append(triage, cliapp.TriageGroup{Heading: "Response", Items: support.MapRows(data)})
	} else {
		triage = append(triage, cliapp.TriageGroup{Heading: "Raw response", Items: []string{string(body)}})
	}

	report := cliapp.OperationalReport{
		Status: status,
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("%s reasoning results --limit 20", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runChainStatus(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reasoning chain-status")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reasoning chain-status <chain-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/reasoning/chain/"+id+"/status", nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Chain: %s", id)},
		ResultsHeading: "Status",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s reasoning results --limit 20", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runResultsList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reasoning results")
	limit := fs.Int("limit", 0, "Maximum rows (0 = API default)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{})
	if *limit > 0 {
		query.Set("limit", strconv.Itoa(*limit))
	}

	body, err := core.Get("/reasoning/results", query)
	if err != nil {
		return err
	}
	var page support.ReasoningResultsPage
	if err := support.Decode(body, &page); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Reasoning results: %d", page.Count)},
		ResultsHeading: "Results",
		Results:        resultRows(page.Results),
		RetrievalHints: []string{
			fmt.Sprintf("%s reasoning result <id>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runResultGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("reasoning result")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: reasoning result <result-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/reasoning/results/"+id, nil)
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Result: %s", id)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s reasoning results --limit 20", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func resultRows(results []support.ReasoningResultSummary) []string {
	if len(results) == 0 {
		return []string{"(no results)"}
	}
	rows := make([]string, 0, len(results))
	for _, r := range results {
		row := fmt.Sprintf("%s | type=%s | success=%t | confidence=%.3f | time=%dms",
			support.ShortID(r.ID), r.Type, r.Success, r.Confidence, r.ExecutionTimeMS)
		if r.Error != "" {
			row += " | error=" + r.Error
		}
		if r.CreatedAt != "" {
			row += " | at=" + r.CreatedAt
		}
		rows = append(rows, row)
	}
	return rows
}
