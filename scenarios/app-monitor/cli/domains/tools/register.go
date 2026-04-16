package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"app-monitor/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `tool` subcommand group covering the Tool Discovery Protocol.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "tool",
		Description: "Inspect and execute discovery-protocol tools",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List available tools", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show one tool definition", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "execute", Aliases: []string{"run"}, Description: "Execute a tool with JSON input", Run: func(args []string) error { return runExecute(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/tools", nil)
	if err != nil {
		return err
	}
	var manifest support.ToolsManifest
	if err := support.Decode(body, &manifest); err != nil {
		return err
	}

	rows := make([]string, 0, len(manifest.Tools))
	for _, t := range manifest.Tools {
		rows = append(rows, fmt.Sprintf("%s | category=%s | %s", t.Name, t.Category, t.Description))
	}
	if len(rows) == 0 {
		rows = []string{"(no tools)"}
	}

	summary := []string{fmt.Sprintf("Tools: %d", len(manifest.Tools))}
	if manifest.Scenario != "" {
		summary = append(summary, fmt.Sprintf("Scenario: %s (v%s)", manifest.Scenario, manifest.Version))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Tools",
		Results:        rows,
		RetrievalHints: []string{fmt.Sprintf("%s tool get <tool-name>", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tool get <tool-name>")
	}
	name := fs.Arg(0)

	body, err := core.Get("/tools/"+name, nil)
	if err != nil {
		return err
	}
	var tool support.ToolManifest
	if err := support.Decode(body, &tool); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Name: %s", tool.Name),
		fmt.Sprintf("Category: %s", tool.Category),
	}
	if tool.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", tool.Description))
	}
	if len(tool.InputSchema) > 0 {
		results = append(results, fmt.Sprintf("Input schema: %s", string(tool.InputSchema)))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Tool: %s", tool.Name)},
		ResultsHeading: "Definition",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s tool execute %s --input-file ./input.json", support.CLIName, tool.Name)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runExecute(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("tool execute")
	inputFile := fs.String("input-file", "", "Path to JSON input for the tool (default: stdin)")
	inputLiteral := fs.String("input", "", "Inline JSON input")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: tool execute <tool-name> [--input-file path | --input '{...}']")
	}
	name := fs.Arg(0)

	var input json.RawMessage
	switch {
	case strings.TrimSpace(*inputLiteral) != "":
		if err := json.Unmarshal([]byte(*inputLiteral), &input); err != nil {
			return fmt.Errorf("parse --input JSON: %w", err)
		}
	case strings.TrimSpace(*inputFile) != "":
		loaded, err := support.ReadJSONFile(*inputFile, true)
		if err != nil {
			return err
		}
		input = loaded
	default:
		input = json.RawMessage(`{}`)
	}

	body, err := core.Request("POST", "/tools/execute", nil, map[string]interface{}{
		"name":  name,
		"input": input,
	})
	if err != nil {
		return err
	}
	var data map[string]interface{}
	if err := support.Decode(body, &data); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Executed tool %s", name)},
		Changes:     support.MapRows(data),
		NextCommand: []string{fmt.Sprintf("%s tool list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}
