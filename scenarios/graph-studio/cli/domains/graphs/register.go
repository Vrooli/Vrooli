package graphs

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"graph-studio/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register builds the `graphs` subcommand group covering the full
// /api/v1/graphs surface. Commands are thin wrappers: flags map to query
// parameters or request bodies, responses are formatted via the standard
// output contracts. Writes that require nested JSON (update, convert, render,
// export) expose a `--body-file` flag rather than hand-assembling bodies.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "graphs",
		Description: "Manage graphs (list, create, get, update, delete, validate, convert, render, export)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List graphs", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", Description: "Create a new graph", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show a graph by ID", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "update", Description: "Update a graph (requires --body-file)", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a graph", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "validate", Description: "Validate a graph against its schema", Run: func(args []string) error { return runValidate(core, args) }},
			{Name: "convert", Description: "Convert a graph to another format", Run: func(args []string) error { return runConvert(core, args) }},
			{Name: "render", Description: "Render a graph as svg/html", Run: func(args []string) error { return runRender(core, args) }},
			{Name: "export", Description: "Export a graph (graphml, gexf, json)", Run: func(args []string) error { return runExport(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs list")
	graphType := fs.String("type", "", "Filter by graph type (plugin id)")
	tag := fs.String("tag", "", "Filter by tag")
	search := fs.String("search", "", "Full-text search across name/description/tags")
	limit := fs.Int("limit", 50, "Max results to return")
	offset := fs.Int("offset", 0, "Pagination offset")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	query := support.BuildQuery(map[string]string{
		"type":   *graphType,
		"tag":    *tag,
		"search": *search,
		"limit":  strconv.Itoa(*limit),
		"offset": strconv.Itoa(*offset),
	})
	body, err := core.Get("/graphs", query)
	if err != nil {
		return err
	}
	var resp support.ListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	var graphs []support.Graph
	if len(resp.Data) > 0 {
		if err := support.Decode(resp.Data, &graphs); err != nil {
			return err
		}
	}

	summary := []string{fmt.Sprintf("Graphs: %d (total in page)", resp.Total)}
	if resp.Limit != "" {
		summary = append(summary, fmt.Sprintf("Limit: %s | Offset: %s", resp.Limit, resp.Offset))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Graphs",
		Results:        graphRows(graphs),
		RetrievalHints: []string{
			fmt.Sprintf("%s graphs get <graph-id>", support.CLIName),
			fmt.Sprintf("%s graphs list --type mind-maps --limit 100", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs create")
	name := fs.String("name", "", "Graph name (required)")
	graphType := fs.String("type", "", "Graph type / plugin id (required)")
	description := fs.String("description", "", "Graph description")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the full CreateGraphRequest (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	// Support positional form: `graphs create <name> <type>` to mirror the legacy bash CLI.
	if *name == "" && fs.NArg() >= 1 {
		*name = fs.Arg(0)
	}
	if *graphType == "" && fs.NArg() >= 2 {
		*graphType = fs.Arg(1)
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *name == "" || *graphType == "" {
			return fmt.Errorf("usage: graphs create <name> <type> | --name <n> --type <t> [--description <d>] [--body-file path]")
		}
		payload = map[string]interface{}{
			"name":        *name,
			"type":        *graphType,
			"description": *description,
		}
	}

	body, err := core.Request("POST", "/graphs", nil, payload)
	if err != nil {
		return err
	}

	var created map[string]interface{}
	if err := support.Decode(body, &created); err != nil {
		return err
	}

	id, _ := created["id"].(string)
	result := []string{fmt.Sprintf("Created graph %s", id)}
	if n, ok := created["name"].(string); ok && n != "" {
		result = append(result, fmt.Sprintf("Name: %s", n))
	}
	if t, ok := created["type"].(string); ok && t != "" {
		result = append(result, fmt.Sprintf("Type: %s", t))
	}

	report := cliapp.MutationReport{
		Result:  result,
		Changes: []string{fmt.Sprintf("Graph %s created", id)},
		NextCommand: []string{
			fmt.Sprintf("%s graphs get %s", support.CLIName, id),
			fmt.Sprintf("%s graphs validate %s", support.CLIName, id),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs get <graph-id>")
	}
	id := fs.Arg(0)

	body, err := core.Get("/graphs/"+id, nil)
	if err != nil {
		return err
	}
	var graph support.Graph
	if err := support.Decode(body, &graph); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("ID: %s", graph.ID),
		fmt.Sprintf("Name: %s", graph.Name),
		fmt.Sprintf("Type: %s", graph.Type),
	}
	if graph.Description != "" {
		results = append(results, fmt.Sprintf("Description: %s", graph.Description))
	}
	if graph.Version != 0 {
		results = append(results, fmt.Sprintf("Version: %d", graph.Version))
	}
	if graph.CreatedBy != "" {
		results = append(results, fmt.Sprintf("Created by: %s", graph.CreatedBy))
	}
	if graph.CreatedAt != nil {
		results = append(results, fmt.Sprintf("Created at: %s", support.FormatTimeValue(*graph.CreatedAt)))
	}
	if graph.UpdatedAt != nil {
		results = append(results, fmt.Sprintf("Updated at: %s", support.FormatTimeValue(*graph.UpdatedAt)))
	}
	if len(graph.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %v", graph.Tags))
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Graph: %s (%s)", graph.Name, graph.Type)},
		ResultsHeading: "Details",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s graphs validate %s", support.CLIName, graph.ID),
			fmt.Sprintf("%s graphs export %s json", support.CLIName, graph.ID),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs update")
	bodyFile := fs.String("body-file", "", "JSON file with the UpdateGraphRequest payload (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs update <graph-id> --body-file PATH")
	}
	id := fs.Arg(0)

	raw, err := support.ReadJSONFile(*bodyFile, true)
	if err != nil {
		return err
	}

	body, err := core.Request("PUT", "/graphs/"+id, nil, raw)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Graph %s updated", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Graph %s updated", id)},
		NextCommand: []string{fmt.Sprintf("%s graphs get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs delete <graph-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("DELETE", "/graphs/"+id, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Graph %s deleted", id)
	}

	report := cliapp.MutationReport{
		Result:      []string{message},
		Changes:     []string{fmt.Sprintf("Graph %s deleted", id)},
		NextCommand: []string{fmt.Sprintf("%s graphs list", support.CLIName)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runValidate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs validate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs validate <graph-id>")
	}
	id := fs.Arg(0)

	body, err := core.Request("POST", "/graphs/"+id+"/validate", nil, nil)
	if err != nil {
		return err
	}
	var result support.ValidationResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	status := "invalid"
	if result.Valid {
		status = "valid"
	}

	triage := []cliapp.TriageGroup{}
	if len(result.Errors) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Errors", Items: result.Errors})
	}
	if len(result.Warnings) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Warnings", Items: result.Warnings})
	}
	if len(triage) == 0 {
		triage = []cliapp.TriageGroup{{Heading: "Details", Items: []string{"(no issues reported)"}}}
	}

	report := cliapp.OperationalReport{
		Status:    []string{fmt.Sprintf("Graph %s: %s", id, status)},
		Triage:    triage,
		NextSteps: []string{fmt.Sprintf("%s graphs get %s", support.CLIName, id)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func runConvert(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs convert")
	target := fs.String("target-format", "", "Target format / plugin id")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the full ConversionRequest (overrides other flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs convert <graph-id> <target-format> | <graph-id> --target-format <f> [--body-file path]")
	}
	id := fs.Arg(0)
	if *target == "" && fs.NArg() >= 2 {
		*target = fs.Arg(1)
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *target == "" {
			return fmt.Errorf("target format required; pass it positionally or via --target-format")
		}
		payload = map[string]interface{}{"target_format": *target}
	}

	body, err := core.Request("POST", "/graphs/"+id+"/convert", nil, payload)
	if err != nil {
		return err
	}
	var resp map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	newID, _ := resp["converted_graph_id"].(string)
	format, _ := resp["format"].(string)
	result := []string{fmt.Sprintf("Converted %s -> %s", id, format)}
	if newID != "" {
		result = append(result, fmt.Sprintf("New graph id: %s", newID))
	}

	report := cliapp.MutationReport{
		Result:      result,
		Changes:     []string{fmt.Sprintf("Created graph %s (format=%s)", newID, format)},
		NextCommand: []string{fmt.Sprintf("%s graphs get %s", support.CLIName, newID)},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runRender(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs render")
	format := fs.String("format", "svg", "Render format (svg|html)")
	output := fs.String("output", "", "Optional output file; defaults to stdout")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the full RenderRequest (overrides --format)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs render <graph-id> [--format svg|html] [--output PATH] [--body-file path]")
	}
	id := fs.Arg(0)

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		payload = map[string]interface{}{"format": *format}
	}

	// The render endpoint returns raw svg/html, not JSON, so write the response
	// body directly without attempting envelope decoding.
	body, err := core.Request("POST", "/graphs/"+id+"/render", nil, payload)
	if err != nil {
		return err
	}
	return support.WriteOutput(*output, body)
}

func runExport(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("graphs export")
	format := fs.String("format", "", "Export format (graphml|gexf|json)")
	output := fs.String("output", "", "Optional output file; when set, only the content is written")
	bodyFile := fs.String("body-file", "", "Optional JSON file with the full ExportRequest (overrides --format)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: graphs export <graph-id> <format> [--output PATH] | <graph-id> --format <f> [--body-file path]")
	}
	id := fs.Arg(0)
	// Positional format mirrors the legacy bash CLI: `graphs export <id> <format> [file]`.
	if *format == "" && fs.NArg() >= 2 {
		*format = fs.Arg(1)
	}
	if *output == "" && fs.NArg() >= 3 {
		*output = fs.Arg(2)
	}

	var payload interface{}
	if *bodyFile != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		payload = raw
	} else {
		if *format == "" {
			return fmt.Errorf("format required; pass it positionally or via --format")
		}
		payload = map[string]interface{}{"format": *format}
	}

	body, err := core.Request("POST", "/graphs/"+id+"/export", nil, payload)
	if err != nil {
		return err
	}
	var resp support.ExportResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	// With --output: write only the exported content so it can be consumed by
	// downstream tooling. Without --output: render the structured report.
	if *output != "" {
		if err := support.WriteOutput(*output, []byte(resp.Content)); err != nil {
			return err
		}
		report := cliapp.MutationReport{
			Result:      []string{fmt.Sprintf("Exported %s as %s to %s", id, resp.Format, *output)},
			Changes:     []string{fmt.Sprintf("Wrote %s bytes", strconv.Itoa(len(resp.Content)))},
			NextCommand: []string{fmt.Sprintf("%s graphs get %s", support.CLIName, id)},
		}
		if *jsonOutput {
			return cliapp.PrintReportJSON(os.Stdout, report)
		}
		return cliapp.RenderMutationReport(os.Stdout, report)
	}

	// No --output: emit the full payload for piping/inspection.
	if *jsonOutput {
		payload, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		fmt.Println(string(payload))
		return nil
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Export: %s (%s)", resp.Filename, resp.Format)},
		ResultsHeading: "Metadata",
		Results: []string{
			fmt.Sprintf("Filename: %s", resp.Filename),
			fmt.Sprintf("Format: %s", resp.Format),
			fmt.Sprintf("MIME type: %s", resp.MimeType),
			fmt.Sprintf("Content bytes: %d", len(resp.Content)),
		},
		RetrievalHints: []string{
			fmt.Sprintf("%s graphs export %s %s --output out.%s", support.CLIName, id, resp.Format, resp.Format),
		},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func graphRows(graphs []support.Graph) []string {
	if len(graphs) == 0 {
		return []string{"(no graphs found)"}
	}
	rows := make([]string, 0, len(graphs))
	for _, g := range graphs {
		updated := "unknown"
		if g.UpdatedAt != nil {
			updated = support.FormatTimeValue(*g.UpdatedAt)
		}
		rows = append(rows, fmt.Sprintf("%s (%s) | type=%s | updated=%s", g.Name, support.ShortID(g.ID), g.Type, updated))
	}
	return rows
}
