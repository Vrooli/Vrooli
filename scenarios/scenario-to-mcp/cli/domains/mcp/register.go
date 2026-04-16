package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"scenario-to-mcp/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `mcp` subcommand group covering MCP endpoint discovery,
// registry, session inspection, and the POST-based add flow. The API is the
// source of truth; this package is a thin wrapper that formats responses
// through the standard output contracts.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "mcp",
		Description: "Inspect and manage MCP endpoints across scenarios",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "endpoints", Aliases: []string{"list"}, Description: "List MCP endpoint status across scenarios", Run: func(args []string) error { return runEndpoints(core, args) }},
			{Name: "registry", Description: "Show MCP registry for service discovery", Run: func(args []string) error { return runRegistry(core, args) }},
			{Name: "scenario", Aliases: []string{"show"}, Description: "Show MCP details for one scenario", Run: func(args []string) error { return runScenario(core, args) }},
			{Name: "session", Description: "Show agent session status", Run: func(args []string) error { return runSession(core, args) }},
			{Name: "add", Description: "Spawn a claude-code agent to add MCP support to a scenario", Run: func(args []string) error { return runAdd(core, args) }},
		},
	}
}

func runEndpoints(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("mcp endpoints")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/mcp/endpoints", nil)
	if err != nil {
		return err
	}
	var resp support.MCPEndpointsResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Scenarios scanned: %d", len(resp.Scenarios))}
	if len(resp.Summary) > 0 {
		keys := make([]string, 0, len(resp.Summary))
		for k := range resp.Summary {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			summary = append(summary, fmt.Sprintf("%s: %d", k, resp.Summary[k]))
		}
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Endpoints",
		Results:        endpointRows(resp.Scenarios),
		RetrievalHints: []string{
			fmt.Sprintf("%s mcp scenario <name>", support.CLIName),
			fmt.Sprintf("%s mcp registry", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runRegistry(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("mcp registry")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/mcp/registry", nil)
	if err != nil {
		return err
	}
	var registry support.MCPRegistry
	if err := support.Decode(body, &registry); err != nil {
		return err
	}

	summary := []string{fmt.Sprintf("Registry version: %s", registry.Version), fmt.Sprintf("Endpoints: %d", len(registry.Endpoints))}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Registry",
		Results:        registryRows(registry.Endpoints),
		RetrievalHints: []string{
			fmt.Sprintf("%s mcp endpoints", support.CLIName),
			fmt.Sprintf("%s mcp scenario <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runScenario(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("mcp scenario")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: mcp scenario <name>")
	}
	name := fs.Arg(0)

	body, err := core.Get("/mcp/scenarios/"+name, nil)
	if err != nil {
		return err
	}
	// The detector returns a free-form JSON object; decode as a map so we can
	// render every field without imposing a rigid schema.
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario: %s", name)},
		ResultsHeading: "Details",
		Results:        support.MapRows(data),
		RetrievalHints: []string{
			fmt.Sprintf("%s mcp endpoints", support.CLIName),
			fmt.Sprintf("%s mcp add %s", support.CLIName, name),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runSession(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("mcp session")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: mcp session <session-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/mcp/sessions/"+id, nil)
	if err != nil {
		return err
	}
	var session support.MCPSession
	if err := support.Decode(body, &session); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", session.ID),
		fmt.Sprintf("Scenario: %s", session.ScenarioName),
		fmt.Sprintf("Status: %s", session.Status),
		fmt.Sprintf("Started: %s", support.FormatTimeValue(session.StartTime)),
	}
	if session.EndTime != nil {
		results = append(results, fmt.Sprintf("Ended: %s", support.FormatTimeValue(*session.EndTime)))
	}
	if session.Logs != "" {
		results = append(results, fmt.Sprintf("Logs: %s", session.Logs))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Session: %s (%s)", support.ShortID(session.ID), session.Status)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s mcp session %s --json", support.CLIName, session.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runAdd(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("mcp add")
	template := fs.String("template", "", "MCP server template to use (e.g. basic-api|data-processor)")
	auto := fs.Bool("auto", false, "Auto-detect and generate optimal MCP config")
	bodyFile := fs.String("body-file", "", "Optional path to a JSON file with the full AddMCPRequest body")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 && *bodyFile == "" {
		return fmt.Errorf("usage: mcp add <scenario> [--template NAME] [--auto] [--body-file PATH]")
	}

	payload, err := buildAddPayload(fs.Arg(0), *template, *auto, *bodyFile)
	if err != nil {
		return err
	}

	body, err := core.Request("POST", "/mcp/add", nil, payload)
	if err != nil {
		return err
	}
	var resp support.MCPAddResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	target := fs.Arg(0)
	if target == "" {
		target = "(from body-file)"
	}

	changes := []string{fmt.Sprintf("Scenario %s: MCP add session created", target)}
	if resp.AgentSessionID != "" {
		changes = append(changes, fmt.Sprintf("Session ID: %s", resp.AgentSessionID))
	}
	if resp.EstimatedTime > 0 {
		changes = append(changes, fmt.Sprintf("Estimated time: %ds", resp.EstimatedTime))
	}

	result := []string{"Claude-code agent spawned"}
	if message := support.EnvelopeMessage(body); message != "" {
		result = []string{message}
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s mcp session %s", support.CLIName, resp.AgentSessionID),
			fmt.Sprintf("%s mcp endpoints", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func buildAddPayload(scenario, template string, auto bool, bodyFile string) (json.RawMessage, error) {
	if bodyFile != "" {
		return support.ReadJSONFile(bodyFile, true)
	}

	payload := map[string]interface{}{
		"scenario_name": scenario,
		"agent_config": map[string]interface{}{
			"auto_detect": auto,
		},
	}
	if template != "" {
		payload["agent_config"].(map[string]interface{})["template"] = template
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	return raw, nil
}

func endpointRows(endpoints []support.MCPEndpoint) []string {
	if len(endpoints) == 0 {
		return []string{"No scenarios scanned"}
	}
	rows := make([]string, 0, len(endpoints))
	for _, e := range endpoints {
		status := e.Status
		if status == "" {
			if e.HasMCP {
				status = "no-status"
			} else {
				status = "no-mcp"
			}
		}
		port := "-"
		if e.MCPPort > 0 {
			port = fmt.Sprintf("%d", e.MCPPort)
		}
		confidence := e.Confidence
		if confidence == "" {
			confidence = "-"
		}
		rows = append(rows, fmt.Sprintf("%s | status=%s | port=%s | tools=%d | confidence=%s",
			e.ScenarioName, status, port, len(e.Tools), confidence))
	}
	return rows
}

func registryRows(endpoints []support.MCPRegistryEndpoint) []string {
	if len(endpoints) == 0 {
		return []string{"(registry empty)"}
	}
	rows := make([]string, 0, len(endpoints))
	for _, e := range endpoints {
		transport := e.Transport
		if transport == "" {
			transport = "-"
		}
		rows = append(rows, fmt.Sprintf("%s | transport=%s | url=%s", e.Name, transport, e.URL))
	}
	return rows
}
